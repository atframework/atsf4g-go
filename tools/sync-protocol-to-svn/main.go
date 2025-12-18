// Copyright 2025 atframework
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type mvPath struct {
	src string
	dst string
}

type ProjectEnv struct {
	ClientProtocolPath string `json:"client.protocol-path"`
}

// rmDirFile 删除目录中的所有文件.
func rmDirFile(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 目录不存在，不报错
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			filePath := filepath.Join(dir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFile 复制文件.
func copyFile(src, dst string) error {
	// 确保目标目录存在
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func loadProjectEnv(envFilePath string) (string, error) {
	data, err := os.ReadFile(envFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read project_env.json: %w", err)
	}

	var env ProjectEnv
	if err := json.Unmarshal(data, &env); err != nil {
		return "", fmt.Errorf("failed to parse project_env.json: %w", err)
	}

	if env.ClientProtocolPath == "" {
		return "", fmt.Errorf("client.protocol-path is empty in project_env.json")
	}

	return env.ClientProtocolPath, nil
}

func main() { // nolint:gocognit
	// 定义命令行参数
	envFile := flag.String("env", "", "Path to project_env.json file (required)")
	projectRoot := flag.String("root", "", "Project root directory (required)")
	flag.Parse()

	// 验证必需参数
	if *envFile == "" || *projectRoot == "" {
		fmt.Println("Error: -env and -root parameters are required")
		fmt.Println("Usage: sync-protocol-to-svn -env <path-to-project_env.json> -root <project-root-dir>")
		fmt.Println("Example: sync-protocol-to-svn -env .vscode/project_env.json -root /path/to/project")
		os.Exit(1)
	}

	rootDir := *projectRoot
	// 转换为绝对路径
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		os.Exit(1)
	}
	rootDir = absRootDir
	fmt.Printf("Using project root: %s\n", rootDir)

	// 同步GIT仓库内的修改到本地的SVN仓库 方便SVN提交
	svnPath, err := loadProjectEnv(*envFile)
	if err != nil {
		fmt.Printf("Error loading client protocol path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Using client protocol path: %s\n", svnPath)

	gitProtocolList := []mvPath{
		{
			src: filepath.Join(rootDir, "/src/component/protocol/public/common/protocol/common"),
			dst: filepath.Join(svnPath, "common"),
		},
		{
			src: filepath.Join(rootDir, "/src/component/protocol/public/config/protocol/config"),
			dst: filepath.Join(svnPath, "config"),
		},
		{
			src: filepath.Join(rootDir, "/src/component/protocol/public/pbdesc/protocol/pbdesc"),
			dst: filepath.Join(svnPath, "pbdesc"),
		},
		{
			src: filepath.Join(rootDir, "/src/lobbysvr/protocol/public/protocol/pbdesc"),
			dst: filepath.Join(svnPath, "pbdesc"),
		},
	}

	// 删除所有文件
	for _, v := range gitProtocolList {
		if err := rmDirFile(v.dst); err != nil {
			fmt.Printf("Warning: failed to clean directory %s: %v\n", v.dst, err)
		}
	}

	// 拷贝所有文件
	for _, v := range gitProtocolList {
		var entries []os.DirEntry
		entries, err = os.ReadDir(v.src)
		if err != nil {
			break
		}

		var files []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".proto") {
				files = append(files, filepath.Join(v.src, e.Name()))
			}
		}

		for _, file := range files {
			fmt.Printf("Copy file %s to %s \n", file, filepath.Join(v.dst, filepath.Base(file)))
			err = copyFile(file, filepath.Join(v.dst, filepath.Base(file)))
			if err != nil {
				break
			}
		}
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
