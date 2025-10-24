package main

import (
	"os"
	"path/filepath"
	"strings"

	project_settings "github.com/atframework/atsf4g-go/tools/project-settings"
)

func deleteGoFiles(dir string) error {
	// 使用 filepath.Walk 递归遍历文件夹
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 如果是文件且以 .pb.go 结尾，删除文件
		if !info.IsDir() && strings.HasSuffix(path, ".pb.go") {
			return os.Remove(path)
		} else if !info.IsDir() && strings.HasSuffix(path, ".go") && strings.Contains(path, "generate_config") {
			return os.Remove(path)
		}
		return nil
	})
}

func main() {
	if len(os.Args) > 1 && os.Args[1] != "" {
		for _, path := range os.Args[1:] {
			deleteGoFiles(path)
		}
	} else {
		deleteGoFiles(filepath.Join(project_settings.GetProjectSourceDir(), "component", "config", "generate_config"))
		deleteGoFiles(filepath.Join(project_settings.GetProjectSourceDir(), "component", "protocol", "private", "common"))
		deleteGoFiles(filepath.Join(project_settings.GetProjectSourceDir(), "component", "protocol", "private", "config"))
		deleteGoFiles(filepath.Join(project_settings.GetProjectSourceDir(), "component", "protocol", "private", "extension"))
		deleteGoFiles(filepath.Join(project_settings.GetProjectSourceDir(), "component", "protocol", "private", "pbdesc"))

		deleteGoFiles(filepath.Join(project_settings.GetProjectSourceDir(), "component", "protocol", "public", "common"))
		deleteGoFiles(filepath.Join(project_settings.GetProjectSourceDir(), "component", "protocol", "public", "config"))
		deleteGoFiles(filepath.Join(project_settings.GetProjectSourceDir(), "component", "protocol", "public", "extension"))
		deleteGoFiles(filepath.Join(project_settings.GetProjectSourceDir(), "component", "protocol", "public", "pbdesc"))

		deleteGoFiles(filepath.Join(project_settings.GetProjectSourceDir(), "lobbysvr", "protocol"))
	}
}
