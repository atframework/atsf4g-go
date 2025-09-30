// Copyright 2025 atframework
package atframework_tools_project_settings

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

var (
	projectPackageName       string = "proy"
	projectRootDir           string
	projectBuildDir          string
	projectTemplateDir       string
	projectGenDir            string
	projectToolsDir          string
	projectInstallTargetDir  string
	projectInstallSourceDir  string
	projectResourceTargetDir string
	projectResourceSourceDir string
	projectSourceDir         string
)

func GetProjectPackageName() string {
	return projectPackageName
}

func findProjectRootDir() string {
	if _, filename, _, ok := runtime.Caller(0); ok {
		checkPath := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(filename))), "tools", "project-settings", "go.mod")
		if _, err := os.Stat(checkPath); err == nil {
			return filepath.Dir(filepath.Dir(filepath.Dir(filename)))
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
		if _, err := os.Stat(filepath.Join(baseDir, "tools", "project-settings", "go.mod")); err == nil {
			return baseDir
		}

		previousDir = baseDir
		baseDir = filepath.Dir(baseDir)
	}

	return cwdDir
}

func PathSetup() error {
	// 路径设置
	projectBaseDir := GetProjectRootDir()

	buildPath := GetProjectBuildDir()
	buildPbdescDir := path.Join(GetProjectResourceTargetDir(), "pbdesc")
	buildBytesDir := path.Join(GetProjectResourceTargetDir(), "excel")
	projectGenDir := GetProjectGenDir()
	projectToolsDir := GetProjectToolsDir()
	xresloaderPath := path.Join(projectBaseDir, "third_party", "xresloader")
	generateForPbPath := path.Join(projectToolsDir, "generate-for-pb")
	pythonBinPath, err := GetPythonPath()
	if err != nil {
		FmtColor(FgRed, "Get Python Path Failed: %v", err)
		os.Exit(1)
	}
	FmtColor(FgGreen, "PYTHON_BIN_PATH: %s", pythonBinPath)

	javaBinPath, err := GetJavaPath()
	if err != nil {
		FmtColor(FgRed, "Get Java Path Failed: %v", err)
		os.Exit(1)
	}
	FmtColor(FgGreen, "JAVA_BIN_PATH: %s", javaBinPath)

	os.Setenv("PROJECT_XRESLOADER_PATH", xresloaderPath)
	os.Setenv("PROJECT_RESOURCE_TARGET_PBDESC_PATH", buildPbdescDir)
	os.Setenv("PROJECT_RESOURCE_TARGET_BYTES_PATH", buildBytesDir)
	os.Setenv("PROJECT_XRESLOADER_XML_TPL", path.Join(projectBaseDir, "src", "component", "protocol", "public", "xresconv.xml.tpl"))
	os.Setenv("PROJECT_BUILD_GEN_PATH", projectGenDir)
	os.Setenv("PROJECT_TOOLS_GENERATE_FOR_PB_PATH", generateForPbPath)
	os.Setenv("PYTHON_BIN_PATH", pythonBinPath)
	os.Setenv("JAVA_BIN_PATH", javaBinPath)
	os.Setenv("PROJECT_TOOLS_DIR", projectToolsDir)
	os.Setenv("PROJECT_ROOT_DIR", projectBaseDir)

	os.MkdirAll(buildPath, os.ModePerm)
	os.MkdirAll(buildPbdescDir, os.ModePerm)
	os.MkdirAll(buildBytesDir, os.ModePerm)
	os.MkdirAll(projectGenDir, os.ModePerm)
	return nil
}

func GetProjectRootDir() string {
	if projectRootDir == "" {
		projectRootDir = findProjectRootDir()
	}
	return projectRootDir
}

func findProjectDir(base string, envVar string, defaultValue string) string {
	var build_dir string
	if envVar != "" {
		build_dir = os.Getenv(envVar)
	}
	if build_dir != "" {
		if path.IsAbs(build_dir) {
			return build_dir
		}
	} else {
		build_dir = defaultValue
	}

	return filepath.Join(base, build_dir)
}

func GetProjectBuildDir() string {
	if projectBuildDir == "" {
		projectBuildDir = findProjectDir(GetProjectRootDir(), "PROJECT_BUILD_DIR", "build")
	}
	return projectBuildDir
}

func GetProjectTemplateDir() string {
	if projectTemplateDir == "" {
		projectTemplateDir = findProjectDir(GetProjectRootDir(), "PROJECT_TEMPLATE_DIR", filepath.Join("src", "template"))
	}
	return projectTemplateDir
}

func GetProjectGenDir() string {
	if projectGenDir == "" {
		projectGenDir = findProjectDir(GetProjectBuildDir(), "PROJECT_GEN_DIR", "_gen")
	}
	return projectGenDir
}

func GetProjectToolsDir() string {
	if projectToolsDir == "" {
		projectToolsDir = findProjectDir(GetProjectRootDir(), "", "tools")
	}
	return projectToolsDir
}

func GetProjectInstallTargetDir() string {
	if projectInstallTargetDir == "" {
		projectInstallTargetDir = findProjectDir(GetProjectBuildDir(), "PROJECT_INSTALL_TARGET_DIR", "install")
	}
	return projectInstallTargetDir
}

func GetProjectInstallSourceDir() string {
	if projectInstallSourceDir == "" {
		projectInstallSourceDir = findProjectDir(GetProjectRootDir(), "PROJECT_INSTALL_SOURCE_DIR", "install")
	}
	return projectInstallSourceDir
}

func GetProjectResourceTargetDir() string {
	if projectResourceTargetDir == "" {
		projectResourceTargetDir = findProjectDir(GetProjectInstallTargetDir(), "PROJECT_INSTALL_RESOURCE_TARGET_DIR", "resource")
	}
	return projectResourceTargetDir
}

func GetProjectResourceSourceDir() string {
	if projectResourceSourceDir == "" {
		projectResourceSourceDir = findProjectDir(GetProjectRootDir(), "PROJECT_INSTALL_RESOURCE_SOURCE_DIR", "resource")
	}
	return projectResourceSourceDir
}

func GetProjectSourceDir() string {
	if projectSourceDir == "" {
		projectSourceDir = findProjectDir(GetProjectRootDir(), "PROJECT_SOURCE_DIR", "src")
	}
	return projectSourceDir
}

func GetXresloaderBinName() string {
	return fmt.Sprintf("xresloader-%s.jar", GetXresloaderVersion())
}

func GetAtdtoolDownloadPath() string {
	return path.Join(GetProjectToolsDir(), "bin", fmt.Sprintf("atdtool_v%s", GetAtdtoolVersion()))
}
