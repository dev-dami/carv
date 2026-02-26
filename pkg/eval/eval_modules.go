package eval

import (
	"strings"

	"github.com/dev-dami/carv/pkg/ast"
	"github.com/dev-dami/carv/pkg/module"
)

var moduleLoader *module.Loader
var currentFile string

func SetModuleLoader(loader *module.Loader) {
	moduleLoader = loader
}

func SetCurrentFile(file string) {
	currentFile = file
}

func evalRequireStatement(node *ast.RequireStatement, env *Environment) Object {
	if module.IsBuiltinModule(node.Path.Value) {
		return evalBuiltinModuleRequire(node, env)
	}

	if moduleLoader == nil {
		return &Error{Message: "module loader not initialized", Line: node.Token.Line, Column: node.Token.Column}
	}

	mod, err := moduleLoader.Load(node.Path.Value, currentFile)
	if err != nil {
		return &Error{Message: "failed to load module: " + err.Error(), Line: node.Token.Line, Column: node.Token.Column}
	}

	modEnv := NewEnvironment()
	result := Eval(mod.Program, modEnv)
	if isError(result) {
		return result
	}

	if node.Alias != nil {
		modObj := &Module{
			Name:    node.Path.Value,
			Exports: make(map[string]Object),
		}
		for name := range mod.Exports {
			if val, ok := modEnv.Get(name); ok {
				modObj.Exports[name] = val
			}
		}
		env.Set(node.Alias.Value, modObj)
		return NIL
	}

	if len(node.Names) > 0 {
		for _, name := range node.Names {
			if !mod.Exports[name.Value] {
				return &Error{Message: "undefined export: " + name.Value, Line: name.Token.Line, Column: name.Token.Column}
			}
			if val, ok := modEnv.Get(name.Value); ok {
				env.Set(name.Value, val)
			} else {
				return &Error{Message: "undefined export: " + name.Value, Line: name.Token.Line, Column: name.Token.Column}
			}
		}
		return NIL
	}

	if node.All {
		for name := range mod.Exports {
			if val, ok := modEnv.Get(name); ok {
				env.Set(name, val)
			}
		}
		return NIL
	}

	modObj := &Module{
		Name:    node.Path.Value,
		Exports: make(map[string]Object),
	}
	for name := range mod.Exports {
		if val, ok := modEnv.Get(name); ok {
			modObj.Exports[name] = val
		}
	}
	env.Set(getModuleName(node.Path.Value), modObj)
	return NIL
}

func evalBuiltinModuleRequire(node *ast.RequireStatement, env *Environment) Object {
	exports, ok := module.BuiltinModuleExports(node.Path.Value)
	if !ok {
		return &Error{Message: "unknown builtin module: " + node.Path.Value, Line: node.Token.Line, Column: node.Token.Column}
	}

	resolve := func(name string) Object {
		if !exports[name] {
			return &Error{Message: "undefined export: " + name, Line: node.Token.Line, Column: node.Token.Column}
		}
		b, exists := builtins[name]
		if !exists {
			return &Error{Message: "builtin export not implemented: " + name, Line: node.Token.Line, Column: node.Token.Column}
		}
		return b
	}

	if node.Alias != nil {
		modObj := &Module{
			Name:    node.Path.Value,
			Exports: make(map[string]Object, len(exports)),
		}
		for name := range exports {
			v := resolve(name)
			if isError(v) {
				return v
			}
			modObj.Exports[name] = v
		}
		env.Set(node.Alias.Value, modObj)
		return NIL
	}

	if len(node.Names) > 0 {
		for _, name := range node.Names {
			v := resolve(name.Value)
			if isError(v) {
				return v
			}
			env.Set(name.Value, v)
		}
		return NIL
	}

	if node.All {
		for name := range exports {
			v := resolve(name)
			if isError(v) {
				return v
			}
			env.Set(name, v)
		}
		return NIL
	}

	modObj := &Module{
		Name:    node.Path.Value,
		Exports: make(map[string]Object, len(exports)),
	}
	for name := range exports {
		v := resolve(name)
		if isError(v) {
			return v
		}
		modObj.Exports[name] = v
	}
	env.Set(getModuleName(node.Path.Value), modObj)
	return NIL
}

func getModuleName(path string) string {
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".carv")
	return name
}
