package vm

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/dev-dami/carv/pkg/ir"
)

type ValueType byte

const (
	ValInt    ValueType = iota
	ValFloat
	ValBool
	ValString
	ValArray
	ValMap
	ValResult
	ValClass
	ValFunc
	ValNil
	ValChar
)

type Value struct {
	Type      ValueType
	Int       int64
	Float     float64
	Bool      bool
	Str       string
	Arr       []Value
	M         map[string]Value
	Result    *ResultVal
	Class     map[string]Value
	ClassName string
	Func      *ir.Function
}

type ResultVal struct {
	IsOk bool
	Ok   *Value
	Err  *Value
}

type Frame struct {
	fn      *ir.Function
	pc      int
	locals  []Value
}

type VM struct {
	stack     []Value
	frames    []Frame
	funcs     map[string]*ir.Function
	module    *ir.Module
	entry     string
	out       io.Writer
	err       io.Writer
	in        io.Reader
	maxStack  int
	trace     bool
}

func New(mod *ir.Module) *VM {
	vm := &VM{
		stack:    make([]Value, 0, 4096),
		frames:   make([]Frame, 0, 256),
		funcs:    mod.Functions,
		module:   mod,
		entry:    mod.Entry,
		out:      os.Stdout,
		err:      os.Stderr,
		in:       os.Stdin,
		maxStack: 65536,
	}
	return vm
}

func (vm *VM) SetOutput(w io.Writer) {
	vm.out = w
}

func (vm *VM) SetTrace(trace bool) {
	vm.trace = trace
}

func (vm *VM) Run() (Value, error) {
	fn, ok := vm.funcs[vm.entry]
	if !ok {
		return NilValue(), fmt.Errorf("entry function %q not found", vm.entry)
	}
	return vm.runFn(fn)
}

func (vm *VM) runFn(fn *ir.Function) (Value, error) {
	locals := make([]Value, len(fn.Locals))
	for i, l := range fn.Locals {
		if l.Type == ir.IRInt {
			locals[i] = IntValue(0)
		} else if l.Type == ir.IRFloat {
			locals[i] = FloatValue(0)
		} else {
			locals[i] = NilValue()
		}
	}

	vm.frames = append(vm.frames, Frame{fn: fn, pc: 0, locals: locals})
	frameIdx := len(vm.frames) - 1

	var result Value
	var err error

	for vm.frames[frameIdx].pc < len(fn.Body) {
		inst := fn.Body[vm.frames[frameIdx].pc]
		vm.frames[frameIdx].pc++

		if vm.trace {
			vm.dumpState(&inst)
		}

		result, err = vm.exec(&inst)
		if err != nil {
			return NilValue(), err
		}
		if result.Type == ValFunc && result.Func != nil {
			break
		}
	}

	vm.frames = vm.frames[:len(vm.frames)-1]
	return result, nil
}

func (vm *VM) exec(inst *ir.Inst) (Value, error) {
	switch inst.Op {
	case ir.OpNop:
		return NilValue(), nil

	case ir.OpConstInt:
		vm.push(IntValue(inst.Arg.Int))
	case ir.OpConstFloat:
		vm.push(FloatValue(inst.Arg.Float))
	case ir.OpConstString:
		vm.push(StrValue(inst.Arg.Str))
	case ir.OpConstBool:
		vm.push(BoolValue(inst.Arg.Bool))
	case ir.OpConstNil:
		vm.push(NilValue())
	case ir.OpConstChar:
		vm.push(IntValue(inst.Arg.Int))

	case ir.OpAlloc:
		vm.setLocal(inst.Label, NilValue())

	case ir.OpLoad:
		vm.push(vm.getLocal(inst.Label))

	case ir.OpStore:
		val := vm.pop()
		vm.setLocal(inst.Label, val)

	case ir.OpLoadRef:
		val := vm.getLocal(inst.Label)
		vm.push(val)

	case ir.OpNeg:
		v := vm.pop()
		switch v.Type {
		case ValInt:
			vm.push(IntValue(-v.Int))
		case ValFloat:
			vm.push(FloatValue(-v.Float))
		}

	case ir.OpNot:
		v := vm.pop()
		vm.push(BoolValue(!v.Bool))

	case ir.OpBitNot:
		v := vm.pop()
		vm.push(IntValue(^v.Int))

	case ir.OpAdd:
		b, a := vm.pop2()
		if a.Type == ValFloat || b.Type == ValFloat {
			vm.push(FloatValue(toFloat(a) + toFloat(b)))
		} else if a.Type == ValString || b.Type == ValString {
			vm.push(StrValue(toString(a) + toString(b)))
		} else {
			vm.push(IntValue(toInt(a) + toInt(b)))
		}

	case ir.OpSub:
		b, a := vm.pop2()
		if a.Type == ValFloat || b.Type == ValFloat {
			vm.push(FloatValue(toFloat(a) - toFloat(b)))
		} else {
			vm.push(IntValue(toInt(a) - toInt(b)))
		}

	case ir.OpMul:
		b, a := vm.pop2()
		if a.Type == ValFloat || b.Type == ValFloat {
			vm.push(FloatValue(toFloat(a) * toFloat(b)))
		} else {
			vm.push(IntValue(toInt(a) * toInt(b)))
		}

	case ir.OpDiv:
		b, a := vm.pop2()
		if a.Type == ValFloat || b.Type == ValFloat {
			vm.push(FloatValue(toFloat(a) / toFloat(b)))
		} else {
			if toInt(b) == 0 {
				return NilValue(), fmt.Errorf("division by zero")
			}
			vm.push(IntValue(toInt(a) / toInt(b)))
		}

	case ir.OpPow:
		b, a := vm.pop2()
		if a.Type == ValFloat || b.Type == ValFloat {
			vm.push(FloatValue(powFloat(toFloat(a), toFloat(b))))
		} else {
			vm.push(IntValue(powInt(toInt(a), toInt(b))))
		}

	case ir.OpMod:
		b, a := vm.pop2()
		if toInt(b) == 0 {
			return NilValue(), fmt.Errorf("modulo by zero")
		}
		vm.push(IntValue(toInt(a) % toInt(b)))

	case ir.OpEq:
		b, a := vm.pop2()
		vm.push(BoolValue(valuesEqual(a, b)))

	case ir.OpNe:
		b, a := vm.pop2()
		vm.push(BoolValue(!valuesEqual(a, b)))

	case ir.OpLt:
		b, a := vm.pop2()
		if a.Type == ValFloat || b.Type == ValFloat {
			vm.push(BoolValue(toFloat(a) < toFloat(b)))
		} else {
			vm.push(BoolValue(toInt(a) < toInt(b)))
		}

	case ir.OpGt:
		b, a := vm.pop2()
		if a.Type == ValFloat || b.Type == ValFloat {
			vm.push(BoolValue(toFloat(a) > toFloat(b)))
		} else {
			vm.push(BoolValue(toInt(a) > toInt(b)))
		}

	case ir.OpLe:
		b, a := vm.pop2()
		if a.Type == ValFloat || b.Type == ValFloat {
			vm.push(BoolValue(toFloat(a) <= toFloat(b)))
		} else {
			vm.push(BoolValue(toInt(a) <= toInt(b)))
		}

	case ir.OpGe:
		b, a := vm.pop2()
		if a.Type == ValFloat || b.Type == ValFloat {
			vm.push(BoolValue(toFloat(a) >= toFloat(b)))
		} else {
			vm.push(BoolValue(toInt(a) >= toInt(b)))
		}

	case ir.OpAnd:
		b, a := vm.pop2()
		vm.push(BoolValue(a.Bool && b.Bool))

	case ir.OpOr:
		b, a := vm.pop2()
		vm.push(BoolValue(a.Bool || b.Bool))

	case ir.OpArrayLit:
		n := int(inst.Arg.Int)
		arr := make([]Value, n)
		for i := n - 1; i >= 0; i-- {
			arr[i] = vm.pop()
		}
		vm.push(ArrayValue(arr))

	case ir.OpArrayGet:
		idx := vm.pop()
		arr := vm.pop()
		i := int(toInt(idx))
		if i < 0 || i >= len(arr.Arr) {
			return NilValue(), fmt.Errorf("array index out of bounds: %d (len %d)", i, len(arr.Arr))
		}
		vm.push(arr.Arr[i])

	case ir.OpArraySet:
		val := vm.pop()
		idx := vm.pop()
		arr := vm.pop()
		i := int(toInt(idx))
		if i < 0 || i >= len(arr.Arr) {
			return NilValue(), fmt.Errorf("array index out of bounds: %d (len %d)", i, len(arr.Arr))
		}
		arr.Arr[i] = val
		vm.push(arr)

	case ir.OpArrayLen:
		arr := vm.pop()
		vm.push(IntValue(int64(len(arr.Arr))))

	case ir.OpMapLit:
		n := int(inst.Arg.Int)
		m := make(map[string]Value)
		for i := 0; i < n; i++ {
			val := vm.pop()
			key := vm.pop()
			m[toString(key)] = val
		}
		vm.push(MapValue(m))

	case ir.OpMapGet:
		key := vm.pop()
		m := vm.pop()
		k := toString(key)
		val, ok := m.M[k]
		if !ok {
			return NilValue(), fmt.Errorf("key %q not found in map", k)
		}
		vm.push(val)

	case ir.OpMapSet:
		val := vm.pop()
		key := vm.pop()
		m := vm.pop()
		m.M[toString(key)] = val
		vm.push(m)

	case ir.OpMapLen:
		m := vm.pop()
		vm.push(IntValue(int64(len(m.M))))

	case ir.OpMapKeys:
		m := vm.pop()
		keys := make([]Value, 0, len(m.M))
		for k := range m.M {
			keys = append(keys, StrValue(k))
		}
		vm.push(ArrayValue(keys))

	case ir.OpMapContains:
		key := vm.pop()
		m := vm.pop()
		_, ok := m.M[toString(key)]
		vm.push(BoolValue(ok))

	case ir.OpStrLen:
		s := vm.pop()
		vm.push(IntValue(int64(len(s.Str))))

	case ir.OpStrIndex:
		idx := vm.pop()
		s := vm.pop()
		i := int(toInt(idx))
		if i < 0 || i >= len(s.Str) {
			return NilValue(), fmt.Errorf("string index out of bounds: %d (len %d)", i, len(s.Str))
		}
		vm.push(IntValue(int64(s.Str[i])))

	case ir.OpStrCat:
		b, a := vm.pop2()
		vm.push(StrValue(toString(a) + toString(b)))

	case ir.OpStrContains:
		sub := vm.pop()
		s := vm.pop()
		vm.push(BoolValue(strings.Contains(s.Str, sub.Str)))

	case ir.OpStrRepeat:
		n := int(toInt(vm.pop()))
		s := vm.pop()
		vm.push(StrValue(strings.Repeat(s.Str, n)))

	case ir.OpCall:
		name := inst.Label
		argc := int(inst.Arg.Int)
		fn, ok := vm.funcs[name]
		if !ok {
			return NilValue(), fmt.Errorf("undefined function: %s", name)
		}

		args := make([]Value, argc)
		for i := argc - 1; i >= 0; i-- {
			args[i] = vm.pop()
		}

		result, err := vm.callFunction(fn, args)
		if err != nil {
			return NilValue(), err
		}
		vm.push(result)

	case ir.OpCallVirt:
		methodName := inst.Label
		argc := int(inst.Arg.Int)

		args := make([]Value, argc)
		for i := argc - 1; i >= 0; i-- {
			args[i] = vm.pop()
		}

		receiver := args[0]
		var fnName string
		if receiver.ClassName != "" {
			fnName = receiver.ClassName + "_" + methodName
		} else {
			fnName = methodName
		}

		fn, ok := vm.funcs[fnName]
		if !ok {
			return NilValue(), fmt.Errorf("undefined function: %s", fnName)
		}

		result, err := vm.callFunction(fn, args)
		if err != nil {
			return NilValue(), err
		}
		vm.push(result)

	case ir.OpReturn:
		return ReturnValue(), nil

	case ir.OpReturnVal:
		val := vm.pop()
		return val, nil

	case ir.OpOk:
		val := vm.pop()
		v := val
		vm.push(Value{Type: ValResult, Result: &ResultVal{IsOk: true, Ok: &v}})

	case ir.OpErr:
		val := vm.pop()
		v := val
		vm.push(Value{Type: ValResult, Result: &ResultVal{IsOk: false, Err: &v}})

	case ir.OpTry:
		val := vm.pop()
		if val.Type != ValResult {
			vm.push(val)
			break
		}
		if !val.Result.IsOk {
			return NilValue(), fmt.Errorf("try on Err: %s", valueStr(*val.Result.Err))
		}
		vm.push(*val.Result.Ok)

	case ir.OpLabel:
	case ir.OpJmp:
		vm.jumpTo(inst.Label)

	case ir.OpJmpIf:
		cond := vm.pop()
		if !cond.Bool {
			vm.jumpTo(inst.Label)
		}

	case ir.OpMatchArmEq:
		val := vm.pop()
		if val.Int == inst.Arg.Int {
			vm.jumpTo(inst.Label)
		}

	case ir.OpMatchResult:
		okLabel := inst.Label
		errLabel := inst.Arg.Str
		val := vm.pop()
		if val.Type == ValResult {
			if val.Result.IsOk {
				vm.push(*val.Result.Ok)
				vm.jumpTo(okLabel)
			} else {
				vm.push(*val.Result.Err)
				vm.jumpTo(errLabel)
			}
		}

	case ir.OpNew:
		vm.push(Value{Type: ValClass, Class: make(map[string]Value), ClassName: inst.Label})

	case ir.OpGetField:
		field := inst.Label
		obj := vm.pop()
		switch obj.Type {
		case ValClass:
			vm.push(obj.Class[field])
		case ValResult:
			switch field {
			case "ok":
				if obj.Result.Ok != nil {
					vm.push(*obj.Result.Ok)
				} else {
					vm.push(NilValue())
				}
			case "err":
				if obj.Result.Err != nil {
					vm.push(*obj.Result.Err)
				} else {
					vm.push(NilValue())
				}
			default:
				return NilValue(), fmt.Errorf("unknown result field: %s", field)
			}
		default:
			return NilValue(), fmt.Errorf("cannot get field %s from %v", field, obj.Type)
		}

	case ir.OpSetField:
		field := inst.Label
		obj := vm.pop()
		if obj.Type == ValClass {
			val := vm.pop()
			obj.Class[field] = val
			vm.push(obj)
		}

	case ir.OpBorrow:
	case ir.OpDeref:

	case ir.OpPrint:
		v := vm.pop()
		fmt.Fprint(vm.out, valueStr(v))

	case ir.OpPrintln:
		v := vm.pop()
		fmt.Fprintln(vm.out, valueStr(v))

	case ir.OpReadLine:
		var s string
		_, err := fmt.Fscanln(vm.in, &s)
		if err != nil {
			s = ""
		}
		vm.push(StrValue(s))

	case ir.OpLen:
		v := vm.pop()
		switch v.Type {
		case ValString:
			vm.push(IntValue(int64(len(v.Str))))
		case ValArray:
			vm.push(IntValue(int64(len(v.Arr))))
		case ValMap:
			vm.push(IntValue(int64(len(v.M))))
		default:
			vm.push(IntValue(0))
		}

	case ir.OpContains:
		sub := vm.pop()
		container := vm.pop()
		switch container.Type {
		case ValString:
			vm.push(BoolValue(strings.Contains(container.Str, sub.Str)))
		case ValArray:
			found := false
			for _, e := range container.Arr {
				if valuesEqual(e, sub) {
					found = true
					break
				}
			}
			vm.push(BoolValue(found))
		case ValMap:
			_, ok := container.M[toString(sub)]
			vm.push(BoolValue(ok))
		default:
			vm.push(BoolValue(false))
		}

	case ir.OpKeys:
		v := vm.pop()
		if v.Type == ValMap {
			keys := make([]Value, 0, len(v.M))
			for k := range v.M {
				keys = append(keys, StrValue(k))
			}
			vm.push(ArrayValue(keys))
		} else {
			vm.push(ArrayValue(nil))
		}

	case ir.OpAssert:
		v := vm.pop()
		if !v.Bool {
			return NilValue(), fmt.Errorf("assertion failed")
		}

	case ir.OpToInt:
		v := vm.pop()
		vm.push(IntValue(toInt(v)))

	case ir.OpToFloat:
		v := vm.pop()
		vm.push(FloatValue(toFloat(v)))

	case ir.OpToString:
		v := vm.pop()
		vm.push(StrValue(valueStr(v)))

	case ir.OpInterp:
		n := int(inst.Arg.Int)
		parts := make([]string, n)
		for i := n - 1; i >= 0; i-- {
			parts[i] = valueStr(vm.pop())
		}
		vm.push(StrValue(strings.Join(parts, "")))

	case ir.OpCastInt:
		v := vm.pop()
		vm.push(IntValue(toInt(v)))

	case ir.OpCastFloat:
		v := vm.pop()
		vm.push(FloatValue(toFloat(v)))

	default:
		return NilValue(), fmt.Errorf("unknown op: %s", inst.Op)
	}

	return NilValue(), nil
}

func (vm *VM) callFunction(fn *ir.Function, args []Value) (Value, error) {
	locals := make([]Value, len(fn.Locals))
	for i, l := range fn.Locals {
		if l.Type == ir.IRInt {
			locals[i] = IntValue(0)
		} else if l.Type == ir.IRFloat {
			locals[i] = FloatValue(0)
		} else {
			locals[i] = NilValue()
		}
	}

	for i := range fn.Params {
		if i < len(args) {
			locals[i] = args[i]
		}
	}

	frame := Frame{fn: fn, pc: 0, locals: locals}
	vm.frames = append(vm.frames, frame)
	frameIdx := len(vm.frames) - 1

	var result Value
	var err error

	for vm.frames[frameIdx].pc < len(fn.Body) {
		inst := fn.Body[vm.frames[frameIdx].pc]
		vm.frames[frameIdx].pc++

		if vm.trace {
			vm.dumpState(&inst)
		}

		result, err = vm.exec(&inst)
		if err != nil {
			return NilValue(), err
		}

		if inst.Op == ir.OpReturn || inst.Op == ir.OpReturnVal {
			break
		}
	}

	vm.frames = vm.frames[:len(vm.frames)-1]

	if result.Type == ValFunc && result.Func != nil {
		return NilValue(), nil
	}
	return result, nil
}

func (vm *VM) jumpTo(label string) {
	frame := &vm.frames[len(vm.frames)-1]
	for i, inst := range frame.fn.Body {
		if inst.Op == ir.OpLabel && inst.Label == label {
			frame.pc = i + 1
			return
		}
	}
}

func (vm *VM) push(v Value) {
	if len(vm.stack) >= vm.maxStack {
		panic("stack overflow")
	}
	vm.stack = append(vm.stack, v)
}

func (vm *VM) pop() Value {
	if len(vm.stack) == 0 {
		panic("stack underflow")
	}
	v := vm.stack[len(vm.stack)-1]
	vm.stack = vm.stack[:len(vm.stack)-1]
	return v
}

func (vm *VM) pop2() (Value, Value) {
	return vm.pop(), vm.pop()
}

func (vm *VM) getLocal(name string) Value {
	if len(vm.frames) == 0 {
		return NilValue()
	}
	frame := &vm.frames[len(vm.frames)-1]
	for i, l := range frame.fn.Locals {
		if l.Name == name {
			return frame.locals[i]
		}
	}
	return NilValue()
}

func (vm *VM) setLocal(name string, val Value) {
	if len(vm.frames) == 0 {
		return
	}
	frame := &vm.frames[len(vm.frames)-1]
	for i, l := range frame.fn.Locals {
		if l.Name == name {
			frame.locals[i] = val
			return
		}
	}
}

func (vm *VM) dumpState(inst *ir.Inst) {
	frame := &vm.frames[len(vm.frames)-1]
	stackVals := make([]string, len(vm.stack))
	for i, v := range vm.stack {
		stackVals[i] = valueStr(v)
	}
		fmt.Fprintf(vm.err, "[pc=%d] %s | stack=[%s] | locals=%v\n",
			frame.pc-1, ir.FormatInst(frame.pc-1, inst), strings.Join(stackVals, ", "), frame.locals)
}

func IntValue(v int64) Value {
	return Value{Type: ValInt, Int: v}
}

func FloatValue(v float64) Value {
	return Value{Type: ValFloat, Float: v}
}

func BoolValue(v bool) Value {
	return Value{Type: ValBool, Bool: v}
}

func StrValue(v string) Value {
	return Value{Type: ValString, Str: v}
}

func ArrayValue(v []Value) Value {
	return Value{Type: ValArray, Arr: v}
}

func MapValue(v map[string]Value) Value {
	return Value{Type: ValMap, M: v}
}

func NilValue() Value {
	return Value{Type: ValNil}
}

func ReturnValue() Value {
	return Value{Type: ValFunc}
}

func toInt(v Value) int64 {
	switch v.Type {
	case ValInt, ValChar:
		return v.Int
	case ValFloat:
		return int64(v.Float)
	case ValBool:
		if v.Bool {
			return 1
		}
		return 0
	}
	return 0
}

func toFloat(v Value) float64 {
	switch v.Type {
	case ValFloat:
		return v.Float
	case ValInt, ValChar:
		return float64(v.Int)
	case ValBool:
		if v.Bool {
			return 1
		}
		return 0
	}
	return 0
}

func toString(v Value) string {
	switch v.Type {
	case ValString:
		return v.Str
	case ValInt, ValChar:
		return fmt.Sprintf("%d", v.Int)
	case ValFloat:
		return fmt.Sprintf("%g", v.Float)
	case ValBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case ValNil:
		return "nil"
	case ValArray:
		parts := make([]string, len(v.Arr))
		for i, e := range v.Arr {
			parts[i] = toString(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case ValMap:
		parts := make([]string, 0, len(v.M))
		for k, val := range v.M {
			parts = append(parts, fmt.Sprintf("%q: %s", k, valueStr(val)))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return "?"
	}
}

func valueStr(v Value) string {
	return toString(v)
}

func powInt(a, b int64) int64 {
	if b < 0 {
		return 0
	}
	result := int64(1)
	for b > 0 {
		if b&1 == 1 {
			result *= a
		}
		a *= a
		b >>= 1
	}
	return result
}

func powFloat(a, b float64) float64 {
	return math.Pow(a, b)
}

func valuesEqual(a, b Value) bool {
	if a.Type != b.Type {
		if a.Type == ValInt && b.Type == ValFloat {
			return float64(a.Int) == b.Float
		}
		if a.Type == ValFloat && b.Type == ValInt {
			return a.Float == float64(b.Int)
		}
		return false
	}
	switch a.Type {
	case ValInt, ValChar:
		return a.Int == b.Int
	case ValFloat:
		return a.Float == b.Float
	case ValBool:
		return a.Bool == b.Bool
	case ValString:
		return a.Str == b.Str
	case ValNil:
		return true
	case ValArray:
		if len(a.Arr) != len(b.Arr) {
			return false
		}
		for i := range a.Arr {
			if !valuesEqual(a.Arr[i], b.Arr[i]) {
				return false
			}
		}
		return true
	case ValMap:
		if len(a.M) != len(b.M) {
			return false
		}
		for k, va := range a.M {
			vb, ok := b.M[k]
			if !ok || !valuesEqual(va, vb) {
				return false
			}
		}
		return true
	case ValResult:
		if a.Result.IsOk != b.Result.IsOk {
			return false
		}
		if a.Result.Ok != nil && b.Result.Ok != nil {
			if !valuesEqual(*a.Result.Ok, *b.Result.Ok) {
				return false
			}
		}
		if a.Result.Err != nil && b.Result.Err != nil {
			if !valuesEqual(*a.Result.Err, *b.Result.Err) {
				return false
			}
		}
		return true
	}
	return false
}

func FormatInst(idx int, inst *ir.Inst) string {
	return ir.FormatInst(idx, inst)
}
