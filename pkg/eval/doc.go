// Package eval provides the tree-walking interpreter for Carv.
//
// Design decisions:
//   - Runtime values are represented by Object implementations (object.go).
//   - Builtins are centralized and module aliases (for example net/web) map to the same runtime
//     primitives where appropriate.
//
// Usage pattern:
//
//	env := eval.NewEnvironment()
//	obj := eval.Eval(program, env)
//	_ = obj // inspect result or error object
package eval
