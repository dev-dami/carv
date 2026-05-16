package resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dev-dami/carv/pkg/module"
)

func TestResolverResolveEmptyConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := &module.Config{
		Package:      module.PackageInfo{Name: "test", Version: "0.1.0"},
		Dependencies: make(map[string]module.Dependency),
	}

	r := New(dir)
	deps, err := r.Resolve(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Fatalf("expected 0 deps, got %d", len(deps))
	}
}

func TestResolverCycleDetection(t *testing.T) {
	dir := t.TempDir()
	modDir := filepath.Join(dir, "carv_modules", "a")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a carv.toml for "a" that depends on itself (cycle).
	tomlContent := `
[package]
name = "a"
version = "0.1.0"

[dependencies]
a = { path = "../a" }
`
	if err := os.WriteFile(filepath.Join(modDir, "carv.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &module.Config{
		Package: module.PackageInfo{Name: "root", Version: "0.1.0"},
		Dependencies: map[string]module.Dependency{
			"a": {Path: "carv_modules/a"},
		},
	}

	r := New(dir)
	_, err := r.Resolve(cfg)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestResolverPathDependency(t *testing.T) {
	dir := t.TempDir()

	// Create a dependency package.
	depDir := filepath.Join(dir, "libs", "mylib")
	if err := os.MkdirAll(depDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[package]
name = "mylib"
version = "1.0.0"
`
	if err := os.WriteFile(filepath.Join(depDir, "carv.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &module.Config{
		Package: module.PackageInfo{Name: "root", Version: "0.1.0"},
		Dependencies: map[string]module.Dependency{
			"mylib": {Path: "libs/mylib"},
		},
	}

	r := New(dir)
	deps, err := r.Resolve(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(deps))
	}
	if deps[0].Name != "mylib" {
		t.Fatalf("expected dep name 'mylib', got %q", deps[0].Name)
	}
}

func TestResolverLockFile(t *testing.T) {
	dir := t.TempDir()

	depDir := filepath.Join(dir, "libs", "mylib")
	if err := os.MkdirAll(depDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `[package]
name = "mylib"
version = "1.0.0"
`
	if err := os.WriteFile(filepath.Join(depDir, "carv.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &module.Config{
		Package: module.PackageInfo{Name: "root", Version: "0.1.0"},
		Dependencies: map[string]module.Dependency{
			"mylib": {Path: "libs/mylib"},
		},
	}

	r := New(dir)
	deps, err := r.Resolve(cfg)
	if err != nil {
		t.Fatal(err)
	}

	lf := r.LockFile()
	if len(lf.Packages) != 1 {
		t.Fatalf("expected 1 locked package, got %d", len(lf.Packages))
	}
	if lf.Packages[0].Name != "mylib" {
		t.Fatalf("expected locked name 'mylib', got %q", lf.Packages[0].Name)
	}

	// Save and reload.
	if err := module.SaveLock(dir, lf); err != nil {
		t.Fatal(err)
	}

	loaded, err := module.LoadLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Packages) != 1 {
		t.Fatalf("expected 1 loaded package, got %d", len(loaded.Packages))
	}

	_ = deps
}

func TestResolverTransitiveDeps(t *testing.T) {
	dir := t.TempDir()

	// Create lib-b.
	libB := filepath.Join(dir, "libs", "lib-b")
	if err := os.MkdirAll(libB, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libB, "carv.toml"), []byte(`[package]
name = "lib-b"
version = "1.0.0"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create lib-a that depends on lib-b.
	libA := filepath.Join(dir, "libs", "lib-a")
	if err := os.MkdirAll(libA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libA, "carv.toml"), []byte(`[package]
name = "lib-a"
version = "1.0.0"

[dependencies]
lib-b = { path = "../lib-b" }
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &module.Config{
		Package: module.PackageInfo{Name: "root", Version: "0.1.0"},
		Dependencies: map[string]module.Dependency{
			"lib-a": {Path: "libs/lib-a"},
		},
	}

	r := New(dir)
	deps, err := r.Resolve(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 root dep, got %d", len(deps))
	}
	if len(deps[0].Deps) != 1 {
		t.Fatalf("expected 1 transitive dep, got %d", len(deps[0].Deps))
	}
	if deps[0].Deps[0].Name != "lib-b" {
		t.Fatalf("expected transitive dep 'lib-b', got %q", deps[0].Deps[0].Name)
	}

	lf := r.LockFile()
	if len(lf.Packages) != 2 {
		t.Fatalf("expected 2 locked packages, got %d", len(lf.Packages))
	}
}
