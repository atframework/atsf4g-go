package libatapp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type logStdoutWriter struct {
	io.Writer
}

func (w *logStdoutWriter) Close() {
}

func (w *logStdoutWriter) Flush() error {
	return nil
}

func NewlogStdoutWriter() *logStdoutWriter {
	return &logStdoutWriter{os.Stdout}
}

type logStderrWriter struct {
	io.Writer
}

func (w *logStderrWriter) Close() {
}

func (w *logStderrWriter) Flush() error {
	return nil
}

func NewlogStderrWriter() *logStderrWriter {
	return &logStderrWriter{os.Stdout}
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

// 需要外部保证穿行 slog即可
type logBufferedRotatingWriter struct {
	path    string
	maxSize uint64
	retain  uint32

	currentFileIndex uint32
	currentSize      uint64
	init             bool

	flushInterval time.Duration
	nextFlushTime time.Time

	// 对于FD的读写都需要加锁
	currentFile *RefFD
	fileMu      sync.RWMutex
}

// NewlogBufferedRotatingWriter 创建新的日志 writer
func NewlogBufferedRotatingWriter(path string, maxSize uint64, retain uint32, flushInterval time.Duration) (*logBufferedRotatingWriter, error) {
	w := &logBufferedRotatingWriter{
		path:          path,
		maxSize:       maxSize,
		retain:        retain,
		flushInterval: flushInterval,
	}
	return w, nil
}

func (w *logBufferedRotatingWriter) openLogFile(destoryContent bool) (*RefFD, error) {
	// 读锁
	w.fileMu.RLock()
	if w.currentFile != nil {
		defer w.fileMu.RUnlock()
		return w.currentFile.Copy(), nil
	}
	w.fileMu.RUnlock()

	// 创建流程
	w.fileMu.Lock()
	defer w.fileMu.Unlock()
	if w.currentFile != nil {
		// 防止多次创建
		return w.currentFile.Copy(), nil
	}

	if !w.init {
		// 第一次创建 不覆盖
		// 找到第一个可以写入的文件
		w.init = true
		var index uint32
		for index = 0; index < w.retain; index++ {
			path := w.getFilename(index)
			info, err := os.Stat(path)
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
		destoryContent = false
	}

	newFile := w.getFilename(w.currentFileIndex)
	dir := filepath.Dir(newFile)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}

	var f *os.File
	if destoryContent {
		f, err = os.OpenFile(newFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return nil, err
		}
	} else {
		f, err = os.OpenFile(newFile, os.O_CREATE|os.O_WRONLY, 0644)
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
	os.Remove(w.path)
	err = os.Link(newFile, w.path)
	if err != nil {
		f.Close()
		return nil, err
	}

	// 创建好文件
	ref := &RefFD{}
	ref.fd = f
	w.currentFile = ref.Copy()
	w.currentSize = uint64(info.Size())

	return ref.Copy(), nil
}

func (w *logBufferedRotatingWriter) getFilename(index uint32) string {
	return fmt.Sprintf("%s.%d", w.path, index)
}

func (w *logBufferedRotatingWriter) rotateFile() error {
	// 仅用于对比是否需要再次 rotate 防止多次进入
	currentFile := w.currentFile
	// 写锁
	w.fileMu.Lock()
	defer w.fileMu.Unlock()
	if currentFile != w.currentFile || w.currentSize < w.maxSize {
		// 已经有人替换了
		return nil
	}

	// 寻找新Index
	w.currentFileIndex++
	if w.currentFileIndex >= w.retain {
		w.currentFileIndex = 0
	}
	if w.currentFile != nil {
		w.currentFile.fd.Write([]byte("Open Next Log File"))
		w.currentFile.Relese()
		w.currentFile = nil
	}
	w.currentSize = 0
	return nil
}

func (w *logBufferedRotatingWriter) mayRotateFile() {
	if w.currentSize >= w.maxSize {
		w.rotateFile()
	}
}

func (w *logBufferedRotatingWriter) Write(p []byte) (int, error) {
	w.mayRotateFile()
	// 这里可能被滚动过
	// 或者第一次进入
	f, err := w.openLogFile(true)
	defer f.Relese()
	if err != nil {
		return 0, err
	}
	// 模拟智能指针手动释放

	n, err := f.fd.Write(p)
	w.currentSize += uint64(n)

	now := time.Now()
	if now.After(w.nextFlushTime) {
		f.fd.Sync()
		w.updateFlushTime(now)
	}
	return n, err
}

func (w *logBufferedRotatingWriter) updateFlushTime(now time.Time) {
	w.nextFlushTime = now.Add(w.flushInterval)
}

// Flush 手动刷新缓冲区
func (w *logBufferedRotatingWriter) Flush() error {
	f, err := w.openLogFile(true)
	if err != nil {
		return err
	}
	defer f.Relese()

	f.fd.Sync()
	w.updateFlushTime(time.Now())
	return nil
}

// 关闭打开的文件
func (w *logBufferedRotatingWriter) Close() {
	w.fileMu.Lock()
	defer w.fileMu.Unlock()
	w.currentFile.Relese()
	w.currentFile = nil
	w.currentSize = 0
}
