// Copyright 2025 atframework
package libatframe_utils_file_system

import (
	"io"
	"os"
)

func ReadAllContent(filePath string) ([]byte, error) {
	var fileSize int64
	if s, err := os.Stat(filePath); err != nil {
		return nil, err
	} else {
		fileSize = s.Size()
	}

	ret := make([]byte, fileSize)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	l, err := f.Read(ret)
	if err != nil && err != io.EOF {
		return nil, err
	}

	if l < int(fileSize) {
		return ret[0:l], os.ErrInvalid
	}

	return ret, nil
}
