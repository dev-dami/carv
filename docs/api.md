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
- `(*Config).Save(dir string) error`
- `module.FindProjectRoot(startDir string) (string, error)`
- `module.DefaultConfig(name string) *module.Config`
- `module.LoadLock(dir string) (*module.LockFile, error)`
- `module.SaveLock(dir string, lf *module.LockFile) error`

Data types:
- `Config` — full carv.toml structure (package info, dependencies, build config, scripts)
- `Dependency` — version, git URL, branch, tag, or local path
- `LockFile` / `LockedPackage` — pinned dependency versions with source and revision

Design notes:
- Supports relative imports, project-local imports, built-in modules, and external packages from `carv_modules/`.
- Lock files enable reproducible builds by pinning exact git revisions.

### `pkg/semver`

Purpose: semantic version parsing and constraint matching for the package manager.

Key API:
- `semver.ParseVersion(s string) (Version, error)`
- `semver.ParseConstraint(s string) (Constraint, error)`
- `semver.ParseConstraints(s string) (Constraints, error)`
- `(Constraint).Matches(v Version) bool`
- `(Constraints).Matches(v Version) bool`
- `semver.Satisfies(versionStr, constraintStr string) (bool, error)`

Supported constraints:
- Comparison: `=`, `!=`, `>`, `<`, `>=`, `<=`
- Caret: `^1.2.3` (compatible with 1.x.x)
- Tilde: `~1.2.3` (approximately 1.2.x)
- Wildcard: `*` (any version)
- Compound: `>=1.0.0, <2.0.0`

### `pkg/resolver`

Purpose: transitive dependency resolution with cycle detection and GitHub tag resolution.

Key API:
- `resolver.New(rootDir string) *Resolver`
- `(*Resolver).Resolve(cfg *module.Config) ([]*ResolvedDep, error)`
- `(*Resolver).LockFile() *module.LockFile`
- `resolver.PrintTree(deps []*ResolvedDep, prefix string)`

Data types:
- `ResolvedDep` — resolved dependency with name, version, source, revision, and transitive deps

Design notes:
- Resolves git dependencies by fetching tags via `git ls-remote` and matching against semver constraints.
- Path-based dependencies are symlinked (with copy fallback).
- Cycle detection via visiting set prevents infinite recursion.

### `pkg/lsp`

Purpose: Language Server Protocol implementation for editor integration.

Key API:
- `lsp.RunServer()` — starts the LSP server on stdin/stdout

Protocol: LSP 3.16 via `opa-oz/glsp`, stdio transport.

Provided capabilities:
- `textDocument/diagnostic` — type errors and warnings
- `textDocument/hover` — type information for symbols
- `textDocument/definition` — go-to-definition using tracked symbol positions
- `textDocument/completion` — keyword and symbol suggestions

---

[← Back to README](../README.md) | **API Reference** | [Language Guide →](language.md)
