// Package codegen lowers typed Carv AST into C.
//
// Design decisions:
//   - Generation is ownership-aware (moves, drops, clones) to preserve language semantics.
//   - Async functions are lowered to frame structs and poll-style state machines.
//   - Interface dispatch is lowered to vtables and fat pointers.
//
// Usage pattern:
// Construct a CGenerator, call Generate on an AST program, then compile emitted C with a C99
// compiler.
package codegen
