// Copyright 2026 atframework
package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/atframework/atframe-utils-go/buildkit"
)

func TestXresconvAssetSpec(t *testing.T) {
	tests := []struct {
		version string
		goos    string
		goarch  string
		want    string
	}{
		{"2.1.0", "windows", "amd64", "https://github.com/owent/xresconv-cli/releases/download/v2.1.0/xresconv-cli-2.1.0-x86_64-pc-windows-msvc.zip"},
		{"v2.1.0", "windows", "arm64", "https://github.com/owent/xresconv-cli/releases/download/v2.1.0/xresconv-cli-2.1.0-aarch64-pc-windows-msvc.zip"},
		{"2.1.0", "linux", "amd64", "https://github.com/owent/xresconv-cli/releases/download/v2.1.0/xresconv-cli-2.1.0-x86_64-unknown-linux-musl.tar.gz"},
		{"2.1.0", "linux", "arm64", "https://github.com/owent/xresconv-cli/releases/download/v2.1.0/xresconv-cli-2.1.0-aarch64-unknown-linux-musl.tar.gz"},
		{"2.1.0", "linux", "riscv64", "https://github.com/owent/xresconv-cli/releases/download/v2.1.0/xresconv-cli-2.1.0-riscv64gc-unknown-linux-gnu.tar.gz"},
		{"2.1.0", "darwin", "amd64", "https://github.com/owent/xresconv-cli/releases/download/v2.1.0/xresconv-cli-2.1.0-x86_64-apple-darwin.tar.gz"},
		{"2.1.0", "darwin", "arm64", "https://github.com/owent/xresconv-cli/releases/download/v2.1.0/xresconv-cli-2.1.0-aarch64-apple-darwin.tar.gz"},
		{"2.1.0", "freebsd", "amd64", "https://github.com/owent/xresconv-cli/releases/download/v2.1.0/xresconv-cli-2.1.0-x86_64-unknown-freebsd.tar.gz"},
		{"2.1.0", "android", "arm64", "https://github.com/owent/xresconv-cli/releases/download/v2.1.0/xresconv-cli-2.1.0-aarch64-linux-android.tar.gz"},
	}

	for _, tc := range tests {
		_, _, _, gotURL, err := xresconvAssetSpec(tc.version, tc.goos, tc.goarch)
		if err != nil {
			t.Fatalf("xresconvAssetSpec(%s, %s, %s) error: %v", tc.version, tc.goos, tc.goarch, err)
		}
		if gotURL != tc.want {
			t.Fatalf("xresconvAssetSpec(%s, %s, %s) = %q, want %q", tc.version, tc.goos, tc.goarch, gotURL, tc.want)
		}
	}

	if _, _, _, _, err := xresconvAssetSpec("2.1.0", "windows", "386"); err == nil {
		t.Fatal("expected error for unsupported windows/386 platform")
	}
}

func TestParseSHA256ChecksumAndVerify(t *testing.T) {
	payload := []byte("hello xresconv-cli")
	sum := sha256.Sum256(payload)
	hexSum := hex.EncodeToString(sum[:])

	parsed, err := parseSHA256Checksum([]byte(strings.ToUpper(hexSum) + "  xresconv-cli-2.1.0.zip\n"))
	if err != nil {
		t.Fatalf("parseSHA256Checksum failed: %v", err)
	}
	if parsed != hexSum {
		t.Fatalf("parsed=%q want=%q", parsed, hexSum)
	}
	if err := verifySHA256(payload, parsed); err != nil {
		t.Fatalf("verifySHA256 failed: %v", err)
	}
	if err := verifySHA256([]byte("tampered"), parsed); err == nil {
		t.Fatal("expected verifySHA256 mismatch error")
	}
	if _, err := parseSHA256Checksum([]byte("not-a-valid-sha256")); err == nil {
		t.Fatal("expected parseSHA256Checksum error for invalid input")
	}
}

func TestExtractBinaryFromZipAndTarGz(t *testing.T) {
	want := []byte("fake-binary-bytes")

	zipBytes := buildTestZip(t, map[string][]byte{
		"../evil.exe":      []byte("bad"),
		"xresconv-cli.exe": want,
	})
	gotZip, err := extractBinaryFromZip(zipBytes, "xresconv-cli.exe")
	if err != nil {
		t.Fatalf("extractBinaryFromZip failed: %v", err)
	}
	if !bytes.Equal(gotZip, want) {
		t.Fatalf("zip extracted %q, want %q", gotZip, want)
	}

	tarGzBytes := buildTestTarGz(t, map[string][]byte{
		"../evil":        []byte("bad"),
		"./xresconv-cli": want,
	})
	gotTar, err := extractBinaryFromTarGz(tarGzBytes, "xresconv-cli")
	if err != nil {
		t.Fatalf("extractBinaryFromTarGz failed: %v", err)
	}
	if !bytes.Equal(gotTar, want) {
		t.Fatalf("tar.gz extracted %q, want %q", gotTar, want)
	}

	missingZip := buildTestZip(t, map[string][]byte{"README.md": []byte("doc")})
	if _, err := extractBinaryFromZip(missingZip, "xresconv-cli.exe"); err == nil {
		t.Fatal("expected error when binary is missing from zip")
	}
}

func TestEnsureXresconvCLIDownloadAndCacheReuse(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	fakeBin210 := buildFakeXresconvBinary(t, tmp, "2.1.0")
	fakeBin201 := buildFakeXresconvBinary(t, tmp, "2.0.1")
	fakeBin210Bytes, err := os.ReadFile(fakeBin210)
	if err != nil {
		t.Fatal(err)
	}

	binName := buildkit.BinName("xresconv-cli")
	var archiveBytes []byte
	var archivePath string
	if runtime.GOOS == "windows" {
		archivePath = "/xresconv-cli-2.1.0.zip"
		archiveBytes = buildTestZip(t, map[string][]byte{binName: fakeBin210Bytes})
	} else {
		archivePath = "/xresconv-cli-2.1.0.tar.gz"
		archiveBytes = buildTestTarGz(t, map[string][]byte{binName: fakeBin210Bytes})
	}
	sum := sha256.Sum256(archiveBytes)
	validSHA := hex.EncodeToString(sum[:]) + "  archive\n"

	var archiveHits atomic.Int32
	var serveBadSHA atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case archivePath:
			archiveHits.Add(1)
			_, _ = w.Write(archiveBytes)
		case archivePath + ".sha256":
			if serveBadSHA.Load() {
				_, _ = w.Write([]byte(strings.Repeat("0", 64) + "  archive\n"))
				return
			}
			_, _ = w.Write([]byte(validSHA))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Isolate PATH and override env vars from host environment.
	t.Setenv("PATH", filepath.Join(tmp, "empty-path"))
	t.Setenv("ATBUILD_XRESCONV_CLI", "")
	t.Setenv("XRESCONV_CLI_BIN", "")
	t.Setenv("ATBUILD_XRESCONV_CLI_URL", srv.URL+archivePath)
	t.Setenv("ATBUILD_XRESCONV_CLI_SHA256_URL", srv.URL+archivePath+".sha256")
	t.Setenv("ATBUILD_XRESCONV_CLI_SHA256", "")

	repoRoot := filepath.Join(tmp, "repo")
	userCache := filepath.Join(tmp, "user-cache")

	newEnv := func() *ProjectEnv {
		env := NewProjectEnv(repoRoot)
		env.XresconvCLIVersion = "2.1.0"
		env.XresconvUserCacheDir = userCache
		return env
	}

	// 1. InspectXresconvCLI on cold state must NOT download anything.
	if _, _, _, ok := newEnv().InspectXresconvCLI(ctx); ok {
		t.Fatal("expected cold InspectXresconvCLI to report not cached")
	}
	if archiveHits.Load() != 0 {
		t.Fatalf("InspectXresconvCLI triggered %d downloads, want 0", archiveHits.Load())
	}

	// 2. Checksum mismatch must fail without writing cache.
	serveBadSHA.Store(true)
	if _, err := newEnv().EnsureXresconvCLI(ctx); err == nil {
		t.Fatal("expected checksum mismatch error")
	}
	if archiveHits.Load() != 1 {
		t.Fatalf("expected 1 download attempt before sha256 failure, got %d", archiveHits.Load())
	}
	serveBadSHA.Store(false)

	// 3. Valid download caches into both project build/tools and persistent user cache.
	got1, err := newEnv().EnsureXresconvCLI(ctx)
	if err != nil {
		t.Fatalf("EnsureXresconvCLI download failed: %v", err)
	}
	if archiveHits.Load() != 2 {
		t.Fatalf("expected 2 total archive requests (1 failed + 1 valid), got %d", archiveHits.Load())
	}
	if !xresconvMatchesVersion(ctx, got1, "2.1.0") {
		t.Fatalf("downloaded binary %s did not report version 2.1.0", got1)
	}

	// 4. Subsequent run reuses project cache with ZERO HTTP downloads.
	got2, err := newEnv().EnsureXresconvCLI(ctx)
	if err != nil {
		t.Fatalf("EnsureXresconvCLI project cache hit failed: %v", err)
	}
	if got2 != got1 {
		t.Fatalf("project cache path changed: got %q, want %q", got2, got1)
	}
	if archiveHits.Load() != 2 {
		t.Fatalf("project cache miss: archiveHits=%d, want 2", archiveHits.Load())
	}

	// 5. Removing build/tools (like `clean -all`) still reuses persistent user cache with ZERO downloads.
	if err := os.RemoveAll(filepath.Join(repoRoot, "build")); err != nil {
		t.Fatal(err)
	}
	got3, err := newEnv().EnsureXresconvCLI(ctx)
	if err != nil {
		t.Fatalf("EnsureXresconvCLI user cache hit failed: %v", err)
	}
	if archiveHits.Load() != 2 {
		t.Fatalf("user cache miss after clean: archiveHits=%d, want 2", archiveHits.Load())
	}
	if !xresconvMatchesVersion(ctx, got3, "2.1.0") {
		t.Fatalf("restored binary %s did not report version 2.1.0", got3)
	}

	// 6. Upstream unversioned cache (<userCache>/bin/xresconv-cli) with matching version is reused.
	if err := os.RemoveAll(filepath.Join(repoRoot, "build")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(userCache); err != nil {
		t.Fatal(err)
	}
	upstreamBin := filepath.Join(userCache, "bin", binName)
	if err := atomicInstallBinary(upstreamBin, fakeBin210Bytes); err != nil {
		t.Fatal(err)
	}
	if _, err := newEnv().EnsureXresconvCLI(ctx); err != nil {
		t.Fatalf("EnsureXresconvCLI upstream cache hit failed: %v", err)
	}
	if archiveHits.Load() != 2 {
		t.Fatalf("upstream matching cache miss: archiveHits=%d, want 2", archiveHits.Load())
	}

	// 7. Outdated upstream unversioned cache (2.0.1) is NOT reused; downloads 2.1.0 instead.
	if err := os.RemoveAll(filepath.Join(repoRoot, "build")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(userCache); err != nil {
		t.Fatal(err)
	}
	fakeBin201Bytes, err := os.ReadFile(fakeBin201)
	if err != nil {
		t.Fatal(err)
	}
	if err := atomicInstallBinary(upstreamBin, fakeBin201Bytes); err != nil {
		t.Fatal(err)
	}
	got4, err := newEnv().EnsureXresconvCLI(ctx)
	if err != nil {
		t.Fatalf("EnsureXresconvCLI upgrade from stale upstream cache failed: %v", err)
	}
	if archiveHits.Load() != 3 {
		t.Fatalf("expected download when upstream cache is 2.0.1, got archiveHits=%d", archiveHits.Load())
	}
	if !xresconvMatchesVersion(ctx, got4, "2.1.0") {
		t.Fatalf("upgraded binary %s did not report version 2.1.0", got4)
	}

	// 8. Explicit env override ATBUILD_XRESCONV_CLI takes precedence with zero downloads.
	t.Setenv("ATBUILD_XRESCONV_CLI", fakeBin210)
	gotEnv, err := newEnv().EnsureXresconvCLI(ctx)
	if err != nil {
		t.Fatalf("ATBUILD_XRESCONV_CLI override failed: %v", err)
	}
	if gotEnv != filepath.Clean(fakeBin210) {
		t.Fatalf("got %q, want %q", gotEnv, fakeBin210)
	}
	if archiveHits.Load() != 3 {
		t.Fatalf("env override triggered download: archiveHits=%d", archiveHits.Load())
	}

	// 9. Missing ATBUILD_XRESCONV_CLI returns explicit error.
	t.Setenv("ATBUILD_XRESCONV_CLI", filepath.Join(tmp, "does-not-exist"))
	if _, err := newEnv().EnsureXresconvCLI(ctx); err == nil {
		t.Fatal("expected error for missing ATBUILD_XRESCONV_CLI path")
	}

	// 10. PATH lookup reuses matching binary with zero downloads.
	t.Setenv("ATBUILD_XRESCONV_CLI", "")
	if err := os.RemoveAll(filepath.Join(repoRoot, "build")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(userCache); err != nil {
		t.Fatal(err)
	}
	pathDir := filepath.Join(tmp, "path-bin")
	if err := atomicInstallBinary(filepath.Join(pathDir, binName), fakeBin210Bytes); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", pathDir)
	gotPath, err := newEnv().EnsureXresconvCLI(ctx)
	if err != nil {
		t.Fatalf("PATH lookup failed: %v", err)
	}
	if gotPath != filepath.Join(pathDir, binName) {
		t.Fatalf("PATH lookup got %q, want %q", gotPath, filepath.Join(pathDir, binName))
	}
	if archiveHits.Load() != 3 {
		t.Fatalf("PATH cache hit triggered download: archiveHits=%d", archiveHits.Load())
	}
}

func buildFakeXresconvBinary(t *testing.T, dir, version string) string {
	t.Helper()
	srcDir := filepath.Join(dir, "fake-"+version)
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := fmt.Sprintf("package main\nimport \"fmt\"\nfunc main() { fmt.Println(%q) }\n", version)
	srcFile := filepath.Join(srcDir, "main.go")
	if err := os.WriteFile(srcFile, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	outBin := filepath.Join(srcDir, buildkit.BinName("xresconv-cli"))
	cmd := exec.Command("go", "build", "-o", outBin, srcFile)
	cmd.Dir = srcDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fake xresconv-cli %s failed: %v\n%s", version, err, string(out))
	}
	return outBin
}

func buildTestZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func buildTestTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0o755,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
