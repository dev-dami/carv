package types

import "github.com/dev-dami/carv/pkg/ast"

type BorrowInfo struct {
	ImmutableCount int
	MutableActive  bool
}

func (c *Checker) pushBorrows() map[string]*BorrowInfo {
	prev := c.borrows
	next := make(map[string]*BorrowInfo, len(prev))
	for name, info := range prev {
		cp := *info
		next[name] = &cp
	}
	c.borrows = next
	return prev
}

func (c *Checker) popBorrows(prev map[string]*BorrowInfo) {
	c.borrows = prev
}

func (c *Checker) hasActiveBorrow(name string) bool {
	if info, exists := c.borrows[name]; exists {
		return info.ImmutableCount > 0 || info.MutableActive
	}
	return false
}

func (c *Checker) checkBorrowExpression(e *ast.BorrowExpression) Type {
	innerType := c.checkExpression(e.Value)

	varName := ""
	if ident, ok := e.Value.(*ast.Identifier); ok {
		varName = ident.Value
	}

	if varName != "" {
		if own, exists := c.ownership[varName]; exists && own.State == Moved {
			line, col := e.Pos()
			c.warning(line, col, "cannot borrow moved value '%s' (moved at line %d)", varName, own.MovedAt)
			return &RefType{Inner: innerType, Mutable: e.Mutable}
		}

		info := c.borrows[varName]
		if info == nil {
			info = &BorrowInfo{}
			c.borrows[varName] = info
		}

		if e.Mutable {
			if info.ImmutableCount > 0 {
				line, col := e.Pos()
				c.warning(line, col, "cannot mutably borrow '%s': already immutably borrowed", varName)
			}
			if info.MutableActive {
				line, col := e.Pos()
				c.warning(line, col, "cannot mutably borrow '%s': already mutably borrowed", varName)
			}
			info.MutableActive = true
		} else {
			if info.MutableActive {
				line, col := e.Pos()
				c.warning(line, col, "cannot immutably borrow '%s': already mutably borrowed", varName)
			}
			info.ImmutableCount++
		}
	}

	return &RefType{Inner: innerType, Mutable: e.Mutable}
}

func (c *Checker) checkDerefExpression(e *ast.DerefExpression) Type {
	innerType := c.checkExpression(e.Value)
	if ref, ok := innerType.(*RefType); ok {
		return ref.Inner
	}
	line, col := e.Pos()
	c.warning(line, col, "dereference of non-reference type %s", innerType.String())
	return innerType
}
