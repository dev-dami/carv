[← Language Guide](language.md) | **Architecture** | [Built-ins →](builtins.md)

---

# Carv Architecture

How the compiler is structured. Mostly notes for myself but might be useful if you're poking around.

## Pipeline

Source → Lexer → Tokens → Parser → AST → Type Checker → Interpreter or C Codegen → GCC/Clang → Binary

The type checker produces a `CheckResult` with type info, ownership tracking, and warnings. Both the interpreter and codegen consume this result.

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

The parser is probably the messiest part of the codebase. It works but could use some cleanup.

### `pkg/types`

Type checker. Walks the AST and validates types, builds symbol tables, tracks ownership.

Produces a `CheckResult` with:
- `NodeTypes`: type of every expression
- `FuncSigs`: function signatures
- `ClassInfo`: class field/method info
- `Errors`: type errors (fatal in codegen)
- `Warnings`: ownership/borrow violations (warnings in interpreter, fatal in codegen)

Implements ownership tracking (move/drop), borrow checking (&T / &mut T), and a warnings system for non-fatal violations.

### `pkg/eval`

Tree-walking interpreter. Useful for quick iteration and testing without going through the C compilation step.

Key files:
- `eval.go` - main evaluation logic
- `object.go` - runtime value types
- `builtins.go` - built-in functions
- `environment.go` - variable scoping

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

#### Interface Codegen

Interfaces compile to a vtable + fat pointer pattern:

1. **Vtable struct**: one function pointer per interface method, all taking `const void* self` as first param
2. **Fat pointer**: `{ const void* data; const Vtable* vt; }` — `_ref` (immutable) and `_mut_ref` (mutable) variants
3. **Impl wrappers**: static functions that cast `const void*` back to the concrete type and call the real method
4. **Vtable instances**: one `static const` vtable per impl, initialized with wrapper function pointers
5. **Cast expressions**: `&obj as &Interface` produces a fat pointer literal `{ .data = obj, .vt = &VT }`
6. **Dynamic dispatch**: `obj.method(args)` on an interface ref becomes `obj.vt->method(obj.data, args)`

Generation order: interface typedefs → impl forward decls → impl bodies → wrappers + vtable instances (all before `main()`)

### `pkg/module`

Module system for loading and resolving dependencies.

Key files:
- `loader.go` - module resolution and loading
- `config.go` - `carv.toml` parsing

Supports:
- Relative imports (`./utils`, `../lib/math`)
- Project-local imports (from `src/` directory)
- Future: external packages (from `carv_modules/`)

### `cmd/carv`

CLI entry point. Handles `run`, `build`, `emit-c`, `repl`, and `init` commands.

## Design Decisions

**Why compile to C?**

Portability mostly. C compilers exist everywhere, and I get optimization for free. Plus it's interesting to see how high-level constructs map to C.

**Why a tree-walking interpreter too?**

Much faster feedback loop during development. Compiling to C means invoking GCC which is slow for quick tests.

**Why semicolons?**

Easier to parse. Maybe I'll add automatic semicolon insertion later, but for now explicit semis keep the parser simple.

## Future Plans

The goal is self-hosting - writing the Carv compiler in Carv. That means I need:

1. ~~Module/import system~~ ✓ Done!
2. ~~String interpolation~~ ✓ Done!
3. ~~Ownership system (move + drop)~~ ✓ Done!
4. ~~Borrowing (&T / &mut T)~~ ✓ Done!
5. ~~Interfaces (interface/impl)~~ ✓ Done!
6. ~~Async/await~~ ✓ Done!
7. Package manager (for external dependencies)
8. Better standard library
9. Then rewrite lexer, parser, codegen in Carv

It's a long road but that's half the fun. Getting closer though!

---

[← Language Guide](language.md) | **Architecture** | [Built-ins →](builtins.md)
