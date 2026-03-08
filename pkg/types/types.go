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

type FutureType struct {
	Inner Type
}

func (f *FutureType) String() string { return "Future<" + f.Inner.String() + ">" }
func (f *FutureType) Equals(other Type) bool {
	if o, ok := other.(*FutureType); ok {
		return f.Inner.Equals(o.Inner)
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

	// Sized integer types
	U8    = &BasicType{Name: "u8"}
	U16   = &BasicType{Name: "u16"}
	U32   = &BasicType{Name: "u32"}
	U64   = &BasicType{Name: "u64"}
	I8    = &BasicType{Name: "i8"}
	I16   = &BasicType{Name: "i16"}
	I32   = &BasicType{Name: "i32"}
	I64   = &BasicType{Name: "i64"}
	F32   = &BasicType{Name: "f32"}
	F64   = &BasicType{Name: "f64"}
	Usize = &BasicType{Name: "usize"}
	Isize = &BasicType{Name: "isize"}
)

func IsNumeric(t Type) bool {
	if b, ok := t.(*BasicType); ok {
		switch b.Name {
		case "int", "float",
			"u8", "u16", "u32", "u64",
			"i8", "i16", "i32", "i64",
			"f32", "f64", "usize", "isize":
			return true
		}
	}
	return false
}

func IsComparable(t Type) bool {
	if b, ok := t.(*BasicType); ok {
		switch b.Name {
		case "int", "float", "string", "char", "bool",
			"u8", "u16", "u32", "u64",
			"i8", "i16", "i32", "i64",
			"f32", "f64", "usize", "isize":
			return true
		}
	}
	return false
}

type VolatileType struct {
	Inner Type
}

func (v *VolatileType) String() string { return "volatile<" + v.Inner.String() + ">" }
func (v *VolatileType) Equals(other Type) bool {
	if o, ok := other.(*VolatileType); ok {
		return v.Inner.Equals(o.Inner)
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
	case t.Equals(U8), t.Equals(U16), t.Equals(U32), t.Equals(U64):
		return CopyType
	case t.Equals(I8), t.Equals(I16), t.Equals(I32), t.Equals(I64):
		return CopyType
	case t.Equals(F32), t.Equals(F64), t.Equals(Usize), t.Equals(Isize):
		return CopyType
	case t.Equals(String):
		return MoveType
	case t.Equals(Any):
		return CopyType // Any is treated as copy for backward compat
	}
	switch t.(type) {
	case *ArrayType, *MapType, *ClassType, *FutureType:
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
