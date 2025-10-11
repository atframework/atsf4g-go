// Copyright 2025 atframework
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	project_settings "github.com/atframework/atsf4g-go/tools/project-settings"
)

type mvPath struct {
	src string
	dst string
}

func main() {
	// 同步GIT仓库内的修改到本地的SVN仓库 方便SVN提交
	svnPath := "D:/SVN/ProjectY/trunk/Protocol/protocol"

	gitProtocolList := []mvPath{
		{
			src: filepath.Join(project_settings.GetProjectRootDir(), "/src/component/protocol/public/common/protocol/common"),
			dst: filepath.Join(svnPath, "common"),
		},
		{
			src: filepath.Join(project_settings.GetProjectRootDir(), "/src/component/protocol/public/config/protocol/config"),
			dst: filepath.Join(svnPath, "config"),
		},
		{
			src: filepath.Join(project_settings.GetProjectRootDir(), "/src/component/protocol/public/pbdesc/protocol/pbdesc"),
			dst: filepath.Join(svnPath, "pbdesc"),
		},
		{
			src: filepath.Join(project_settings.GetProjectRootDir(), "/src/lobbysvr/protocol/public/protocol/pbdesc"),
			dst: filepath.Join(svnPath, "pbdesc"),
		},
	}

	// 删除所有文件
	for _, v := range gitProtocolList {
		project_settings.RmDirFile(v.dst)
	}

	// 拷贝所有文件
	var err error
	for _, v := range gitProtocolList {
		var entries []os.DirEntry
		entries, err = os.ReadDir(v.src)
		if err != nil {
			break
		}

		var files []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".proto") {
				files = append(files, filepath.Join(v.src, e.Name()))
			}
		}

		for _, file := range files {
			fmt.Printf("Copy file %s to %s \n", file, filepath.Join(v.dst, filepath.Base(file)))
			err = project_settings.CopyFile(file, filepath.Join(v.dst, filepath.Base(file)))
			if err != nil {
				break
			}
		}
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
