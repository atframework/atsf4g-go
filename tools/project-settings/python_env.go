// Copyright 2025 atframework
package atframework_tools_project_settings

import (
	"fmt"
	"os"
	"os/exec"
	"path"
)

var pythonPath string

func GetPythonPath() (string, error) {
	if pythonPath != "" {
		return pythonPath, nil
	}

	// 自定义变量
	pythonEnvPath := os.Getenv("PYTHON_BIN_PATH")
	if pythonEnvPath != "" {
		pythonPath = pythonEnvPath
		return pythonPath, nil
	}

	// pyenv
	pyEnv := os.Getenv("PYENV_ROOT")
	if pyEnv != "" {
		testPythonPath := path.Join(pyEnv, "bin", binName("python3"))
		if _, err := os.Stat(testPythonPath); err != nil {
			pythonPath = testPythonPath
			return pythonPath, nil
		}

		testPythonPath = path.Join(pyEnv, "bin", binName("python"))
		if _, err := os.Stat(testPythonPath); err != nil {
			pythonPath = testPythonPath
			return pythonPath, nil
		}
	}

	// virtual env
	pyEnv = os.Getenv("VIRTUAL_ENV")
	if pyEnv != "" {
		testPythonPath := path.Join(pyEnv, "bin", binName("python3"))
		if _, err := os.Stat(testPythonPath); err != nil {
			pythonPath = testPythonPath
			return pythonPath, nil
		}

		testPythonPath = path.Join(pyEnv, "bin", binName("python"))
		if _, err := os.Stat(testPythonPath); err != nil {
			pythonPath = testPythonPath
			return pythonPath, nil
		}
	}

	// PYTHONHOME
	pyEnv = os.Getenv("PYTHONHOME")
	if pyEnv != "" {
		testPythonPath := path.Join(pyEnv, "bin", binName("python3"))
		if _, err := os.Stat(testPythonPath); err != nil {
			pythonPath = testPythonPath
			return pythonPath, nil
		}

		testPythonPath = path.Join(pyEnv, "bin", binName("python"))
		if _, err := os.Stat(testPythonPath); err != nil {
			pythonPath = testPythonPath
			return pythonPath, nil
		}
	}

	// 检查Python
	pythonFindExecutable, err := exec.LookPath("python3")
	if err == nil {
		pythonPath = pythonFindExecutable
		return pythonFindExecutable, nil
	}

	pythonFindExecutable, err = exec.LookPath("python")
	if err == nil {
		pythonPath = pythonFindExecutable
		return pythonFindExecutable, nil
	}

	return "", fmt.Errorf("python3/python Not Found")
}
