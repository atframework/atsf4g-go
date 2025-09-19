package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	atframe_utils "github.com/atframework/atframe-utils-go"
)

func guessBinDir() string {
	if _, filename, _, ok := runtime.Caller(0); ok {
		if _, err := os.Stat(filename); err == nil {
			return filepath.Join(filepath.Dir(filepath.Dir(filename)), "bin")
		}
	}

	exePath, err := os.Executable()
	if err != nil {
		return exePath
	}

	cwdDir, _ := os.Getwd()
	baseDir := cwdDir
	previousDir := baseDir + "_"
	for i := 0; previousDir != baseDir && previousDir != ""; i++ {
		if _, err := os.Stat(filepath.Join(baseDir, "tools", "generate", "go.mod")); err == nil {
			return filepath.Join(baseDir, "tools", "bin")
		}

		previousDir = baseDir
		baseDir = filepath.Dir(baseDir)
	}

	return cwdDir
}

func main() {
	scanDirs := []string{"../../"}
	if len(os.Args) > 1 && os.Args[1] != "" {
		scanDirs = os.Args[1:]
	}

	toolsBinDir := guessBinDir()
	if toolsBinDir == "" {
		fmt.Fprintf(os.Stderr, "Cannot guess tools bin dir\n")
		os.Exit(1)
	}

	log.Println("Tools bin dir:", toolsBinDir)

	protocBin := atframe_utils.EnsureProtocExecutable(toolsBinDir)
	// 将protocBin的上级目录加入PATH
	binDir := filepath.Dir(protocBin)
	os.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	pendingGoTidy := make(map[string]bool)
	runCache := make(map[string]bool)

	// 扫描所有 generate.atfw.go 文件
	var matches []string
	for _, scanDir := range scanDirs {
		err := filepath.WalkDir(scanDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			baseName := filepath.Base(path)
			if strings.HasPrefix(baseName, "generate.atfw.") || strings.HasSuffix(baseName, ".go") {
				absPath, err := filepath.Abs(path)
				if err != nil {
					if runCache[absPath] {
						return nil
					}
					matches = append(matches, absPath)
					runCache[absPath] = true
				} else {
					if runCache[path] {
						return nil
					}
					matches = append(matches, path)
					runCache[path] = true
				}
			}
			return nil
		})

		if err != nil || len(matches) == 0 {
			fmt.Fprintf(os.Stderr, "Scan generate.atfw.go failed: %v\n", err)
			os.Exit(1)
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i] < matches[j]
	})

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
