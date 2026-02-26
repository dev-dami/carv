package types

import "github.com/dev-dami/carv/pkg/ast"

func (c *Checker) checkAwaitExpression(e *ast.AwaitExpression) Type {
	if !c.inAsyncFn {
		line, col := e.Pos()
		c.error(line, col, "await can only be used inside async functions")
		return Any
	}

	for name, info := range c.borrows {
		if info.ImmutableCount > 0 || info.MutableActive {
			line, col := e.Pos()
			c.error(line, col, "borrow of '%s' cannot be held across await point", name)
		}
	}

	innerType := c.checkExpression(e.Value)
	if ft, ok := innerType.(*FutureType); ok {
		return ft.Inner
	}

	line, col := e.Pos()
	c.error(line, col, "await requires Future type, got %s", innerType.String())
	return Any
}
