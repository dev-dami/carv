# Embedded AOT Overhaul Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Transform Carv from a general-purpose language into an embedded-focused AOT-only language targeting ARM Cortex-M (M0–M7), removing the tree-walker interpreter, replacing the pipe operator with method chaining, and adding sized integer types.

**Architecture:** Evolutionary changes across all compiler passes (lexer → parser → AST → checker → codegen). Remove the `pkg/eval/` package entirely. Add new sized-integer token types and propagate them through the type system and C codegen. Replace pipe operator (`|>`) parsing with method chaining (`.` already works). Remove `run`/`repl` commands from CLI.

**Tech Stack:** Go, C codegen targeting `arm-none-eabi-gcc`

---

### Task 1: Remove the tree-walker interpreter

Delete `pkg/eval/` entirely and remove all references to it from `cmd/carv/main.go`.

**Files:**
- Delete: `pkg/eval/` (all files)
- Modify: `cmd/carv/main.go`

**Step 1: Delete the eval package**

```bash
rm -rf pkg/eval/
```

**Step 2: Update main.go — remove `run`, `repl`, and eval imports**

Remove the `eval` import and the `runFile`, `runRepl` functions. Update the CLI dispatch:

```go
// cmd/carv/main.go
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
	case "init":
		initProject()
	default:
		if strings.HasSuffix(os.Args[1], ".carv") {
			buildFile(os.Args[1])
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
  build <file> Compile to native binary via C
  emit-c <file> Output generated C code
  init         Initialize a new Carv project with carv.toml
  version      Print version info
  help         Show this help

Examples:
  carv build hello.carv
  carv emit-c hello.carv
  carv hello.carv
  carv init`)
}
```

Keep `initProject`, `emitC`, `buildFile`, and `runCmd` functions unchanged.

**Step 3: Verify compilation**

```bash
cd /home/damilare/Downloads/chessable && go build ./...
```

Expected: builds cleanly with no eval references.

**Step 4: Run existing tests (expect eval tests gone)**

```bash
go test ./pkg/lexer/ ./pkg/parser/ ./pkg/ast/ ./pkg/types/ ./pkg/codegen/ ./pkg/module/
```

Expected: all remaining tests pass.

**Step 5: Commit**

```bash
git add -A && git commit -m "refactor: remove tree-walker interpreter, AOT-only"
```

---

### Task 2: Remove pipe operator, keep method chaining

Remove `|>` and `<|` from lexer, parser, AST, checker, and codegen. Method chaining via `.` already works.

**Files:**
- Modify: `pkg/lexer/token.go` — remove TOKEN_PIPE, TOKEN_PIPE_BACK
- Modify: `pkg/lexer/lexer.go` — remove `|>` and `<|` scanning
- Modify: `pkg/ast/ast.go` — remove PipeExpression
- Modify: `pkg/parser/parser.go` — remove pipe infix registration
- Modify: `pkg/parser/pratt.go` — remove PIPE precedence, parsePipeExpression
- Modify: `pkg/types/checker.go` — remove checkPipeExpression
- Modify: `pkg/codegen/cgen.go` — remove generatePipeExpression and PipeExpression cases
- Delete/update: test files that test pipes

**Step 1: Remove pipe tokens from lexer**

In `token.go`: remove `TOKEN_PIPE` and `TOKEN_PIPE_BACK` constants and their entries in `tokenNames`.

In `lexer.go`: remove the `|>` and `<|` scanning branches.

**Step 2: Remove PipeExpression from AST**

In `ast.go`: remove the `PipeExpression` struct and its methods.

**Step 3: Remove pipe from parser**

In `parser.go`: remove lines registering `TOKEN_PIPE` and `TOKEN_PIPE_BACK` as infix operators.

In `pratt.go`: remove `PIPE` from precedence enum, remove TOKEN_PIPE/TOKEN_PIPE_BACK from precedences map, remove `parsePipeExpression` function.

**Step 4: Remove pipe from checker**

In `checker.go`: remove the `*ast.PipeExpression` case and `checkPipeExpression` method.

**Step 5: Remove pipe from codegen**

In `cgen.go`: remove the `*ast.PipeExpression` cases and `generatePipeExpression` method.

**Step 6: Update/remove pipe tests**

Remove pipe test functions from `parser_test.go`, `eval_test.go` (already gone), `ast_test.go`, `codegen/cgen_test.go`.

**Step 7: Verify**

```bash
go build ./... && go test ./...
```

**Step 8: Commit**

```bash
git add -A && git commit -m "refactor: remove pipe operator in favor of method chaining"
```

---

### Task 3: Add sized integer types to lexer and type system

Add `u8`, `u16`, `u32`, `u64`, `i8`, `i16`, `i32`, `i64`, `f32`, `f64`, `usize`, `isize` as first-class types. `int` stays as alias for `i32`, `float` stays as alias for `f32`.

**Files:**
- Modify: `pkg/lexer/token.go` — add new type tokens
- Modify: `pkg/lexer/lexer.go` — no changes needed (keywords map handles it)
- Modify: `pkg/parser/parser_types.go` — parse new type tokens
- Modify: `pkg/parser/parser.go` — register new type tokens as prefix parsers
- Modify: `pkg/types/types.go` — add new BasicType singletons and update IsNumeric/IsComparable
- Modify: `pkg/types/checker.go` — register new types in scope, update numeric checks
- Modify: `pkg/codegen/cgen.go` — map new types to C stdint.h types

**Step 1: Add type tokens**

In `token.go`, add after existing type tokens:

```go
TOKEN_U8_TYPE     // u8
TOKEN_U16_TYPE    // u16
TOKEN_U32_TYPE    // u32
TOKEN_U64_TYPE    // u64
TOKEN_I8_TYPE     // i8
TOKEN_I16_TYPE    // i16
TOKEN_I32_TYPE    // i32
TOKEN_I64_TYPE    // i64
TOKEN_F32_TYPE    // f32
TOKEN_F64_TYPE    // f64
TOKEN_USIZE_TYPE  // usize
TOKEN_ISIZE_TYPE  // isize
```

Add to `tokenNames` and `keywords` maps.

**Step 2: Add type singletons**

In `types.go`:

```go
var (
    U8    = &BasicType{Name: "u8"}
    U16   = &BasicType{Name: "u16"}
    U32   = &BasicType{Name: "u32"}
    U64   = &BasicType{Name: "u64"}
    I8    = &BasicType{Name: "i8"}
    I16   = &BasicType{Name: "i16"}
    I32   = &BasicType{Name: "i32"}
    I64   = &BasicType{Name: "i64"}
    F32   = &BasicType{Name: "f32"}
    F64   = &BasicType{Name: "f64"}
    Usize = &BasicType{Name: "usize"}
    Isize = &BasicType{Name: "isize"}
)
```

Update `IsNumeric` and `IsComparable` to recognize all new types.
Update `Category` — all sized ints and floats are CopyType.
Make `int` equivalent to `i32` and `float` equivalent to `f32` in type equality.

**Step 3: Parse new types**

In `parser_types.go`, add cases for each new token type.
In `parser.go`, register each new type token with `p.parseTypeAsIdentifier`.

**Step 4: Map to C types in codegen**

In `cgen.go`, update `checkerTypeToCString`:

```go
case t.Equals(types.U8):   return "uint8_t"
case t.Equals(types.U16):  return "uint16_t"
case t.Equals(types.U32):  return "uint32_t"
case t.Equals(types.U64):  return "uint64_t"
case t.Equals(types.I8):   return "int8_t"
case t.Equals(types.I16):  return "int16_t"
case t.Equals(types.I32):  return "int32_t"
case t.Equals(types.I64):  return "int64_t"
case t.Equals(types.F32):  return "float"
case t.Equals(types.F64):  return "double"
case t.Equals(types.Usize): return "size_t"
case t.Equals(types.Isize): return "ptrdiff_t"
```

Ensure generated C includes `#include <stdint.h>`.

**Step 5: Add tests**

Write a parser test that parses `let x: u32 = 42;` and verifies the type.
Write a codegen test that generates `uint32_t x = 42;`.

**Step 6: Verify**

```bash
go build ./... && go test ./...
```

**Step 7: Commit**

```bash
git add -A && git commit -m "feat: add sized integer types (u8-u64, i8-i64, f32, f64, usize, isize)"
```

---

### Task 4: Add `volatile<T>` type for memory-mapped I/O

Add a `volatile` wrapper type that generates `volatile` qualifiers in C.

**Files:**
- Modify: `pkg/lexer/token.go` — add TOKEN_VOLATILE keyword
- Modify: `pkg/ast/types.go` — add VolatileType node
- Modify: `pkg/parser/parser_types.go` — parse `volatile<T>`
- Modify: `pkg/types/types.go` — add VolatileType
- Modify: `pkg/types/checker.go` — handle volatile
- Modify: `pkg/codegen/cgen.go` — emit `volatile` qualifier

**Step 1: Add volatile keyword token**

In `token.go`: add `TOKEN_VOLATILE` and register `"volatile"` in keywords map.

**Step 2: Add VolatileType AST node**

In `ast/types.go`:

```go
type VolatileType struct {
    Token lexer.Token
    Inner TypeExpr
}
func (vt *VolatileType) typeExprNode()        {}
func (vt *VolatileType) TokenLiteral() string { return vt.Token.Literal }
func (vt *VolatileType) Pos() (int, int)      { return vt.Token.Line, vt.Token.Column }
```

**Step 3: Add VolatileType to type system**

In `types/types.go`:

```go
type VolatileType struct {
    Inner Type
}
func (v *VolatileType) String() string { return "volatile<" + v.Inner.String() + ">" }
func (v *VolatileType) Equals(other Type) bool {
    if o, ok := other.(*VolatileType); ok {
        return v.Inner.Equals(o.Inner)
    }
    return false
}
```

**Step 4: Parse volatile<T>**

In `parser_types.go`, add:

```go
case lexer.TOKEN_VOLATILE:
    vt := &ast.VolatileType{Token: p.curToken}
    if !p.expectPeek(lexer.TOKEN_LT) {
        return nil
    }
    p.nextToken()
    vt.Inner = p.parseTypeExpr()
    if !p.expectPeek(lexer.TOKEN_GT) {
        return nil
    }
    return vt
```

**Step 5: Map to C**

In `checkerTypeToCString`:

```go
if vol, ok := t.(*types.VolatileType); ok {
    return "volatile " + checkerTypeToCString(vol.Inner)
}
```

**Step 6: Verify**

```bash
go build ./... && go test ./...
```

**Step 7: Commit**

```bash
git add -A && git commit -m "feat: add volatile<T> type for memory-mapped I/O"
```

---

### Task 5: Add `#[packed]` attribute for structs

Add a `packed` attribute to classes that generates `__attribute__((packed))` in C — essential for register maps.

**Files:**
- Modify: `pkg/ast/statements.go` — add Packed field to ClassStatement
- Modify: `pkg/parser/parser_decls.go` — parse `#[packed]` before class
- Modify: `pkg/codegen/cgen.go` — emit packed attribute

**Step 1: Add Packed field to ClassStatement**

In `ast/statements.go`, add `Packed bool` to the ClassStatement struct.

**Step 2: Parse #[packed] attribute**

In the parser, before `parseClassStatement`, detect `#[packed]`:
- When `#` token is encountered before `class`, parse the attribute tag
- This requires adding `TOKEN_HASH` to the lexer (if not present)
- Alternative: use a keyword `packed class` instead of annotation syntax

Simpler approach: add `packed` keyword and parse `packed class Point { ... }`.

In `token.go`: add `TOKEN_PACKED` keyword.
In `parser.go`: handle `TOKEN_PACKED` before class in `parseStatement()`.

**Step 3: Emit packed in codegen**

When generating a class/struct with Packed=true, append `__attribute__((packed))` to the struct definition.

**Step 4: Verify**

```bash
go build ./... && go test ./...
```

**Step 5: Commit**

```bash
git add -A && git commit -m "feat: add packed class support for register maps"
```

---

### Task 6: Add `static` variable declarations

Allow `static` keyword on `let`/`const` to generate C `static` variables — placed in BSS/data sections.

**Files:**
- Modify: `pkg/ast/statements.go` — add Static field to LetStatement
- Modify: `pkg/parser/parser.go` — parse `static let ...` / `static const ...`
- Modify: `pkg/codegen/cgen.go` — emit `static` qualifier

**Step 1: Add Static field**

`LetStatement` and `ConstStatement` already exist. Add `Static bool` to both.

**Step 2: Parse static declarations**

In `parseStatement()`, handle `TOKEN_STATIC`:
```go
case lexer.TOKEN_STATIC:
    p.nextToken()
    switch p.curToken.Type {
    case lexer.TOKEN_LET, lexer.TOKEN_MUT:
        stmt := p.parseLetStatement()
        if stmt != nil { stmt.Static = true }
        return stmt
    case lexer.TOKEN_CONST:
        stmt := p.parseConstStatement()
        if stmt != nil { stmt.Static = true }
        return stmt
    }
```

**Step 3: Emit static in codegen**

Prefix the C declaration with `static` when the flag is set.

**Step 4: Verify and commit**

```bash
go build ./... && go test ./...
git add -A && git commit -m "feat: add static variable declarations"
```

---

### Task 7: Update build command for ARM cross-compilation

Add `--target arm` flag to `carv build` that uses `arm-none-eabi-gcc` instead of `gcc`.

**Files:**
- Modify: `cmd/carv/main.go` — add target flag parsing
- Modify: `pkg/codegen/cgen.go` — add embedded C preamble (no stdlib, stdint only)

**Step 1: Add target flag**

Parse `carv build --target arm <file>` and pass target to buildFile.

**Step 2: Select compiler based on target**

```go
if target == "arm" {
    compiler = "arm-none-eabi-gcc"
    flags = []string{"-mcpu=cortex-m4", "-mthumb", "-Os", "-ffreestanding", "-nostdlib"}
}
```

**Step 3: Adjust codegen preamble for embedded**

When targeting embedded, emit `#include <stdint.h>` instead of full libc headers. Skip string/print builtins that depend on libc.

**Step 4: Verify and commit**

```bash
go build ./... && go test ./...
git add -A && git commit -m "feat: add ARM cross-compilation target"
```

---

### Task 8: Update README and docs

Update README to reflect the new embedded focus, remove interpreter references, update syntax examples.

**Step 1: Update README.md**

- Change tagline to "A memory-safe language for embedded systems that compiles to C"
- Remove `run` and `repl` from commands
- Add sized types examples
- Remove pipe operator examples, show method chaining
- Add embedded hardware targeting section

**Step 2: Commit**

```bash
git add README.md && git commit -m "docs: update README for embedded AOT focus"
```
