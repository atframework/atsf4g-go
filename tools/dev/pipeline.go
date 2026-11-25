// Copyright 2026 atframework
package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	texttemplate "text/template"

	"github.com/atframework/atframe-utils-go/buildkit"
	utils_template "github.com/atframework/atframe-utils-go/template"
)

// RunProto compiles third_party, submodule, component, and all auto-discovered service protos
// in staged concurrent batches with SHA-256 incremental caching.
func (e *ProjectEnv) RunProto(ctx context.Context, force bool, serviceFilter []string) error {
	cache, err := e.ensureCache()
	if err != nil {
		return err
	}

	xresCore := "mod:github.com/xresloader/xresloader"
	xresCode := "mod:github.com/xresloader/xres-code-generator"
	atbusProto := "mod:github.com/atframework/libatbus-go/protocol"
	atappProto := "mod:github.com/atframework/libatapp-go/protocol"

	compPub := filepath.ToSlash(filepath.Join("src", "component", "protocol", "public"))
	compPriv := filepath.ToSlash(filepath.Join("src", "component", "protocol", "private"))

	targets := []buildkit.ProtoTarget{
		// Stage 0: third-party xresloader & atframework base protocols
		{
			Name:       "xresloader-code",
			Stage:      0,
			WorkDir:    "third_party/xresloader/protocols/code",
			ProtoPaths: []string{"."},
			Files:      []string{"protocol/extension/xrescode_extensions_v3.proto"},
			GoOut:      ".",
			MutableOut: ".",
		},
		{
			Name:       "xresloader-core-config",
			Stage:      0,
			WorkDir:    "third_party/xresloader/protocols/core",
			ProtoPaths: []string{"."},
			Files:      []string{"protocol/config/pb_header_v3.proto"},
			GoOut:      ".",
			MutableOut: ".",
		},
		{
			Name:       "xresloader-core-extension",
			Stage:      0,
			WorkDir:    "third_party/xresloader/protocols/core",
			ProtoPaths: []string{"."},
			Files: []string{
				"protocol/extension/v3/xresloader.proto",
				"protocol/extension/v3/xresloader_ue.proto",
			},
			GoOut:      ".",
			MutableOut: ".",
		},
		{
			Name:       "libatbus-protocol",
			Stage:      0,
			WorkDir:    "atframework/libatbus-go/protocol",
			ProtoPaths: []string{"."},
			Files:      []string{"*.proto"},
			GoOut:      ".",
			MutableOut: ".",
		},
		{
			Name:       "libatapp-protocol",
			Stage:      0,
			WorkDir:    "atframework/libatapp-go/protocol",
			ProtoPaths: []string{"."},
			Files:      []string{"atframe/*.proto"},
			GoOut:      ".",
			MutableOut: ".",
		},
		{
			Name:          "atframework-all-pb",
			Stage:         0,
			WorkDir:       ".",
			ProtoPaths:    []string{atbusProto, atappProto},
			Files:         []string{"atframework/libatbus-go/protocol/libatbus_protocol.proto", "atframework/libatapp-go/protocol/atframe/atapp_conf.proto"},
			DescriptorOut: "build/install/resource/pbdesc/atframework-all.pb",
		},
		{
			Name:       "third-party-extension-pb",
			Stage:      0,
			WorkDir:    ".",
			ProtoPaths: []string{xresCore, xresCode},
			Files: []string{
				"third_party/xresloader/protocols/core/protocol/extension/v3/xresloader.proto",
				"third_party/xresloader/protocols/core/protocol/extension/v3/xresloader_ue.proto",
				"third_party/xresloader/protocols/core/protocol/config/pb_header_v3.proto",
				"third_party/xresloader/protocols/code/protocol/extension/xrescode_extensions_v3.proto",
			},
			DescriptorOut: "build/install/resource/pbdesc/third_party-extension.pb",
		},

		// Stage 1: component public & private protocols + descriptors
		{
			Name:          "component-public-extension",
			Stage:         1,
			WorkDir:       compPub,
			ProtoPaths:    []string{"extension"},
			Files:         []string{"extension/protocol/extension/*.proto"},
			GoOut:         "extension",
			MutableOut:    "extension",
			DescriptorOut: "build/install/resource/pbdesc/public-extension.pb",
		},
		{
			Name:          "component-public-common",
			Stage:         1,
			WorkDir:       compPub,
			ProtoPaths:    []string{"common", "extension", xresCore, xresCode},
			Files:         []string{"common/protocol/common/*.proto"},
			GoOut:         "common",
			MutableOut:    "common",
			DescriptorOut: "build/install/resource/pbdesc/public-common.pb",
		},
		{
			Name:       "component-public-config",
			Stage:      1,
			WorkDir:    compPub,
			ProtoPaths: []string{"config", "common", "extension", atbusProto, atappProto, xresCore, xresCode},
			Files:      []string{"config/protocol/config/*.proto"},
			GoOut:      "config",
			MutableOut: "config",
		},
		{
			Name:          "component-public-pbdesc",
			Stage:         1,
			WorkDir:       compPub,
			ProtoPaths:    []string{"pbdesc", "common", "extension", xresCore, xresCode},
			Files:         []string{"pbdesc/protocol/pbdesc/*.proto"},
			GoOut:         "pbdesc",
			MutableOut:    "pbdesc",
			DescriptorOut: "build/install/resource/pbdesc/public-pbdesc.pb",
		},
		{
			Name:       "component-public-config-pb",
			Stage:      1,
			WorkDir:    compPub,
			ProtoPaths: []string{"config", "common", "extension", atbusProto, atappProto, xresCore, xresCode},
			Files: []string{
				"extension/protocol/extension/*.proto",
				"common/protocol/common/*.proto",
				"config/protocol/config/*.proto",
				"../../../../third_party/xresloader/protocols/core/protocol/extension/v3/xresloader.proto",
				"../../../../third_party/xresloader/protocols/core/protocol/extension/v3/xresloader_ue.proto",
				"../../../../third_party/xresloader/protocols/core/protocol/config/pb_header_v3.proto",
				"../../../../third_party/xresloader/protocols/code/protocol/extension/xrescode_extensions_v3.proto",
			},
			DescriptorOut: "build/install/resource/pbdesc/public-config.pb",
		},
		{
			Name:       "component-private-extension",
			Stage:      1,
			WorkDir:    compPriv,
			ProtoPaths: []string{"extension", "../public/extension"},
			Files:      []string{"extension/protocol/extension/*.proto"},
			GoOut:      "extension",
			MutableOut: "extension",
		},
		{
			Name:       "component-private-common",
			Stage:      1,
			WorkDir:    compPriv,
			ProtoPaths: []string{"common", "extension", xresCore, "../public/common", "../public/extension"},
			Files:      []string{"common/protocol/common/*.proto"},
			GoOut:      "common",
			MutableOut: "common",
		},
		{
			Name:       "component-private-config",
			Stage:      1,
			WorkDir:    compPriv,
			ProtoPaths: []string{"config", "common", xresCore, "extension", "../public/config", "../public/common", "../public/extension", atbusProto, atappProto},
			Files:      []string{"config/protocol/config/*.proto"},
			GoOut:      "config",
			MutableOut: "config",
		},
		{
			Name:       "component-private-pbdesc",
			Stage:      1,
			WorkDir:    compPriv,
			ProtoPaths: []string{"pbdesc", "common", xresCore, "extension", "../public/pbdesc", "../public/common", "../public/extension", atbusProto, atappProto},
			Files:      []string{"pbdesc/protocol/pbdesc/*.proto"},
			GoOut:      "pbdesc",
			MutableOut: "pbdesc",
		},
		{
			Name:       "component-private-log",
			Stage:      1,
			WorkDir:    compPriv,
			ProtoPaths: []string{"log", "common", xresCore, "extension", "../public/log", "../public/common", "../public/extension", "../public/pbdesc", atbusProto, atappProto},
			Files:      []string{"log/protocol/log/*.proto"},
			GoOut:      "log",
			MutableOut: "log",
		},
		{
			Name:          "component-private-pbdesc-pb",
			Stage:         1,
			WorkDir:       compPriv,
			ProtoPaths:    []string{"pbdesc", "config", "common", "extension", "../public/common", "../public/pbdesc", "../public/extension", xresCore},
			Files:         []string{"pbdesc/protocol/pbdesc/*.proto", "extension/protocol/extension/*.proto"},
			DescriptorOut: "build/install/resource/pbdesc/private-pbdesc.pb",
		},
	}

	// Stage 2: auto-discovered business services (concurrent across 20+ services)
	allSvcs, err := e.DiscoverServices()
	if err != nil {
		return err
	}
	svcs, err := e.filterServices(allSvcs, serviceFilter)
	if err != nil {
		return err
	}

	baseSvcProtoPaths := []string{
		filepath.Join(e.RootDir, "src", "component", "protocol", "public", "pbdesc"),
		filepath.Join(e.RootDir, "src", "component", "protocol", "public", "config"),
		filepath.Join(e.RootDir, "src", "component", "protocol", "public", "common"),
		filepath.Join(e.RootDir, "src", "component", "protocol", "public", "extension"),
		filepath.Join(e.RootDir, "src", "component", "protocol", "private", "config"),
		filepath.Join(e.RootDir, "src", "component", "protocol", "private", "common"),
		filepath.Join(e.RootDir, "src", "component", "protocol", "private", "extension"),
		xresCore,
		xresCode,
		atbusProto,
		atappProto,
	}
	// Expose every service's public protocol root so cross-service SS RPC protos resolve cleanly.
	for _, s := range allSvcs {
		pubRoot := filepath.Join(s.AbsDir, "protocol", "public")
		if _, statErr := os.Stat(pubRoot); statErr == nil {
			baseSvcProtoPaths = append(baseSvcProtoPaths, pubRoot)
		}
	}

	for _, svc := range svcs {
		if !svc.HasProto {
			continue
		}
		privRoot := filepath.Join(svc.AbsDir, "protocol", "private")
		pubRoot := filepath.Join(svc.AbsDir, "protocol", "public")
		svcProtoPaths := append([]string{}, baseSvcProtoPaths...)
		if _, statErr := os.Stat(privRoot); statErr == nil {
			svcProtoPaths = append(svcProtoPaths, privRoot)
		}
		if _, statErr := os.Stat(pubRoot); statErr == nil && !slices.Contains(svcProtoPaths, pubRoot) {
			svcProtoPaths = append(svcProtoPaths, pubRoot)
		}

		var descInputs []string
		privGlob := filepath.Join(privRoot, "protocol", "pbdesc", "*.proto")
		if matches, _ := filepath.Glob(privGlob); len(matches) > 0 {
			targets = append(targets, buildkit.ProtoTarget{
				Name:       svc.Name + "-private-proto",
				Stage:      2,
				WorkDir:    svc.RelDir,
				ProtoPaths: svcProtoPaths,
				Files:      []string{"protocol/private/protocol/pbdesc/*.proto"},
				GoOut:      "protocol/private",
				MutableOut: "protocol/private",
			})
			descInputs = append(descInputs, "protocol/private/protocol/pbdesc/*.proto")
		}
		pubGlob := filepath.Join(pubRoot, "protocol", "pbdesc", "*.proto")
		if matches, _ := filepath.Glob(pubGlob); len(matches) > 0 {
			targets = append(targets, buildkit.ProtoTarget{
				Name:       svc.Name + "-public-proto",
				Stage:      2,
				WorkDir:    svc.RelDir,
				ProtoPaths: svcProtoPaths,
				Files:      []string{"protocol/public/protocol/pbdesc/*.proto"},
				GoOut:      "protocol/public",
				MutableOut: "protocol/public",
			})
			descInputs = append(descInputs, "protocol/public/protocol/pbdesc/*.proto")
		}
		if len(descInputs) > 0 {
			targets = append(targets, buildkit.ProtoTarget{
				Name:          svc.Name + "-pbdesc",
				Stage:         2,
				WorkDir:       svc.RelDir,
				ProtoPaths:    svcProtoPaths,
				Files:         descInputs,
				DescriptorOut: fmt.Sprintf("build/install/resource/pbdesc/%s.pb", svc.Name),
			})
		}
	}

	runner := &buildkit.ProtoRunner{
		RootDir:          e.RootDir,
		ProtocBin:        e.ProtocBin,
		ProtocVersion:    e.ProtocVersion,
		ProtocGenGoBin:   e.ProtocGenGoBin,
		ProtocGenMutable: e.ProtocGenMutable,
		ModuleDirs:       e.ModuleDirs,
		Cache:            cache,
		Force:            force,
	}
	return runner.RunTargets(ctx, targets)
}

// RunConfig converts Excel tables via xresconv-cli/xresloader and generates Go config loaders via xrescode-gen.py,
// skipping JVM/Python invocation when SHA-256 input digest is unchanged.
func (e *ProjectEnv) RunConfig(ctx context.Context, force bool) error {
	cache, err := e.ensureCache()
	if err != nil {
		return err
	}

	xmlTplPath := filepath.Join(e.RootDir, "src", "component", "protocol", "public", "xresconv.xml.tpl")
	xresconvXML := filepath.Join(e.ResourceDir, "xresconv.xml")
	publicConfigPb := filepath.Join(e.PbdescDir, "public-config.pb")
	generatedXML := filepath.Join(e.GenDir, "xresconv.xml")
	genConfigDir := filepath.Join(e.RootDir, "src", "component", "config", "generate_config")

	// 1. Render build/_gen/xresconv.xml + copy validator.yaml & version.txt (fast in-process).
	if err := e.renderXresconvXML(xmlTplPath, xresconvXML, publicConfigPb, generatedXML); err != nil {
		return err
	}

	// 2. Excel -> bytes/json conversion (cached).
	excelDigest, err := buildkit.HashInputs(
		[]string{generatedXML, xresconvXML, filepath.Join(e.ResourceDir, "validator.yaml"), publicConfigPb, e.ExcelSrcDir},
		nil,
		[]string{"xresloader:" + e.XresloaderVersion, "xresconv-cli:" + e.XresconvCLIVersion},
	)
	if err != nil {
		return fmt.Errorf("hash excel config inputs: %w", err)
	}
	excelMarker := filepath.Join(e.CacheDir, "excel.stamp")
	if !force && cache.IsUpToDate("config:excel", excelDigest, []string{excelMarker}) && dirHasFiles(e.ExcelOutDir) {
		fmt.Println("[config] skip excel conversion (up-to-date)")
	} else {
		xresconvBin, err := e.EnsureXresconvCLI(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("[config] converting Excel tables via xresconv-cli %s...\n", e.XresconvCLIVersion)
		cmd := exec.CommandContext(ctx, xresconvBin, "--java-path", e.JavaBin, generatedXML)
		cmd.Dir = e.RootDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("xresconv-cli failed: %w", err)
		}
		_ = os.MkdirAll(e.CacheDir, 0o755)
		_ = os.WriteFile(excelMarker, []byte(excelDigest+"\n"), 0o644)
		cache.Record("config:excel", excelDigest, []string{excelMarker})
	}

	// 3. Go config loader generation via xrescode-gen.py (cached).
	loaderDigest, err := buildkit.HashInputs(
		[]string{publicConfigPb, filepath.Join(e.TemplateDir, "config_group.go.mako"), filepath.Join(e.TemplateDir, "config_set.go.mako")},
		nil,
		[]string{"xrescode-gen:v1"},
	)
	if err != nil {
		return fmt.Errorf("hash config loader inputs: %w", err)
	}
	configGroupOut := filepath.Join(genConfigDir, "config_group.go")
	if !force && cache.IsUpToDate("config:go-loaders", loaderDigest, []string{configGroupOut}) {
		fmt.Println("[config] skip Go config loader codegen (up-to-date)")
	} else {
		fmt.Println("[config] generating Go config loaders via xrescode-gen.py...")
		if err := os.MkdirAll(genConfigDir, 0o755); err != nil {
			return err
		}
		xresGenScript := filepath.Join(e.XresloaderDir, "xres-code-generator", "xrescode-gen.py")
		cmd := e.Python.ExecCmd(
			ctx,
			xresGenScript,
			"-i", e.TemplateDir,
			"-p", publicConfigPb,
			"-o", genConfigDir,
			"-g", "config_group.go.mako:config_group.go",
			"-l", `config_set.go.mako:${"config_set_{0}.go".format(loader.get_go_pb_name())}`,
			"-t", "server",
		)
		cmd.Dir = e.RootDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("xrescode-gen.py failed: %w", err)
		}
		cache.Record("config:go-loaders", loaderDigest, []string{configGroupOut})
	}

	return cache.Save()
}

func (e *ProjectEnv) renderXresconvXML(tplPath, xresconvXML, publicConfigPb, outXML string) error {
	tmpl, err := template.ParseFiles(tplPath)
	if err != nil {
		return fmt.Errorf("parse %s: %w", tplPath, err)
	}
	vars := struct {
		XRESCONV_XML_PATH     string
		XRESCONV_EXE_PATH     string
		XRESCONV_CONFIG_PB    string
		XRESCONV_BYTES_OUTPUT string
		XRESCONV_EXECL_SRC    string
	}{
		XRESCONV_XML_PATH:     xresconvXML,
		XRESCONV_EXE_PATH:     e.XresloaderJar,
		XRESCONV_CONFIG_PB:    publicConfigPb,
		XRESCONV_BYTES_OUTPUT: e.ExcelOutDir,
		XRESCONV_EXECL_SRC:    e.ExcelSrcDir,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return fmt.Errorf("render xresconv.xml: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outXML), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outXML, buf.Bytes(), 0o644); err != nil {
		return err
	}

	validatorSrc := filepath.Join(e.ResourceDir, "validator.yaml")
	if _, err := os.Stat(validatorSrc); err == nil {
		if err := buildkit.CopyFile(validatorSrc, filepath.Join(e.GenDir, "validator.yaml")); err != nil {
			return err
		}
	}
	versionSrc := filepath.Join(e.ExcelSrcDir, "version.txt")
	if _, err := os.Stat(versionSrc); err == nil {
		if err := buildkit.CopyFile(versionSrc, filepath.Join(e.ExcelOutDir, "version.txt")); err != nil {
			return err
		}
	}
	return nil
}

// RunMakoCodegen renders generate-for-pb.yaml from all matching generate-for-pb.atfw.*.tpl files
// and invokes tools/generate-for-pbtool/mako-generator.py (cached by SHA-256).
func (e *ProjectEnv) RunMakoCodegen(ctx context.Context, force bool, serviceFilter []string) error {
	cache, err := e.ensureCache()
	if err != nil {
		return err
	}

	scanDirs := []string{filepath.Join(e.RootDir, "src", "component")}
	allSvcs, err := e.DiscoverServices()
	if err != nil {
		return err
	}
	svcs, err := e.filterServices(allSvcs, serviceFilter)
	if err != nil {
		return err
	}
	for _, s := range svcs {
		scanDirs = append(scanDirs, s.AbsDir)
	}

	tplFiles, err := collectPbTemplateFiles(scanDirs)
	if err != nil {
		return err
	}
	if len(tplFiles) == 0 {
		return nil
	}

	pbFiles, err := collectPbDescriptors(e.PbdescDir)
	if err != nil {
		return err
	}
	if len(pbFiles) == 0 {
		return fmt.Errorf("no .pb descriptor files found in %s (run proto stage first)", e.PbdescDir)
	}

	hashInputs := append([]string{}, pbFiles...)
	hashInputs = append(hashInputs, tplFiles...)
	hashInputs = append(hashInputs, e.TemplateDir)
	digest, err := buildkit.HashInputs(hashInputs, []string{".pb", ".tpl", ".mako"}, []string{"mako-codegen:v1", strings.Join(serviceFilter, ",")})
	if err != nil {
		return fmt.Errorf("hash mako codegen inputs: %w", err)
	}

	yamlOut := filepath.Join(e.GenDir, "generate-for-pb.yaml")
	expectedOutputs := []string{
		yamlOut,
		filepath.Join(e.RootDir, "src", "component", "db", "local_db.go"),
		filepath.Join(e.RootDir, "src", "component", "db", "global_db.go"),
	}
	for _, s := range svcs {
		if s.Name == "lobbysvr" {
			expectedOutputs = append(expectedOutputs,
				filepath.Join(s.AbsDir, "app", "register_lobbyclientservice.go"),
				filepath.Join(s.AbsDir, "rpc", "lobbyclientservice", "session_downstream_api.go"),
			)
		}
	}

	if !force && cache.IsUpToDate("codegen:mako", digest, expectedOutputs) {
		fmt.Println("[codegen] skip Mako RPC/DB generation (up-to-date)")
		return nil
	}

	if err := e.renderGenerateForPbYAML(tplFiles, pbFiles, yamlOut); err != nil {
		return err
	}

	fmt.Printf("[codegen] running mako-generator.py (%d rule templates, %d .pb descriptors)...\n", len(tplFiles), len(pbFiles))
	makoDir := filepath.Join(e.RootDir, "tools", "generate-for-pbtool")
	cmd := e.Python.ExecCmd(ctx, filepath.Join(makoDir, "mako-generator.py"), "--project-dir", e.RootDir, "-c", yamlOut)
	cmd.Dir = makoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mako-generator.py failed: %w", err)
	}

	cache.Record("codegen:mako", digest, expectedOutputs)
	return cache.Save()
}

const pbToolYAMLTemplate = `# Auto-generated by tools/dev (do not edit manually).
configure:
  encoding: "UTF-8"
  output_directory: "{{ .project_build_dir }}"
  overwrite: false
  paths: []
  package_prefix: []
  protocol_input_pb_file: "{{ .protocol_input_pb_file }}"
  protocol_project_directory: "{{ .project_root_dir }}"
{{- with .protocol_external_pb_files }}
  protocol_external_pb_files:
  {{- range $_, $value := . }}
    - "{{ $value }}"
  {{- end }}
{{- end }}

rules:
{{- range $rule := .rules }}
  # rule for {{ $rule.Key }}
  {{- $rule.Body | trim | nindent 2 }}
{{- end }}
`

func (e *ProjectEnv) renderGenerateForPbYAML(tplFiles, pbFiles []string, outPath string) error {
	type renderedRule struct {
		Key  string
		Body string
	}
	var rules []renderedRule
	for _, tf := range tplFiles {
		t, err := texttemplate.ParseFiles(tf)
		if err != nil {
			return fmt.Errorf("parse rule template %s: %w", tf, err)
		}
		var buf bytes.Buffer
		err = t.Execute(&buf, map[string]any{
			"project_root_dir":               filepath.ToSlash(e.RootDir),
			"project_build_dir":              filepath.ToSlash(e.BuildDir),
			"project_namespace":              e.Namespace,
			"project_current_configure_dir":  filepath.ToSlash(filepath.Dir(tf)),
			"project_current_configure_file": tf,
			"project_template_dir":           filepath.ToSlash(e.TemplateDir),
		})
		if err != nil {
			return fmt.Errorf("execute rule template %s: %w", tf, err)
		}
		body := strings.TrimSpace(buf.String())
		if !strings.HasPrefix(body, "- ") {
			body = "- " + body
		}
		rules = append(rules, renderedRule{
			Key:  filepath.ToSlash(tf),
			Body: body,
		})
	}

	baseTmpl, err := texttemplate.New("generate-for-pb").Funcs(utils_template.TemplateFuncMap()).Parse(pbToolYAMLTemplate)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	if err := baseTmpl.Execute(&out, map[string]any{
		"project_root_dir":           filepath.ToSlash(e.RootDir),
		"project_build_dir":          filepath.ToSlash(e.BuildDir),
		"protocol_input_pb_file":     filepath.ToSlash(pbFiles[0]),
		"protocol_external_pb_files": toSlashSlice(pbFiles[1:]),
		"rules":                      rules,
	}); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, out.Bytes(), 0o644)
}

func collectPbTemplateFiles(scanDirs []string) ([]string, error) {
	seen := make(map[string]bool)
	var files []string
	for _, dir := range scanDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(dir, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return walkErr
			}
			name := d.Name()
			if strings.HasPrefix(name, "generate-for-pb.atfw.") &&
				(strings.HasSuffix(name, ".yaml.tpl") || strings.HasSuffix(name, ".yml.tpl")) {
				clean := filepath.Clean(p)
				if !seen[clean] {
					seen[clean] = true
					files = append(files, clean)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	slices.Sort(files)
	return files, nil
}

func collectPbDescriptors(pbDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(pbDir, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		if filepath.Ext(d.Name()) == ".pb" {
			files = append(files, filepath.Clean(p))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(files)
	return files, nil
}

func toSlashSlice(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = filepath.ToSlash(s)
	}
	return out
}

func dirHasFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) > 0
}
