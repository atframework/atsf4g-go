package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: lint-runner <work-dir> <report-dir> <lint-dir1> [lint-dir2] ...")
		os.Exit(1)
	}

	workDir := os.Args[1]
	reportDir := os.Args[2]
	lintDirs := os.Args[3:]

	// 标准化路径
	workDir = filepath.Clean(workDir)
	reportDir = filepath.Clean(reportDir)

	// 验证工作目录是否存在
	if info, err := os.Stat(workDir); err != nil || !info.IsDir() {
		fmt.Printf("Error: work directory does not exist or is not accessible: %s\n", workDir)
		os.Exit(1)
	}

	// 改变工作目录
	if err := os.Chdir(workDir); err != nil {
		fmt.Printf("Failed to change to work directory: %v\n", err)
		fmt.Printf("Attempted path: %s\n", workDir)
		os.Exit(1)
	}

	// 创建报告目录
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		fmt.Printf("Failed to create report directory: %v\n", err)
		os.Exit(1)
	}

	// 清理旧报告
	cleanReportDir(reportDir)

	reportFile := filepath.Join(reportDir, "golangci-lint-report.txt")
	reportF, err := os.Create(reportFile)
	if err != nil {
		fmt.Printf("Failed to create report file: %v\n", err)
		os.Exit(1)
	}
	defer reportF.Close()

	// 写入报告头
	fmt.Fprintf(reportF, "golangci-lint Report - %s\n", time.Now().Format(time.RFC1123))
	fmt.Fprintf(reportF, "============================================\n")

	// 遍历每个目录运行 lint
	for _, dir := range lintDirs {
		// 检查 go.mod 是否存在
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err != nil {
			continue
		}

		projectName := filepath.Base(dir)
		moduleReportFile := filepath.Join(reportDir, projectName+"-lint-report.txt")

		// 运行 golangci-lint
		cmd := exec.Command("golangci-lint", "run", "./...")
		cmd.Dir = dir

		output, _ := cmd.CombinedOutput()

		// 写入单个模块报告
		if moduleF, err := os.Create(moduleReportFile); err == nil {
			fmt.Fprintf(moduleF, "================================================\n")
			fmt.Fprintf(moduleF, "Project: %s\n", projectName)
			fmt.Fprintf(moduleF, "Path: %s\n", dir)
			fmt.Fprintf(moduleF, "Time: %s\n", time.Now().Format(time.RFC1123))
			fmt.Fprintf(moduleF, "================================================\n")
			fmt.Fprintf(moduleF, "\n")
			moduleF.Write(output)
			moduleF.Close()

			// 写入主报告
			fmt.Fprintf(reportF, "\n### %s (%s)\n", projectName, dir)
			fmt.Fprintf(reportF, "---\n")

			// 跳过头 5 行，写入其余内容
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				if lineNum > 5 {
					fmt.Fprintf(reportF, "%s\n", scanner.Text())
				}
			}
		}
	}

	fmt.Println("✅ golangci-lint completed")
	fmt.Printf("📄 Report: %s\n", reportFile)
}

func cleanReportDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			os.RemoveAll(path)
		} else {
			os.Remove(path)
		}
	}
}

func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return dir
}
