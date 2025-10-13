package libatapp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type logStdoutWriter struct {
	io.Writer
}

func (w *logStdoutWriter) Close() error {
	return nil
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

func (w *logStderrWriter) Close() error {
	return nil
}

func (w *logStderrWriter) Flush() error {
	return nil
}

func NewlogStderrWriter() *logStderrWriter {
	return &logStderrWriter{os.Stdout}
}

// 需要外部保证穿行 slog即可
type logBufferedRotatingWriter struct {
	path    string
	maxSize uint64
	retain  uint32

	currentFile *os.File
	fileMu      sync.Mutex

	currentSize uint64

	flushInterval time.Duration
	lastFlushTime time.Duration
}

// NewlogBufferedRotatingWriter 创建新的日志 writer
func NewlogBufferedRotatingWriter(path string, maxSize uint64, retain uint32, flushInterval time.Duration) (*logBufferedRotatingWriter, error) {
	w := &logBufferedRotatingWriter{
		path:          path,
		maxSize:       maxSize,
		retain:        retain,
		flushInterval: flushInterval,
		lastFlushTime: 0,
	}

	if _, err := w.openLogFile(); err != nil {
		return nil, err
	}
	return w, nil
}

// getFilename 根据日期和序号生成文件名
func (w *logBufferedRotatingWriter) getFilename(index uint32) string {
	return fmt.Sprintf("%s.%d", w.path, index)
}

// rotateFile 执行日志文件滚动
func (w *logBufferedRotatingWriter) rotateFile() error {
	w.fileMu.Lock()
	defer w.fileMu.Unlock()

	{
		// 关闭旧文件
		currentFile := w.currentFile
		w.currentFile = nil
		w.currentSize = 0

		if currentFile != nil {
			currentFile.Close()
		}
	}

	// 从后往前重命名（.N → .N+1）
	for i := w.retain - 1; ; i-- {
		oldPath := w.getFilename(i)
		if _, err := os.Stat(oldPath); err == nil {
			if i == w.retain-1 {
				os.Remove(oldPath)
			} else {
				os.Rename(oldPath, w.getFilename(i+1))
			}
		}
		if i == 0 {
			break
		}
	}

	newFile := w.getFilename(0)
	dir := filepath.Dir(newFile)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(newFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	info, err := os.Stat(newFile)
	if err != nil {
		return err
	}

	w.currentSize = uint64(info.Size())
	w.currentFile = f
	return nil
}

func (w *logBufferedRotatingWriter) mayRotateFile() {
	if w.currentSize > w.maxSize {
		w.rotateFile()
	}
}

func (w *logBufferedRotatingWriter) openLogFile() (*os.File, error) {
	if w.currentFile != nil {
		return w.currentFile, nil
	}

	// 对文件进行操作
	w.fileMu.Lock()
	defer w.fileMu.Unlock()

	w.currentSize = 0

	newFile := w.getFilename(0)
	dir := filepath.Dir(newFile)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(newFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(newFile)
	if err != nil {
		return nil, err
	}

	w.currentSize = uint64(info.Size())
	w.currentFile = f

	return w.currentFile, err
}

func (w *logBufferedRotatingWriter) Write(p []byte) (int, error) {
	w.mayRotateFile()
	f, err := w.openLogFile()
	if err != nil {
		return 0, err
	}

	n, err := f.Write(p)
	w.currentSize += uint64(n)
	return n, err
}

// Flush 手动刷新缓冲区
func (w *logBufferedRotatingWriter) Flush() error {
	currentFile := w.currentFile
	if currentFile != nil {
		return currentFile.Sync()
	}
	return nil
}

func (w *logBufferedRotatingWriter) Close() error {
	return nil
}
