// Copyright 2026 atframework
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/atframework/atframe-utils-go/buildkit"
	"golang.org/x/sync/errgroup"
)

// ServiceTarget represents an auto-discovered business process service under src/.
type ServiceTarget struct {
	Name      string
	AbsDir    string
	RelDir    string // slash-separated path relative to RootDir (e.g. "src/lobbysvr")
	HasProto  bool
	HasPbTpl  bool
	BinOutput string
}

// ProjectEnv holds all resolved paths and toolchain state in memory across pipeline stages.
type ProjectEnv struct {
	RootDir              string
	BuildDir             string
	GenDir               string
	CacheDir             string
	ToolsDir             string
	InstallDir           string
	PbdescDir            string
	ExcelOutDir          string
	ResourceDir          string
	ExcelSrcDir          string
	TemplateDir          string
	XresloaderDir        string
	ProtocVersion        string
	XresloaderVersion    string
	XresconvCLIVersion   string
	AtdtoolVersion       string
	Namespace            string
	XresconvUserCacheDir string // Optional override for persistent user cache (defaults to OS user cache dir)

	// Populated dynamically in memory by PrepareAll / EnsureProtoToolchain:
	ProtocBin        string
	ProtocGenGoBin   string
	ProtocGenMutable string
	JavaBin          string
	JavaMajor        int
	XresloaderJar    string
	XresconvCLIBin   string
	AtdtoolBin       string
	Python           *buildkit.PythonRunner
	ModuleDirs       map[string]string
	Cache            *buildkit.CacheStore
}

// NewProjectEnv constructs a ProjectEnv with standard directory conventions.
func NewProjectEnv(rootDir string) *ProjectEnv {
	buildDir := filepath.Join(rootDir, "build")
	installDir := filepath.Join(buildDir, "install")
	return &ProjectEnv{
		RootDir:            rootDir,
		BuildDir:           buildDir,
		GenDir:             filepath.Join(buildDir, "_gen"),
		CacheDir:           filepath.Join(rootDir, ".cache", "atbuild"),
		ToolsDir:           filepath.Join(buildDir, "tools"),
		InstallDir:         installDir,
		PbdescDir:          filepath.Join(installDir, "resource", "pbdesc"),
		ExcelOutDir:        filepath.Join(installDir, "resource", "excel"),
		ResourceDir:        filepath.Join(rootDir, "resource"),
		ExcelSrcDir:        filepath.Join(rootDir, "resource", "ExcelTables"),
		TemplateDir:        filepath.Join(rootDir, "src", "template"),
		XresloaderDir:      filepath.Join(rootDir, "third_party", "xresloader"),
		ProtocVersion:      envOrDefault("ENV_PROTOC_VERSION", buildkit.DefaultProtocVersion),
		XresloaderVersion:  envOrDefault("ENV_XRESLOADER_VERSION", buildkit.DefaultXresloaderVersion),
		XresconvCLIVersion: normalizeVersion(envOrDefault("ENV_XRESCONV_CLI_VERSION", DefaultXresconvCLIVersion)),
		AtdtoolVersion:     envOrDefault("ENV_ADTOOLS_VERSION", buildkit.DefaultAtdtoolVersion),
		Namespace:          "proy",
	}
}

func (e *ProjectEnv) ensureCache() (*buildkit.CacheStore, error) {
	if e.Cache != nil {
		return e.Cache, nil
	}
	store, err := buildkit.OpenCacheStore(filepath.Join(e.CacheDir, "atsf4g-manifest.json"))
	if err != nil {
		return nil, err
	}
	e.Cache = store
	return store, nil
}

// EnsureProtoToolchain resolves protoc, protoc-gen-go, protoc-gen-mutable, and Go module dirs.
func (e *ProjectEnv) EnsureProtoToolchain(ctx context.Context) error {
	if e.ProtocBin == "" {
		p, err := buildkit.EnsureProtoc(ctx, e.ProtocVersion, e.ToolsDir)
		if err != nil {
			return err
		}
		e.ProtocBin = p
	}
	binFallback := filepath.Join(e.ToolsDir, "bin")
	if e.ProtocGenGoBin == "" {
		p, err := buildkit.ResolveGoTool(ctx, e.RootDir, "protoc-gen-go", "", "google.golang.org/protobuf/cmd/protoc-gen-go", binFallback)
		if err != nil {
			return err
		}
		e.ProtocGenGoBin = p
	}
	if e.ProtocGenMutable == "" {
		p, err := buildkit.ResolveGoTool(ctx, e.RootDir, "protoc-gen-mutable", "", "github.com/atframework/atframe-utils-go/protoc-gen-mutable", binFallback)
		if err != nil {
			return err
		}
		e.ProtocGenMutable = p
	}
	if e.ModuleDirs == nil {
		dirs, err := buildkit.ResolveModuleDirs(ctx, e.RootDir)
		if err != nil {
			return err
		}
		e.ModuleDirs = dirs
	}
	return nil
}

// PrepareAll ensures all required build tools exist and are cached.
func (e *ProjectEnv) PrepareAll(ctx context.Context, includeExternal bool) error {
	for _, d := range []string{e.BuildDir, e.GenDir, e.ToolsDir, e.PbdescDir, e.ExcelOutDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	if err := e.EnsureProtoToolchain(ctx); err != nil {
		return err
	}
	if !includeExternal {
		return nil
	}

	if e.JavaBin == "" {
		javaBin, major, err := buildkit.ResolveJava(ctx, buildkit.DefaultJavaMinMajorVersion)
		if err != nil {
			return err
		}
		e.JavaBin = javaBin
		e.JavaMajor = major
	}

	if e.XresloaderJar == "" {
		xresURL := fmt.Sprintf("https://github.com/xresloader/xresloader/releases/download/v%s/xresloader-%s.jar",
			e.XresloaderVersion, e.XresloaderVersion)
		if override := strings.TrimSpace(os.Getenv("ATBUILD_XRESLOADER_URL")); override != "" {
			xresURL = override
		}
		jarPath, err := buildkit.EnsureDownloadAsset(ctx, buildkit.AssetSpec{
			Name:        "xresloader",
			Version:     e.XresloaderVersion,
			URL:         xresURL,
			ToolsDir:    e.ToolsDir,
			InstallPath: filepath.Join(e.InstallDir, "xresloader", "xresloader.jar"),
		})
		if err != nil {
			return err
		}
		e.XresloaderJar = jarPath
	}

	if e.XresconvCLIBin == "" {
		if _, err := e.EnsureXresconvCLI(ctx); err != nil {
			return err
		}
	}

	if e.AtdtoolBin == "" {
		ext := "tar.gz"
		if runtime.GOOS == "windows" {
			ext = "zip"
		}
		atdURL := fmt.Sprintf("https://github.com/atframework/atdtool/releases/download/v%s/atdtool-%s-%s.%s",
			e.AtdtoolVersion, runtime.GOOS, runtime.GOARCH, ext)
		if override := strings.TrimSpace(os.Getenv("ATBUILD_ATDTOOL_URL")); override != "" {
			atdURL = override
		}
		binPath, err := buildkit.EnsureDownloadAsset(ctx, buildkit.AssetSpec{
			Name:        "atdtool",
			Version:     e.AtdtoolVersion,
			URL:         atdURL,
			ToolsDir:    e.ToolsDir,
			InstallPath: filepath.Join(e.InstallDir, "atdtool", buildkit.BinName("atdtool")),
		})
		if err != nil {
			return err
		}
		e.AtdtoolBin = binPath
	}

	if e.Python == nil {
		py, err := buildkit.ResolvePythonRunner(ctx, e.RootDir, filepath.Join(e.RootDir, "third_party", "python_env", "requirements.txt"), "")
		if err != nil {
			return err
		}
		e.Python = py
	}

	// Write lightweight compatibility settings for helper scripts like doc/scripts/generate_proto_docs.py.
	return e.writeCompatSettings()
}

func (e *ProjectEnv) writeCompatSettings() error {
	type toolEntry struct {
		Path    string `json:"path"`
		Version string `json:"version"`
	}
	payload := struct {
		Tools map[string]toolEntry `json:"tools"`
	}{
		Tools: map[string]toolEntry{
			"protoc":       {Path: e.ProtocBin, Version: e.ProtocVersion},
			"java":         {Path: e.JavaBin, Version: fmt.Sprintf("%d", e.JavaMajor)},
			"xresloader":   {Path: e.XresloaderJar, Version: e.XresloaderVersion},
			"xresconv-cli": {Path: e.XresconvCLIBin, Version: e.XresconvCLIVersion},
			"atdtool":      {Path: e.AtdtoolBin, Version: e.AtdtoolVersion},
		},
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(e.BuildDir, "build-settings.json"), append(data, '\n'), 0o644)
}

// DiscoverServices auto-discovers all business server directories under src/<svc>
// or src/services/<svc> containing a main.go file (supporting 20+ services without config edits).
func (e *ProjectEnv) DiscoverServices() ([]ServiceTarget, error) {
	scanRoots := []string{
		filepath.Join(e.RootDir, "src"),
		filepath.Join(e.RootDir, "src", "services"),
	}
	skipNames := map[string]bool{
		"component": true,
		"template":  true,
		"services":  true,
	}

	seen := make(map[string]bool)
	var services []ServiceTarget
	for _, root := range scanRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", root, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() || skipNames[entry.Name()] {
				continue
			}
			svcDir := filepath.Join(root, entry.Name())
			if _, err := os.Stat(filepath.Join(svcDir, "main.go")); err != nil {
				continue
			}
			name := entry.Name()
			if seen[name] {
				continue
			}
			seen[name] = true
			rel, _ := filepath.Rel(e.RootDir, svcDir)
			binName := name
			if !strings.HasSuffix(binName, "d") {
				binName += "d"
			}
			_, protoErr := os.Stat(filepath.Join(svcDir, "protocol"))
			_, tplErr := os.Stat(filepath.Join(svcDir, "generate-for-pb.atfw.yaml.tpl"))
			services = append(services, ServiceTarget{
				Name:      name,
				AbsDir:    svcDir,
				RelDir:    filepath.ToSlash(rel),
				HasProto:  protoErr == nil,
				HasPbTpl:  tplErr == nil,
				BinOutput: filepath.Join(e.InstallDir, name, "bin", buildkit.BinName(binName)),
			})
		}
	}

	slices.SortFunc(services, func(a, b ServiceTarget) int {
		return strings.Compare(a.Name, b.Name)
	})
	return services, nil
}

func (e *ProjectEnv) filterServices(all []ServiceTarget, names []string) ([]ServiceTarget, error) {
	if len(names) == 0 {
		return all, nil
	}
	byName := make(map[string]ServiceTarget, len(all))
	for _, s := range all {
		byName[s.Name] = s
	}
	out := make([]ServiceTarget, 0, len(names))
	for _, n := range names {
		svc, ok := byName[n]
		if !ok {
			return nil, fmt.Errorf("unknown service %q (discovered: %d services)", n, len(all))
		}
		out = append(out, svc)
	}
	return out, nil
}

// BuildOptions controls Go binary compilation for business services.
type BuildOptions struct {
	Services      []string
	Mode          string
	Version       string
	ConfigVersion string
	Branch        string
	Race          bool
}

// BuildServices compiles all selected services concurrently into build/install/<svc>/bin/.
func (e *ProjectEnv) BuildServices(ctx context.Context, opts BuildOptions) error {
	allSvcs, err := e.DiscoverServices()
	if err != nil {
		return err
	}
	svcs, err := e.filterServices(allSvcs, opts.Services)
	if err != nil {
		return err
	}
	if len(svcs) == 0 {
		return fmt.Errorf("no business services discovered under src/")
	}

	commit := gitOutput(ctx, e.RootDir, "rev-parse", "HEAD")
	if commit == "" {
		commit = "unknown"
	}
	branch := opts.Branch
	if branch == "" {
		branch = gitOutput(ctx, e.RootDir, "rev-parse", "--abbrev-ref", "HEAD")
		if branch == "" {
			branch = "unknown"
		}
	}
	buildTime := time.Now().UTC().Format(time.RFC3339)

	isDebug := !strings.EqualFold(opts.Mode, "Release")
	modeLabel := "Release"
	buildTag := "release"
	if isDebug {
		modeLabel = "Debug"
		buildTag = "debug"
	}

	ldflags := strings.Join([]string{
		"-X", "github.com/atframework/libatapp-go.Version=" + opts.Version,
		"-X", "github.com/atframework/libatapp-go.Commit=" + commit,
		"-X", "github.com/atframework/libatapp-go.Branch=" + branch,
		"-X", "github.com/atframework/libatapp-go.BuildTime=" + buildTime,
		"-X", "github.com/atframework/libatapp-go.ConfigVer=" + opts.ConfigVersion,
		"-X", "github.com/atframework/libatapp-go.BuildMode=" + modeLabel,
	}, " ")

	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(max(1, runtime.NumCPU()/2))
	for _, svc := range svcs {
		s := svc
		eg.Go(func() error {
			if err := os.MkdirAll(filepath.Dir(s.BinOutput), 0o755); err != nil {
				return err
			}
			args := []string{"build", "-tags=" + buildTag, "-ldflags=" + ldflags}
			if isDebug {
				args = append(args, "-gcflags=all=-N -l")
			}
			if opts.Race || (isDebug && runtime.GOOS != "windows") {
				args = append(args, "-race")
			}
			args = append(args, "-o", s.BinOutput, "./"+s.RelDir)

			fmt.Printf("[build] %s (%s) -> %s\n", s.Name, modeLabel, s.BinOutput)
			cmd := exec.CommandContext(egCtx, "go", args...)
			cmd.Dir = e.RootDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("go build %s failed: %w", s.Name, err)
			}
			return nil
		})
	}
	return eg.Wait()
}

// InstallDeployAssets copies cloud-native charts/values and top-level launch scripts into build/install/.
func (e *ProjectEnv) InstallDeployAssets() error {
	cloudSrc := filepath.Join(e.RootDir, "install", "cloud-native")
	if _, err := os.Stat(cloudSrc); err == nil {
		if err := buildkit.CopyDir(cloudSrc, filepath.Join(e.InstallDir, "deploy")); err != nil {
			return fmt.Errorf("copy cloud-native deploy assets: %w", err)
		}
	}
	scriptSrc := filepath.Join(e.RootDir, "script")
	if _, err := os.Stat(scriptSrc); err == nil {
		if err := buildkit.CopyTopLevelFiles(scriptSrc, e.InstallDir); err != nil {
			return fmt.Errorf("copy launch scripts: %w", err)
		}
	}
	return nil
}

// VerifySubmodulesStandalone runs GOWORK=off build/test across the 3 standalone atframework repositories.
func (e *ProjectEnv) VerifySubmodulesStandalone(ctx context.Context, runTests bool) error {
	submodules := []string{
		filepath.Join(e.RootDir, "atframework", "atframe-utils-go"),
		filepath.Join(e.RootDir, "atframework", "atframe-utils-go", "buildkit"),
		filepath.Join(e.RootDir, "atframework", "libatbus-go"),
		filepath.Join(e.RootDir, "atframework", "libatbus-go", "sample", "sample_usage_01"),
		filepath.Join(e.RootDir, "atframework", "libatapp-go"),
	}
	if err := buildkit.VerifyStandalone(ctx, submodules, runTests); err != nil {
		return err
	}
	// Verify integration sub-module compiles standalone (tests require embedded etcd runtime so build-verify by default).
	integrationDir := filepath.Join(e.RootDir, "atframework", "libatapp-go", "etcd_module_v2", "integration")
	return buildkit.VerifyStandalone(ctx, []string{integrationDir}, false)
}

// BuildDocs runs doc/scripts/generate_proto_docs.py and mkdocs build via uv (with docs dependency group).
func (e *ProjectEnv) BuildDocs(ctx context.Context) error {
	if err := e.EnsureProtoToolchain(ctx); err != nil {
		return err
	}
	_ = e.writeCompatSettings()

	py, err := buildkit.ResolvePythonRunner(ctx, e.RootDir, filepath.Join(e.RootDir, "doc", "requirements.txt"), "docs")
	if err != nil {
		return err
	}

	docDir := filepath.Join(e.RootDir, "doc")
	envVars := append(os.Environ(), "PATH="+filepath.Dir(e.ProtocBin)+string(os.PathListSeparator)+os.Getenv("PATH"))

	genCmd := py.ExecCmd(ctx, filepath.Join(docDir, "scripts", "generate_proto_docs.py"))
	genCmd.Dir = docDir
	genCmd.Env = envVars
	genCmd.Stdout = os.Stdout
	genCmd.Stderr = os.Stderr
	if err := genCmd.Run(); err != nil {
		return fmt.Errorf("generate_proto_docs.py failed: %w", err)
	}

	mkdocsCmd := py.ExecCmd(ctx, "-m", "mkdocs", "build")
	mkdocsCmd.Dir = docDir
	mkdocsCmd.Env = envVars
	mkdocsCmd.Stdout = os.Stdout
	mkdocsCmd.Stderr = os.Stderr
	if err := mkdocsCmd.Run(); err != nil {
		return fmt.Errorf("mkdocs build failed: %w", err)
	}
	return nil
}

// SyncProtoToSVN copies public .proto definitions from component and all discovered services to SVN client path.
func (e *ProjectEnv) SyncProtoToSVN(envFile string) error {
	if !filepath.IsAbs(envFile) {
		envFile = filepath.Join(e.RootDir, envFile)
	}
	data, err := os.ReadFile(envFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", envFile, err)
	}
	var cfg struct {
		ClientProtocolPath string `json:"client.protocol-path"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse %s: %w", envFile, err)
	}
	if cfg.ClientProtocolPath == "" {
		return fmt.Errorf("client.protocol-path is empty in %s", envFile)
	}

	type copyPair struct{ src, dst string }
	pairs := []copyPair{
		{filepath.Join(e.RootDir, "src", "component", "protocol", "public", "common", "protocol", "common"), filepath.Join(cfg.ClientProtocolPath, "common")},
		{filepath.Join(e.RootDir, "src", "component", "protocol", "public", "config", "protocol", "config"), filepath.Join(cfg.ClientProtocolPath, "config")},
		{filepath.Join(e.RootDir, "src", "component", "protocol", "public", "pbdesc", "protocol", "pbdesc"), filepath.Join(cfg.ClientProtocolPath, "pbdesc")},
	}
	svcs, err := e.DiscoverServices()
	if err != nil {
		return err
	}
	for _, s := range svcs {
		pubDir := filepath.Join(s.AbsDir, "protocol", "public", "protocol", "pbdesc")
		if _, statErr := os.Stat(pubDir); statErr == nil {
			pairs = append(pairs, copyPair{pubDir, filepath.Join(cfg.ClientProtocolPath, "pbdesc")})
		}
	}

	cleanedDst := make(map[string]bool)
	for _, pair := range pairs {
		if !cleanedDst[pair.dst] {
			cleanedDst[pair.dst] = true
			if entries, err := os.ReadDir(pair.dst); err == nil {
				for _, ent := range entries {
					if !ent.IsDir() {
						_ = os.Remove(filepath.Join(pair.dst, ent.Name()))
					}
				}
			}
		}
	}
	for _, pair := range pairs {
		entries, err := os.ReadDir(pair.src)
		if err != nil {
			return fmt.Errorf("read %s: %w", pair.src, err)
		}
		for _, ent := range entries {
			if !ent.IsDir() && strings.HasSuffix(ent.Name(), ".proto") {
				if err := buildkit.CopyFile(filepath.Join(pair.src, ent.Name()), filepath.Join(pair.dst, ent.Name())); err != nil {
					return err
				}
			}
		}
	}
	fmt.Printf("[sync-proto-svn] synced protocols to %s\n", cfg.ClientProtocolPath)
	return nil
}

// Clean removes generated Go/pb files, caches, or full build outputs.
func (e *ProjectEnv) Clean(all, pb, cache bool) error {
	if pb || all {
		srcDir := filepath.Join(e.RootDir, "src")
		removed, err := buildkit.RemoveMatchingFiles(srcDir, func(p, name string) bool {
			if strings.HasSuffix(name, ".pb.go") {
				return true
			}
			slash := filepath.ToSlash(p)
			if strings.HasSuffix(name, ".go") && strings.Contains(slash, "/config/generate_config/") {
				return true
			}
			return false
		})
		if err != nil {
			return err
		}
		fmt.Printf("[clean] removed %d generated Go/pb files under src/\n", removed)
		_ = os.RemoveAll(e.GenDir)
	}
	if cache || all {
		fmt.Printf("[clean] removing %s\n", e.CacheDir)
		_ = os.RemoveAll(e.CacheDir)
	}
	if all {
		fmt.Printf("[clean] removing %s\n", e.BuildDir)
		if err := os.RemoveAll(e.BuildDir); err != nil {
			return err
		}
	}
	return nil
}

func gitOutput(ctx context.Context, dir string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
