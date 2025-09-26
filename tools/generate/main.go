package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
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

func generateAtfwGo(scanDirs []string) {
	pendingGoTidy := make(map[string]bool)
	runCache := make(map[string]bool)

	// 扫描所有 generate.atfw.go 文件
	type matchPath struct {
		number int
		path   string
	}
	var matches []matchPath
	for _, scanDir := range scanDirs {
		err := filepath.WalkDir(scanDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			baseName := filepath.Base(path)
			if strings.HasPrefix(baseName, "generate.atfw.") && strings.HasSuffix(baseName, ".go") {
				absPath, err := filepath.Abs(path)

				re := regexp.MustCompile(`\.(\d+)\.go$`)
				numberStrMatch := re.FindStringSubmatch(baseName)
				number, _ := strconv.Atoi(numberStrMatch[1])

				if err != nil {
					if runCache[absPath] {
						return nil
					}
					matches = append(matches, matchPath{number, absPath})
					runCache[absPath] = true
				} else {
					if runCache[path] {
						return nil
					}
					matches = append(matches, matchPath{number, absPath})
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
		if matches[i].number < matches[j].number {
			return true
		} else if matches[i].number > matches[j].number {
			return false
		}
		return matches[i].path < matches[j].path
	})

	for _, file := range matches {
		// 执行 go generate
		if err := runGoGenerate(file.path); err != nil {
			fmt.Fprintf(os.Stderr, "Run go generate failed: %v\n", err)
			os.Exit(2)
		} else {
			goModDir := findFirstGoMod(filepath.Dir(file.path))
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

type XresloaderXmlVar struct {
	XRESCONV_XML_PATH     string
	XRESCONV_EXE_PATH     string
	XRESCONV_CONFIG_PB    string
	XRESCONV_BYTES_OUTPUT string
	XRESCONV_EXECL_SRC    string
}

func copyFile(src, dest string) error {
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

func generateXresloaderXml() {
	projectBaseDir := os.Getenv("ProjectBasePath")
	buildPbdescDir := os.Getenv("BuildPbdescPath")
	buildPath := os.Getenv("ProjectBuildPath")
	xresloaderXmlTpl := os.Getenv("XresloaderXmlTpl")
	resourcePath := os.Getenv("ResourcePath")
	excelGenBytePath := os.Getenv("ExcelGenBytePath")

	// 解析模板
	tmpl, err := template.ParseFiles(xresloaderXmlTpl)
	if err != nil {
		log.Fatal("Error parsing template: ", err)
		os.Exit(1)
	}

	// 定义模板替换的数据
	data := XresloaderXmlVar{
		XRESCONV_XML_PATH:     resourcePath + "/xresconv.xml",
		XRESCONV_EXE_PATH:     projectBaseDir + "/tools/xresloader-2.20.1.jar",
		XRESCONV_CONFIG_PB:    buildPbdescDir + "/config.pb",
		XRESCONV_BYTES_OUTPUT: buildPath + "/excel",
		XRESCONV_EXECL_SRC:    resourcePath + "/ExcelTables",
	}

	// 输出到新的文件
	outputFile, err := os.Create(excelGenBytePath + "/xresconv.xml")
	if err != nil {
		log.Fatal("Error creating output file: ", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	// 执行模板并将输出写入文件
	err = tmpl.Execute(outputFile, data)
	if err != nil {
		log.Fatal("Error executing template: ", err)
		os.Exit(1)
	}

	log.Println("xresconv.xml generated successfully.")

	// 拷贝 validator.yaml
	copyFile(resourcePath+"/validator.yaml", excelGenBytePath+"/validator.yaml")
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

	_, err := os.Stat("../../third_party/xresloader/xres-code-generator/xrescode-gen.py")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Not Found xres-code-generator xrescode-gen.py\n")
		os.Exit(1)
	}

	log.Println("Tools bin dir:", toolsBinDir)

	protocBin := atframe_utils.EnsureProtocExecutable(toolsBinDir)
	// 将protocBin的上级目录加入PATH
	binDir := filepath.Dir(protocBin)
	os.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	generateXresloaderXml()
	generateAtfwGo(scanDirs)
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
