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

type noCopy struct{}

type RefFD struct {
	_      noCopy
	fd     *os.File
	refCnt atomic.Int32
}

func (f *RefFD) Copy() *RefFD {
	f.refCnt.Add(1)
	return f
}
func (f *RefFD) Relese() {
	value := f.refCnt.Add(-1)
	if value == 0 {
		f.fd.Sync()
		f.fd.Close()
	}
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

	currentFileName    string
	currentFileIndex   uint32
	currentSize        atomic.Uint64
	firstFile          bool
	needTruncateOnOpen bool

	flushInterval time.Duration
	nextFlushTime time.Time

	timeRotateCheckInterval int64
	lastCheckRotateTime     int64
	currentTimeRotateTime   time.Time

	// 对于FD的读写都需要加锁
	currentFile atomic.Pointer[RefFD]
	fileMu      sync.RWMutex
}

var intervalLut = [128]int64{}

const (
	secondsPerMinute = int64(time.Minute / time.Second)
	secondsPerHour   = int64(time.Hour / time.Second)
)

// NewLogBufferedRotatingWriter 创建新的日志 writer
func NewLogBufferedRotatingWriter(getTime GetTime, fileName string, fileAlias string, maxSize uint64, retain uint32, flushInterval time.Duration) (*LogBufferedRotatingWriter, error) {
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
	return w, nil
}

func (w *LogBufferedRotatingWriter) openLogFile(truncate bool) (*RefFD, error) {
	// 读锁
	w.fileMu.RLock()
	if w.currentFile.Load() != nil {
		defer w.fileMu.RUnlock()
		return w.currentFile.Load().Copy(), nil
	}
	w.fileMu.RUnlock()

	// 创建流程
	w.fileMu.Lock()
	defer w.fileMu.Unlock()
	if w.currentFile.Load() != nil {
		// 防止多次创建
		return w.currentFile.Load().Copy(), nil
	}

	now := w.GetSysNow()

	if !w.firstFile {
		// 第一次创建 不覆盖
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
		truncate = false
	}

	truncate = truncate || w.needTruncateOnOpen

	newFile := w.getFilename(w.currentFileIndex, now)
	dir := filepath.Dir(newFile)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}

	var f *os.File
	if truncate {
		f, err = os.OpenFile(newFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return nil, err
		}
	} else {
		f, err = os.OpenFile(newFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
	}
	info, err := os.Stat(newFile)
	if err != nil {
		f.Close()
		return nil, err
	}

	// 创建硬链接
	linkFileName := w.getLinkFilename(now)
	if linkFileName != "" {
		os.Remove(linkFileName)
		os.Link(newFile, linkFileName)
	}

	// 创建好文件
	ref := &RefFD{}
	ref.fd = f
	w.currentFileName = newFile
	w.currentFile.Store(ref.Copy())
	w.currentSize.Store(uint64(info.Size()))
	w.needTruncateOnOpen = false

	w.currentTimeRotateTime = now

	return ref.Copy(), nil
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

func (w *LogBufferedRotatingWriter) rotateFile() error {
	// 仅用于对比是否需要再次 rotate 防止多次进入
	currentFile := w.currentFile.Load()
	// 写锁
	w.fileMu.Lock()
	defer w.fileMu.Unlock()
	if currentFile != w.currentFile.Load() {
		// 已经有人替换了
		return nil
	}

	// 寻找新Index
	w.currentFileIndex++
	if w.currentFileIndex >= w.retain {
		w.currentFileIndex = 0
	}
	if w.currentFile.Load() != nil {
		w.currentFile.Load().Relese()
		w.currentFile.Store(nil)
	}
	w.currentSize.Store(0)
	w.needTruncateOnOpen = true
	return nil
}

func (w *LogBufferedRotatingWriter) mayRotateFile() {
	if w.needRotateFile() {
		w.rotateFile()
	}
}

func (w *LogBufferedRotatingWriter) needRotateFile() bool {
	if w.currentSize.Load() >= w.maxSize {
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

func (w *LogBufferedRotatingWriter) Write(p []byte) (int, error) {
	w.mayRotateFile()
	// 这里可能被滚动过
	// 或者第一次进入
	f, err := w.openLogFile(false)
	if err != nil {
		fmt.Println("open File Failed", err)
		return 0, err
	}
	defer f.Relese()
	// 模拟智能指针手动释放

	n, err := f.fd.Write(p)
	en, _ := f.fd.Write(endLine)
	w.currentSize.Add(uint64(n + en))

	now := w.GetSysNow()
	if now.After(w.nextFlushTime) {
		w.updateFlushTime(now)
		f.fd.Sync()
	}
	return n, err
}

func (w *LogBufferedRotatingWriter) updateFlushTime(now time.Time) {
	w.nextFlushTime = now.Add(w.flushInterval)
}

// Flush 手动刷新缓冲区
func (w *LogBufferedRotatingWriter) Flush() error {
	f, err := w.openLogFile(false)
	if err != nil {
		return err
	}
	defer f.Relese()

	w.updateFlushTime(w.GetSysNow())
	f.fd.Sync()
	return nil
}

// 关闭打开的文件
func (w *LogBufferedRotatingWriter) Close() {
	w.fileMu.Lock()
	defer w.fileMu.Unlock()
	if w.currentFile.Load() != nil {
		w.currentFile.Load().Relese()
		w.currentFile.Store(nil)
	}
	w.currentSize.Store(0)
}
