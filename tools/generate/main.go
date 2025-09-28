package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	atframe_utils "github.com/atframework/atframe-utils-go"
	project_settings "github.com/atframework/atsf4g-go/tools/project-settings"
)

func guessBinDir() string {
	return filepath.Join(project_settings.GetProjectToolsDir(), "bin")
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

func generateXresloaderXml(projectGenDir string) {
	projectBaseDir := project_settings.GetProjectRootDir()
	buildPbdescDir := os.Getenv("PROJECT_BUILD_PBDESC_PATH")
	buildBytesPath := os.Getenv("PROJECT_BUILD_BYTES_PATH")
	xresloaderXmlTpl := os.Getenv("PROJECT_XRESLOADER_XML_TPL")
	resourcePath := project_settings.GetProjectResourceSourceDir()

	// 解析模板
	tmpl, err := template.ParseFiles(xresloaderXmlTpl)
	if err != nil {
		log.Fatal("Error parsing template: ", err)
		os.Exit(1)
	}

	// 定义模板替换的数据
	data := XresloaderXmlVar{
		XRESCONV_XML_PATH:     path.Join(resourcePath, "xresconv.xml"),
		XRESCONV_EXE_PATH:     path.Join(projectBaseDir, "tools", "xresloader-2.20.1.jar"),
		XRESCONV_CONFIG_PB:    path.Join(buildPbdescDir, "config.pb"),
		XRESCONV_BYTES_OUTPUT: buildBytesPath,
		XRESCONV_EXECL_SRC:    path.Join(resourcePath, "ExcelTables"),
	}

	// 输出到新的文件
	outputFile, err := os.Create(path.Join(projectGenDir, "xresconv.xml"))
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
	project_settings.CopyFile(path.Join(resourcePath, "validator.yaml"), path.Join(projectGenDir, "validator.yaml"))
}

func installAtdtool() {
	// 拷贝工具
	project_settings.CopyDir(path.Join(project_settings.GetProjectInstallSourceDir(), "atdtool"), path.Join(project_settings.GetProjectInstallTargetDir(), "atdtool"))
	// 拷贝配置文件
	project_settings.CopyDir(path.Join(project_settings.GetProjectInstallSourceDir(), "cloud-native"), path.Join(project_settings.GetProjectInstallTargetDir(), "deploy"))
}

func main() {
	err := project_settings.PathSetup()
	if err != nil {
		os.Exit(1)
	}

	scanDirs := []string{"../../"}
	runAllTools := true
	if len(os.Args) > 1 && os.Args[1] != "" {
		scanDirs = os.Args[1:]
		runAllTools = false
	}

	toolsBinDir := guessBinDir()
	if toolsBinDir == "" {
		fmt.Fprintf(os.Stderr, "Cannot guess tools bin dir\n")
		os.Exit(1)
	}

	protocBin := atframe_utils.EnsureProtocExecutable(toolsBinDir)
	// 将protocBin的上级目录加入PATH
	binDir := filepath.Dir(protocBin)
	os.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if runAllTools {
		_, err := os.Stat(path.Join(project_settings.GetProjectRootDir(), "third_party", "xresloader", "xres-code-generator", "xrescode-gen.py"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Not Found xres-code-generator xrescode-gen.py\n")
			os.Exit(1)
		}

		log.Println("Tools bin dir:", toolsBinDir)
		generateXresloaderXml(project_settings.GetProjectGenDir())
	}

	generateAtfwGo(scanDirs)
	installAtdtool()
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
