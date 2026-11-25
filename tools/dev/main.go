// Copyright 2026 atframework
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/atframework/atframe-utils-go/buildkit"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	ctx := context.Background()
	rootDir, err := locateProjectRoot()
	if err != nil {
		fatalf("locate project root: %v", err)
	}

	env := NewProjectEnv(rootDir)
	sub := os.Args[1]
	args := os.Args[2:]

	switch sub {
	case "prepare":
		err = cmdPrepare(ctx, env, args)
	case "proto":
		err = cmdProto(ctx, env, args)
	case "config":
		err = cmdConfig(ctx, env, args)
	case "codegen":
		err = cmdCodegen(ctx, env, args)
	case "build":
		err = cmdBuild(ctx, env, args)
	case "test":
		err = cmdTest(ctx, env, args)
	case "lint":
		err = cmdLint(ctx, env, args)
	case "verify-standalone":
		err = cmdVerifyStandalone(ctx, env, args)
	case "docs":
		err = cmdDocs(ctx, env, args)
	case "sync-proto-svn":
		err = cmdSyncProtoSVN(env, args)
	case "clean":
		err = cmdClean(env, args)
	case "ci":
		err = cmdCI(ctx, env, args)
	case "doctor":
		err = cmdDoctor(ctx, env)
	case "-h", "--help", "help":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", sub)
		printUsage()
		os.Exit(2)
	}

	if err != nil {
		fatalf("%v", err)
	}
}

func printUsage() {
	fmt.Print(`dev - unified cross-platform Go build, codegen, lint, and CI driver for atsf4g-go

Usage:
  go tool dev <command> [flags]

Commands:
  prepare            Verify/download build tools (protoc, plugins, xresloader, atdtool, Java, uv/Python)
  proto              Generate protobuf & *_mutable.pb.go + .pb descriptors (incremental SHA-256 cached)
  config             Convert Excel configs via xresloader & generate Go config loaders (cached)
  codegen            Generate RPC/DB Go files from .pb descriptors & Mako templates (cached)
  build              Incremental codegen + parallel Go build for discovered services + deploy packaging
  test               Run Go unit tests (supports --affected <git-ref>, --race, --with-submodules)
  lint               Run golangci-lint with strict exit code & depguard architecture enforcement
  verify-standalone  Verify GOWORK=off standalone compilation across the 3 atframework submodules
  docs               Generate proto Markdown docs and build MkDocs site via uv
  sync-proto-svn     Sync public .proto files of component and all services to client SVN path
  clean              Clean generated code (--pb), cache (--cache), or full build/ (--all)
  ci                 Run full CI pipeline (prepare -> build -> verify-standalone -> test -> lint)
  doctor             Inspect toolchain paths, discovered services, and cache/GOCACHEPROG status
`)
}

func cmdPrepare(ctx context.Context, env *ProjectEnv, args []string) error {
	fs := flag.NewFlagSet("prepare", flag.ExitOnError)
	skipOptional := fs.Bool("skip-optional", false, "skip optional tools (xresloader/atdtool/java/python)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return env.PrepareAll(ctx, !*skipOptional)
}

func cmdProto(ctx context.Context, env *ProjectEnv, args []string) error {
	fs := flag.NewFlagSet("proto", flag.ExitOnError)
	force := fs.Bool("force", false, "force regeneration ignoring SHA-256 cache")
	serviceFilter := fs.String("service", "", "comma-separated service names to include (empty = all)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := env.EnsureProtoToolchain(ctx); err != nil {
		return err
	}
	return env.RunProto(ctx, *force, parseCSV(*serviceFilter))
}

func cmdConfig(ctx context.Context, env *ProjectEnv, args []string) error {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	force := fs.Bool("force", false, "force Excel/config regeneration ignoring cache")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := env.PrepareAll(ctx, true); err != nil {
		return err
	}
	if err := env.RunProto(ctx, false, nil); err != nil {
		return err
	}
	return env.RunConfig(ctx, *force)
}

func cmdCodegen(ctx context.Context, env *ProjectEnv, args []string) error {
	fs := flag.NewFlagSet("codegen", flag.ExitOnError)
	force := fs.Bool("force", false, "force Mako codegen ignoring cache")
	serviceFilter := fs.String("service", "", "comma-separated service names (empty = all)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := env.PrepareAll(ctx, true); err != nil {
		return err
	}
	svcs := parseCSV(*serviceFilter)
	if err := env.RunProto(ctx, false, svcs); err != nil {
		return err
	}
	if err := env.RunConfig(ctx, false); err != nil {
		return err
	}
	return env.RunMakoCodegen(ctx, *force, svcs)
}

func cmdBuild(ctx context.Context, env *ProjectEnv, args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	serviceFilter := fs.String("service", "", "comma-separated service names (empty = all discovered services)")
	mode := fs.String("mode", envOrDefault("BUILD_MODE", "Release"), "build mode: Release or Debug")
	version := fs.String("version", envOrDefault("BUILD_VERSION", "v0.0.0"), "embedded build version")
	configVer := fs.String("config-version", envOrDefault("CONFIG_VERSION", "v0.0.0"), "embedded config version")
	branch := fs.String("branch", os.Getenv("GIT_BRANCH"), "embedded git branch (defaults to current branch)")
	forceCodegen := fs.Bool("force-codegen", false, "force full proto/config/mako regeneration")
	skipCodegen := fs.Bool("skip-codegen", false, "skip code generation and only compile Go binaries")
	skipDeploy := fs.Bool("skip-deploy", false, "skip copying cloud-native charts and launch scripts")
	race := fs.Bool("race", false, "enable Go race detector")
	if err := fs.Parse(args); err != nil {
		return err
	}

	selectedNames := parseCSV(*serviceFilter)
	if !*skipCodegen {
		if err := env.PrepareAll(ctx, true); err != nil {
			return err
		}
		if err := env.RunProto(ctx, *forceCodegen, selectedNames); err != nil {
			return err
		}
		if err := env.RunConfig(ctx, *forceCodegen); err != nil {
			return err
		}
		if err := env.RunMakoCodegen(ctx, *forceCodegen, selectedNames); err != nil {
			return err
		}
	}

	opts := BuildOptions{
		Services:      selectedNames,
		Mode:          *mode,
		Version:       *version,
		ConfigVersion: *configVer,
		Branch:        *branch,
		Race:          *race,
	}
	if err := env.BuildServices(ctx, opts); err != nil {
		return err
	}
	if !*skipDeploy {
		if err := env.InstallDeployAssets(); err != nil {
			return err
		}
	}
	return nil
}

func cmdTest(ctx context.Context, env *ProjectEnv, args []string) error {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	affected := fs.String("affected", "", "run tests only for Go packages affected since git ref")
	race := fs.Bool("race", false, "enable race detector")
	verbose := fs.Bool("v", false, "verbose output")
	timeout := fs.String("timeout", "10m", "test timeout")
	withSubmodules := fs.Bool("with-submodules", false, "also test atframework submodules")
	if err := fs.Parse(args); err != nil {
		return err
	}

	pkgs := fs.Args()
	if len(pkgs) == 0 {
		pkgs = []string{"./src/...", "./tools/..."}
	}
	if err := buildkit.RunTests(ctx, buildkit.TestOptions{
		WorkDir:       env.RootDir,
		Packages:      pkgs,
		AffectedSince: *affected,
		Race:          *race,
		Verbose:       *verbose,
		Timeout:       *timeout,
	}); err != nil {
		return err
	}

	if *withSubmodules {
		return env.VerifySubmodulesStandalone(ctx, true)
	}
	return nil
}

func cmdLint(ctx context.Context, env *ProjectEnv, args []string) error {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	reportDir := fs.String("report-dir", filepath.Join("build", "lint_report"), "directory for lint report output")
	affected := fs.String("affected", "", "lint only Go packages affected since git ref")
	if err := fs.Parse(args); err != nil {
		return err
	}

	pkgs := fs.Args()
	if len(pkgs) == 0 {
		pkgs = []string{"./src/...", "./tools/..."}
	}
	if *affected != "" {
		aff, err := buildkit.AffectedPackages(ctx, env.RootDir, *affected, pkgs)
		if err != nil {
			return err
		}
		if len(aff) == 0 {
			fmt.Printf("[lint] no affected Go packages since %s, skipping\n", *affected)
			return nil
		}
		pkgs = aff
	}

	absReport := *reportDir
	if absReport != "" && !filepath.IsAbs(absReport) {
		absReport = filepath.Join(env.RootDir, absReport)
	}

	return buildkit.RunLint(ctx, buildkit.LintOptions{
		WorkDir:    env.RootDir,
		ConfigPath: filepath.Join(env.RootDir, ".golangci.yaml"),
		ReportDir:  absReport,
		Packages:   pkgs,
	})
}

func cmdVerifyStandalone(ctx context.Context, env *ProjectEnv, args []string) error {
	fs := flag.NewFlagSet("verify-standalone", flag.ExitOnError)
	runTests := fs.Bool("test", false, "also run GOWORK=off go test ./... in each submodule")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return env.VerifySubmodulesStandalone(ctx, *runTests)
}

func cmdDocs(ctx context.Context, env *ProjectEnv, args []string) error {
	fs := flag.NewFlagSet("docs", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	return env.BuildDocs(ctx)
}

func cmdSyncProtoSVN(env *ProjectEnv, args []string) error {
	fs := flag.NewFlagSet("sync-proto-svn", flag.ExitOnError)
	envFile := fs.String("env", filepath.Join(".vscode", "project_env.json"), "path to project_env.json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return env.SyncProtoToSVN(*envFile)
}

func cmdClean(env *ProjectEnv, args []string) error {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	all := fs.Bool("all", false, "remove build/, .cache/, and generated code")
	pb := fs.Bool("pb", false, "remove generated .pb.go, generate_config, and build/_gen")
	cache := fs.Bool("cache", false, "remove incremental build cache only")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*all && !*pb && !*cache {
		*pb = true
	}
	return env.Clean(*all, *pb, *cache)
}

func cmdCI(ctx context.Context, env *ProjectEnv, args []string) error {
	fs := flag.NewFlagSet("ci", flag.ExitOnError)
	mode := fs.String("mode", envOrDefault("BUILD_MODE", "Release"), "build mode")
	changedSince := fs.String("changed-since", "", "git ref for affected package testing/linting")
	runDocs := fs.Bool("docs", false, "also build MkDocs documentation site")
	runLint := fs.Bool("lint", true, "run golangci-lint (fails CI on lint errors)")
	standaloneTests := fs.Bool("standalone-tests", true, "run GOWORK=off tests in all 3 atframework submodules")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := cmdBuild(ctx, env, []string{"-mode", *mode}); err != nil {
		return err
	}
	if err := env.VerifySubmodulesStandalone(ctx, *standaloneTests); err != nil {
		return err
	}

	testArgs := []string{"-timeout", "10m"}
	if *changedSince != "" {
		testArgs = append(testArgs, "-affected", *changedSince)
	}
	if err := cmdTest(ctx, env, testArgs); err != nil {
		return err
	}
	if *runLint {
		lintArgs := []string{}
		if *changedSince != "" {
			lintArgs = append(lintArgs, "-affected", *changedSince)
		}
		if err := cmdLint(ctx, env, lintArgs); err != nil {
			return err
		}
	}
	if *runDocs {
		if err := env.BuildDocs(ctx); err != nil {
			return err
		}
	}
	fmt.Println("[ci] all checks passed")
	return nil
}

func cmdDoctor(ctx context.Context, env *ProjectEnv) error {
	goVer, _ := exec.CommandContext(ctx, "go", "version").Output()
	fmt.Printf("ProjectRoot:  %s\n", env.RootDir)
	fmt.Printf("OS/Arch:      %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Go:           %s\n", strings.TrimSpace(string(goVer)))
	fmt.Printf("GOCACHEPROG:  %s\n", envOrDefault("GOCACHEPROG", "(local GOCACHE)"))

	if javaBin, major, err := buildkit.ResolveJava(ctx, buildkit.DefaultJavaMinMajorVersion); err == nil {
		fmt.Printf("Java:         JDK %d (%s)\n", major, javaBin)
	} else {
		fmt.Printf("Java:         NOT READY (%v)\n", err)
	}
	if uvBin, err := exec.LookPath("uv"); err == nil {
		fmt.Printf("Python/uv:    %s\n", uvBin)
	} else if py, err := exec.LookPath("python"); err == nil {
		fmt.Printf("Python:       %s (uv not installed)\n", py)
	} else {
		fmt.Printf("Python/uv:    missing\n")
	}
	if lintBin, err := exec.LookPath("golangci-lint"); err == nil {
		fmt.Printf("golangci-lint:%s\n", lintBin)
	} else {
		fmt.Printf("golangci-lint:missing\n")
	}

	svcs, err := env.DiscoverServices()
	if err != nil {
		return err
	}
	names := make([]string, 0, len(svcs))
	for _, s := range svcs {
		names = append(names, fmt.Sprintf("%s (%s)", s.Name, s.RelDir))
	}
	fmt.Printf("Services (%d): %s\n", len(svcs), strings.Join(names, ", "))
	return nil
}

func locateProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err2 := os.Stat(filepath.Join(dir, "src", "component")); err2 == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate atsf4g-go root (containing go.mod and src/component) from %s", cwd)
		}
		dir = parent
	}
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[dev] ERROR: "+format+"\n", args...)
	os.Exit(1)
}
