package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

func main() {
	// 执行多个构建流程

	// 路径设置
	projectBaseDir, _ := filepath.Abs("../../")

	buildPath := projectBaseDir + "/build"
	xresloaderPath := projectBaseDir + "/third_party/xresloader"
	buildPbdescDir := buildPath + "/pbdesc"
	excelGenBytePath := buildPath + "/_gen"
	resourcePath := projectBaseDir + "/resource"

	os.Setenv("ProjectBasePath", projectBaseDir)
	os.Setenv("ProjectBuildPath", buildPath)
	os.Setenv("XresloaderPath", xresloaderPath)
	os.Setenv("BuildPbdescPath", buildPbdescDir)
	os.Setenv("XresloaderXmlTpl", projectBaseDir+"/src/component/protocol/public/xresconv.xml.tpl")
	os.Setenv("ExcelGenBytePath", excelGenBytePath)
	os.Setenv("ResourcePath", resourcePath)

	os.Mkdir(buildPath, os.ModePerm)
	os.Mkdir(buildPbdescDir, os.ModePerm)
	os.Mkdir(excelGenBytePath, os.ModePerm)

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
	{
		// 安装protoc
		cmd := exec.Command("go", "install", "google.golang.org/protobuf/cmd/protoc-gen-go@latest")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = projectBaseDir
		err := cmd.Run()
		if err != nil {
			fmtColorRed("Run install protoc Error %s", err)
			os.Exit(1)
		}
		fmtColorGreen("Run Install Protoc Success")
	}

	// 2.generate
	{
		cmd := exec.Command("go", "run", ".")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = projectBaseDir + "/tools/generate"
		err := cmd.Run()
		if err != nil {
			fmtColorRed("Run generate Error %s", err)
			os.Exit(1)
		}
		fmtColorGreen("Run generate Success")
	}
	// 3.build
	buildBinPath := buildPath + "/bin"
	os.Mkdir(buildBinPath, os.ModePerm)
	{
		srcPath := "/src/lobbysvr"
		cmd := exec.Command("go", "build", "-o", buildBinPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = projectBaseDir + srcPath
		err := cmd.Run()
		if err != nil {
			fmtColorRed("Run build Error %s", err)
			os.Exit(1)
		}
		fmtColorGreen("Run build %s Success", srcPath)
	}
	// 4.CI....
}
