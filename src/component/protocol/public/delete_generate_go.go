package main

import (
	"os"
	"path/filepath"
)

func deleteGoFiles(dir string) error {
	// 使用 filepath.Walk 递归遍历文件夹
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 如果是文件且以 .go 结尾，删除文件
		if !info.IsDir() && filepath.Ext(path) == ".go" {
			return os.Remove(path)
		}
		return nil
	})
}


func main() {
	deleteGoFiles("common")
	deleteGoFiles("config")
	deleteGoFiles("pbdesc")
}
