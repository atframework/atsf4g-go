package atframework_tools_project_settings

import (
	"os"
	"os/exec"
	"path"
)

var javaExecPath string

func GetJavaPath() (string, error) {
	if javaExecPath != "" {
		return javaExecPath, nil
	}

	javaExecEnv := os.Getenv("JAVA_BIN_PATH")
	if javaExecEnv != "" {
		javaExecPath = javaExecEnv
		return javaExecPath, nil
	}

	// 检查Java
	javaFindExecutable, err := exec.LookPath("java")
	if err == nil {
		javaExecPath = javaFindExecutable
		return javaExecPath, nil
	}

	javaHome := os.Getenv("JAVA_HOME")
	if javaHome != "" {
		testJavaPath := path.Join(javaHome, "bin", binName("java"))
		if _, err := os.Stat(testJavaPath); err == nil {
			javaExecPath = testJavaPath
			return javaExecPath, nil
		}
		testJavaPath = path.Join(javaHome, "jre", "bin", binName("java"))
		if _, err := os.Stat(testJavaPath); err == nil {
			javaExecPath = testJavaPath
			return javaExecPath, nil
		}
	}

	return "", err
}
