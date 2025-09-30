// Copyright 2025 atframework
package main

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"

	project_settings "github.com/atframework/atsf4g-go/tools/project-settings"
)

func buildService(projectBaseDir string, buildPath string, sourcePath string, outputPath string) error {
	outputBinPath := path.Join(buildPath, "install", outputPath, "bin")
	os.Mkdir(outputBinPath, os.ModePerm)

	workDir := path.Join(projectBaseDir, sourcePath)
	var cmd *exec.Cmd
	if outputBinRelPath, err := filepath.Rel(workDir, outputBinPath); err == nil {
		project_settings.FmtColorCyan("Run: go build -o %s", path.Join(outputBinRelPath, project_settings.ServiceBinName(path.Base(sourcePath))))
		cmd = exec.Command("go", "build", "-o", path.Join(outputBinRelPath, project_settings.ServiceBinName(path.Base(sourcePath))))
	} else {
		project_settings.FmtColorCyan("Run: go build -o %s", path.Join(outputBinPath, project_settings.ServiceBinName(path.Base(sourcePath))))
		cmd = exec.Command("go", "build", "-o", path.Join(outputBinPath, project_settings.ServiceBinName(path.Base(sourcePath))))
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = workDir
	err := cmd.Run()
	if err != nil {
		project_settings.FmtColorRed("Run build Error %s", err)
		return err
	}

	project_settings.FmtColorGreen("Run build %s to %s success", sourcePath, outputBinPath)
	return nil
}

func main() {
	// 执行多个构建流程
	os.Setenv("PROJECT_DISABLE_GENERATE_GO_TIDY", "true")

	// 1.generate
	{
		cmd := exec.Command("go", "run", ".")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = path.Join(project_settings.GetProjectRootDir(), "tools", "generate")
		err := cmd.Run()
		if err != nil {
			project_settings.FmtColorRed("Run generate Error %s", err)
			os.Exit(1)
		}
		project_settings.FmtColorGreen("Run generate success")
	}

	// 2.go mod
	project_settings.RunGoModTidy(project_settings.GetProjectSourceDir())
	project_settings.FmtColorGreen("Run Go Mod success")

	// 3.build
	exitCode := 0
	if buildService(project_settings.GetProjectRootDir(), project_settings.GetProjectBuildDir(), path.Join("src", "lobbysvr"), "lobbysvr") != nil {
		exitCode = 1
	}
	if buildService(project_settings.GetProjectRootDir(), project_settings.GetProjectBuildDir(), path.Join("src", "robot"), "robot") != nil {
		exitCode = 1
	}

	project_settings.FmtColorGreen("Build success")

	// 4.CI....

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
