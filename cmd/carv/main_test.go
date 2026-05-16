package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildCLI(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	bin := filepath.Join(tmpDir, "carv-test")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build CLI: %v\n%s", err, string(out))
	}
	return bin
}

func runCLI(t *testing.T, bin string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	return string(out), exitCode
}

func TestVersion(t *testing.T) {
	bin := buildCLI(t)
	out, code := runCLI(t, bin, "version")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(out, "carv") {
		t.Fatalf("expected version output to contain 'carv', got: %s", out)
	}
}

func TestHelp(t *testing.T) {
	bin := buildCLI(t)
	out, code := runCLI(t, bin, "help")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	for _, want := range []string{"build", "emit-c", "init", "add", "remove", "install", "version"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected help to mention %q, got:\n%s", want, out)
		}
	}
}

func TestNoArgs(t *testing.T) {
	bin := buildCLI(t)
	out, code := runCLI(t, bin)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected usage output, got: %s", out)
	}
}

func TestUnknownCommand(t *testing.T) {
	bin := buildCLI(t)
	_, code := runCLI(t, bin, "foobar")
	if code != 1 {
		t.Fatalf("expected exit code 1 for unknown command, got %d", code)
	}
}

func TestBuildMissingFile(t *testing.T) {
	bin := buildCLI(t)
	_, code := runCLI(t, bin, "build")
	if code != 1 {
		t.Fatalf("expected exit code 1 for missing file, got %d", code)
	}
}

func TestEmitCMissingFile(t *testing.T) {
	bin := buildCLI(t)
	_, code := runCLI(t, bin, "emit-c")
	if code != 1 {
		t.Fatalf("expected exit code 1 for missing file, got %d", code)
	}
}

func TestInitCreatesProject(t *testing.T) {
	bin := buildCLI(t)
	tmpDir := t.TempDir()

	cmd := exec.Command(bin, "init")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("carv init failed: %v\n%s", err, string(out))
	}

	cfgPath := filepath.Join(tmpDir, "carv.toml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatal("expected carv.toml to be created")
	}

	srcPath := filepath.Join(tmpDir, "src", "main.carv")
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		t.Fatal("expected src/main.carv to be created")
	}
}

func TestEmitCGeneratesOutput(t *testing.T) {
	bin := buildCLI(t)
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "hello.carv")
	if err := os.WriteFile(src, []byte(`let x = 42;`), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	out, code := runCLI(t, bin, "emit-c", src)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(out, "#include") {
		t.Fatalf("expected C output to contain #include, got:\n%s", out)
	}
	if !strings.Contains(out, "int main") {
		t.Fatalf("expected C output to contain main, got:\n%s", out)
	}
}

func TestBuildGeneratesBinary(t *testing.T) {
	bin := buildCLI(t)
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "hello.carv")
	if err := os.WriteFile(src, []byte(`let x = 42;`), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	outPath := filepath.Join(tmpDir, "hello")
	_, code := runCLI(t, bin, "build", src)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatal("expected built binary to exist")
	}
}
