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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/atframework/atframe-utils-go/buildkit"
)

const (
	// DefaultXresconvCLIVersion is the pinned prebuilt Rust release of owent/xresconv-cli.
	DefaultXresconvCLIVersion = "2.1.0"

	maxXresconvArchiveSize      = 128 * 1024 * 1024
	maxXresconvBinarySize       = 128 * 1024 * 1024
	maxXresconvChecksumSize     = 8 * 1024
	xresconvDownloadTimeout     = 120 * time.Second
	xresconvVersionProbeTimeout = 10 * time.Second
)

func normalizeVersion(v string) string {
	s := strings.TrimSpace(v)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	return s
}

// xresconvAssetSpec maps a version and Go OS/Arch pair to the official GitHub release asset.
// On Linux x64/arm64, musl static builds are preferred to avoid host glibc version mismatches.
func xresconvAssetSpec(version, goos, goarch string) (target, ext, filename, downloadURL string, err error) {
	cleanVer := normalizeVersion(version)
	if cleanVer == "" {
		cleanVer = DefaultXresconvCLIVersion
	}

	switch goos {
	case "windows":
		switch goarch {
		case "amd64":
			target, ext = "x86_64-pc-windows-msvc", ".zip"
		case "arm64":
			target, ext = "aarch64-pc-windows-msvc", ".zip"
		}
	case "linux":
		switch goarch {
		case "amd64":
			target, ext = "x86_64-unknown-linux-musl", ".tar.gz"
		case "arm64":
			target, ext = "aarch64-unknown-linux-musl", ".tar.gz"
		case "riscv64":
			target, ext = "riscv64gc-unknown-linux-gnu", ".tar.gz"
		}
	case "darwin":
		switch goarch {
		case "amd64":
			target, ext = "x86_64-apple-darwin", ".tar.gz"
		case "arm64":
			target, ext = "aarch64-apple-darwin", ".tar.gz"
		}
	case "freebsd":
		if goarch == "amd64" {
			target, ext = "x86_64-unknown-freebsd", ".tar.gz"
		}
	case "android":
		if goarch == "arm64" {
			target, ext = "aarch64-linux-android", ".tar.gz"
		}
	}

	if target == "" {
		return "", "", "", "", fmt.Errorf(
			"unsupported platform %s/%s for xresconv-cli %s (set XRESCONV_CLI_BIN or ATBUILD_XRESCONV_CLI to use a custom binary)",
			goos, goarch, cleanVer,
		)
	}

	filename = fmt.Sprintf("xresconv-cli-%s-%s%s", cleanVer, target, ext)
	downloadURL = fmt.Sprintf("https://github.com/owent/xresconv-cli/releases/download/v%s/%s", cleanVer, filename)
	return target, ext, filename, downloadURL, nil
}

func xresconvUserCacheBase(override string) string {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return filepath.Clean(trimmed)
	}
	if runtime.GOOS == "windows" {
		if localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); localAppData != "" {
			return filepath.Join(localAppData, "xresconv-cli")
		}
	} else {
		if xdg := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME")); xdg != "" {
			return filepath.Join(xdg, "xresconv-cli")
		}
	}
	if ucd, err := os.UserCacheDir(); err == nil && strings.TrimSpace(ucd) != "" {
		return filepath.Join(ucd, "xresconv-cli")
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".cache", "xresconv-cli")
	}
	return ""
}

func xresconvUpstreamUnversionedBins(override string) []string {
	binName := buildkit.BinName("xresconv-cli")
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return []string{filepath.Join(filepath.Clean(trimmed), "bin", binName)}
	}

	var candidates []string
	addCandidate := func(p string) {
		p = filepath.Clean(p)
		for _, existing := range candidates {
			if existing == p {
				return
			}
		}
		candidates = append(candidates, p)
	}

	if base := xresconvUserCacheBase(""); base != "" {
		addCandidate(filepath.Join(base, "bin", binName))
	}
	// On macOS, upstream xresconv-cli's legacy Python shim cached under ~/.cache/xresconv-cli/bin
	// rather than ~/Library/Caches/xresconv-cli/bin.
	if runtime.GOOS != "windows" {
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			addCandidate(filepath.Join(home, ".cache", "xresconv-cli", "bin", binName))
		}
	}
	return candidates
}

// InspectXresconvCLI checks whether a matching xresconv-cli binary is already available
// locally without downloading or modifying any files.
func (e *ProjectEnv) InspectXresconvCLI(ctx context.Context) (binPath, version, source string, ok bool) {
	wantVer := normalizeVersion(e.XresconvCLIVersion)
	if wantVer == "" {
		wantVer = DefaultXresconvCLIVersion
	}

	if envPath := strings.TrimSpace(os.Getenv("ATBUILD_XRESCONV_CLI")); envPath != "" {
		if isRegularFile(envPath) {
			ver, _ := probeXresconvVersion(ctx, envPath)
			return filepath.Clean(envPath), ver, "ATBUILD_XRESCONV_CLI", true
		}
	}
	if envPath := strings.TrimSpace(os.Getenv("XRESCONV_CLI_BIN")); envPath != "" {
		if isRegularFile(envPath) {
			ver, _ := probeXresconvVersion(ctx, envPath)
			return filepath.Clean(envPath), ver, "XRESCONV_CLI_BIN", true
		}
	}

	platformTag := runtime.GOOS + "-" + runtime.GOARCH
	binName := buildkit.BinName("xresconv-cli")

	projDir := filepath.Join(e.ToolsDir, "xresconv-cli", wantVer, platformTag)
	projBin := filepath.Join(projDir, binName)
	projReady := filepath.Join(projDir, ".ready")
	if isRegularFile(projBin) && (isRegularFile(projReady) || xresconvMatchesVersion(ctx, projBin, wantVer)) {
		return filepath.Clean(projBin), wantVer, "project-cache", true
	}

	if userBase := xresconvUserCacheBase(e.XresconvUserCacheDir); userBase != "" {
		userDir := filepath.Join(userBase, wantVer, platformTag)
		userBin := filepath.Join(userDir, binName)
		userReady := filepath.Join(userDir, ".ready")
		if isRegularFile(userBin) && (isRegularFile(userReady) || xresconvMatchesVersion(ctx, userBin, wantVer)) {
			return filepath.Clean(userBin), wantVer, "user-cache", true
		}
	}

	for _, cand := range xresconvUpstreamUnversionedBins(e.XresconvUserCacheDir) {
		if isRegularFile(cand) && xresconvMatchesVersion(ctx, cand, wantVer) {
			return filepath.Clean(cand), wantVer, "upstream-user-cache", true
		}
	}

	if pathBin, err := exec.LookPath("xresconv-cli"); err == nil && xresconvMatchesVersion(ctx, pathBin, wantVer) {
		return filepath.Clean(pathBin), wantVer, "PATH", true
	}

	return "", "", "", false
}

// EnsureXresconvCLI resolves the native xresconv-cli binary for the current platform,
// reusing local/user caches or PATH whenever a matching version is already present,
// and downloading + SHA-256 verifying from GitHub Releases only on cache miss.
//
// Resolution order:
//  1. Already resolved in memory (e.XresconvCLIBin)
//  2. ATBUILD_XRESCONV_CLI (error if explicitly set to a missing path)
//  3. XRESCONV_CLI_BIN (upstream-compatible; warns and falls back if missing)
//  4. Project tool cache: build/tools/xresconv-cli/<version>/<os>-<arch>/xresconv-cli(.exe)
//  5. User versioned cache: <userCacheDir>/xresconv-cli/<version>/<os>-<arch>/xresconv-cli(.exe)
//  6. Upstream unversioned cache: <userCacheDir>/xresconv-cli/bin/xresconv-cli(.exe) (version matched)
//  7. System PATH (version matched)
//  8. On-demand download for current platform (+ SHA-256 verification) into project & user cache
func (e *ProjectEnv) EnsureXresconvCLI(ctx context.Context) (string, error) {
	if e.XresconvCLIBin != "" && isRegularFile(e.XresconvCLIBin) {
		return filepath.Clean(e.XresconvCLIBin), nil
	}

	wantVer := normalizeVersion(e.XresconvCLIVersion)
	if wantVer == "" {
		wantVer = DefaultXresconvCLIVersion
	}

	if envPath := strings.TrimSpace(os.Getenv("ATBUILD_XRESCONV_CLI")); envPath != "" {
		if isRegularFile(envPath) {
			e.XresconvCLIBin = filepath.Clean(envPath)
			return e.XresconvCLIBin, nil
		}
		return "", fmt.Errorf("ATBUILD_XRESCONV_CLI points to non-existent file: %s", envPath)
	}

	if envPath := strings.TrimSpace(os.Getenv("XRESCONV_CLI_BIN")); envPath != "" {
		if isRegularFile(envPath) {
			e.XresconvCLIBin = filepath.Clean(envPath)
			return e.XresconvCLIBin, nil
		}
		fmt.Fprintf(os.Stderr, "[dev] WARNING: XRESCONV_CLI_BIN not found: %s; falling back to cache/download\n", envPath)
	}

	platformTag := runtime.GOOS + "-" + runtime.GOARCH
	binName := buildkit.BinName("xresconv-cli")

	projDir := filepath.Join(e.ToolsDir, "xresconv-cli", wantVer, platformTag)
	projBin := filepath.Join(projDir, binName)
	projReady := filepath.Join(projDir, ".ready")

	// 1) Project-local version+platform cache (fast os.Stat hit when .ready is present).
	if isRegularFile(projBin) {
		if isRegularFile(projReady) {
			ensureExecutableBit(projBin)
			e.XresconvCLIBin = filepath.Clean(projBin)
			return e.XresconvCLIBin, nil
		}
		if xresconvMatchesVersion(ctx, projBin, wantVer) {
			ensureExecutableBit(projBin)
			_ = writeReadyMarker(projReady, projBin)
			e.XresconvCLIBin = filepath.Clean(projBin)
			return e.XresconvCLIBin, nil
		}
	}

	// 2) Persistent user-level versioned cache (survives `go tool dev clean -all` and shared across worktrees).
	userBase := xresconvUserCacheBase(e.XresconvUserCacheDir)
	var userDir, userBin, userReady string
	if userBase != "" {
		userDir = filepath.Join(userBase, wantVer, platformTag)
		userBin = filepath.Join(userDir, binName)
		userReady = filepath.Join(userDir, ".ready")
		if isRegularFile(userBin) && (isRegularFile(userReady) || xresconvMatchesVersion(ctx, userBin, wantVer)) {
			ensureExecutableBit(userBin)
			if !isRegularFile(userReady) {
				_ = writeReadyMarker(userReady, userBin)
			}
			fmt.Printf("[dev] reusing cached xresconv-cli %s from %s\n", wantVer, userBin)
			if err := materializeCachedBinary(userBin, projBin, projReady); err == nil {
				e.XresconvCLIBin = filepath.Clean(projBin)
				return e.XresconvCLIBin, nil
			}
			e.XresconvCLIBin = filepath.Clean(userBin)
			return e.XresconvCLIBin, nil
		}
	}

	// 3) Upstream unversioned user cache (only reused when exact version matches).
	for _, cand := range xresconvUpstreamUnversionedBins(e.XresconvUserCacheDir) {
		if isRegularFile(cand) && xresconvMatchesVersion(ctx, cand, wantVer) {
			fmt.Printf("[dev] reusing cached xresconv-cli %s from %s\n", wantVer, cand)
			if userBin != "" {
				_ = materializeCachedBinary(cand, userBin, userReady)
			}
			if err := materializeCachedBinary(cand, projBin, projReady); err == nil {
				e.XresconvCLIBin = filepath.Clean(projBin)
				return e.XresconvCLIBin, nil
			}
			e.XresconvCLIBin = filepath.Clean(cand)
			return e.XresconvCLIBin, nil
		}
	}

	// 4) System PATH (only reused when exact version matches).
	if pathBin, err := exec.LookPath("xresconv-cli"); err == nil && xresconvMatchesVersion(ctx, pathBin, wantVer) {
		fmt.Printf("[dev] using xresconv-cli %s from PATH (%s)\n", wantVer, pathBin)
		e.XresconvCLIBin = filepath.Clean(pathBin)
		return e.XresconvCLIBin, nil
	}

	// 5) On-demand download for the current platform with SHA-256 verification.
	assetURL := strings.TrimSpace(os.Getenv("ATBUILD_XRESCONV_CLI_URL"))
	shaURL := strings.TrimSpace(os.Getenv("ATBUILD_XRESCONV_CLI_SHA256_URL"))
	expectedSHA := strings.TrimSpace(os.Getenv("ATBUILD_XRESCONV_CLI_SHA256"))
	var archiveExt string

	if assetURL != "" {
		lowerURL := strings.ToLower(assetURL)
		switch {
		case strings.HasSuffix(lowerURL, ".zip"):
			archiveExt = ".zip"
		case strings.HasSuffix(lowerURL, ".tar.gz"), strings.HasSuffix(lowerURL, ".tgz"):
			archiveExt = ".tar.gz"
		default:
			archiveExt = ""
		}
	} else {
		var err error
		_, archiveExt, _, assetURL, err = xresconvAssetSpec(wantVer, runtime.GOOS, runtime.GOARCH)
		if err != nil {
			return "", err
		}
		if expectedSHA == "" && shaURL == "" {
			shaURL = assetURL + ".sha256"
		}
	}

	fmt.Printf("[dev] downloading xresconv-cli %s (%s) from %s\n", wantVer, platformTag, assetURL)
	archiveData, err := downloadBytesWithRetry(ctx, assetURL, maxXresconvArchiveSize, 3)
	if err != nil {
		return "", fmt.Errorf("download xresconv-cli %s: %w", wantVer, err)
	}

	if expectedSHA != "" || shaURL != "" {
		wantDigest := expectedSHA
		if wantDigest == "" {
			shaBytes, err := downloadBytesWithRetry(ctx, shaURL, maxXresconvChecksumSize, 3)
			if err != nil {
				return "", fmt.Errorf("download xresconv-cli %s sha256: %w", wantVer, err)
			}
			wantDigest, err = parseSHA256Checksum(shaBytes)
			if err != nil {
				return "", fmt.Errorf("parse xresconv-cli %s sha256 from %s: %w", wantVer, shaURL, err)
			}
		} else {
			wantDigest, err = parseSHA256Checksum([]byte(wantDigest))
			if err != nil {
				return "", fmt.Errorf("invalid ATBUILD_XRESCONV_CLI_SHA256: %w", err)
			}
		}
		if err := verifySHA256(archiveData, wantDigest); err != nil {
			return "", fmt.Errorf("verify xresconv-cli %s checksum: %w", wantVer, err)
		}
	}

	var binData []byte
	switch archiveExt {
	case ".zip":
		binData, err = extractBinaryFromZip(archiveData, binName)
	case ".tar.gz":
		binData, err = extractBinaryFromTarGz(archiveData, binName)
	default:
		binData = archiveData
	}
	if err != nil {
		return "", fmt.Errorf("extract xresconv-cli %s: %w", wantVer, err)
	}

	if err := atomicInstallBinary(projBin, binData); err != nil {
		return "", fmt.Errorf("install xresconv-cli %s to %s: %w", wantVer, projBin, err)
	}
	if err := writeReadyMarker(projReady, projBin); err != nil {
		return "", fmt.Errorf("write xresconv-cli ready marker: %w", err)
	}

	// Best-effort mirror into persistent user cache so clean builds and sibling worktrees skip re-downloading.
	if userBin != "" {
		if err := atomicInstallBinary(userBin, binData); err == nil {
			_ = writeReadyMarker(userReady, userBin)
		}
	}

	e.XresconvCLIBin = filepath.Clean(projBin)
	return e.XresconvCLIBin, nil
}

func probeXresconvVersion(ctx context.Context, binPath string) (string, error) {
	probeCtx, cancel := context.WithTimeout(ctx, xresconvVersionProbeTimeout)
	defer cancel()

	out, err := exec.CommandContext(probeCtx, binPath, "--version").Output()
	if err != nil {
		return "", err
	}
	ver := parseXresconvVersion(string(out))
	if ver == "" {
		return "", fmt.Errorf("empty version output from %s", binPath)
	}
	return ver, nil
}

func parseXresconvVersion(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		token := fields[len(fields)-1]
		return normalizeVersion(token)
	}
	return ""
}

func xresconvMatchesVersion(ctx context.Context, binPath, want string) bool {
	got, err := probeXresconvVersion(ctx, binPath)
	if err != nil {
		return false
	}
	cleanWant := normalizeVersion(want)
	if cleanWant == "" {
		return got != ""
	}
	return got == cleanWant
}

func parseSHA256Checksum(content []byte) (string, error) {
	fields := strings.Fields(string(content))
	if len(fields) == 0 {
		return "", errors.New("empty checksum content")
	}
	digest := strings.ToLower(fields[0])
	if len(digest) != 64 {
		return "", fmt.Errorf("unexpected sha256 length %d", len(digest))
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", fmt.Errorf("invalid sha256 hex digest: %w", err)
	}
	return digest, nil
}

func verifySHA256(data []byte, expectedHex string) error {
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if actual != strings.ToLower(expectedHex) {
		return fmt.Errorf("sha256 mismatch: got %s, want %s", actual, strings.ToLower(expectedHex))
	}
	return nil
}

func downloadBytesWithRetry(ctx context.Context, url string, maxSize int64, attempts int) ([]byte, error) {
	if attempts < 1 {
		attempts = 1
	}
	client := &http.Client{Timeout: xresconvDownloadTimeout}
	var lastErr error

	for i := 0; i < attempts; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "atsf4g-go-dev")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if i+1 < attempts {
				if waitErr := sleepWithContext(ctx, time.Duration(1<<uint(i))*time.Second); waitErr != nil {
					return nil, waitErr
				}
			}
			continue
		}

		body, readErr := readLimited(resp.Body, maxSize)
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusOK && readErr == nil {
			return body, nil
		}
		if readErr != nil {
			lastErr = readErr
		} else {
			lastErr = fmt.Errorf("HTTP %s", resp.Status)
		}

		// Do not retry non-recoverable 4xx responses except 429 Too Many Requests.
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			break
		}
		if i+1 < attempts {
			wait := retryDelay(resp.Header.Get("Retry-After"), i)
			if waitErr := sleepWithContext(ctx, wait); waitErr != nil {
				return nil, waitErr
			}
		}
	}
	return nil, lastErr
}

func readLimited(r io.Reader, maxSize int64) ([]byte, error) {
	if maxSize <= 0 {
		return io.ReadAll(r)
	}
	data, err := io.ReadAll(io.LimitReader(r, maxSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxSize {
		return nil, fmt.Errorf("response exceeds limit of %d bytes", maxSize)
	}
	return data, nil
}

func retryDelay(retryAfterHeader string, attempt int) time.Duration {
	if secs, err := strconv.Atoi(strings.TrimSpace(retryAfterHeader)); err == nil && secs > 0 {
		d := time.Duration(secs) * time.Second
		if d > 30*time.Second {
			return 30 * time.Second
		}
		return d
	}
	return time.Duration(1<<uint(attempt)) * time.Second
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func extractBinaryFromZip(archiveData []byte, binName string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archiveData), int64(len(archiveData)))
	if err != nil {
		return nil, err
	}

	var exact *zip.File
	var fallback *zip.File
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || f.Mode()&os.ModeSymlink != 0 {
			continue
		}
		cleaned, ok := cleanSafeArchivePath(f.Name)
		if !ok {
			continue
		}
		if cleaned == binName {
			if exact != nil {
				return nil, fmt.Errorf("multiple %s entries in zip archive", binName)
			}
			exact = f
		} else if path.Base(cleaned) == binName {
			if fallback == nil {
				fallback = f
			}
		}
	}

	target := exact
	if target == nil {
		target = fallback
	}
	if target == nil {
		return nil, fmt.Errorf("binary %q not found in zip archive", binName)
	}
	if target.UncompressedSize64 == 0 || target.UncompressedSize64 > maxXresconvBinarySize {
		return nil, fmt.Errorf("invalid binary size %d in zip archive", target.UncompressedSize64)
	}

	rc, err := target.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	data, err := readLimited(rc, maxXresconvBinarySize)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("binary %q is empty in zip archive", binName)
	}
	return data, nil
}

func extractBinaryFromTarGz(archiveData []byte, binName string) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(archiveData))
	if err != nil {
		return nil, err
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	var exactData []byte
	var fallbackData []byte

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		cleaned, ok := cleanSafeArchivePath(hdr.Name)
		if !ok {
			continue
		}
		isExact := cleaned == binName
		isFallback := !isExact && path.Base(cleaned) == binName && fallbackData == nil
		if !isExact && !isFallback {
			continue
		}
		if hdr.Size <= 0 || hdr.Size > maxXresconvBinarySize {
			return nil, fmt.Errorf("invalid binary size %d in tar.gz archive", hdr.Size)
		}
		data, err := readLimited(tr, maxXresconvBinarySize)
		if err != nil {
			return nil, err
		}
		if isExact {
			if exactData != nil {
				return nil, fmt.Errorf("multiple %s entries in tar.gz archive", binName)
			}
			exactData = data
		} else {
			fallbackData = data
		}
	}

	if exactData != nil {
		return exactData, nil
	}
	if fallbackData != nil {
		return fallbackData, nil
	}
	return nil, fmt.Errorf("binary %q not found in tar.gz archive", binName)
}

func cleanSafeArchivePath(raw string) (string, bool) {
	normalized := strings.ReplaceAll(raw, "\\", "/")
	normalized = strings.TrimPrefix(normalized, "./")
	if normalized == "" || strings.HasPrefix(normalized, "/") {
		return "", false
	}
	cleaned := path.Clean(normalized)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func atomicInstallBinary(destPath string, data []byte) error {
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".xresconv-cli-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	ensureExecutableBit(tmpPath)

	if err := os.Rename(tmpPath, destPath); err != nil {
		// On Windows, a concurrent or currently executing identical binary cannot be overwritten;
		// succeed cleanly if the existing file already has identical SHA-256 content.
		if existing, readErr := os.ReadFile(destPath); readErr == nil && sha256.Sum256(existing) == sha256.Sum256(data) {
			return nil
		}
		return err
	}
	cleanupTmp = false
	return nil
}

func materializeCachedBinary(srcPath, destPath, readyPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	if err := atomicInstallBinary(destPath, data); err != nil {
		return err
	}
	return writeReadyMarker(readyPath, destPath)
}

func writeReadyMarker(readyPath, binPath string) error {
	if err := os.MkdirAll(filepath.Dir(readyPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(readyPath, []byte(filepath.Clean(binPath)+"\n"), 0o644)
}

func ensureExecutableBit(p string) {
	if runtime.GOOS != "windows" {
		_ = os.Chmod(p, 0o755)
	}
}

func isRegularFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}
