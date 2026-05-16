package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dev-dami/carv/pkg/codegen"
	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/lsp"
	"github.com/dev-dami/carv/pkg/module"
	"github.com/dev-dami/carv/pkg/parser"
	"github.com/dev-dami/carv/pkg/resolver"
	"github.com/dev-dami/carv/pkg/types"
)

const version = "0.6.0"

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
	case "repl":
		runREPL()
	case "lsp":
		runLSP()
	case "pkg":
		handlePkg()
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
  repl            Start interactive REPL
  lsp             Start language server for editor integration
  pkg             Package manager (see below)
  version         Print version info
  help            Show this help

Package Management:
  carv add <name> [--git <url>] [--path <localpath>] [--version <ver>]
  carv remove <name>
  carv install
  carv pkg list              List installed dependencies
  carv pkg info <name>       Show details about a dependency
  carv pkg update [name]     Update dependencies to latest matching versions
  carv pkg publish           Publish current package to GitHub registry

Examples:
  carv build hello.carv
  carv emit-c hello.carv
  carv hello.carv
  carv init
  carv add mylib --git https://github.com/user/mylib
  carv remove mylib
  carv install
  carv pkg list
  carv pkg update
  carv pkg publish`)
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

	if len(cfg.Dependencies) == 0 {
		fmt.Println("No dependencies to install.")
		return
	}

	res := resolver.New(root)
	deps, err := res.Resolve(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolution failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("\nResolved dependencies:")
	resolver.PrintTree(deps, "  ")

	lf := res.LockFile()
	if err := module.SaveLock(root, lf); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write carv.lock: %s\n", err)
	}
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
func runLSP() {
	lsp.RunServer()
}

func handlePkg() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: carv pkg <list|info|update|publish> [args]")
		fmt.Fprintln(os.Stderr, "\nSubcommands:")
		fmt.Fprintln(os.Stderr, "  list              List installed dependencies")
		fmt.Fprintln(os.Stderr, "  info <name>       Show details about a dependency")
		fmt.Fprintln(os.Stderr, "  update [name]     Update dependencies to latest versions")
		fmt.Fprintln(os.Stderr, "  publish           Publish package to GitHub registry")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "list":
		pkgList()
	case "info":
		pkgInfo()
	case "update":
		pkgUpdate()
	case "publish":
		pkgPublish()
	default:
		fmt.Fprintf(os.Stderr, "unknown pkg subcommand: %s\n", os.Args[2])
		os.Exit(1)
	}
}

func pkgList() {
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

	fmt.Printf("Dependencies for %s:\n\n", cfg.Package.Name)

	// Load lock file for installed status.
	lf, _ := module.LoadLock(root)
	locked := make(map[string]module.LockedPackage)
	for _, p := range lf.Packages {
		locked[p.Name] = p
	}

	if len(cfg.Dependencies) == 0 {
		fmt.Println("  (none)")
		return
	}

	for name, dep := range cfg.Dependencies {
		ver := dep.Version
		if ver == "" {
			ver = "*"
		}
		status := "not installed"
		if lp, ok := locked[name]; ok {
			status = fmt.Sprintf("installed (%s)", lp.Version)
			if lp.Revision != "" && len(lp.Revision) >= 8 {
				status += fmt.Sprintf(" [%s]", lp.Revision[:8])
			}
		}

		source := dep.Git
		if source == "" && dep.Path != "" {
			source = dep.Path
		}

		fmt.Printf("  %-20s %-12s %-30s %s\n", name, ver, source, status)
	}
}

func pkgInfo() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: carv pkg info <name>")
		os.Exit(1)
	}

	name := os.Args[3]

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
	if err != nil || cfg == nil {
		fmt.Fprintln(os.Stderr, "error: no carv.toml found")
		os.Exit(1)
	}

	dep, ok := cfg.Dependencies[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "dependency %q not found\n", name)
		os.Exit(1)
	}

	lf, _ := module.LoadLock(root)
	var locked *module.LockedPackage
	for i, p := range lf.Packages {
		if p.Name == name {
			locked = &lf.Packages[i]
			break
		}
	}

	fmt.Printf("Name:        %s\n", name)
	fmt.Printf("Version:     %s\n", dep.Version)
	if dep.Git != "" {
		fmt.Printf("Git:         %s\n", dep.Git)
	}
	if dep.Path != "" {
		fmt.Printf("Path:        %s\n", dep.Path)
	}
	if dep.Branch != "" {
		fmt.Printf("Branch:      %s\n", dep.Branch)
	}
	if dep.Tag != "" {
		fmt.Printf("Tag:         %s\n", dep.Tag)
	}
	if locked != nil {
		fmt.Printf("Locked:      %s\n", locked.Version)
		if locked.Revision != "" {
			fmt.Printf("Revision:    %s\n", locked.Revision)
		}
		fmt.Printf("Source:      %s\n", locked.Source)
	}

	// Show transitive deps if installed.
	pkgDir := filepath.Join(root, "carv_modules", name)
	pkgCfg, err := module.LoadConfig(pkgDir)
	if err == nil && pkgCfg != nil && len(pkgCfg.Dependencies) > 0 {
		fmt.Printf("\nDependencies of %s:\n", name)
		for depName := range pkgCfg.Dependencies {
			fmt.Printf("  - %s\n", depName)
		}
	}
}

func pkgUpdate() {
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
	if err != nil || cfg == nil {
		fmt.Fprintln(os.Stderr, "error: no carv.toml found")
		os.Exit(1)
	}

	_ = os.Remove(filepath.Join(root, "carv.lock"))

	if len(os.Args) >= 4 {
		name := os.Args[3]
		_ = os.RemoveAll(filepath.Join(root, "carv_modules", name))
		fmt.Printf("Updating %s...\n", name)
	} else {
		_ = os.RemoveAll(filepath.Join(root, "carv_modules"))
		fmt.Println("Updating all dependencies...")
	}

	res := resolver.New(root)
	deps, err := res.Resolve(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolution failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("\nResolved dependencies:")
	resolver.PrintTree(deps, "  ")

	lf := res.LockFile()
	if err := module.SaveLock(root, lf); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write carv.lock: %s\n", err)
	}
}

func pkgPublish() {
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
	if err != nil || cfg == nil {
		fmt.Fprintln(os.Stderr, "error: no carv.toml found. Run 'carv init' first.")
		os.Exit(1)
	}

	if cfg.Package.Name == "" {
		fmt.Fprintln(os.Stderr, "error: package name not set in carv.toml")
		os.Exit(1)
	}
	if cfg.Package.Version == "" {
		fmt.Fprintln(os.Stderr, "error: package version not set in carv.toml")
		os.Exit(1)
	}

	// Check for git remote.
	remoteURL, err := getGitRemote(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: package directory is not a git repo with a remote")
		fmt.Fprintln(os.Stderr, "       Publish requires a git remote for GitHub releases")
		os.Exit(1)
	}

	// Create version tag.
	tag := "v" + cfg.Package.Version
	fmt.Printf("Publishing %s %s to %s...\n", cfg.Package.Name, tag, remoteURL)

	// Check if tag exists.
	if tagExists(root, tag) {
		fmt.Fprintf(os.Stderr, "tag %s already exists. Bump version in carv.toml first.\n", tag)
		os.Exit(1)
	}

	// Create and push tag.
	if err := runCmd("git", "-C", root, "tag", tag); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create tag: %s\n", err)
		os.Exit(1)
	}

	if err := runCmd("git", "-C", root, "push", "origin", tag); err != nil {
		fmt.Fprintf(os.Stderr, "failed to push tag: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Published %s %s\n", cfg.Package.Name, tag)
	fmt.Println("Create a GitHub release at:")
	fmt.Printf("  https://github.com/%s/releases/new?tag=%s\n", extractRepoPath(remoteURL), tag)
}

func getGitRemote(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func tagExists(dir, tag string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", tag)
	return cmd.Run() == nil
}

func extractRepoPath(url string) string {
	url = strings.TrimSuffix(url, ".git")
	if strings.HasPrefix(url, "git@github.com:") {
		return strings.TrimPrefix(url, "git@github.com:")
	}
	if strings.HasPrefix(url, "https://github.com/") {
		return strings.TrimPrefix(url, "https://github.com/")
	}
	return url
}
