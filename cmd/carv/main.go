package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dev-dami/carv/pkg/codegen"
	"github.com/dev-dami/carv/pkg/eval"
	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/parser"
	"github.com/dev-dami/carv/pkg/types"
)

const version = "0.0.1-dev"

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
		runFile(os.Args[2])
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
	default:
		if strings.HasSuffix(os.Args[1], ".carv") {
			runFile(os.Args[1])
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
  repl         Start interactive REPL
  version      Print version info
  help         Show this help

Examples:
  carv run hello.carv
  carv build hello.carv
  carv hello.carv
  carv repl`)
}

func runFile(filename string) {
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
	cCode := gen.Generate(program)

	baseName := strings.TrimSuffix(filename, ".carv")
	cFile := baseName + ".c"
	outFile := baseName

	if err := os.WriteFile(cFile, []byte(cCode), 0644); err != nil {
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

	if filepath, err := exec.LookPath(name); err == nil {
		cmd.Path = filepath
	}

	return cmd.Run()
}
