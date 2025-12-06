package libatframe_utils

import (
	buildsetting "github.com/atframework/atframe-utils-go/build_setting"
)

type BuildMananger interface {
	Init() error
	SetDocDir(dir string) error

	SetTool(toolName string, version string, path string) error
	GetToolPath(toolName string) (string, error)
	ResetTool(toolName string) error

	ListTools() (map[string]string, error)
}

var NewBuildManager = buildsetting.NewManagerInDir
var BuildManagerLoad = buildsetting.NewManagerLoadExistSettingsFile
