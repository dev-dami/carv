<p align="center">
  <img src="assets/logo.svg" alt="Carv Logo" width="400">
</p>

<p align="center">
  <strong>A memory-safe language for embedded systems that compiles to C</strong>
</p>

<p align="center">
  <a href="#what-it-does">Features</a> •
  <a href="#quick-look">Quick Look</a> •
  <a href="#building">Building</a> •
  <a href="#where-things-stand">Status</a> •
  <a href="#docs">Docs</a>
</p>

---

# Carv

A programming language designed for embedded systems. Compiles to C, runs natively on bare metal.

---

## The Story

This started as a hobby project called **dyms** back in September. After a few rewrites between Go and Rust, it became **Carv** — a language that compiles to C with the goal of being safe, small, and suitable for microcontrollers.

The focus is now squarely on **embedded systems**: sized integer types, volatile memory access, packed structs for register maps, and cross-compilation to ARM targets.

---

## What It Does

Carv compiles to C and targets embedded hardware. No interpreter, no runtime — just clean C output that works with `gcc` or `arm-none-eabi-gcc`.

Features that actually work:
- **Sized integer types** (`u8`, `u16`, `u32`, `u64`, `i8`, `i16`, `i32`, `i64`, `f32`, `f64`, `usize`, `isize`)
- **`volatile<T>`** for memory-mapped I/O
- **`packed` classes** for register maps (`__attribute__((packed))`)
- **`static` variables** for BSS/data section placement
- **ARM cross-compilation** (`carv build --target arm`)
- Static typing with inference
- Method chaining with `.`
- `let` / `mut` / `const` with proper immutability enforcement
- Compound assignment (`+=`, `-=`, `*=`, `/=`, `%=`, `&=`, `|=`, `^=`)
- Classes with methods
- Result types (`Ok`/`Err`) with pattern matching
- `for-in` loops over arrays, strings, and maps
- Closures with environment capture
- Module system with `require`
- String interpolation with `f"hello {name}"`
- Ownership system (move semantics)
- Borrowing (`&T` / `&mut T`)
- Interfaces (`interface` / `impl` with vtable-based dispatch)
- Async/await (compiles to state machines)
- 40+ built-in functions

---

## Quick Look

```carv
// sized types for embedded
let counter: u32 = 0;
let flags: u8 = 0xFF;
let temperature: f32 = 23.5;

// volatile for hardware registers (memory-mapped I/O)
let status_reg: volatile<u32>;

// packed struct for register maps
packed class GPIO_Regs {
    moder:   u32 = 0
    otyper:  u32 = 0
    ospeedr: u32 = 0
    pupdr:   u32 = 0
    idr:     u32 = 0
    odr:     u32 = 0
}

// static variable declarations
static let boot_count: int = 0;

// method chaining
let result = sensor.read().calibrate().to_celsius();

// ownership: move semantics
let s = "hello";
let t = s;              // s is moved, now invalid

// error handling without exceptions
fn divide(a: i32, b: i32) {
    if b == 0 {
        return Err("division by zero");
    }
    return Ok(a / b);
}

let x = divide(10, 2)?;

// classes
class Point {
    x: i32 = 0
    y: i32 = 0
}

// closures
let multiplier = 3;
let triple = fn(x: i32) -> i32 {
    return x * multiplier;
};
```

### Modules

```carv
// math.carv
pub fn add(a: i32, b: i32) -> i32 {
    return a + b;
}

// main.carv
require { add } from "./math";
println(f"1 + 2 = {add(1, 2)}");
```

---

## Building

```bash
git clone https://github.com/dev-dami/carv
cd carv
make build
```

Then:
```bash
./build/carv build file.carv               # compile to binary (host)
./build/carv build --target arm file.carv   # compile for ARM Cortex-M
./build/carv emit-c file.carv              # emit generated C source
./build/carv repl                          # start interactive REPL
./build/carv lsp                           # start language server
./build/carv init                          # create new project with carv.toml
./build/carv add mylib --git <url>         # add a dependency
./build/carv install                       # install all dependencies
./build/carv pkg list                      # list installed packages
./build/carv pkg update                    # update dependencies
./build/carv pkg publish                   # publish package to GitHub
```

For ARM targets, you need `arm-none-eabi-gcc` installed.

## Quick Install

No need to clone or build — grab a pre-built binary directly:

```bash
# Install latest release
curl -fsSL https://raw.githubusercontent.com/dev-dami/carv/main/scripts/install.sh | bash

# Install a specific version
curl -fsSL https://raw.githubusercontent.com/dev-dami/carv/main/scripts/install.sh | bash -s -- v0.5.1-beta
```

The script auto-detects your OS/architecture, downloads the matching binary from GitHub releases, and installs it to `/usr/local/bin` (or `~/.local/bin`).

Supported platforms: **Linux** (amd64, arm64), **macOS** (amd64, arm64), **Windows** (amd64).

---

## Where Things Stand

### Core Language
- [x] Lexer, parser, type checker
- [x] C code generation (AOT only)
- [x] Static typing with inference

### Embedded Features
- [x] Sized integer types (`u8`-`u64`, `i8`-`i64`, `f32`, `f64`, `usize`, `isize`)
- [x] `volatile<T>` for memory-mapped I/O
- [x] `packed` classes for register maps
- [x] `static` variable declarations
- [x] ARM cross-compilation (`--target arm`)

### Data Types & Structures
- [x] Primitives (int, float, string, bool, char + all sized variants)
- [x] Arrays and hash maps
- [x] Result types (`Ok`/`Err`) with pattern matching
- [x] Classes with methods

### Memory & Ownership
- [x] Ownership system (move semantics)
- [x] Borrowing (`&T` / `&mut T`)
- [x] Arena allocator in codegen
- [x] Automatic drop insertion

### Functional Features
- [x] First-class functions
- [x] Closures with capture
- [x] Method chaining (`.`)
- [x] Higher-order functions

### Advanced Features
- [x] Interfaces (`interface`/`impl` with vtables)
- [x] Module system (`require`)
- [x] String interpolation (`f"..."`)
- [x] Async/await (state-machine codegen)

### Tooling
- [x] Project config (`carv.toml`)
- [x] Build scripts
- [x] REPL with history, tab completion, `:type`, `:load`, `:clear`
- [x] LSP server (diagnostics, hover, go-to-definition, completion)
- [x] VS Code extension (syntax highlighting + LSP client)
- [x] Package manager (`carv pkg`, semver resolution, lock files, GitHub registry)
- [ ] HAL modules (GPIO, UART, SPI, I2C, Timers)
- [ ] Self-hosting

---

## REPL

```bash
$ carv repl
carv 0.5.1-beta - Type :help for commands
>>> let x = 42;
>>> :type x
i32
>>> :load script.carv
>>> :clear
```

REPL features: command history, tab completion, multi-line input, `:help`, `:type`, `:clear`, `:load`.

## Package Manager

Carv has a built-in package manager with semver resolution and lock files:

```bash
carv add mylib --git https://github.com/user/mylib --version "^1.0.0"
carv install
carv pkg list
carv pkg update
carv pkg publish
```

Dependencies are resolved transitively, with cycle detection. Lock files (`carv.lock`) pin exact versions for reproducible builds. Packages are published via git tags on GitHub.

## LSP & Editor Support

Start the language server with `carv lsp` (stdio transport). Provides:
- **Diagnostics** — type errors, ownership violations, warnings
- **Hover** — type info on hover
- **Go-to-definition** — jump to symbol definition
- **Completion** — keyword and symbol suggestions

A VS Code extension is included in `editors/vscode/` with syntax highlighting and LSP client integration.

---

## Docs

- [Language Guide](docs/language.md)
- [API Reference](docs/api.md)
- [Architecture](docs/architecture.md)
- [Built-ins](docs/builtins.md)
- [Examples](docs/examples.md)
- [Contributing](CONTRIBUTING.md)

---

## License

MIT

---

*Built for reading sensors, and writing to registers - without the footguns of raw C.*
