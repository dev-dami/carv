package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dev-dami/carv/pkg/codegen"
	"github.com/dev-dami/carv/pkg/eval"
	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/module"
	"github.com/dev-dami/carv/pkg/parser"
	"github.com/dev-dami/carv/pkg/types"
)

const version = "0.2.0"

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
	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: carv run <file.carv>")
			os.Exit(1)
		}
		runFile(os.Args[2], os.Args[3:])
	case "build":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: carv build <file.carv>")
			os.Exit(1)
		}
		buildFile(os.Args[2])
	case "emit-c":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: carv emit-c <file.carv>")
			os.Exit(1)
		}
		emitC(os.Args[2])
	case "repl":
		runRepl()
	case "init":
		initProject()
	default:
		if strings.HasSuffix(os.Args[1], ".carv") {
			runFile(os.Args[1], os.Args[2:])
		} else {
			fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Println(`carv - a memory-safe, concurrent scripting language

Usage:
  carv <command> [arguments]

Commands:
  run <file>   Run a Carv source file (interpreted)
  build <file> Compile to native binary via C
  emit-c <file> Output generated C code
  init         Initialize a new Carv project with carv.toml
  repl         Start interactive REPL
  version      Print version info
  help         Show this help

Examples:
  carv run hello.carv
  carv build hello.carv
  carv hello.carv
  carv init
  carv repl`)
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
	fmt.Println("\nRun your project with:")
	fmt.Println("  carv run src/main.carv")
}

func runFile(filename string, programArgs []string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %s\n", err)
		os.Exit(1)
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to resolve path: %s\n", err)
		absPath = filename
	}

	projectRoot, err := module.FindProjectRoot(filepath.Dir(absPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to find project root: %s\n", err)
		projectRoot = filepath.Dir(absPath)
	}

	cfg, err := module.LoadConfig(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load carv.toml: %s\n", err)
	}

	loader := module.NewLoader(projectRoot)
	if cfg != nil {
		loader.SetConfig(cfg)
	}

	eval.SetModuleLoader(loader)
	eval.SetCurrentFile(absPath)
	eval.SetArgs(programArgs)

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

	env := eval.NewEnvironment()
	result := eval.Eval(program, env)

	if result != nil {
		if errObj, ok := result.(*eval.Error); ok {
			fmt.Fprintln(os.Stderr, errObj.Inspect())
			os.Exit(1)
		}
	}
}

func runRepl() {
	scanner := bufio.NewScanner(os.Stdin)
	env := eval.NewEnvironment()

	fmt.Printf("Carv %s REPL\n", version)
	fmt.Println("Type 'exit' or Ctrl+D to quit")

	for {
		fmt.Print(">>> ")
		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		if line == "exit" || line == "quit" {
			break
		}
		if strings.TrimSpace(line) == "" {
			continue
		}

		l := lexer.New(line)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			for _, msg := range p.Errors() {
				fmt.Fprintln(os.Stderr, msg)
			}
			continue
		}

		result := eval.Eval(program, env)
		if result != nil {
			if result.Type() != eval.NIL_OBJ {
				fmt.Println(result.Inspect())
			}
		}
	}

	fmt.Println("\nGoodbye!")
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

	gen := codegen.NewCGenerator()
	gen.SetTypeInfo(checker.TypeInfo())
	cCode := gen.Generate(program)
	fmt.Print(cCode)
}

func buildFile(filename string) {
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

	cmd := fmt.Sprintf("gcc -O2 -o %s %s", outFile, cFile)
	fmt.Printf("Compiling: %s\n", cmd)

	if err := runCmd("gcc", "-O2", "-o", outFile, cFile); err != nil {
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
