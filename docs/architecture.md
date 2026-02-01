[← Language Guide](language.md) | **Architecture** | [Built-ins →](builtins.md)

---

# Carv Architecture

How the compiler is structured. Mostly notes for myself but might be useful if you're poking around.

## Pipeline

```
Source Code (.carv)
       │
       ▼
    Lexer (pkg/lexer)
       │
       ▼
    Tokens
       │
       ▼
    Parser (pkg/parser)
       │
       ▼
    AST (pkg/ast)
       │
       ▼
    Type Checker (pkg/types)
       │
       ├──────────────────┐
       ▼                  ▼
  Interpreter         Code Generator
  (pkg/eval)          (pkg/codegen)
       │                  │
       │                  │
  Module Loader           │
  (pkg/module)            │
       │                  │
       ▼                  ▼
    Output            C Source
                          │
                          ▼
                     GCC/Clang
                          │
                          ▼
                      Binary
```

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

Type checker. Walks the AST and validates types, builds symbol tables.

Currently pretty basic - doesn't do full type inference, mostly just checks that operations are valid.

### `pkg/eval`

Tree-walking interpreter. Useful for quick iteration and testing without going through the C compilation step.

Key files:
- `eval.go` - main evaluation logic
- `object.go` - runtime value types
- `builtins.go` - built-in functions
- `environment.go` - variable scoping

### `pkg/codegen`

Generates C code from the AST. The generated C is not pretty but it works.

Currently targets C99. The runtime includes an arena allocator and helper macros for strings, arrays, maps, and Result types.

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
3. Package manager (for external dependencies)
4. Better standard library
5. Then rewrite lexer, parser, codegen in Carv

It's a long road but that's half the fun. Getting closer though!

---

[← Language Guide](language.md) | **Architecture** | [Built-ins →](builtins.md)
