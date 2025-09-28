package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	project_settings "github.com/atframework/atsf4g-go/tools/project-settings"
)

func fmtColorInner(color int, str string) {
	fmt.Printf("\033[1;%s;40m%s\033[0m\n", strconv.Itoa(color), str)
}

func fmtColor(color int, format string, a ...any) {
	fmtColorInner(color, fmt.Sprintf(format, a...))
}

func fmtColorRed(format string, a ...any) {
	fmtColorInner(FgRed, fmt.Sprintf(format, a...))
}

func fmtColorGreen(format string, a ...any) {
	fmtColorInner(FgGreen, fmt.Sprintf(format, a...))
}

func fmtColorCyan(format string, a ...any) {
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

func binName(name string) string {
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(name), ".exe") {
		return name + ".exe"
	}

	return name
}

func serviceBinName(name string) string {
	if !strings.HasSuffix(name, "d") {
		name = name + "d"
	}

	return binName(name)
}

func buildService(projectBaseDir string, buildPath string, sourcePath string, outputPath string) error {
	outputBinPath := path.Join(buildPath, "install", outputPath, "bin")
	os.Mkdir(outputBinPath, os.ModePerm)

	workDir := path.Join(projectBaseDir, sourcePath)
	var cmd *exec.Cmd
	if outputBinRelPath, err := filepath.Rel(workDir, outputBinPath); err == nil {
		fmtColorCyan("Run: go build -o %s", path.Join(outputBinRelPath, serviceBinName(path.Base(sourcePath))))
		cmd = exec.Command("go", "build", "-o", path.Join(outputBinRelPath, serviceBinName(path.Base(sourcePath))))
	} else {
		fmtColorCyan("Run: go build -o %s", path.Join(outputBinPath, serviceBinName(path.Base(sourcePath))))
		cmd = exec.Command("go", "build", "-o", path.Join(outputBinPath, serviceBinName(path.Base(sourcePath))))
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = workDir
	err := cmd.Run()
	if err != nil {
		fmtColorRed("Run build Error %s", err)
		return err
	}

	fmtColorGreen("Run build %s to %s success", sourcePath, outputBinPath)
	return nil
}

func main() {
	// 执行多个构建流程

	// 路径设置
	projectBaseDir := project_settings.GetProjectRootDir()

	buildPath := project_settings.GetProjectBuildDir()
	xresloaderPath := path.Join(projectBaseDir, "third_party", "xresloader")
	buildPbdescDir := path.Join(project_settings.GetProjectResourceTargetDir(), "pbdesc")
	buildBytesDir := path.Join(project_settings.GetProjectResourceTargetDir(), "excel")
	excelGenBytePath := project_settings.GetProjectGenDir()
	resourcePath := project_settings.GetProjectResourceSourceDir()
	generateForPbPath := path.Join(project_settings.GetProjectToolsDir(), "generate-for-pb")

	os.Setenv("ProjectBasePath", projectBaseDir)
	os.Setenv("ProjectBuildPath", buildPath)
	os.Setenv("XresloaderPath", xresloaderPath)
	os.Setenv("BuildPbdescPath", buildPbdescDir)
	os.Setenv("BuildBytesPath", buildBytesDir)
	os.Setenv("XresloaderXmlTpl", path.Join(projectBaseDir, "src", "component", "protocol", "public", "xresconv.xml.tpl"))
	os.Setenv("ExcelGenBytePath", excelGenBytePath)
	os.Setenv("ResourcePath", resourcePath)
	os.Setenv("GenerateForPbPath", generateForPbPath)

	os.MkdirAll(buildPath, os.ModePerm)
	os.MkdirAll(buildPbdescDir, os.ModePerm)
	os.MkdirAll(buildBytesDir, os.ModePerm)
	os.MkdirAll(excelGenBytePath, os.ModePerm)

	// 1.初始化
	{
		// 检查Python
		pythonExecutable, err := exec.LookPath("python3")
		if err != nil {
			fmt.Println("Python Not Found:", err)
			os.Exit(1)
		}
		fmtColor(FgGreen, "PythonExecutable:%s", pythonExecutable)
		os.Setenv("PythonExecutable", pythonExecutable)
	}
	{
		// 检查Java
		javaExecutable, err := exec.LookPath("java")
		if err != nil {
			fmt.Println("Java Not Found:", err)
			os.Exit(1)
		}
		fmtColor(FgGreen, "JavaExecutable:%s", javaExecutable)
		os.Setenv("JavaExecutable", javaExecutable)
	}

	// 2.generate
	{
		cmd := exec.Command("go", "run", ".")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = path.Join(projectBaseDir, "tools", "generate")
		err := cmd.Run()
		if err != nil {
			fmtColorRed("Run generate Error %s", err)
			os.Exit(1)
		}
		fmtColorGreen("Run generate success")
	}

	// 3.build
	exitCode := 0
	if buildService(projectBaseDir, buildPath, path.Join("src", "lobbysvr"), "lobbysvr") != nil {
		exitCode = 1
	}

	// 4.CI....

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
