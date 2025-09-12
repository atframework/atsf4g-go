package main

import (
	"os"
	"strings"

	atframe_utils "github.com/atframework/atframe-utils-go"
)

func main() {
	includeDirs := []string{}
	protoDirs := []string{}
	extFlags := []string{}
	for _, arg := range os.Args[4:] {
		if strings.HasPrefix(arg, "I:") {
			includeDirs = append(includeDirs, arg[2:])
		} else if strings.HasPrefix(arg, "P:") {
			protoDirs = append(protoDirs, arg[2:])
		} else {
			extFlags = append(extFlags, arg)
		}
	}

	atframe_utils.RunProcScanFiles(os.Args[1], os.Args[2], os.Args[3], includeDirs, protoDirs, extFlags)
}
