package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"

	atframe_utils "github.com/atframework/atframe-utils-go"
)

func main() {
	scanDir := "../../"
	if len(os.Args) > 1 && os.Args[1] != "" {
		scanDir = os.Args[1]
	}

	var protocBin string
	if _, filename, _, ok := runtime.Caller(1); ok {
		protocBin = atframe_utils.EnsureProtocExecutable(path.Join(filepath.Dir(filename), "bin"))
	} else {
		protocBin = atframe_utils.EnsureProtocExecutable(path.Join("..", "bin"))
	}
	// 将protocBin的上级目录加入PATH
	binDir := filepath.Dir(protocBin)
	os.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	pendingGoTidy := make(map[string]bool)

	// 扫描所有 generate.atfw.go 文件
	var matches []string
	err := filepath.WalkDir(scanDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "generate.atfw.go" {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil || len(matches) == 0 {
		fmt.Fprintf(os.Stderr, "Scan generate.atfw.go failed: %v\n", err)
		os.Exit(1)
	}
	for _, file := range matches {
		// 执行 go generate
		if err := runGoGenerate(file); err != nil {
			fmt.Fprintf(os.Stderr, "Run go generate failed: %v\n", err)
			os.Exit(2)
		} else {
			goModDir := findFirstGoMod(filepath.Dir(file))
			if goModDir != "" {
				pendingGoTidy[goModDir] = true
			}
		}
	}

	for dir := range pendingGoTidy {
		if err := runGoTidy(dir); err != nil {
			fmt.Fprintf(os.Stderr, "Run go mod tidy failed: %v\n", err)
			os.Exit(4)
		}
	}
}

func runGoGenerate(target string) error {
	// 使用os/exec执行 go generate
	cmd := exec.Command("go", "generate", filepath.Base(target))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = filepath.Dir(target)

	fmt.Printf("Run go generate %s on %s\n", filepath.Base(target), cmd.Dir)
	return cmd.Run()
}

func findFirstGoMod(baseDir string) string {
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

func runGoTidy(baseDir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = baseDir

	fmt.Printf("Run go mod tidy on %s\n", baseDir)
	return cmd.Run()
}
