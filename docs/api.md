[← Back to README](../README.md) | **API Reference** | [Language Guide →](language.md)

---

# Carv API Reference

This document explains the public Go package APIs used by the Carv CLI and tooling integrations.

## Compiler Pipeline

`source -> lexer -> parser -> types -> eval/codegen`

Typical flow in tools:

1. Create a lexer from source text.
2. Parse into an AST program.
3. Run type checking and inspect structured issues.
4. Execute with the interpreter (`pkg/eval`) or generate C (`pkg/codegen`).

## Package Guide

### `pkg/lexer`

Purpose: tokenize source with line/column tracking.

Key API:
- `lexer.New(input string) *Lexer`
- `(*Lexer).NextToken() lexer.Token`
- `lexer.LookupIdent(string) lexer.TokenType`

Usage pattern:

```go
l := lexer.New(src)
for tok := l.NextToken(); tok.Type != lexer.TOKEN_EOF; tok = l.NextToken() {
    // consume tokens
}
```

### `pkg/parser`

Purpose: syntax analysis with Pratt expression parsing + recursive descent statements/declarations.

Key API:
- `parser.New(l *lexer.Lexer) *Parser`
- `(*Parser).ParseProgram() *ast.Program`
- `(*Parser).Errors() []string`

Usage pattern:

```go
p := parser.New(lexer.New(src))
prog := p.ParseProgram()
if len(p.Errors()) > 0 {
    // handle syntax errors
}
```

### `pkg/types`

Purpose: semantic analysis (typing, ownership/borrow checks, interface validation, async checks).

Key API:
- `types.NewChecker() *types.Checker`
- `(*Checker).Check(program *ast.Program) *types.CheckResult`
- `(*Checker).ErrorIssues() []types.CheckIssue`
- `(*Checker).WarningIssues() []types.CheckIssue`

Structured diagnostics:
- `CheckIssue{Line, Column, Kind, Message}`

Design notes:
- Ownership/borrow/interface/async rules are intentionally separated into focused files.
- `Errors()`/`Warnings()` remain available as formatted string compatibility helpers.

### `pkg/eval`

Purpose: tree-walking interpreter and runtime object system.

Key API:
- `eval.NewEnvironment() *eval.Environment`
- `eval.NewEnclosedEnvironment(outer *Environment) *Environment`
- `eval.Eval(node ast.Node, env *Environment) eval.Object`
- `eval.SetArgs(args []string)`
- `eval.SetModuleLoader(loader *module.Loader)`

Design notes:
- Built-ins include string/array/map/file/process/env helpers plus TCP primitives (`tcp_listen`, `tcp_accept`, `tcp_read`, `tcp_write`, `tcp_close`).
- `net` and `web` module aliases expose the same TCP primitives.

### `pkg/codegen`

Purpose: emit C99 code from typed AST.

Key API:
- `codegen.NewCGenerator() *codegen.CGenerator`
- `(*CGenerator).Generate(program *ast.Program) (string, error)`

Design notes:
- Preserves ownership semantics through generated move/drop/clone logic.
- Lowers async functions to frame structs + poll state machines.
- Lowers interfaces to vtables and fat pointers.

### `pkg/module`

Purpose: module resolution/loading and project configuration (`carv.toml`).

Key API:
- `module.NewLoader(basePath string) *module.Loader`
- `(*Loader).Load(importPath string, fromFile string) (*module.Module, error)`
- `module.LoadConfig(dir string) (*module.Config, error)`
- `module.FindProjectRoot(startDir string) (string, error)`

Design notes:
- Supports relative imports, project-local imports, and built-in modules (`net`, `web`).

### `pkg/ast`

Purpose: shared AST node definitions used by parser, checker, eval, and codegen.

Usage pattern:
- Parsers construct AST nodes.
- All downstream passes switch on concrete node types.
- Source locations are retained for diagnostics.

---

[← Back to README](../README.md) | **API Reference** | [Language Guide →](language.md)
