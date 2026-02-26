package eval

import "github.com/dev-dami/carv/pkg/ast"

func evalIntegerLiteral(node *ast.IntegerLiteral) Object {
	return &Integer{Value: node.Value}
}

func evalFloatLiteral(node *ast.FloatLiteral) Object {
	return &Float{Value: node.Value}
}

func evalStringLiteral(node *ast.StringLiteral) Object {
	return &String{Value: node.Value}
}

func evalCharLiteral(node *ast.CharLiteral) Object {
	return &Char{Value: node.Value}
}

func evalBoolLiteral(node *ast.BoolLiteral) Object {
	return nativeBoolToBooleanObject(node.Value)
}

func evalNilLiteral(_ *ast.NilLiteral) Object {
	return NIL
}
