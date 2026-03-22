package module

import (
	"errors"
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

// ---------- IsBuiltinModule ----------

func TestIsBuiltinModule(t *testing.T) {
	builtins := []string{"net", "web", "gpio", "uart", "spi", "i2c", "timer"}
	for _, name := range builtins {
		if !IsBuiltinModule(name) {
			t.Errorf("expected %q to be a builtin module", name)
		}
	}

	nonBuiltins := []string{"math", "os", "http", "json", "", "NET", "Web"}
	for _, name := range nonBuiltins {
		if IsBuiltinModule(name) {
			t.Errorf("expected %q to NOT be a builtin module", name)
		}
	}
}

// ---------- LoadLock / SaveLock ----------

func TestLoadLockNotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-lock-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	lf, err := LoadLock(tmpDir)
	if err != nil {
		t.Fatalf("LoadLock error: %v", err)
	}
	if lf == nil {
		t.Fatal("expected empty LockFile, got nil")
	}
	if len(lf.Packages) != 0 {
		t.Errorf("expected 0 packages, got %d", len(lf.Packages))
	}
}

func TestSaveLockAndLoadLock(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-lock-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	lf := &LockFile{
		Packages: []LockedPackage{
			{Name: "foo", Version: "1.2.3", Source: "git", Revision: "abc123"},
			{Name: "bar", Version: "0.1.0", Source: "path", Revision: ""},
		},
	}

	err = SaveLock(tmpDir, lf)
	if err != nil {
		t.Fatalf("SaveLock error: %v", err)
	}

	// Verify file exists
	lockPath := filepath.Join(tmpDir, "carv.lock")
	if _, statErr := os.Stat(lockPath); os.IsNotExist(statErr) {
		t.Fatal("carv.lock was not created")
	}

	// Load it back
	loaded, err := LoadLock(tmpDir)
	if err != nil {
		t.Fatalf("LoadLock error: %v", err)
	}

	if len(loaded.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(loaded.Packages))
	}

	if loaded.Packages[0].Name != "foo" || loaded.Packages[0].Version != "1.2.3" {
		t.Errorf("unexpected first package: %+v", loaded.Packages[0])
	}
	if loaded.Packages[1].Name != "bar" || loaded.Packages[1].Version != "0.1.0" {
		t.Errorf("unexpected second package: %+v", loaded.Packages[1])
	}
}

func TestLoadLockInvalidTOML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-lock-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	lockPath := filepath.Join(tmpDir, "carv.lock")
	err = os.WriteFile(lockPath, []byte("this is not valid toml [[["), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadLock(tmpDir)
	if err == nil {
		t.Error("expected error for invalid TOML lock file")
	}
}

func TestSaveLockBadDir(t *testing.T) {
	err := SaveLock("/nonexistent/dir/that/should/not/exist", &LockFile{})
	if err == nil {
		t.Error("expected error when saving to nonexistent directory")
	}
}

// ---------- resolvePath ----------

func TestResolvePathParentRelative(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-resolve-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create lib/utils.carv
	libDir := filepath.Join(tmpDir, "lib")
	err = os.MkdirAll(libDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	utilsPath := filepath.Join(libDir, "utils.carv")
	err = os.WriteFile(utilsPath, []byte("pub fn helper() {}"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// Create src/main.carv
	srcDir := filepath.Join(tmpDir, "src")
	err = os.MkdirAll(srcDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(srcDir, "main.carv")
	err = os.WriteFile(mainPath, []byte("let x = 1;"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	mod, err := loader.Load("../lib/utils", mainPath)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if mod == nil {
		t.Fatal("expected module, got nil")
	}
	if !mod.Exports["helper"] {
		t.Error("expected 'helper' to be exported")
	}
}

func TestResolvePathRelativeWithExtension(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-resolve-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modPath := filepath.Join(tmpDir, "mymod.carv")
	err = os.WriteFile(modPath, []byte("pub let val = 1;"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	mainPath := filepath.Join(tmpDir, "main.carv")
	err = os.WriteFile(mainPath, []byte("let x = 1;"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	// Load with .carv extension already present
	mod, err := loader.Load("./mymod.carv", mainPath)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if mod == nil {
		t.Fatal("expected module, got nil")
	}
}

func TestResolvePathRelativeFromSameDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-resolve-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	modPath := filepath.Join(tmpDir, "mymod.carv")
	err = os.WriteFile(modPath, []byte("pub let val = 1;"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	// fromFile is in the same directory as the module
	mod, err := loader.Load("./mymod", filepath.Join(tmpDir, "caller.carv"))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if mod == nil {
		t.Fatal("expected module, got nil")
	}
}

func TestResolvePathProjectSrc(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-resolve-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create src/utils.carv - the project-local path
	srcDir := filepath.Join(tmpDir, "src")
	err = os.MkdirAll(srcDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(srcDir, "utils.carv"), []byte("pub fn util() {}"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	// Non-relative, non-builtin import that exists in src/
	mod, err := loader.Load("utils", filepath.Join(tmpDir, "main.carv"))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if mod == nil {
		t.Fatal("expected module, got nil")
	}
	if !mod.Exports["util"] {
		t.Error("expected 'util' to be exported")
	}
}

func TestResolvePathFallbackToBase(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-resolve-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create helpers.carv at base (not in src/)
	err = os.WriteFile(filepath.Join(tmpDir, "helpers.carv"), []byte("pub fn help() {}"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	// Non-relative, non-builtin, not in src/ -> falls back to basePath/importPath.carv
	mod, err := loader.Load("helpers", filepath.Join(tmpDir, "main.carv"))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if mod == nil {
		t.Fatal("expected module, got nil")
	}
	if !mod.Exports["help"] {
		t.Error("expected 'help' to be exported")
	}
}

// ---------- resolvePackage ----------

func TestResolvePackageModCarv(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-pkg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create carv_modules/mypkg/mod.carv
	pkgDir := filepath.Join(tmpDir, "carv_modules", "mypkg")
	err = os.MkdirAll(pkgDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(pkgDir, "mod.carv"), []byte("pub fn init() {}"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	loader.SetConfig(&Config{
		Dependencies: map[string]Dependency{
			"mypkg": {Version: "1.0.0"},
		},
	})

	mod, err := loader.Load("mypkg", filepath.Join(tmpDir, "main.carv"))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if mod == nil {
		t.Fatal("expected module, got nil")
	}
	if !mod.Exports["init"] {
		t.Error("expected 'init' to be exported")
	}
}

func TestResolvePackageIndexCarv(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-pkg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create carv_modules/mypkg/index.carv (no mod.carv)
	pkgDir := filepath.Join(tmpDir, "carv_modules", "mypkg")
	err = os.MkdirAll(pkgDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(pkgDir, "index.carv"), []byte("pub fn start() {}"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	loader.SetConfig(&Config{
		Dependencies: map[string]Dependency{
			"mypkg": {Version: "1.0.0"},
		},
	})

	mod, err := loader.Load("mypkg", filepath.Join(tmpDir, "main.carv"))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if mod == nil {
		t.Fatal("expected module, got nil")
	}
	if !mod.Exports["start"] {
		t.Error("expected 'start' to be exported")
	}
}

func TestResolvePackageFallbackToName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-pkg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create carv_modules/mypkg/mypkg.carv (no mod.carv, no index.carv)
	pkgDir := filepath.Join(tmpDir, "carv_modules", "mypkg")
	err = os.MkdirAll(pkgDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(pkgDir, "mypkg.carv"), []byte("pub fn run() {}"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	loader.SetConfig(&Config{
		Dependencies: map[string]Dependency{
			"mypkg": {Version: "1.0.0"},
		},
	})

	mod, err := loader.Load("mypkg", filepath.Join(tmpDir, "main.carv"))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if mod == nil {
		t.Fatal("expected module, got nil")
	}
	if !mod.Exports["run"] {
		t.Error("expected 'run' to be exported")
	}
}

func TestResolvePackageWithLocalPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-pkg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a local package at a relative path
	localPkgDir := filepath.Join(tmpDir, "local_libs", "mypkg")
	err = os.MkdirAll(localPkgDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(localPkgDir, "mod.carv"), []byte("pub fn local() {}"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	loader.SetConfig(&Config{
		Dependencies: map[string]Dependency{
			"mypkg": {Path: "local_libs/mypkg"},
		},
	})

	mod, err := loader.Load("mypkg", filepath.Join(tmpDir, "main.carv"))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if mod == nil {
		t.Fatal("expected module, got nil")
	}
	if !mod.Exports["local"] {
		t.Error("expected 'local' to be exported")
	}
}

func TestResolvePackageWithAbsolutePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-pkg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a local package at an absolute path
	absPkgDir := filepath.Join(tmpDir, "abs_libs", "mypkg")
	err = os.MkdirAll(absPkgDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(absPkgDir, "mod.carv"), []byte("pub fn absolute() {}"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	loader.SetConfig(&Config{
		Dependencies: map[string]Dependency{
			"mypkg": {Path: absPkgDir},
		},
	})

	mod, err := loader.Load("mypkg", filepath.Join(tmpDir, "main.carv"))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if mod == nil {
		t.Fatal("expected module, got nil")
	}
	if !mod.Exports["absolute"] {
		t.Error("expected 'absolute' to be exported")
	}
}

// ---------- Load edge cases ----------

func TestLoaderLoadCachedBuiltinModule(t *testing.T) {
	loader := NewLoader("/tmp/test")

	// Load builtin module twice; second should be cached
	mod1, err := loader.Load("gpio", "/tmp/main.carv")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	mod2, err := loader.Load("gpio", "/tmp/main.carv")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if mod1 != mod2 {
		t.Error("expected cached builtin module to return same pointer")
	}
}

func TestLoaderLoadCachedFileModule(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-cache-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(filepath.Join(tmpDir, "mod.carv"), []byte("pub fn f() {}"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	mainPath := filepath.Join(tmpDir, "main.carv")

	mod1, err := loader.Load("./mod", mainPath)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	mod2, err := loader.Load("./mod", mainPath)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if mod1 != mod2 {
		t.Error("expected cached file module to return same pointer")
	}
}

func TestLoaderLoadFileNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-notfound-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	loader := NewLoader(tmpDir)
	_, err = loader.Load("./nonexistent", filepath.Join(tmpDir, "main.carv"))
	if err == nil {
		t.Error("expected error for nonexistent module file")
	}
}

func TestLoaderLoadParseError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-parse-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write invalid carv source that triggers parse errors
	err = os.WriteFile(filepath.Join(tmpDir, "bad.carv"), []byte("fn {}{}{}}}}}"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	_, err = loader.Load("./bad", filepath.Join(tmpDir, "main.carv"))
	if err == nil {
		t.Error("expected parse error")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("expected *ParseError, got %T", err)
	}
}

// ---------- LoadConfig edge cases ----------

func TestLoadConfigInvalidTOML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-badcfg-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(filepath.Join(tmpDir, "carv.toml"), []byte("[[[invalid toml"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadConfig(tmpDir)
	if err == nil {
		t.Error("expected error for invalid TOML config")
	}
}

func TestConfigSaveWithDependencies(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-savedep-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &Config{
		Package: PackageInfo{
			Name:        "deptest",
			Version:     "2.0.0",
			Description: "test with deps",
			Authors:     []string{"tester"},
			License:     "MIT",
		},
		Dependencies: map[string]Dependency{
			"foo": {Version: "1.0.0", Git: "https://example.com/foo.git"},
			"bar": {Path: "../bar"},
		},
		DevDeps: map[string]Dependency{
			"testlib": {Version: "0.1.0"},
		},
		Build: BuildConfig{
			Output:    "out",
			Target:    "arm",
			Optimize:  true,
			Debug:     true,
			Includes:  []string{"include/"},
			Libraries: []string{"libfoo"},
		},
		Scripts: map[string]string{
			"test": "carv test",
		},
	}

	err = cfg.Save(tmpDir)
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	if loaded.Package.Name != "deptest" {
		t.Errorf("expected name 'deptest', got %q", loaded.Package.Name)
	}
	if loaded.Dependencies["foo"].Git != "https://example.com/foo.git" {
		t.Errorf("expected foo git dep, got %+v", loaded.Dependencies["foo"])
	}
	if loaded.Dependencies["bar"].Path != "../bar" {
		t.Errorf("expected bar path dep, got %+v", loaded.Dependencies["bar"])
	}
	if loaded.DevDeps["testlib"].Version != "0.1.0" {
		t.Errorf("unexpected dev dep: %+v", loaded.DevDeps["testlib"])
	}
}

func TestConfigSaveBadDir(t *testing.T) {
	cfg := DefaultConfig("test")
	err := cfg.Save("/nonexistent/dir/that/should/not/exist")
	if err == nil {
		t.Error("expected error when saving to nonexistent directory")
	}
}

// ---------- FindProjectRoot edge cases ----------

func TestFindProjectRootNoConfig(t *testing.T) {
	// Create a directory tree with no carv.toml anywhere
	tmpDir, err := os.MkdirTemp("", "carv-noroot-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "a", "b", "c")
	err = os.MkdirAll(subDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// FindProjectRoot traverses to filesystem root and returns it
	root, err := FindProjectRoot(subDir)
	if err != nil {
		t.Fatalf("FindProjectRoot error: %v", err)
	}

	// It should return the filesystem root "/" when no carv.toml is found
	if root != "/" {
		t.Errorf("expected filesystem root '/', got %q", root)
	}
}

func TestFindProjectRootFromConfigDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carv-rootdir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	err = os.WriteFile(filepath.Join(tmpDir, "carv.toml"), []byte("[package]\nname=\"test\"\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// Start from the directory that contains carv.toml
	root, err := FindProjectRoot(tmpDir)
	if err != nil {
		t.Fatalf("FindProjectRoot error: %v", err)
	}

	absDir, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatalf("filepath.Abs error: %v", err)
	}
	if root != absDir {
		t.Errorf("expected %q, got %q", absDir, root)
	}
}

// ---------- NewLoader with relative path ----------

func TestNewLoaderRelativePath(t *testing.T) {
	loader := NewLoader(".")
	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}
	// basePath should be converted to absolute
	if loader.basePath == "." {
		t.Error("expected basePath to be resolved to absolute path")
	}
}

// ---------- BuiltinModuleExports for non-existent ----------

func TestBuiltinModuleExportsNonExistent(t *testing.T) {
	exports, ok := BuiltinModuleExports("nonexistent")
	if ok {
		t.Error("expected ok to be false for nonexistent module")
	}
	if exports != nil {
		t.Error("expected nil exports for nonexistent module")
	}
}

// ---------- Load all builtin modules ----------

func TestLoaderLoadAllBuiltinModules(t *testing.T) {
	builtins := []string{"gpio", "uart", "spi", "i2c", "timer"}
	for _, name := range builtins {
		loader := NewLoader("/tmp/test")
		mod, err := loader.Load(name, "/tmp/main.carv")
		if err != nil {
			t.Fatalf("Load(%q) error: %v", name, err)
		}
		if mod == nil {
			t.Fatalf("Load(%q) returned nil", name)
		}
		if !mod.IsBuiltin {
			t.Errorf("expected %q to be builtin", name)
		}
		if len(mod.Exports) == 0 {
			t.Errorf("expected %q to have exports", name)
		}
	}
}
