[← Built-ins](docs/builtins.md) | **Contributing** | [Back to README →](README.md)

---

# Contributing to Carv

Hey, thanks for the interest! This is a hobby project so things are pretty informal around here.

## The Basics

1. **Fork & clone** the repo
2. **Make your changes** on a branch
3. **Run tests** with `make test`
4. **Open a PR** with a clear description

That's pretty much it.

## What I'm Looking For

### Bugs
Found something broken? Open an issue or PR. Even better if you include a minimal `.carv` file that reproduces it.

### Ideas
Have a feature idea? Open an issue first so we can discuss. I might have opinions (or it might conflict with future plans).

### Documentation
Docs can always be better. Typo fixes, clarifications, examples - all welcome.

### Code
If you want to contribute code:
- Keep changes focused (one feature/fix per PR)
- Follow existing code style (just look at nearby code)
- Add tests if you're adding features

## What I'm Not Looking For

- Major architectural changes without discussion
- Features that complicate the language significantly
- Aggressive optimization (clarity > speed for now)

## Code Style

No strict rules, just:
- Go code should pass `go fmt` and `go vet`
- Keep functions reasonably sized
- Comments for non-obvious things

## Running Tests

```bash
make test        # run all tests
make build       # build the binary
make examples    # run example programs
```

## Project Structure

```
cmd/carv/       # CLI entry point
pkg/
  lexer/        # tokenizer
  parser/       # parser
  ast/          # AST definitions
  types/        # type checker (ownership tracking, borrow checking)
  eval/         # interpreter
  codegen/      # C code generator (ownership-aware, borrow support)
  module/       # module loader & carv.toml config
examples/       # example programs
docs/           # documentation
```

**Ownership & Borrowing**: The type checker (`pkg/types`) tracks ownership (move/drop) and enforces borrow rules. The codegen (`pkg/codegen`) emits ownership-aware C code with proper drop calls and borrow support (&T / &mut T).

## Response Time

I work on this when I have time and energy, so response times vary. Don't take it personally if I'm slow - I'll get to it eventually.

## License

By contributing, you agree that your contributions will be licensed under MIT.

---

Thanks for helping make Carv better!

---

[← Built-ins](docs/builtins.md) | **Contributing** | [Back to README →](README.md)
