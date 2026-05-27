[← Language Guide](language.md) | [API Reference](api.md) | **Architecture** | [Built-ins →](builtins.md)

---

# Carv Architecture

How the compiler is structured. Mostly notes for myself but might be useful if you're poking around.

## Pipeline

```
Source → Lexer → Tokens → Parser → AST → Type Checker → IR → (VM | C Codegen → GCC/Clang → Binary)
```

- **`carv run`**: type checker → IR → VM (fast iteration, no C compiler needed)
- **`carv build`**: type checker → IR → C Codegen → GCC/Clang → native binary

The type checker produces a `CheckResult` with type info, ownership tracking, and warnings. IR lowering converts the typed AST into a flat instruction list consumed by either the VM or C codegen.

## Package Overview

### `pkg/lexer`

Breaks source code into tokens. Pretty standard lexer - handles keywords, operators, literals, etc.

Key files:
- `lexer.go` - the actual lexer
- `token.go` - token types and keyword lookup

### `pkg/ast`

Abstract Syntax Tree definitions. Every syntactic construct has a corresponding AST node.

Key files:
- `ast.go` - expressions (literals, operators, calls, etc.)
- `statements.go` - statements (let, return, if, for, etc.)
- `types.go` - type expressions

### `pkg/parser`

Pratt parser (operator precedence parsing) for expressions, recursive descent for statements.

Key files:
- `parser.go` - parser setup + statement parsing
- `pratt.go` - expression parsing and precedence logic
- `parser_decls.go` - class/interface/impl parsing
- `parser_types.go` - type expression parsing

### `pkg/types`

Type checker. Walks the AST and validates types, builds symbol tables, tracks ownership.

Produces a `CheckResult` with:
- `NodeTypes`: type of every expression
- `FuncSigs`: function signatures
- `ClassInfo`: class field/method info
- `Errors`: type errors (fatal in codegen)
- `Warnings`: ownership/borrow violations (treated as fatal in codegen)

Implements ownership tracking (move/drop), borrow checking (&T / &mut T), and a warnings system for non-fatal violations.

Key files:
- `checker.go` - checker core + diagnostics
- `ownership.go` - move/ownership rules
- `borrow.go` - borrow state and borrow checks
- `interface.go` - interface + impl validation
- `async.go` - async/await validation

### `pkg/codegen`

Generates C code from the AST. The generated C is not pretty but it works.

Currently targets C99. Key features:
- **Scope stack**: tracks variable lifetimes for drop insertion
- **Preamble buffer**: emits runtime helpers (carv_string, carv_array, etc.)
- **carv_string struct**: `{char* data; size_t len; bool owned;}`
- **Single-exit functions**: all returns become `goto __carv_exit` with drops at exit label
- **Ownership-aware code generation**: emits `carv_string_move()`, `carv_string_drop()`, `carv_string_clone()`
- **Borrow support**: `&T` → `const T*`, `&mut T` → `T*`
- **Interface dispatch**: vtable-based dynamic dispatch via fat pointers
- **Arena allocator**: used for all owned heap values
- **Async/await lowering**: `async fn` to frame structs + poll state machines
- **Async runtime bootstrap**: generated `main()` drives `async fn carv_main()` via event loop

#### Interface Codegen

Interfaces compile to a vtable + fat pointer pattern:

1. **Vtable struct**: one function pointer per interface method, all taking `const void* self` as first param
2. **Fat pointer**: `{ const void* data; const Vtable* vt; }` — `_ref` (immutable) and `_mut_ref` (mutable) variants
3. **Impl wrappers**: static functions that cast `const void*` back to the concrete type and call the real method
4. **Vtable instances**: one `static const` vtable per impl, initialized with wrapper function pointers
5. **Cast expressions**: `&obj as &Interface` produces a fat pointer literal `{ .data = obj, .vt = &VT }`
6. **Dynamic dispatch**: `obj.method(args)` on an interface ref becomes `obj.vt->method(obj.data, args)`

Generation order: interface typedefs → impl forward decls → impl bodies → wrappers + vtable instances (all before `main()`)

### `pkg/ir`

Intermediate Representation — a flat instruction list that decouples the type checker from both the VM and C codegen.

Key files:
- `ir.go` — IR instruction types (73 opcodes: arithmetic, control flow, memory, objects, etc.)
- `lower.go` — lower typed AST → IR instruction sequence
- `ssa.go` — SSA construction and def-use chain analysis
- `optimize.go` — peephole and constant-folding optimizations
- `scheduler.go` — instruction scheduling for execution order

The IR uses a register-based format with `ValueRef` (SSA value references) and `BlockRef` (basic block references). All control flow is explicit via branch/phi instructions.

### `pkg/vm`

Bytecode-style virtual machine that executes IR directly. No separate bytecode encoding — the IR instruction list IS the bytecode.

Key files:
- `vm.go` — main interpreter loop (all 73 opcodes)
- `value.go` — runtime value representation (int, float, string, array, map, object, etc.)
- `thread.go` — goroutine-friendly thread-safe execution
- `debug.go` — debug/trace helpers for inspecting VM state

The VM supports: all arithmetic/logic ops, heap allocation (arrays, maps, strings), function calls (native + Carv-defined), virtual dispatch via interface calls, closures (function values), class method dispatch, and the 10 built-in functions (`print`, `println`, `len`, `contains`, `keys`, `int`, `float`, `string`, `readline`, `assert`).

### `pkg/module`

Module system for loading and resolving dependencies.

Key files:
- `loader.go` - module resolution and loading
- `config.go` - `carv.toml` parsing
- `lock.go` - `carv.lock` lock file handling
- `builtin_modules.go` - built-in module exports (fs, net, gpio, uart, spi, i2c, timer)

Supports:
- Relative imports (`./utils`, `../lib/math`)
- Project-local imports (from `src/` directory)
- External packages (from `carv_modules/`)
- Built-in standard modules (`fs`, `net`, `gpio`, `uart`, `spi`, `i2c`, `timer`)

### `pkg/semver`

Semantic version parsing and constraint matching for the package manager.

Supports:
- Version parsing: `1.0.0`, `v1.2.3-beta.1`, `0.1.0+build.42`
- Constraint operators: `=`, `!=`, `>`, `<`, `>=`, `<=`, `^`, `~`, `*`
- Caret ranges: `^1.2.3` := `>=1.2.3, <2.0.0`
- Tilde ranges: `~1.2.3` := `>=1.2.3, <1.3.0`
- Comma/space-separated compound constraints: `>=1.0.0, <2.0.0`

### `pkg/resolver`

Transitive dependency resolution with cycle detection and GitHub tag resolution.

Key features:
- Resolves dependencies from `carv.toml`, including transitive deps
- Cycle detection via visiting set
- Semver constraint matching against git tags (`git ls-remote --tags`)
- Lock file generation from resolved tree
- Path-based and git-based dependency sources

### `pkg/lsp`

Language Server Protocol implementation using `opa-oz/glsp` (protocol 3.16, stdio transport).

Provides:
- **Diagnostics** — type errors and warnings on file open/change
- **Hover** — type information for symbols
- **Go-to-definition** — jump to symbol definition using tracked positions
- **Completion** — keyword and symbol suggestions

### `cmd-carv`

CLI entry point. Handles `run`, `build`, `emit-c`, `repl`, `lsp`, `init`, `add`, `remove`, `install`, and `pkg` commands.

- `carv run <file>` — parse, type-check, lower to IR, execute in the built-in VM
- `carv build <file>` — parse, type-check, lower to IR, generate C code, compile with GCC/Clang

Subcommands:
- `carv pkg list` — list installed dependencies with lock status
- `carv pkg info <name>` — show dependency details including transitive deps
- `carv pkg update [name]` — update dependencies to latest matching versions
- `carv pkg publish` — create git tag for GitHub release registry

## Design Decisions

**Why compile to C?**

Portability mostly. C compilers exist everywhere, and I get optimization for free. Plus it's interesting to see how high-level constructs map to C.

**Why a VM?**

Fast iteration. `carv run` skips the C compiler entirely — parse, type-check, and execute in milliseconds. The VM is register-based and executes the IR directly (no separate bytecode). It's not as fast as compiled C code, but for development it's much more convenient.

**Why semicolons?**

Easier to parse. Maybe I'll add automatic semicolon insertion later, but for now explicit semis keep the parser simple.

## Future Plans

The goal is self-hosting - writing the Carv compiler in Carv itself. That means I need:

1. ~~Module/import system~~ ✓ Done!
2. ~~String interpolation~~ ✓ Done!
3. ~~Ownership system (move + drop)~~ ✓ Done!
4. ~~Borrowing (&T / &mut T)~~ ✓ Done!
5. ~~Interfaces (interface/impl)~~ ✓ Done!
6. ~~Async/await~~ ✓ Done!
7. ~~IR + Virtual Machine~~ ✓ Done!
8. Package manager (for external dependencies)
9. Better standard library
10. Then rewrite lexer, parser, codegen in Carv

It's a long road but that's half the fun. Getting closer though!

---

[← Language Guide](language.md) | [API Reference](api.md) | **Architecture** | [Built-ins →](builtins.md)
