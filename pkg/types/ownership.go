package types

import "github.com/dev-dami/carv/pkg/ast"

// OwnershipState tracks whether a variable's value has been moved.
type OwnershipState int

const (
	Owned OwnershipState = iota // variable holds its value
	Moved                       // value has been moved to another binding
)

// VarOwnership tracks ownership metadata for a variable.
type VarOwnership struct {
	State   OwnershipState
	MovedAt int    // line number where the move occurred
	MovedTo string // what it was moved to (variable name or "function call")
}

func (c *Checker) pushOwnership() map[string]*VarOwnership {
	prev := c.ownership
	next := make(map[string]*VarOwnership, len(prev))
	for name, ownership := range prev {
		next[name] = ownership
	}
	c.ownership = next
	return prev
}

func (c *Checker) popOwnership(prev map[string]*VarOwnership) {
	c.ownership = prev
}

func (c *Checker) trackOwnership(name string, t Type) {
	if IsMoveType(t) {
		c.ownership[name] = &VarOwnership{State: Owned}
		return
	}
	delete(c.ownership, name)
}

func (c *Checker) markMoved(name string, line int, movedTo string) {
	if own, exists := c.ownership[name]; exists && own.State == Owned {
		own.State = Moved
		own.MovedAt = line
		own.MovedTo = movedTo
	}
}

func (c *Checker) markMoveFromExpression(expr ast.Expression, line int, movedTo string) {
	if ident, ok := expr.(*ast.Identifier); ok {
		c.markMoved(ident.Value, line, movedTo)
	}
}
