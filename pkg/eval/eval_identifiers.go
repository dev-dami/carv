package eval

import "github.com/dev-dami/carv/pkg/ast"

func evalIdentifier(node *ast.Identifier, env *Environment) Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}

	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}

	return &Error{Message: "identifier not found: " + node.Value, Line: node.Token.Line, Column: node.Token.Column}
}
