package main

import (
	"os"
	"os/exec"
	"path"
)

func main() {
	PYTHON_BIN_PATH := os.Getenv("PYTHON_BIN_PATH")
	XresloaderPath := os.Getenv("PROJECT_XRESLOADER_PATH")
	PROJECT_RESOURCE_TARGET_PBDESC_PATH := os.Getenv("PROJECT_RESOURCE_TARGET_PBDESC_PATH")

	cmd := exec.Command(PYTHON_BIN_PATH, path.Join(XresloaderPath, "xres-code-generator", "xrescode-gen.py"),
		"-i", path.Join("..", "..", "..", "template"),
		"-p", path.Join(PROJECT_RESOURCE_TARGET_PBDESC_PATH, "public-config.pb"),
		"-o", path.Join("..", "generate_config"),
		"-g", path.Join(XresloaderPath, "xres-code-generator", "template", "config_group.go.mako:config_group.go"),
		"-l", path.Join(XresloaderPath, "xres-code-generator", "template", "config_set.go.mako:${\"config_set_{0}.go\".format(loader.get_go_pb_name())}"),
		"-t", "server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
