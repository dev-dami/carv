package main

import (
	"fmt"
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
  build <file>  Compile to native binary via C
  emit-c <file> Output generated C code
  init          Initialize a new Carv project with carv.toml
  version       Print version info
  help          Show this help

Examples:
  carv build hello.carv
  carv emit-c hello.carv
  carv hello.carv
  carv init`)
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
		flags = []string{"-mcpu=cortex-m4", "-mthumb", "-Os", "-ffreestanding", "-nostdlib", "-o", outFile, cFile}
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
