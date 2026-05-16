package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dev-dami/carv/pkg/codegen"
	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/parser"
	"github.com/dev-dami/carv/pkg/types"
	"github.com/peterh/liner"
)

const replHistoryFile = ".carv_repl_history"

var replCommands = []struct {
	Name        string
	Description string
}{
	{":help", "Show this help message"},
	{":type <expr>", "Type-check an expression without running it"},
	{":load <file>", "Load and run a .carv file"},
	{":clear", "Clear the screen"},
	{":quit / :exit", "Exit the REPL"},
}

var carvKeywords = []string{
	"let", "mut", "const", "fn", "if", "else", "for", "in", "while", "loop",
	"return", "break", "continue", "match", "class", "interface", "impl",
	"async", "await", "spawn", "try", "new", "cast", "self", "static",
	"volatile", "packed", "unsafe", "asm", "require", "true", "false", "nil",
	"void", "int", "bool", "string", "char", "any",
	"u8", "u16", "u32", "u64", "i8", "i16", "i32", "i64",
	"f32", "f64", "usize", "isize",
	"println", "print", "len", "push", "pop", "range", "typeof",
}

func runREPL() {
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	historyPath := filepath.Join(os.Getenv("HOME"), replHistoryFile)
	if f, err := os.Open(historyPath); err == nil {
		_, _ = line.ReadHistory(f)
		f.Close()
	}
	defer func() {
		if f, err := os.Create(historyPath); err == nil {
			_, _ = line.WriteHistory(f)
			f.Close()
		}
	}()

	line.SetCompleter(func(input string) []string {
		lower := strings.ToLower(input)
		var completions []string
		for _, kw := range carvKeywords {
			if strings.HasPrefix(kw, lower) {
				completions = append(completions, kw)
			}
		}
		for _, cmd := range replCommands {
			if strings.HasPrefix(cmd.Name, lower) {
				completions = append(completions, cmd.Name)
			}
		}
		return completions
	})

	fmt.Printf("Carv REPL v%s\nType :help for available commands.\n\n", version)

	var buffer strings.Builder
	multiLine := false
	var sessionDecls strings.Builder

	for {
		prompt := ">>> "
		if multiLine {
			prompt = "... "
		}

		input, err := line.Prompt(prompt)
		if err != nil {
			fmt.Println()
			break
		}

		line.AppendHistory(input)

		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			continue
		}

		if multiLine {
			buffer.WriteString("\n")
			buffer.WriteString(trimmed)
			if !needsMoreInput(buffer.String()) {
				multiLine = false
				evalREPL(line, buffer.String(), &sessionDecls)
				buffer.Reset()
			}
			continue
		}

		if strings.HasPrefix(trimmed, ":") {
			handleCommand(trimmed)
			continue
		}

		if needsMoreInput(trimmed) {
			buffer.WriteString(trimmed)
			multiLine = true
			continue
		}

		evalREPL(line, trimmed, &sessionDecls)
	}
}

func isPrintCall(input string) bool {
	trimmed := strings.TrimSpace(input)
	return strings.HasPrefix(trimmed, "print(") || strings.HasPrefix(trimmed, "println(")
}

func isAssignmentExpr(input string) bool {
	trimmed := strings.TrimSpace(input)
	if strings.HasSuffix(trimmed, ";") {
		trimmed = strings.TrimSpace(trimmed[:len(trimmed)-1])
	}
	for _, op := range []string{"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "="} {
		idx := strings.Index(trimmed, op)
		if idx > 0 && !strings.Contains(trimmed[:idx], "(") {
			return true
		}
	}
	return false
}

func needsMoreInput(input string) bool {
	openBrace := strings.Count(input, "{") - strings.Count(input, "}")
	openParen := strings.Count(input, "(") - strings.Count(input, ")")
	openBracket := strings.Count(input, "[") - strings.Count(input, "]")
	return openBrace > 0 || openParen > 0 || openBracket > 0
}

func handleCommand(cmd string) {
	parts := strings.SplitN(cmd, " ", 2)
	switch parts[0] {
	case ":help":
		fmt.Println("Carv REPL Commands:")
		fmt.Println("-------------------")
		for _, c := range replCommands {
			fmt.Printf("  %-20s %s\n", c.Name, c.Description)
		}
	case ":clear":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		_ = cmd.Run()
	case ":quit", ":exit":
		fmt.Println("Bye!")
		os.Exit(0)
	case ":type":
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			fmt.Fprintln(os.Stderr, "Usage: :type <expression>")
			return
		}
		typeCheckREPL(parts[1])
	case ":load":
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			fmt.Fprintln(os.Stderr, "Usage: :load <file.carv>")
			return
		}
		loadFileREPL(strings.TrimSpace(parts[1]))
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nType :help for available commands.\n", parts[0])
	}
}

func typeCheckREPL(input string) {
	src := input
	if !strings.Contains(input, "fn ") && !strings.Contains(input, "let ") && !strings.Contains(input, "class ") {
		src = "let _tmp = " + input + ";"
	}

	l := lexer.New(src)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Fprintln(os.Stderr, e)
		}
		return
	}

	checker := types.NewChecker()
	if checker.Check(program) {
		fmt.Println("OK")
	} else {
		for _, e := range checker.Errors() {
			fmt.Fprintln(os.Stderr, e)
		}
	}
}

func loadFileREPL(filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %s\n", err)
		return
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Fprintln(os.Stderr, e)
		}
		return
	}

	checker := types.NewChecker()
	if !checker.Check(program) {
		for _, e := range checker.Errors() {
			fmt.Fprintln(os.Stderr, e)
		}
		return
	}

	gen := codegen.NewCGenerator()
	gen.SetTypeInfo(checker.TypeInfo())
	cCode := gen.Generate(program)

	tmpDir, err := os.MkdirTemp("", "carv-repl-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating temp dir: %s\n", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	cFile := filepath.Join(tmpDir, "out.c")
	bin := filepath.Join(tmpDir, "out")
	if err := os.WriteFile(cFile, []byte(cCode), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing C file: %s\n", err)
		return
	}

	if out, err := exec.Command("gcc", "-O2", "-o", bin, cFile).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "gcc error: %s\n%s", err, string(out))
		return
	}

	if out, err := exec.Command(bin).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "runtime error: %s\n%s", err, string(out))
	} else {
		fmt.Print(string(out))
	}
}

func evalREPL(line *liner.State, input string, sessionDecls *strings.Builder) {
	src := input
	hasMain := strings.Contains(input, "fn main") || strings.Contains(input, "fn carv_main")
	isDecl := strings.HasPrefix(strings.TrimSpace(input), "let ") ||
		strings.HasPrefix(strings.TrimSpace(input), "mut ") ||
		strings.HasPrefix(strings.TrimSpace(input), "const ") ||
		strings.HasPrefix(strings.TrimSpace(input), "fn ") ||
		strings.HasPrefix(strings.TrimSpace(input), "class ") ||
		strings.HasPrefix(strings.TrimSpace(input), "interface ") ||
		strings.HasPrefix(strings.TrimSpace(input), "impl ")

	if isDecl {
		testSrc := input
		if !strings.HasSuffix(strings.TrimSpace(input), ";") && !strings.HasSuffix(strings.TrimSpace(input), "}") {
			testSrc = input + ";"
		}
		l := lexer.New(testSrc)
		p := parser.New(l)
		p.ParseProgram()
		if len(p.Errors()) > 0 {
			for _, e := range p.Errors() {
				fmt.Fprintln(os.Stderr, e)
			}
			return
		}

		sessionDecls.WriteString(input)
		if !strings.HasSuffix(strings.TrimSpace(input), ";") && !strings.HasSuffix(strings.TrimSpace(input), "}") {
			sessionDecls.WriteString(";")
		}
		sessionDecls.WriteString("\n")
		fmt.Println("OK")
		return
	}

	if !hasMain && !strings.Contains(input, "fn ") && !strings.Contains(input, "class ") && !strings.Contains(input, "interface ") {
		if isAssignmentExpr(input) {
			src = input + ";"
		} else if isPrintCall(input) || strings.HasSuffix(strings.TrimSpace(input), ";") {
			src = input
			if !strings.HasSuffix(strings.TrimSpace(input), ";") {
				src = input + ";"
			}
		} else {
			src = "println(" + input + ");"
		}
	}

	fullSrc := sessionDecls.String() + src

	l := lexer.New(fullSrc)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Fprintln(os.Stderr, e)
		}
		return
	}

	checker := types.NewChecker()
	if !checker.Check(program) {
		for _, e := range checker.Errors() {
			fmt.Fprintln(os.Stderr, e)
		}
		return
	}

	gen := codegen.NewCGenerator()
	gen.SetTypeInfo(checker.TypeInfo())
	cCode := gen.Generate(program)

	tmpDir, err := os.MkdirTemp("", "carv-repl-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating temp dir: %s\n", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	cFile := filepath.Join(tmpDir, "out.c")
	bin := filepath.Join(tmpDir, "out")
	if err := os.WriteFile(cFile, []byte(cCode), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing C file: %s\n", err)
		return
	}

	if out, err := exec.Command("gcc", "-O2", "-o", bin, cFile).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "gcc error: %s\n%s", err, string(out))
		return
	}

	if out, err := exec.Command(bin).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "runtime error: %s\n%s", err, string(out))
	} else {
		fmt.Print(string(out))
	}
}
