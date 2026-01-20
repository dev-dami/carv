package eval

import (
	"fmt"
	"strings"

	"github.com/dev-dami/carv/pkg/ast"
)

type ObjectType string

const (
	INTEGER_OBJ      ObjectType = "INTEGER"
	FLOAT_OBJ        ObjectType = "FLOAT"
	BOOLEAN_OBJ      ObjectType = "BOOLEAN"
	STRING_OBJ       ObjectType = "STRING"
	CHAR_OBJ         ObjectType = "CHAR"
	NIL_OBJ          ObjectType = "NIL"
	RETURN_VALUE_OBJ ObjectType = "RETURN_VALUE"
	ERROR_OBJ        ObjectType = "ERROR"
	FUNCTION_OBJ     ObjectType = "FUNCTION"
	BUILTIN_OBJ      ObjectType = "BUILTIN"
	ARRAY_OBJ        ObjectType = "ARRAY"
	BREAK_OBJ        ObjectType = "BREAK"
	CONTINUE_OBJ     ObjectType = "CONTINUE"
	OK_OBJ           ObjectType = "OK"
	ERR_OBJ          ObjectType = "ERR"
	INSTANCE_OBJ     ObjectType = "INSTANCE"
	CLASS_OBJ        ObjectType = "CLASS"
	MAP_OBJ          ObjectType = "MAP"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

type Float struct {
	Value float64
}

func (f *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string  { return fmt.Sprintf("%g", f.Value) }

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

type Char struct {
	Value rune
}

func (c *Char) Type() ObjectType { return CHAR_OBJ }
func (c *Char) Inspect() string  { return string(c.Value) }

type Nil struct{}

func (n *Nil) Type() ObjectType { return NIL_OBJ }
func (n *Nil) Inspect() string  { return "nil" }

type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

type Error struct {
	Message string
	Line    int
	Column  int
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string {
	if e.Line > 0 {
		return fmt.Sprintf("error at %d:%d: %s", e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("error: %s", e.Message)
}

type Function struct {
	Parameters []*ast.Parameter
	Body       *ast.BlockStatement
	Env        *Environment
	Name       string
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	var params []string
	for _, p := range f.Parameters {
		params = append(params, p.Name.Value)
	}
	return fmt.Sprintf("fn(%s) { ... }", strings.Join(params, ", "))
}

type BuiltinFunction func(args ...Object) Object

type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

type Array struct {
	Elements []Object
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	var elements []string
	for _, e := range a.Elements {
		elements = append(elements, e.Inspect())
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

type Break struct{}

func (b *Break) Type() ObjectType { return BREAK_OBJ }
func (b *Break) Inspect() string  { return "break" }

type Continue struct{}

func (c *Continue) Type() ObjectType { return CONTINUE_OBJ }
func (c *Continue) Inspect() string  { return "continue" }

type Ok struct {
	Value Object
}

func (o *Ok) Type() ObjectType { return OK_OBJ }
func (o *Ok) Inspect() string  { return fmt.Sprintf("Ok(%s)", o.Value.Inspect()) }

type Err struct {
	Value Object
}

func (e *Err) Type() ObjectType { return ERR_OBJ }
func (e *Err) Inspect() string  { return fmt.Sprintf("Err(%s)", e.Value.Inspect()) }

type Class struct {
	Name    string
	Fields  map[string]Object
	Methods map[string]*Function
}

func (c *Class) Type() ObjectType { return CLASS_OBJ }
func (c *Class) Inspect() string  { return fmt.Sprintf("<class %s>", c.Name) }

type Instance struct {
	Class  *Class
	Fields map[string]Object
}

func (i *Instance) Type() ObjectType { return INSTANCE_OBJ }
func (i *Instance) Inspect() string {
	return fmt.Sprintf("<%s instance>", i.Class.Name)
}

type Method struct {
	Instance *Instance
	Fn       *Function
}

func (m *Method) Type() ObjectType { return FUNCTION_OBJ }
func (m *Method) Inspect() string  { return fmt.Sprintf("<method %s>", m.Fn.Name) }

// HashKey is used as a key in maps
type HashKey struct {
	Type  ObjectType
	Value uint64
}

// Hashable objects can be used as map keys
type Hashable interface {
	HashKey() HashKey
}

func (i *Integer) HashKey() HashKey {
	return HashKey{Type: i.Type(), Value: uint64(i.Value)}
}

func (b *Boolean) HashKey() HashKey {
	var value uint64
	if b.Value {
		value = 1
	} else {
		value = 0
	}
	return HashKey{Type: b.Type(), Value: value}
}

func (s *String) HashKey() HashKey {
	h := uint64(0)
	for _, c := range s.Value {
		h = h*31 + uint64(c)
	}
	return HashKey{Type: s.Type(), Value: h}
}

// MapPair holds a key-value pair in a map
type MapPair struct {
	Key   Object
	Value Object
}

// Map is a hash map object
type Map struct {
	Pairs map[HashKey]MapPair
}

func (m *Map) Type() ObjectType { return MAP_OBJ }
func (m *Map) Inspect() string {
	var pairs []string
	for _, pair := range m.Pairs {
		pairs = append(pairs, fmt.Sprintf("%s: %s", pair.Key.Inspect(), pair.Value.Inspect()))
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

var (
	NIL   = &Nil{}
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
	BREAK = &Break{}
	CONT  = &Continue{}
)
