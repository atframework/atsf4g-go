package libatapp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
)

var endLine = []byte("\u001e\n")

type LogStdoutWriter struct {
	writer io.Writer
}

func (w *LogStdoutWriter) Write(p []byte) (n int, err error) {
	n, err = w.writer.Write(p)
	w.writer.Write(endLine)
	return
}

func (w *LogStdoutWriter) Close() {
}

func (w *LogStdoutWriter) Flush() error {
	return nil
}

func NewlogStdoutWriter() *LogStdoutWriter {
	return &LogStdoutWriter{os.Stdout}
}

type LogStderrWriter struct {
	writer io.Writer
}

func (w *LogStderrWriter) Write(p []byte) (n int, err error) {
	n, err = w.writer.Write(p)
	w.writer.Write(endLine)
	return
}

func (w *LogStderrWriter) Close() {
}

func (w *LogStderrWriter) Flush() error {
	return nil
}

func NewlogStderrWriter() *LogStderrWriter {
	return &LogStderrWriter{os.Stdout}
}

const defaultRingBufferSlots = 512

// 触发立即刷新的阈值（缓冲区使用率）
const flushThresholdRatio = 0.35

type LogRingBufferSlot struct {
	data  *logBuffer
	ready atomic.Bool // 标记数据是否已写入完成
}

type LogRingBuffer struct {
	slots        []LogRingBufferSlot
	slotCount    uint64
	writePos     atomic.Uint64 // 下一个写入位置
	readPos      atomic.Uint64 // 下一个读取位置
	droppedCount atomic.Uint64 // 丢失的日志条数
	buffPool     sync.Pool
}

func NewLogRingBuffer(slotCount uint64) *LogRingBuffer {
	if slotCount == 0 {
		slotCount = defaultRingBufferSlots
	}

	rb := &LogRingBuffer{
		slots:     make([]LogRingBufferSlot, slotCount),
		slotCount: slotCount,
		buffPool: sync.Pool{
			New: func() any {
				b := make([]byte, 0, 2048)
				return (*logBuffer)(&b)
			},
		},
	}

	return rb
}

// Write 写入数据到环形缓冲区（无锁）
// 返回写入的字节数和是否成功
func (rb *LogRingBuffer) Write(p []byte) (int, bool) {
	for {
		writePos := rb.writePos.Load()
		readPos := rb.readPos.Load()

		// 检查缓冲区是否已满
		// 保留一个槽位用于区分满和空的状态
		nextWritePos := (writePos + 1) % rb.slotCount
		if nextWritePos == readPos%rb.slotCount {
			// 缓冲区已满，丢弃数据
			rb.droppedCount.Add(1)
			return 0, false
		}

		// 尝试获取写入槽位
		if rb.writePos.CompareAndSwap(writePos, writePos+1) {
			slotIndex := writePos % rb.slotCount
			slot := &rb.slots[slotIndex]

			// 写入数据
			dataLen := len(p)
			slot.data = rb.buffPool.Get().(*logBuffer)

			slot.data.Write(p[:dataLen])

			// 标记数据已就绪
			slot.ready.Store(true)

			return dataLen, true
		}
		// CAS 失败，重试
	}
}

// ReadAll 读取所有可用数据（仅由单个消费者协程调用，无需CAS）
// 返回数据切片和丢失的日志数量
func (rb *LogRingBuffer) ReadAll() ([]*logBuffer, uint64) {
	var result []*logBuffer

	for {
		readPos := rb.readPos.Load()
		writePos := rb.writePos.Load()

		// 检查是否有数据可读
		if readPos >= writePos {
			break
		}

		slotIndex := readPos % rb.slotCount
		slot := &rb.slots[slotIndex]

		// 检查数据是否已就绪
		if !slot.ready.Load() {
			// 数据还未就绪，等待
			break
		}

		// 直接移动读取位置（单消费者无需CAS）
		rb.readPos.Store(readPos + 1)
		result = append(result, slot.data)
		// 重置就绪标记
		slot.ready.Store(false)
	}

	// 获取并重置丢失计数
	dropped := rb.droppedCount.Swap(0)

	return result, dropped
}

// PendingCount 返回当前待处理的槽位数量
func (rb *LogRingBuffer) PendingCount() uint64 {
	writePos := rb.writePos.Load()
	readPos := rb.readPos.Load()
	if writePos >= readPos {
		return writePos - readPos
	}
	return 0
}

// FlushThreshold 返回触发立即刷新的阈值
func (rb *LogRingBuffer) FlushThreshold() uint64 {
	return uint64(float64(rb.slotCount) * flushThresholdRatio)
}

type GetTime interface {
	GetSysNow() time.Time
}

type DefaultGetTime struct{}

func (d *DefaultGetTime) GetSysNow() time.Time {
	return time.Now()
}

type LogBufferedRotatingWriter struct {
	GetTime
	fileNameFormat  string
	fileAliasFormat string
	maxSize         uint64
	retain          uint32

	// 以下字段仅由后台协程访问，无需加锁
	currentFileName    string
	currentFileIndex   uint32
	currentSize        uint64
	currentFile        *os.File
	firstFile          bool
	needTruncateOnOpen bool

	flushInterval time.Duration

	timeRotateCheckInterval int64
	lastCheckRotateTime     int64
	currentTimeRotateTime   time.Time

	// 环形缓冲区（多生产者写入，单消费者读取）
	ringBuffer *LogRingBuffer

	// 后台刷新 goroutine 控制
	stopCh   chan struct{}
	stoppedC chan struct{}
	flushCh  chan struct{} // 用于手动触发刷新
	started  atomic.Bool
}

var intervalLut = [128]int64{}

const (
	secondsPerMinute = int64(time.Minute / time.Second)
	secondsPerHour   = int64(time.Hour / time.Second)
)

// NewLogBufferedRotatingWriter 创建新的日志 writer
func NewLogBufferedRotatingWriter(getTime GetTime, fileName string, fileAlias string, maxSize uint64, retain uint32,
	flushInterval time.Duration, bufferSlotSize uint64) (*LogBufferedRotatingWriter, error) {
	if lu.IsNil(getTime) {
		getTime = &DefaultGetTime{}
	}
	w := &LogBufferedRotatingWriter{
		GetTime:         getTime,
		fileNameFormat:  fileName,
		fileAliasFormat: fileAlias,
		maxSize:         maxSize,
		retain:          retain,
		flushInterval:   flushInterval,
		ringBuffer:      NewLogRingBuffer(bufferSlotSize),
		stopCh:          make(chan struct{}),
		stoppedC:        make(chan struct{}),
		flushCh:         make(chan struct{}, 1), // 缓冲通道，避免阻塞
	}

	if intervalLut['S'] == 0 {
		intervalLut['f'] = 1
		intervalLut['R'] = secondsPerMinute
		intervalLut['T'] = 1
		intervalLut['F'] = secondsPerHour
		intervalLut['S'] = 1
		intervalLut['M'] = secondsPerMinute
		intervalLut['I'] = secondsPerHour
		intervalLut['H'] = secondsPerHour
		intervalLut['w'] = secondsPerHour
		intervalLut['d'] = secondsPerHour
		intervalLut['j'] = secondsPerHour
		intervalLut['m'] = secondsPerHour
		intervalLut['y'] = secondsPerHour
		intervalLut['Y'] = secondsPerHour
	}

	for i := 0; i+1 < len(fileName); i++ {
		if fileName[i] == '%' {
			checked := fileName[i+1]
			if checked > 0 && checked < 128 {
				if v := intervalLut[checked]; v > 0 && (w.timeRotateCheckInterval == 0 || v < w.timeRotateCheckInterval) {
					w.timeRotateCheckInterval = v
				}
			}
		}
	}

	// 启动后台刷新 goroutine
	w.startFlushRoutine()

	return w, nil
}

// startFlushRoutine 启动后台刷新协程
func (w *LogBufferedRotatingWriter) startFlushRoutine() {
	if w.started.Swap(true) {
		return
	}

	go func() {
		defer close(w.stoppedC)

		ticker := time.NewTicker(w.flushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-w.stopCh:
				// 退出前刷新剩余数据
				w.flushRingBuffer()
				w.closeCurrentFile()
				return
			case <-ticker.C:
				w.flushRingBuffer()
			case <-w.flushCh:
				w.flushRingBuffer()
			}
		}
	}()
}

// flushRingBuffer 将环形缓冲区的数据刷新到文件（仅由后台协程调用）
func (w *LogBufferedRotatingWriter) flushRingBuffer() {
	data, dropped := w.ringBuffer.ReadAll()
	if len(data) == 0 {
		return
	}

	// 检查是否需要轮转文件
	w.mayRotateFile()

	// 确保文件已打开
	if err := w.ensureFileOpen(); err != nil {
		fmt.Println("open File Failed", err)
		for _, d := range data {
			d.Free()
		}
		return
	}

	// 如果有丢失的日志，在最后一条日志内容后添加警告信息
	if dropped > 0 {
		data[len(data)-1].WriteString(fmt.Sprintf("[ERROR][LOG] DROPPED %d LOGS DUE TO BUFFER OVERFLOW!!!", dropped))
	}

	// 写入缓冲区中的数据
	for _, d := range data {
		n, _ := w.currentFile.Write(*d)
		w.currentSize += uint64(n)
		n, _ = w.currentFile.Write(endLine)
		w.currentSize += uint64(n)
		// 写完后归还buffer到池
		d.Free()
	}

	// 同步到磁盘
	w.currentFile.Sync()
}

// ensureFileOpen 确保当前文件已打开（仅由后台协程调用）
func (w *LogBufferedRotatingWriter) ensureFileOpen() error {
	if w.currentFile != nil {
		return nil
	}

	now := w.GetSysNow()

	if !w.firstFile {
		// 第一次创建，不覆盖
		// 找到第一个可以写入的文件
		w.firstFile = true
		var index uint32
		for index = 0; index < w.retain; index++ {
			fileName := w.getFilename(index, now)
			info, err := os.Stat(fileName)
			if err != nil {
				break
			}

			if uint64(info.Size()) < w.maxSize {
				break
			}
		}
		if index >= w.retain {
			index = 0
		}
		// 修正Index
		w.currentFileIndex = index
		w.needTruncateOnOpen = false
	}

	newFile := w.getFilename(w.currentFileIndex, now)
	dir := filepath.Dir(newFile)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	var f *os.File
	if w.needTruncateOnOpen {
		f, err = os.OpenFile(newFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
	} else {
		f, err = os.OpenFile(newFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
	}

	info, err := os.Stat(newFile)
	if err != nil {
		f.Close()
		return err
	}

	// 创建硬链接
	linkFileName := w.getLinkFilename(now)
	if linkFileName != "" {
		os.Remove(linkFileName)
		os.Link(newFile, linkFileName)
	}

	// 设置当前文件
	w.currentFile = f
	w.currentFileName = newFile
	w.currentSize = uint64(info.Size())
	w.needTruncateOnOpen = false
	w.currentTimeRotateTime = now

	return nil
}

// closeCurrentFile 关闭当前文件（仅由后台协程调用）
func (w *LogBufferedRotatingWriter) closeCurrentFile() {
	if w.currentFile != nil {
		w.currentFile.Sync()
		w.currentFile.Close()
		w.currentFile = nil
	}
	w.currentSize = 0
}

func (w *LogBufferedRotatingWriter) getFilename(index uint32, now time.Time) string {
	return LogFormat(w.fileNameFormat, &strings.Builder{}, CallerInfo{
		Now:         now,
		RotateIndex: index,
	}, nil)
}

func (w *LogBufferedRotatingWriter) getLinkFilename(now time.Time) string {
	return LogFormat(w.fileAliasFormat, &strings.Builder{}, CallerInfo{
		Now: now,
	}, nil)
}

// rotateFile 执行文件轮转（仅由后台协程调用）
func (w *LogBufferedRotatingWriter) rotateFile() {
	// 关闭当前文件
	w.closeCurrentFile()

	// 寻找新Index
	w.currentFileIndex++
	if w.currentFileIndex >= w.retain {
		w.currentFileIndex = 0
	}
	w.needTruncateOnOpen = true
}

func (w *LogBufferedRotatingWriter) mayRotateFile() {
	if w.needRotateFile() {
		w.rotateFile()
	}
}

func (w *LogBufferedRotatingWriter) needRotateFile() bool {
	if w.currentSize >= w.maxSize {
		return true
	}
	if w.currentFileName == "" {
		return false
	}
	if w.timeRotateCheckInterval != 0 {
		now := w.GetSysNow()
		if now.Unix()/int64(w.timeRotateCheckInterval) != w.lastCheckRotateTime/int64(w.timeRotateCheckInterval) {
			w.lastCheckRotateTime = now.Unix()
			if w.currentFileName != w.getFilename(w.currentFileIndex, now) {
				// 文件名变化 Rotating
				return true
			}
		}
	}
	return false
}

// triggerFlush 非阻塞地触发刷新
func (w *LogBufferedRotatingWriter) triggerFlush() {
	select {
	case w.flushCh <- struct{}{}:
	default:
		// 通道已满，说明已有刷新请求等待处理
	}
}

func (w *LogBufferedRotatingWriter) Write(p []byte) (int, error) {
	// 写入到环形缓冲区（无锁）
	n, ok := w.ringBuffer.Write(p)
	if !ok {
		// 缓冲区满，数据被丢弃（droppedCount 已在 ringBuffer.Write 中增加）
		return 0, nil
	}

	// 检查缓冲区占用，超过阈值则触发刷新
	if w.ringBuffer.PendingCount() >= w.ringBuffer.FlushThreshold() {
		w.triggerFlush()
	}

	return n, nil
}

// Flush 手动触发刷新缓冲区（非阻塞）
func (w *LogBufferedRotatingWriter) Flush() error {
	if !w.started.Load() {
		return nil
	}
	w.triggerFlush()
	return nil
}

// Close 关闭打开的文件
func (w *LogBufferedRotatingWriter) Close() {
	// 停止后台刷新协程
	if w.started.Load() {
		close(w.stopCh)
	}
}
