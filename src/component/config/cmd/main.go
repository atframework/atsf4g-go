package main

import (
	"os"
	"os/exec"
	"path"
)

func main() {
	PythonExecutable := os.Getenv("PythonExecutable")
	XresloaderPath := os.Getenv("XresloaderPath")
	BuildPbdescPath := os.Getenv("BuildPbdescPath")

	cmd := exec.Command(PythonExecutable, path.Join(XresloaderPath, "xres-code-generator", "xrescode-gen.py"),
		"-i", path.Join("..", "..", "..", "template"),
		"-p", path.Join(BuildPbdescPath, "config.pb"),
		"-o", path.Join("..", "generate_config"),
		"-g", path.Join(XresloaderPath, "xres-code-generator", "template", "config_group.go.mako:config_group.go"),
		"-l", path.Join(XresloaderPath, "xres-code-generator", "template", "config_set.go.mako:${\"config_set_{0}.go\".format(loader.get_go_pb_name())}"),
		"-t", "server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
