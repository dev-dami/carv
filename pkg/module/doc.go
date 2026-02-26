// Package module resolves and loads Carv modules.
//
// Design decisions:
//   - Resolution supports project-local imports, relative imports, and built-in modules.
//   - carv.toml loading is separated from loader state so CLI and tooling can share config logic.
//
// Usage pattern:
//
//	loader := module.NewLoader(basePath)
//	mod, err := loader.Load(importPath, fromFile)
//	_ = mod
//	_ = err
package module
