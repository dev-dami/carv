package module

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader("/tmp/test")

	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}

	if loader.loadedFiles == nil {
		t.Error("loadedFiles map not initialized")
	}
}

func TestLoaderSetConfig(t *testing.T) {
	loader := NewLoader("/tmp/test")
	cfg := &Config{
		Package: PackageInfo{Name: "test"},
	}

	loader.SetConfig(cfg)

	if loader.config != cfg {
		t.Error("SetConfig did not set config")
	}
}

func TestLoaderGetLoadedModules(t *testing.T) {
	loader := NewLoader("/tmp/test")

	modules := loader.GetLoadedModules()

	if modules == nil {
		t.Error("GetLoadedModules returned nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("myproject")

	if cfg.Package.Name != "myproject" {
		t.Errorf("expected package name 'myproject', got %q", cfg.Package.Name)
	}

	if cfg.Package.Version != "0.1.0" {
		t.Errorf("expected version '0.1.0', got %q", cfg.Package.Version)
	}

	if cfg.Package.Entry != "src/main.carv" {
		t.Errorf("expected entry 'src/main.carv', got %q", cfg.Package.Entry)
	}

	if cfg.Build.Output != "build" {
		t.Errorf("expected build output 'build', got %q", cfg.Build.Output)
	}

	if !cfg.Build.Optimize {
		t.Error("expected optimize to be true")
	}

	if cfg.Scripts["build"] != "carv build src/main.carv" {
		t.Errorf("expected build script, got %q", cfg.Scripts["build"])
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Create a temp directory with carv.toml
	tmpDir, err := os.MkdirTemp("", "carv-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create carv.toml
	configPath := filepath.Join(tmpDir, "carv.toml")
	err = os.WriteFile(configPath, []byte("[package]\nname = \"test\"\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "src", "lib")
	err = os.MkdirAll(subDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Test finding project root from subdirectory
	root, err := FindProjectRoot(subDir)
	if err != nil {
		t.Fatal(err)
	}

	absRoot, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if root != absRoot {
		t.Errorf("expected root %q, got %q", absRoot, root)
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temp directory with carv.toml
	tmpDir, err := os.MkdirTemp("", "carv-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create carv.toml
	configContent := `[package]
name = "testpkg"
version = "1.0.0"
description = "A test package"

[build]
output = "dist"
optimize = true
`
	configPath := filepath.Join(tmpDir, "carv.toml")
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig returned nil config")
	}

	if cfg.Package.Name != "testpkg" {
		t.Errorf("expected package name 'testpkg', got %q", cfg.Package.Name)
	}

	if cfg.Package.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", cfg.Package.Version)
	}

	if cfg.Build.Output != "dist" {
		t.Errorf("expected build output 'dist', got %q", cfg.Build.Output)
	}
}

func TestLoadConfigNotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("expected no error for missing config, got: %v", err)
	}

	if cfg != nil {
		t.Error("expected nil config for missing file")
	}
}

func TestConfigSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig("savetest")
	err = cfg.Save(tmpDir)
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, "carv.toml")
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		t.Error("carv.toml was not created")
	}

	// Load it back and verify
	loaded, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig error after save: %v", err)
	}

	if loaded.Package.Name != "savetest" {
		t.Errorf("expected name 'savetest', got %q", loaded.Package.Name)
	}
}

func TestLoaderLoadRelativeModule(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a module file
	mathContent := `pub fn add(a: int, b: int) -> int {
    return a + b;
}
`
	mathPath := filepath.Join(tmpDir, "math.carv")
	err = os.WriteFile(mathPath, []byte(mathContent), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// Create main file
	mainPath := filepath.Join(tmpDir, "main.carv")
	err = os.WriteFile(mainPath, []byte("let x = 1;"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	mod, err := loader.Load("./math", mainPath)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if mod == nil {
		t.Fatal("Load returned nil module")
	}

	if mod.Program == nil {
		t.Error("module program is nil")
	}

	// Check exports
	if !mod.Exports["add"] {
		t.Error("expected 'add' to be exported")
	}
}

func TestParseError(t *testing.T) {
	e := &ParseError{
		Path:   "/tmp/test.carv",
		Errors: []string{"error 1", "error 2"},
	}

	msg := e.Error()
	if msg != "parse error in /tmp/test.carv: error 1; error 2" {
		t.Errorf("unexpected error message: %s", msg)
	}
}

func TestExtractExports(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a module with various exports
	content := `
pub fn publicFunc() {}
fn privateFunc() {}
pub const PUBLIC_CONST = 42;
const PRIVATE_CONST = 0;
pub let publicVar = 1;
let privateVar = 2;
pub class PublicClass {}
class PrivateClass {}
`
	modPath := filepath.Join(tmpDir, "test.carv")
	err = os.WriteFile(modPath, []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	mainPath := filepath.Join(tmpDir, "main.carv")
	err = os.WriteFile(mainPath, []byte("let x = 1;"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	mod, err := loader.Load("./test", mainPath)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Check public exports
	expectedExports := []string{"publicFunc", "PUBLIC_CONST", "publicVar", "PublicClass"}
	for _, name := range expectedExports {
		if !mod.Exports[name] {
			t.Errorf("expected %q to be exported", name)
		}
	}

	// Check private items are not exported
	unexpectedExports := []string{"privateFunc", "PRIVATE_CONST", "privateVar", "PrivateClass"}
	for _, name := range unexpectedExports {
		if mod.Exports[name] {
			t.Errorf("expected %q to NOT be exported", name)
		}
	}
}

func TestLoaderLoadBuiltinNetModule(t *testing.T) {
	loader := NewLoader("/tmp/test")
	mod, err := loader.Load("net", "/tmp/main.carv")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if mod == nil {
		t.Fatal("expected builtin module, got nil")
	}
	if !mod.IsBuiltin {
		t.Fatal("expected net module to be builtin")
	}

	expected := []string{"tcp_listen", "tcp_accept", "tcp_read", "tcp_write", "tcp_close"}
	for _, name := range expected {
		if !mod.Exports[name] {
			t.Fatalf("expected builtin net export %q", name)
		}
	}
}

func TestLoaderLoadBuiltinWebModule(t *testing.T) {
	loader := NewLoader("/tmp/test")
	mod, err := loader.Load("web", "/tmp/main.carv")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if mod == nil {
		t.Fatal("expected builtin module, got nil")
	}
	if !mod.IsBuiltin {
		t.Fatal("expected web module to be builtin")
	}

	expected := []string{"tcp_listen", "tcp_accept", "tcp_read", "tcp_write", "tcp_close"}
	for _, name := range expected {
		if !mod.Exports[name] {
			t.Fatalf("expected builtin web export %q", name)
		}
	}
}
