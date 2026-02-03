package types

type Type interface {
	String() string
	Equals(Type) bool
}

type BasicType struct {
	Name string
}

func (b *BasicType) String() string { return b.Name }
func (b *BasicType) Equals(other Type) bool {
	if o, ok := other.(*BasicType); ok {
		return b.Name == o.Name
	}
	return false
}

type ArrayType struct {
	Element Type
}

func (a *ArrayType) String() string { return "[]" + a.Element.String() }
func (a *ArrayType) Equals(other Type) bool {
	if o, ok := other.(*ArrayType); ok {
		return a.Element.Equals(o.Element)
	}
	return false
}

type FunctionType struct {
	Params []Type
	Return Type
}

func (f *FunctionType) String() string {
	s := "fn("
	for i, p := range f.Params {
		if i > 0 {
			s += ", "
		}
		s += p.String()
	}
	s += ") -> " + f.Return.String()
	return s
}

func (f *FunctionType) Equals(other Type) bool {
	o, ok := other.(*FunctionType)
	if !ok {
		return false
	}
	if len(f.Params) != len(o.Params) {
		return false
	}
	for i, p := range f.Params {
		if !p.Equals(o.Params[i]) {
			return false
		}
	}
	return f.Return.Equals(o.Return)
}

type ChannelType struct {
	Element Type
}

func (c *ChannelType) String() string { return "chan " + c.Element.String() }
func (c *ChannelType) Equals(other Type) bool {
	if o, ok := other.(*ChannelType); ok {
		return c.Element.Equals(o.Element)
	}
	return false
}

type OptionalType struct {
	Inner Type
}

func (o *OptionalType) String() string { return o.Inner.String() + "?" }
func (o *OptionalType) Equals(other Type) bool {
	if ot, ok := other.(*OptionalType); ok {
		return o.Inner.Equals(ot.Inner)
	}
	return false
}

type ClassType struct {
	Name   string
	Fields map[string]Type
}

func (c *ClassType) String() string { return c.Name }
func (c *ClassType) Equals(other Type) bool {
	if o, ok := other.(*ClassType); ok {
		return c.Name == o.Name
	}
	return false
}

type InterfaceType struct {
	Name    string
	Methods map[string]*FunctionType
}

func (i *InterfaceType) String() string { return i.Name }
func (i *InterfaceType) Equals(other Type) bool {
	if o, ok := other.(*InterfaceType); ok {
		return i.Name == o.Name
	}
	return false
}

type MapType struct {
	Key   Type
	Value Type
}

func (m *MapType) String() string { return "{" + m.Key.String() + ": " + m.Value.String() + "}" }
func (m *MapType) Equals(other Type) bool {
	if o, ok := other.(*MapType); ok {
		return m.Key.Equals(o.Key) && m.Value.Equals(o.Value)
	}
	return false
}

type RefType struct {
	Inner   Type
	Mutable bool
}

func (r *RefType) String() string {
	if r.Mutable {
		return "&mut " + r.Inner.String()
	}
	return "&" + r.Inner.String()
}

func (r *RefType) Equals(other Type) bool {
	if o, ok := other.(*RefType); ok {
		return r.Mutable == o.Mutable && r.Inner.Equals(o.Inner)
	}
	return false
}

type ModuleType struct {
	Name    string
	Exports map[string]Type
}

func (m *ModuleType) String() string { return "module " + m.Name }
func (m *ModuleType) Equals(other Type) bool {
	if o, ok := other.(*ModuleType); ok {
		return m.Name == o.Name
	}
	return false
}

var (
	Int    = &BasicType{Name: "int"}
	Float  = &BasicType{Name: "float"}
	Bool   = &BasicType{Name: "bool"}
	String = &BasicType{Name: "string"}
	Char   = &BasicType{Name: "char"}
	Void   = &BasicType{Name: "void"}
	Any    = &BasicType{Name: "any"}
	Nil    = &BasicType{Name: "nil"}
)

func IsNumeric(t Type) bool {
	if b, ok := t.(*BasicType); ok {
		return b.Name == "int" || b.Name == "float"
	}
	return false
}

func IsComparable(t Type) bool {
	if b, ok := t.(*BasicType); ok {
		switch b.Name {
		case "int", "float", "string", "char", "bool":
			return true
		}
	}
	return false
}

// TypeCategory classifies types for ownership semantics
type TypeCategory int

const (
	CopyType TypeCategory = iota // int, float, bool, char — implicit copy on assignment
	MoveType                     // string, []T, {K:V}, class instances — move on assignment
)

// Category returns whether a type is Copy or Move
func Category(t Type) TypeCategory {
	if t == nil {
		return CopyType
	}
	switch {
	case t.Equals(Int), t.Equals(Float), t.Equals(Bool), t.Equals(Char), t.Equals(Void), t.Equals(Nil):
		return CopyType
	case t.Equals(String):
		return MoveType
	case t.Equals(Any):
		return CopyType // Any is treated as copy for backward compat
	}
	switch t.(type) {
	case *ArrayType, *MapType, *ClassType:
		return MoveType
	case *RefType:
		return CopyType
	default:
		return CopyType
	}
}

// IsCopyType returns true if the type is implicitly copied on assignment
func IsCopyType(t Type) bool {
	return Category(t) == CopyType
}

// IsMoveType returns true if assignment moves ownership (source becomes invalid)
func IsMoveType(t Type) bool {
	return Category(t) == MoveType
}
