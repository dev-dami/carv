// Package ast defines Carv's syntax tree nodes.
//
// Design decisions:
//   - Every node carries token/position metadata so diagnostics can point to source locations.
//   - Expressions, statements, and type expressions are modeled as separate interfaces to keep
//     parsing, checking, and codegen passes explicit.
//
// Usage pattern:
// The parser builds an *ast.Program, and later passes (types, eval, codegen) walk the same tree.
// Most integrations only need to switch on concrete node types.
package ast
