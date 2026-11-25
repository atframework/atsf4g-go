// Copyright 2026 atframework
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverServicesSupportsFlatAndNestedServicesDirs(t *testing.T) {
	tmp := t.TempDir()
	// Existing flat layout: src/lobbysvr/main.go
	if err := os.MkdirAll(filepath.Join(tmp, "src", "lobbysvr", "protocol"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "src", "lobbysvr", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Future 20+ services layout: src/services/matchsvr/main.go
	if err := os.MkdirAll(filepath.Join(tmp, "src", "services", "matchsvr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "src", "services", "matchsvr", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Non-service dirs must be ignored
	if err := os.MkdirAll(filepath.Join(tmp, "src", "component"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "src", "template"), 0o755); err != nil {
		t.Fatal(err)
	}

	env := NewProjectEnv(tmp)
	svcs, err := env.DiscoverServices()
	if err != nil {
		t.Fatal(err)
	}
	if len(svcs) != 2 {
		t.Fatalf("expected 2 discovered services, got %d: %+v", len(svcs), svcs)
	}
	if svcs[0].Name != "lobbysvr" || !svcs[0].HasProto || svcs[0].RelDir != "src/lobbysvr" {
		t.Fatalf("unexpected first service: %+v", svcs[0])
	}
	if svcs[1].Name != "matchsvr" || svcs[1].HasProto || svcs[1].RelDir != "src/services/matchsvr" {
		t.Fatalf("unexpected second service: %+v", svcs[1])
	}
}
