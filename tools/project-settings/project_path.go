package atframework_tools_project_settings

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

var (
	projectRootDir           string
	projectBuildDir          string
	projectGenDir            string
	projectToolsDir          string
	projectInstallTargetDir  string
	projectInstallSourceDir  string
	projectResourceTargetDir string
	projectResourceSourceDir string
)

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
