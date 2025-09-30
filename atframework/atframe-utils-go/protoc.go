package libatframe_utils

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// 固定 protoc 版本，建议与你的 protoc-gen-go 版本兼容
const (
	protocVersion             = "32.1"
	protocGoPluginVersion     = "1.36.9"
	protocGoGrpcPluginVersion = "v1.5.1"
)

// releases 列表：https://github.com/protocolbuffers/protobuf/releases

func EnsureProtocExecutable(binDir string) string {
	return ensureProtoc(protocVersion, binDir)
}

func RunProcScanFiles(cwd string, binDir string, outDir string, includePaths []string, protoDirs []string, extFlags []string) error {
	var allProtoFiles []string
	for _, dir := range protoDirs {
		files, err := listProtoFiles(dir)
		if err != nil {
			return fmt.Errorf("list proto files: %v", err)
		}
		allProtoFiles = append(allProtoFiles, files...)
	}

	return RunProc(cwd, binDir, outDir, includePaths, allProtoFiles, extFlags)
}

func RunProc(cwd string, binDir string, outDir string, includePaths []string, protoPaths []string, extFlags []string) error {
	repoRoot := mustAbs(cwd)
	mustMkdirAll(outDir)

	// 1) 确保 protoc 可用（不存在则下载解压）
	protocBin := ensureProtoc(protocVersion, binDir)

	// 2) 确保 go 插件（不存在则 go install）
	ensureGoInstall("protoc-gen-go", "google.golang.org/protobuf/cmd/protoc-gen-go@{"+protocGoPluginVersion+"}")
	// ensureGoInstall("protoc-gen-go-grpc", "google.golang.org/grpc/cmd/protoc-gen-go-grpc@{"+protocGoGrpcPluginVersion+"}")

	// 3) 构造参数并执行
	includeArgs := []string{}
	for _, p := range includePaths {
		includeArgs = append(includeArgs, "-I", p)
	}
	outArgs := []string{
		"--go_out=paths=source_relative:" + outDir,
		// "--go-grpc_out=paths=source_relative,require_unimplemented_servers=false:" + outDir,
	}
	if len(protoPaths) == 0 {
		return fmt.Errorf("no .proto files found; nothing to do")
	}

	args := append(includeArgs, append(outArgs, protoPaths...)...)
	if extFlags != nil {
		args = append(args, extFlags...)
	}
	log.Printf("Running protoc (%s):\n  %s", protocBin, strings.Join(args, " "))

	cmd := exec.Command(protocBin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = repoRoot
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("protoc failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	return nil
}

// =============== Protoc 下载/解压 ===============

func ensureProtoc(version string, binDir string) string {
	// 若系统 PATH 已存在 protoc，且版本 >= 需要版本，可直接用
	if p, err := exec.LookPath(binName("protoc")); err == nil {
		if ok := isProtocVersionAtLeast(p, versionMajor(version)); ok {
			return p
		}
		log.Printf("system protoc version is lower than %s, downloading portable protoc...", version)
	}

	osName, arch := runtime.GOOS, runtime.GOARCH
	assetURL, isZip, err := protocAssetURL(version, osName, arch)
	if err != nil {
		log.Fatalf("resolve protoc asset: %v", err)
	}

	cacheRoot := binDir
	targetDir := filepath.Join(cacheRoot, "protoc", version, osName+"-"+arch)
	protocPath := filepath.Join(targetDir, binName("protoc"))
	protocFallbackPath := filepath.Join(targetDir, "bin", binName("protoc"))

	if FileExists(protocPath) {
		return protocPath
	}

	if FileExists(protocFallbackPath) {
		return protocFallbackPath
	}

	log.Printf("Downloading protoc %s for %s/%s\nURL: %s", version, osName, arch, assetURL)
	mustMkdirAll(targetDir)

	// 下载到内存（也可落盘后再解压）
	data := MustHTTPGet(assetURL)

	// 解压
	if isZip {
		UnzipToDir(data, targetDir)
	} else {
		UntarGzToDir(data, targetDir)
	}

	if !FileExists(protocPath) && FileExists(protocFallbackPath) {
		os.Rename(protocFallbackPath, protocPath)
	}

	// 包结构通常为：bin/protoc, include/***
	// 我们确保可执行权限（*nix）
	if runtime.GOOS != "windows" {
		if err := os.Chmod(protocPath, 0o755); err != nil {
			log.Printf("chmod protoc: %v", err)
		}
	}

	// 简单存在性检查
	if !FileExists(protocPath) {
		log.Fatalf("protoc not found after extraction at: %s", protocPath)
	}
	return protocPath
}

func protocAssetURL(version, osName, arch string) (string, bool, error) {
	// 资产命名参考官方 release
	// Linux: protoc-<ver>-linux-x86_64.zip
	// macOS: protoc-<ver>-osx-universal_binary.zip 或 osx-aarch_64.zip（新版本多为 universal_binary）
	// Windows: protoc-<ver>-win64.zip / win32.zip（老版本），新版本统一为 win64
	base := "https://github.com/protocolbuffers/protobuf/releases/download/v" + version + "/"

	switch osName {
	case "linux":
		// 只处理常见架构
		switch arch {
		case "amd64":
			return base + "protoc-" + version + "-linux-x86_64.zip", true, nil
		case "arm64":
			// 新版本支持 aarch_64
			return base + "protoc-" + version + "-linux-aarch_64.zip", true, nil
		}
	case "darwin":
		// 官方常见为通用二进制
		return base + "protoc-" + version + "-osx-universal_binary.zip", true, nil
	case "windows":
		// 官方多为 win64（x86_64）
		if arch == "amd64" || arch == "arm64" {
			// 目前提供 win64 包；在 arm64 上也能运行 x64 仿真，或使用本机 arm64 包（若存在）
			return base + "protoc-" + version + "-win64.zip", true, nil
		}
	}
	return "", false, fmt.Errorf("unsupported platform %s/%s or asset not mapped", osName, arch)
}

func MustHTTPGet(url string) []byte {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Fatalf("download failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("bad status: %s", resp.Status)
	}
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		log.Fatalf("read body: %v", err)
	}
	return buf.Bytes()
}

func UnzipToDir(data []byte, dest string) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		log.Fatalf("read zip: %v", err)
	}
	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)
		// 防止 zip slip
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
			log.Fatalf("illegal path in zip: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			mustMkdirAll(target)
			continue
		}
		mustMkdirAll(filepath.Dir(target))
		rc, err := f.Open()
		if err != nil {
			log.Fatalf("open zip file: %v", err)
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			log.Fatalf("create file: %v", err)
		}
		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			log.Fatalf("write file: %v", err)
		}
		rc.Close()
		out.Close()
	}
}

func UntarGzToDir(data []byte, dest string) {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("read gzip: %v", err)
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatalf("read tar: %v", err)
		}
		target := filepath.Join(dest, hdr.Name)
		// 防止 tar slip
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
			log.Fatalf("illegal path in tar: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			mustMkdirAll(target)
		case tar.TypeReg:
			mustMkdirAll(filepath.Dir(target))
			out, err := os.Create(target)
			if err != nil {
				log.Fatalf("create file: %v", err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				log.Fatalf("write file: %v", err)
			}
			out.Close()
		default:
			// 忽略其他类型
		}
	}
}

// =============== 生成辅助工具 ===============

func ensureGoInstall(bin, moduleAt string) {
	if _, err := exec.LookPath(binName(bin)); err == nil {
		return
	}
	log.Printf("%s not found, installing: go install %s", bin, moduleAt)
	cmd := exec.Command("go", "install", moduleAt)
	cmd.Env = os.Environ()
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		log.Printf("go install output:\n%s", out.String())
		log.Fatalf("failed to install %s: %v", bin, err)
	}
	// 再查一遍
	if _, err := exec.LookPath(binName(bin)); err != nil {
		log.Fatalf("after install, %s still not found in PATH: %v", bin, err)
	}
}

func listProtoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".proto") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func mustAbs(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		log.Fatalf("abs path: %v", err)
	}
	return abs
}

func mustMkdirAll(p string) {
	if err := os.MkdirAll(p, 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}
}

func FileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func PathExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func prependPath(env []string, dir string) []string {
	if dir == "" {
		return env
	}
	const key = "PATH"
	sep := string(os.PathListSeparator)
	for i, kv := range env {
		if strings.HasPrefix(kv, key+"=") {
			parts := strings.SplitN(kv, "=", 2)
			cur := parts[1]
			// 已包含则不重复添加
			for _, p := range filepath.SplitList(cur) {
				if sameDir(p, dir) {
					return env
				}
			}
			env[i] = key + "=" + dir + sep + cur
			return env
		}
	}
	return append(env, key+"="+dir)
}

func sameDir(a, b string) bool {
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	if err1 != nil || err2 != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return filepath.Clean(ra) == filepath.Clean(rb)
}

func binName(name string) string {
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(name), ".exe") {
		return name + ".exe"
	}
	return name
}

func isProtocVersionAtLeast(protocPath string, wantMajor int) bool {
	out, err := exec.Command(protocPath, "--version").Output()
	if err != nil {
		return false
	}
	// 输出示例：libprotoc 27.1
	var major int
	_, _ = fmt.Sscanf(string(out), "libprotoc %d", &major)
	return major >= wantMajor
}

func versionMajor(v string) int {
	var m int
	fmt.Sscanf(v, "%d", &m)
	return m
}
