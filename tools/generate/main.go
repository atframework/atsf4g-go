// Copyright 2025 atframework
package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	atframe_utils "github.com/atframework/atframe-utils-go"
	project_settings "github.com/atframework/atsf4g-go/tools/project-settings"
)

func guessBinDir() string {
	return filepath.Join(project_settings.GetProjectToolsDir(), "bin")
}

func clearGenerateFile(scanDirs []string) {
	prefix := []string{
		"run", ".",
	}
	deleteDir := append(prefix, scanDirs...)
	cmd := exec.Command("go", deleteDir...)
	cmd.Env = os.Environ()
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Dir = filepath.Join(project_settings.GetProjectToolsDir(), "delete-generate-go")
	if err := cmd.Run(); err != nil {
		log.Printf("go delete-generate-go output:\n%s", out.String())
		log.Fatalf("failed to delete-generate-go: %v", err)
		project_settings.FmtColorFprintRed(os.Stderr, "delete-generate-go%s", out.String())
		project_settings.FmtColorFprintRed(os.Stderr, "failed to delete-generate-go: %v", err)
		os.Exit(1)
	}
	project_settings.FmtColorFprintGreen(os.Stdout, "delete-generate-go Success path:%s", scanDirs)
}

func generateAtfwGo(scanDirs []string) {
	pendingGoTidy := make(map[string]bool)
	runCache := make(map[string]bool)
	disableGoTidy := os.Getenv("PROJECT_DISABLE_GENERATE_GO_TIDY") == "true"

	// 扫描所有 generate.atfw.go 文件
	type matchPath struct {
		number int
		path   string
	}
	var matches []matchPath
	for _, scanDir := range scanDirs {
		// Ensure the scan path exists before walking; filepath.WalkDir can handle files,
		// but report a clear error early if the path does not exist.
		if _, serr := os.Stat(scanDir); serr != nil {
			project_settings.FmtColorFprintRed(os.Stderr, "Scan path does not exist: %s\n", scanDir)
			continue
		}
		err := filepath.WalkDir(scanDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			baseName := filepath.Base(path)
			if strings.HasSuffix(baseName, ".go") && strings.HasPrefix(baseName, "generate.atfw.") {
				for !strings.HasPrefix(baseName, "generate.atfw.windows.") || runtime.GOOS == "windows" {

					if strings.HasPrefix(baseName, "generate.atfw.linux.") && runtime.GOOS == "windows" {
						// linux 只有Windows不跑
						break
					}
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
					break
				}
			}
			return nil
		})

		if err != nil || len(matches) == 0 {
			project_settings.FmtColorFprintRed(os.Stderr, "Scan generate.atfw.go failed: %v\n", err)
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
			project_settings.FmtColorFprintRed(os.Stderr, "Run go generate failed: %v\n", err)
			os.Exit(2)
		} else if !disableGoTidy {
			goModDir := project_settings.FindFirstGoMod(filepath.Dir(file.path))
			if goModDir != "" {
				pendingGoTidy[goModDir] = true
			}
		}
	}

	for dir := range pendingGoTidy {
		if err := project_settings.RunGoTidy(dir); err != nil {
			project_settings.FmtColorFprintRed(os.Stderr, "Run go mod tidy failed: %v\n", err)
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
	buildPbdescDir := os.Getenv("PROJECT_RESOURCE_TARGET_PBDESC_PATH")
	buildBytesPath := os.Getenv("PROJECT_RESOURCE_TARGET_BYTES_PATH")
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
		XRESCONV_EXE_PATH:     path.Join(projectBaseDir, "tools", "bin", project_settings.GetXresloaderBinName()),
		XRESCONV_CONFIG_PB:    path.Join(buildPbdescDir, "public-config.pb"),
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

	project_settings.FmtColorPrintGreen("xresconv.xml generated successfully.")

	// 拷贝 validator.yaml
	project_settings.CopyFile(path.Join(resourcePath, "validator.yaml"), path.Join(projectGenDir, "validator.yaml"))
}

func installAtdtool() {
	// 拷贝工具
	project_settings.CopyDir(path.Join(project_settings.GetAtdtoolDownloadPath()), path.Join(project_settings.GetProjectInstallTargetDir(), "atdtool"))
	// 拷贝配置文件
	project_settings.CopyDir(path.Join(project_settings.GetProjectInstallSourceDir(), "cloud-native"), path.Join(project_settings.GetProjectInstallTargetDir(), "deploy"))
}

func installScript() {
	// 拷贝脚本
	project_settings.CopyDir(project_settings.GetProjectScriptDir(), project_settings.GetProjectInstallTargetDir())
}

func main() {
	// 什么都不传 执行这个oldBuild
	oldBuild()
}

func oldBuild() {
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
		project_settings.FmtColorFprintRed(os.Stderr, "Cannot guess tools bin dir\n")
		os.Exit(1)
	}

	atframe_utils.EnsureProtocGenGo()
	protocBin := atframe_utils.EnsureProtocExecutable(toolsBinDir)
	// 将protocBin的上级目录加入PATH
	binDir := filepath.Dir(protocBin)
	os.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	{
		// 拷贝转表所需文件
		_, err := os.Stat(path.Join(project_settings.GetProjectRootDir(), "third_party", "xresloader", "xres-code-generator", "xrescode-gen.py"))
		if err != nil {
			project_settings.FmtColorFprintRed(os.Stderr, "Not Found xres-code-generator xrescode-gen.py: git submodule init git submodule update\n")
			os.Exit(1)
		}

		project_settings.FmtColorPrintYellow("Tools bin dir:", toolsBinDir)
		generateXresloaderXml(project_settings.GetProjectGenDir())
	}

	// 下载组件
	{
		// Xresloader
		binPath := path.Join(project_settings.GetProjectToolsDir(), "bin", project_settings.GetXresloaderBinName())
		if !atframe_utils.FileExists(binPath) {
			project_settings.FmtColorPrintGreen("Download xresloader")
			data := atframe_utils.MustHTTPGet(fmt.Sprintf("https://github.com/owent/xresloader/releases/download/v%s/%s", project_settings.GetXresloaderVersion(), project_settings.GetXresloaderBinName()))
			out, err := os.Create(binPath)
			if err != nil {
				log.Fatalf("create file: %v", err)
			}
			defer out.Close()
			if _, err := out.Write(data); err != nil {
				log.Fatalf("write file: %v", err)
			}
		}
	}
	{
		// atdtool win64
		if runtime.GOOS == "windows" && !atframe_utils.FileExists(path.Join(project_settings.GetAtdtoolDownloadPath(), "bin", "atdtool.exe")) {
			project_settings.FmtColorPrintGreen("Download atdtool.exe")
			data := atframe_utils.MustHTTPGet(fmt.Sprintf("https://github.com/atframework/atdtool/releases/download/v%s/atdtool-windows-amd64.zip", project_settings.GetAtdtoolVersion()))
			atframe_utils.UnzipToDir(data, project_settings.GetAtdtoolDownloadPath())
		}
	}
	{
		// atdtool linux
		if runtime.GOOS == "linux" && !atframe_utils.FileExists(path.Join(project_settings.GetAtdtoolDownloadPath(), "bin", "atdtool")) {
			project_settings.FmtColorPrintGreen("Download atdtool")
			data := atframe_utils.MustHTTPGet(fmt.Sprintf("https://github.com/atframework/atdtool/releases/download/v%s/atdtool-linux-amd64.tar.gz", project_settings.GetAtdtoolVersion()))
			atframe_utils.UntarGzToDir(data, project_settings.GetAtdtoolDownloadPath())
		}
	}

	// Install protoc-gen-mutable
	{
		cmd := exec.Command("go", "install", "./protoc-gen-mutable")
		cmd.Env = os.Environ()
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.Dir = filepath.Join(project_settings.GetProjectAtframeworkDir(), "atframe-utils-go")
		if err := cmd.Run(); err != nil {
			log.Printf("go install output:\n%s", out.String())
			log.Fatalf("failed to install protoc-gen-mutable: %v", err)
			project_settings.FmtColorFprintRed(os.Stderr, "go install output%s", out.String())
			project_settings.FmtColorFprintRed(os.Stderr, "failed to install protoc-gen-mutable: %v", err)
			os.Exit(1)
		}
		project_settings.FmtColorFprintGreen(os.Stdout, "install protoc-gen-mutable Success")
	}

	if runAllTools {
		clearGenerateFile(nil)
	} else {
		clearGenerateFile(scanDirs)
	}
	generateAtfwGo(scanDirs)
	installAtdtool()
	installScript()
}

func runGoGenerate(target string) error {
	// 使用os/exec执行 go generate
	cmd := exec.Command("go", "generate", filepath.Base(target))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = filepath.Dir(target)

	project_settings.FmtColorPrintYellow("Run go generate %s on %s", filepath.Base(target), cmd.Dir)
	return cmd.Run()
}
