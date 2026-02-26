package eval

import (
	"strings"

	"github.com/dev-dami/carv/pkg/ast"
)

func evalInterpolatedString(node *ast.InterpolatedString, env *Environment) Object {
	var builder strings.Builder

	for _, part := range node.Parts {
		evaluated := Eval(part, env)
		if isError(evaluated) {
			return evaluated
		}
		builder.WriteString(evaluated.Inspect())
	}

	return &String{Value: builder.String()}
}
