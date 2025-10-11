package atframework_tools_project_settings

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func binName(name string) string {
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(name), ".exe") {
		return name + ".exe"
	}

	return name
}

func ServiceBinName(name string) string {
	if !strings.HasSuffix(name, "d") {
		name = name + "d"
	}

	return binName(name)
}

func RmDirFile(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(path, entry.Name())
		err := os.RemoveAll(path) // 删除文件或子目录
		if err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}
	return nil
}

func CopyFile(src, dest string) error {
	// 打开源文件
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// 创建目标文件
	destinationFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destinationFile.Close()

	// 使用 io.Copy 拷贝文件内容
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

func CopyDir(srcDir, dstDir string) error {
	// 获取源目录中的文件信息
	files, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	// 创建目标目录
	err = os.MkdirAll(dstDir, os.ModePerm)
	if err != nil {
		return err
	}

	// 遍历源目录中的文件
	for _, file := range files {
		srcPath := filepath.Join(srcDir, file.Name())
		dstPath := filepath.Join(dstDir, file.Name())

		// 如果是目录，递归调用 CopyDir
		if file.IsDir() {
			err := CopyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			// 如果是文件，调用 CopyFile
			err := CopyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func FindFirstGoMod(baseDir string) string {
	previousDir := baseDir + "_"
	for i := 0; previousDir != baseDir && previousDir != ""; i++ {
		if _, err := os.Stat(filepath.Join(baseDir, "go.mod")); err == nil {
			return baseDir
		}

		previousDir = baseDir
		baseDir = filepath.Dir(baseDir)
	}

	return ""
}

func RunGoModTidy(scanDir string) {
	pendingGoTidy := make(map[string]bool)

	filepath.WalkDir(scanDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		goModDir := FindFirstGoMod(filepath.Dir(path))
		if goModDir != "" {
			pendingGoTidy[goModDir] = true
		}
		return nil
	})

	for dir := range pendingGoTidy {
		if err := RunGoTidy(dir); err != nil {
			fmt.Fprintf(os.Stderr, "Run go mod tidy failed: %v\n", err)
			os.Exit(4)
		}
	}
}

func RunGoTidy(baseDir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = baseDir

	fmt.Printf("Run go mod tidy on %s\n", baseDir)
	return cmd.Run()
}

func fmtColorInner(color int, str string) {
	fmt.Printf("\033[1;%s;40m%s\033[0m\n", strconv.Itoa(color), str)
}

func FmtColor(color int, format string, a ...any) {
	fmtColorInner(color, fmt.Sprintf(format, a...))
}

func FmtColorRed(format string, a ...any) {
	fmtColorInner(FgRed, fmt.Sprintf(format, a...))
}

func FmtColorGreen(format string, a ...any) {
	fmtColorInner(FgGreen, fmt.Sprintf(format, a...))
}

func FmtColorCyan(format string, a ...any) {
	fmtColorInner(FgCyan, fmt.Sprintf(format, a...))
}

const (
	FgBlack int = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)
