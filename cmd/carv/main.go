package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dev-dami/carv/pkg/codegen"
	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/module"
	"github.com/dev-dami/carv/pkg/parser"
	"github.com/dev-dami/carv/pkg/types"
)

const version = "0.3.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "version", "-v", "--version":
		fmt.Printf("carv %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	case "build":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: carv build [--target arm] <file.carv>")
			os.Exit(1)
		}
		target := ""
		fileArg := os.Args[2]
		if os.Args[2] == "--target" {
			if len(os.Args) < 5 {
				fmt.Fprintln(os.Stderr, "usage: carv build --target <arm|host> <file.carv>")
				os.Exit(1)
			}
			target = os.Args[3]
			fileArg = os.Args[4]
		}
		buildFile(fileArg, target)
	case "emit-c":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: carv emit-c <file.carv>")
			os.Exit(1)
		}
		emitC(os.Args[2])
	case "init":
		initProject()
	case "add":
		addPackage()
	case "remove":
		removePackage()
	case "install":
		installPackages()
	default:
		if strings.HasSuffix(os.Args[1], ".carv") {
			buildFile(os.Args[1], "")
		} else {
			fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Println(`carv - a memory-safe language for embedded systems

Usage:
  carv <command> [arguments]

Commands:
  build <file>    Compile to native binary via C
  emit-c <file>   Output generated C code
  init            Initialize a new Carv project with carv.toml
  add <name>      Add a dependency to carv.toml
  remove <name>   Remove a dependency from carv.toml
  install         Install all dependencies from carv.toml
  version         Print version info
  help            Show this help

Package Management:
  carv add <name> [--git <url>] [--path <localpath>] [--version <ver>]
  carv remove <name>
  carv install

Examples:
  carv build hello.carv
  carv emit-c hello.carv
  carv hello.carv
  carv init
  carv add mylib --git https://github.com/user/mylib
  carv remove mylib
  carv install`)
}

func initProject() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting current directory: %s\n", err)
		os.Exit(1)
	}

	name := filepath.Base(cwd)
	cfg := module.DefaultConfig(name)

	if err := cfg.Save(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "error creating carv.toml: %s\n", err)
		os.Exit(1)
	}

	srcDir := filepath.Join(cwd, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating src directory: %s\n", err)
		os.Exit(1)
	}

	mainFile := filepath.Join(srcDir, "main.carv")
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		mainContent := `// Welcome to Carv!

fn main() {
    let name = "World";
    println(f"Hello, {name}!");
}

main();
`
		if err := os.WriteFile(mainFile, []byte(mainContent), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error creating main.carv: %s\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Initialized Carv project '%s'\n", name)
	fmt.Println("  Created carv.toml")
	fmt.Println("  Created src/main.carv")
	fmt.Println("\nBuild your project with:")
	fmt.Println("  carv build src/main.carv")
}

func emitC(filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %s\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, msg := range p.Errors() {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(1)
	}

	checker := types.NewChecker()
	if !checker.Check(program) {
		for _, msg := range checker.Errors() {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(1)
	}

	if len(checker.Warnings()) > 0 {
		for _, msg := range checker.Warnings() {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(1)
	}

	gen := codegen.NewCGenerator()
	gen.SetTypeInfo(checker.TypeInfo())
	cCode := gen.Generate(program)
	fmt.Print(cCode)
}

func buildFile(filename string, target string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %s\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, msg := range p.Errors() {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(1)
	}

	checker := types.NewChecker()
	if !checker.Check(program) {
		for _, msg := range checker.Errors() {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(1)
	}

	if len(checker.Warnings()) > 0 {
		for _, msg := range checker.Warnings() {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(1)
	}

	gen := codegen.NewCGenerator()
	gen.SetTypeInfo(checker.TypeInfo())
	cCode := gen.Generate(program)

	baseName := strings.TrimSuffix(filename, ".carv")
	cFile := baseName + ".c"
	outFile := baseName

	if err := os.WriteFile(cFile, []byte(cCode), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing C file: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s\n", cFile)

	var compiler string
	var flags []string

	switch target {
	case "arm":
		compiler = "arm-none-eabi-gcc"
		outFile += ".elf"
		flags = []string{"-mcpu=cortex-m4", "-mthumb", "-Os", "-ffreestanding", "-nostdlib", "-DCARV_TARGET_ARM", "-o", outFile, cFile}
	default:
		compiler = "gcc"
		flags = []string{"-O2", "-o", outFile, cFile}
	}

	fmt.Printf("Compiling: %s %s\n", compiler, strings.Join(flags, " "))

	if err := runCmd(compiler, flags...); err != nil {
		fmt.Fprintf(os.Stderr, "compilation failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Built %s\n", outFile)
}

func runCmd(name string, args ...string) error {
	cmd := &exec.Cmd{
		Path:   name,
		Args:   append([]string{name}, args...),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if execPath, err := exec.LookPath(name); err == nil {
		cmd.Path = execPath
	}

	return cmd.Run()
}

// parsePkgFlags parses --git, --path, --version flags from os.Args starting at the given index.
func parsePkgFlags(args []string) (gitURL, localPath, ver string) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--git":
			if i+1 < len(args) {
				gitURL = args[i+1]
				i++
			}
		case "--path":
			if i+1 < len(args) {
				localPath = args[i+1]
				i++
			}
		case "--version":
			if i+1 < len(args) {
				ver = args[i+1]
				i++
			}
		}
	}
	return
}

func addPackage() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: carv add <name> [--git <url>] [--path <localpath>] [--version <ver>]")
		os.Exit(1)
	}

	name := os.Args[2]
	gitURL, localPath, ver := parsePkgFlags(os.Args[3:])

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	root, err := module.FindProjectRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error finding project root: %s\n", err)
		os.Exit(1)
	}

	cfg, err := module.LoadConfig(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading carv.toml: %s\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		fmt.Fprintln(os.Stderr, "error: no carv.toml found. Run 'carv init' first.")
		os.Exit(1)
	}

	dep := module.Dependency{
		Version: ver,
		Git:     gitURL,
		Path:    localPath,
	}

	if cfg.Dependencies == nil {
		cfg.Dependencies = make(map[string]module.Dependency)
	}
	cfg.Dependencies[name] = dep

	if err := cfg.Save(root); err != nil {
		fmt.Fprintf(os.Stderr, "error saving carv.toml: %s\n", err)
		os.Exit(1)
	}

	// If it's a git dependency, clone it immediately.
	if gitURL != "" {
		modDir := filepath.Join(root, "carv_modules", name)
		if _, err := os.Stat(modDir); err == nil {
			fmt.Printf("  %s already installed, skipping clone\n", name)
		} else {
			if err := gitClone(gitURL, "", modDir); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to clone %s: %s\n", name, err)
			}
		}
	}

	fmt.Printf("Added dependency '%s' to carv.toml\n", name)
}

func removePackage() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: carv remove <name>")
		os.Exit(1)
	}

	name := os.Args[2]

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	root, err := module.FindProjectRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error finding project root: %s\n", err)
		os.Exit(1)
	}

	cfg, err := module.LoadConfig(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading carv.toml: %s\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		fmt.Fprintln(os.Stderr, "error: no carv.toml found. Run 'carv init' first.")
		os.Exit(1)
	}

	if _, ok := cfg.Dependencies[name]; !ok {
		fmt.Fprintf(os.Stderr, "dependency '%s' not found in carv.toml\n", name)
		os.Exit(1)
	}

	delete(cfg.Dependencies, name)

	if err := cfg.Save(root); err != nil {
		fmt.Fprintf(os.Stderr, "error saving carv.toml: %s\n", err)
		os.Exit(1)
	}

	// Remove the installed module directory.
	modDir := filepath.Join(root, "carv_modules", name)
	if err := os.RemoveAll(modDir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %s\n", modDir, err)
	}

	fmt.Printf("Removed dependency '%s'\n", name)
}

func installPackages() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	root, err := module.FindProjectRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error finding project root: %s\n", err)
		os.Exit(1)
	}

	cfg, err := module.LoadConfig(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading carv.toml: %s\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		fmt.Fprintln(os.Stderr, "error: no carv.toml found. Run 'carv init' first.")
		os.Exit(1)
	}

	modsDir := filepath.Join(root, "carv_modules")
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating carv_modules: %s\n", err)
		os.Exit(1)
	}

	if len(cfg.Dependencies) == 0 {
		fmt.Println("No dependencies to install.")
		return
	}

	var lockPkgs []module.LockedPackage
	installed := 0
	skipped := 0

	for name, dep := range cfg.Dependencies {
		destDir := filepath.Join(modsDir, name)

		if _, err := os.Stat(destDir); err == nil {
			fmt.Printf("  %s already installed, skipping\n", name)
			skipped++
			// Still record in lock file.
			lp := module.LockedPackage{
				Name:    name,
				Version: dep.Version,
			}
			if dep.Git != "" {
				lp.Source = "git+" + dep.Git
				lp.Revision = getGitRevision(destDir)
			} else if dep.Path != "" {
				lp.Source = "path+" + dep.Path
			}
			lockPkgs = append(lockPkgs, lp)
			continue
		}

		if dep.Git != "" {
			branch := dep.Branch
			if branch == "" && dep.Tag != "" {
				branch = dep.Tag
			}
			fmt.Printf("  Cloning %s from %s...\n", name, dep.Git)
			if err := gitClone(dep.Git, branch, destDir); err != nil {
				fmt.Fprintf(os.Stderr, "  error installing %s: %s\n", name, err)
				continue
			}
			lp := module.LockedPackage{
				Name:     name,
				Version:  dep.Version,
				Source:   "git+" + dep.Git,
				Revision: getGitRevision(destDir),
			}
			lockPkgs = append(lockPkgs, lp)
			installed++
		} else if dep.Path != "" {
			srcPath := dep.Path
			if !filepath.IsAbs(srcPath) {
				srcPath = filepath.Join(root, srcPath)
			}
			fmt.Printf("  Linking %s from %s...\n", name, srcPath)
			if err := os.Symlink(srcPath, destDir); err != nil {
				// Fallback to copy.
				fmt.Printf("  Symlink failed, copying instead...\n")
				if err := copyDir(srcPath, destDir); err != nil {
					fmt.Fprintf(os.Stderr, "  error installing %s: %s\n", name, err)
					continue
				}
			}
			lp := module.LockedPackage{
				Name:    name,
				Version: dep.Version,
				Source:  "path+" + dep.Path,
			}
			lockPkgs = append(lockPkgs, lp)
			installed++
		} else {
			fmt.Fprintf(os.Stderr, "  %s: no git or path source specified, skipping\n", name)
			skipped++
		}
	}

	// Write lock file.
	lf := &module.LockFile{Packages: lockPkgs}
	if err := module.SaveLock(root, lf); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write carv.lock: %s\n", err)
	}

	fmt.Printf("\nInstalled %d, skipped %d dependencies.\n", installed, skipped)
}

// gitClone clones a git repository. If branch is non-empty, it clones that branch.
func gitClone(url, branch, destDir string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}

	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, url, destDir)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// getGitRevision returns the HEAD commit hash for a git repo directory.
func getGitRevision(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// copyDir recursively copies a directory tree.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		return copyFile(path, targetPath)
	})
}

// copyFile copies a single file.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
