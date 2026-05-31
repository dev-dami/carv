package vm

import (
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/dev-dami/carv/pkg/ir"
)

type ValueType byte

const (
	ValInt ValueType = iota
	ValFloat
	ValBool
	ValString
	ValArray
	ValMap
	ValResult
	ValClass
	ValFunc
	ValFuture
	ValNil
	ValChar
)

func (vt ValueType) String() string {
	switch vt {
	case ValInt:
		return "int"
	case ValFloat:
		return "float"
	case ValBool:
		return "bool"
	case ValString:
		return "string"
	case ValArray:
		return "array"
	case ValMap:
		return "map"
	case ValResult:
		return "result"
	case ValClass:
		return "class"
	case ValFunc:
		return "fn"
	case ValFuture:
		return "future"
	case ValNil:
		return "nil"
	case ValChar:
		return "char"
	default:
		return "?"
	}
}

type Value struct {
	Type      ValueType
	Int       int64
	Float     float64
	Bool      bool
	Str       string
	Arr       []Value
	M         map[string]Value
	Result    *ResultVal
	Future    *FutureVal
	Class     map[string]Value
	ClassName string
	Func      *ir.Function
	Captures  []Value // captured variables for closures
}

type FutureVal struct {
	Value Value
}

type ResultVal struct {
	IsOk bool
	Ok   *Value
	Err  *Value
}

type Frame struct {
	fn     *ir.Function
	pc     int
	locals []Value
}

type VM struct {
	stack         []Value
	frames        []Frame
	funcs         map[string]*ir.Function
	module        *ir.Module
	entry         string
	out           io.Writer
	err           io.Writer
	in            io.Reader
	maxStack      int
	trace         bool
	tcpHandles    map[int]interface{}
	nextTCPHandle int
}

func New(mod *ir.Module) *VM {
	vm := &VM{
		stack:         make([]Value, 0, 4096),
		frames:        make([]Frame, 0, 256),
		funcs:         mod.Functions,
		module:        mod,
		entry:         mod.Entry,
		out:           os.Stdout,
		err:           os.Stderr,
		in:            os.Stdin,
		maxStack:      65536,
		tcpHandles:    make(map[int]interface{}),
		nextTCPHandle: 1,
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
		frame := &vm.frames[len(vm.frames)-1]
		idx := inst.Arg.Idx
		if idx < 0 || idx >= len(frame.locals) {
			return NilValue(), fmt.Errorf("invalid local index %d", idx)
		}
		frame.locals[idx] = NilValue()

	case ir.OpLoad:
		frame := &vm.frames[len(vm.frames)-1]
		idx := inst.Arg.Idx
		if idx < 0 || idx >= len(frame.locals) {
			return NilValue(), fmt.Errorf("invalid local index %d", idx)
		}
		vm.push(frame.locals[idx])

	case ir.OpStore:
		val := vm.pop()
		frame := &vm.frames[len(vm.frames)-1]
		idx := inst.Arg.Idx
		if idx < 0 || idx >= len(frame.locals) {
			return NilValue(), fmt.Errorf("invalid local index %d", idx)
		}
		frame.locals[idx] = val

	case ir.OpLoadRef:
		frame := &vm.frames[len(vm.frames)-1]
		idx := inst.Arg.Idx
		if idx < 0 || idx >= len(frame.locals) {
			return NilValue(), fmt.Errorf("invalid local index %d", idx)
		}
		vm.push(frame.locals[idx])

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
		m := make(map[string]Value, n)
		for i := 0; i < n; i++ {
			val := vm.pop()
			key := vm.pop()
			m[toString(key)] = val
		}
		vm.push(MapValue(m))

	case ir.OpMapGet:
		key := vm.pop()
		m := vm.pop()
		k := key.Str
		if key.Type != ValString {
			k = toString(key)
		}
		val, ok := m.M[k]
		if !ok {
			return NilValue(), fmt.Errorf("key %q not found in map", k)
		}
		vm.push(val)

	case ir.OpMapSet:
		val := vm.pop()
		key := vm.pop()
		m := vm.pop()
		k := key.Str
		if key.Type != ValString {
			k = toString(key)
		}
		m.M[k] = val
		vm.push(m)

	case ir.OpMapLen:
		m := vm.pop()
		vm.push(IntValue(int64(len(m.M))))

	case ir.OpMapKeys:
		m := vm.pop()
		keys := make([]Value, len(m.M))
		i := 0
		for k := range m.M {
			keys[i] = StrValue(k)
			i++
		}
		vm.push(ArrayValue(keys))

	case ir.OpMapContains:
		key := vm.pop()
		m := vm.pop()
		k := key.Str
		if key.Type != ValString {
			k = toString(key)
		}
		_, ok := m.M[k]
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

		result, err := vm.callFunction(fn, args, nil)
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

		result, err := vm.callFunction(fn, args, nil)
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
		s := v.Str
		switch v.Type {
		case ValString:
			// keep existing string value
		case ValInt, ValChar:
			s = strconv.FormatInt(v.Int, 10)
		case ValFloat:
			s = strconv.FormatFloat(v.Float, 'g', -1, 64)
		case ValBool:
			if v.Bool {
				s = "true"
			} else {
				s = "false"
			}
		case ValNil:
			s = "nil"
		default:
			s = valueStr(v)
		}
		vm.push(StrValue(s))

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

	// --- String builtins ---
	case ir.OpStrSplit:
		sep := vm.pop()
		s := vm.pop()
		parts := strings.Split(s.Str, sep.Str)
		arr := make([]Value, len(parts))
		for i, p := range parts {
			arr[i] = StrValue(p)
		}
		vm.push(ArrayValue(arr))

	case ir.OpStrJoin:
		sep := vm.pop()
		arr := vm.pop()
		parts := make([]string, len(arr.Arr))
		for i, v := range arr.Arr {
			parts[i] = toString(v)
		}
		vm.push(StrValue(strings.Join(parts, sep.Str)))

	case ir.OpStrTrim:
		s := vm.pop()
		vm.push(StrValue(strings.TrimSpace(s.Str)))

	case ir.OpStrSubstr:
		end := int(toInt(vm.pop()))
		start := int(toInt(vm.pop()))
		s := vm.pop()
		if start < 0 {
			start = 0
		}
		if end > len(s.Str) {
			end = len(s.Str)
		}
		if start >= end {
			vm.push(StrValue(""))
		} else {
			vm.push(StrValue(s.Str[start:end]))
		}

	case ir.OpStrStartsWith:
		prefix := vm.pop()
		s := vm.pop()
		vm.push(BoolValue(strings.HasPrefix(s.Str, prefix.Str)))

	case ir.OpStrEndsWith:
		suffix := vm.pop()
		s := vm.pop()
		vm.push(BoolValue(strings.HasSuffix(s.Str, suffix.Str)))

	case ir.OpStrReplace:
		new := vm.pop()
		old := vm.pop()
		s := vm.pop()
		vm.push(StrValue(strings.ReplaceAll(s.Str, old.Str, new.Str)))

	case ir.OpStrIndexOf:
		sub := vm.pop()
		s := vm.pop()
		idx := strings.Index(s.Str, sub.Str)
		vm.push(IntValue(int64(idx)))

	case ir.OpStrToUpper:
		s := vm.pop()
		vm.push(StrValue(strings.ToUpper(s.Str)))

	case ir.OpStrToLower:
		s := vm.pop()
		vm.push(StrValue(strings.ToLower(s.Str)))

	case ir.OpOrd:
		ch := vm.pop()
		s := toString(ch)
		if len(s) > 0 {
			vm.push(IntValue(int64(s[0])))
		} else {
			vm.push(IntValue(0))
		}

	case ir.OpChr:
		code := int(toInt(vm.pop()))
		if code >= 0 && code <= 0x10FFFF {
			vm.push(IntValue(int64(code)))
		} else {
			vm.push(IntValue(0))
		}

	case ir.OpCharAt:
		idx := int(toInt(vm.pop()))
		s := vm.pop()
		if idx >= 0 && idx < len(s.Str) {
			vm.push(IntValue(int64(s.Str[idx])))
		} else {
			vm.push(IntValue(0))
		}

	// --- Array builtins ---
	case ir.OpArrayPush:
		item := vm.pop()
		arr := vm.pop()
		newArr := make([]Value, len(arr.Arr)+1)
		copy(newArr, arr.Arr)
		newArr[len(arr.Arr)] = item
		vm.push(ArrayValue(newArr))

	case ir.OpArrayPushLocal:
		item := vm.pop()
		frame := &vm.frames[len(vm.frames)-1]
		idx := inst.Arg.Idx
		if idx < 0 || idx >= len(frame.locals) {
			return NilValue(), fmt.Errorf("invalid local index %d", idx)
		}
		cur := frame.locals[idx]
		arr := cur.Arr
		n := len(arr)

		// In-place append if capacity allows. Carv has move semantics so
		// this local should be the sole reference to arr.
		if n < cap(arr) {
			frame.locals[idx] = Value{Type: ValArray, Arr: append(arr, item)}
			return NilValue(), nil
		}

		newCap := n * 2
		if newCap < n+16 {
			newCap = n + 16
		}
		newArr := make([]Value, n+1, newCap)
		copy(newArr, arr)
		newArr[n] = item
		frame.locals[idx] = Value{Type: ValArray, Arr: newArr}

	case ir.OpArrayHead:
		arr := vm.pop()
		if len(arr.Arr) == 0 {
			vm.push(NilValue())
		} else {
			vm.push(arr.Arr[0])
		}

	case ir.OpArrayTail:
		arr := vm.pop()
		if len(arr.Arr) <= 1 {
			vm.push(ArrayValue(nil))
		} else {
			tail := make([]Value, len(arr.Arr)-1)
			copy(tail, arr.Arr[1:])
			vm.push(ArrayValue(tail))
		}

	// --- Map builtins ---
	case ir.OpMapValues:
		m := vm.pop()
		vals := make([]Value, len(m.M))
		i := 0
		for _, v := range m.M {
			vals[i] = v
			i++
		}
		vm.push(ArrayValue(vals))

	case ir.OpMapDelete:
		key := vm.pop()
		m := vm.pop()
		keyStr := key.Str
		if key.Type != ValString {
			keyStr = toString(key)
		}
		newMap := make(map[string]Value, len(m.M))
		for k, v := range m.M {
			if k != keyStr {
				newMap[k] = v
			}
		}
		vm.push(MapValue(newMap))

	// --- Type / conversion ---
	case ir.OpTypeOf:
		v := vm.pop()
		typeName := "unknown"
		switch v.Type {
		case ValInt:
			typeName = "int"
		case ValFloat:
			typeName = "float"
		case ValBool:
			typeName = "bool"
		case ValString:
			typeName = "string"
		case ValArray:
			typeName = "array"
		case ValMap:
			typeName = "map"
		case ValResult:
			typeName = "result"
		case ValClass:
			typeName = "class"
		case ValFunc:
			typeName = "fn"
		case ValNil:
			typeName = "nil"
		case ValChar:
			typeName = "char"
		}
		vm.push(StrValue(typeName))

	case ir.OpParseInt:
		s := vm.pop()
		n, err := strconv.ParseInt(strings.TrimSpace(s.Str), 10, 64)
		if err != nil {
			vm.push(IntValue(0))
		} else {
			vm.push(IntValue(n))
		}

	case ir.OpParseFloat:
		s := vm.pop()
		f, err := strconv.ParseFloat(strings.TrimSpace(s.Str), 64)
		if err != nil {
			vm.push(FloatValue(0))
		} else {
			vm.push(FloatValue(f))
		}

	// --- File I/O builtins ---
	case ir.OpReadFile:
		path := vm.pop()
		data, err := os.ReadFile(path.Str)
		if err != nil {
			vm.push(StrValue(""))
		} else {
			vm.push(StrValue(string(data)))
		}

	case ir.OpWriteFile:
		content := vm.pop()
		path := vm.pop()
		_ = os.WriteFile(path.Str, []byte(content.Str), 0644)

	case ir.OpAppendFile:
		content := vm.pop()
		path := vm.pop()
		f, err := os.OpenFile(path.Str, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			_, _ = f.WriteString(content.Str)
			_ = f.Close()
		}

	case ir.OpFileExists:
		path := vm.pop()
		_, err := os.Stat(path.Str)
		vm.push(BoolValue(err == nil))

	case ir.OpMkDir:
		path := vm.pop()
		_ = os.MkdirAll(path.Str, 0755)

	case ir.OpRemoveFile:
		path := vm.pop()
		_ = os.Remove(path.Str)

	case ir.OpRenameFile:
		newPath := vm.pop()
		oldPath := vm.pop()
		_ = os.Rename(oldPath.Str, newPath.Str)

	case ir.OpReadDir:
		path := vm.pop()
		entries, err := os.ReadDir(path.Str)
		if err != nil {
			vm.push(ArrayValue(nil))
		} else {
			names := make([]Value, len(entries))
			for i, e := range entries {
				names[i] = StrValue(e.Name())
			}
			vm.push(ArrayValue(names))
		}

	case ir.OpCwd:
		dir, err := os.Getwd()
		if err != nil {
			vm.push(StrValue(""))
		} else {
			vm.push(StrValue(dir))
		}

	// --- Process builtins ---
	case ir.OpArgs:
		args := make([]Value, len(os.Args))
		for i, a := range os.Args {
			args[i] = StrValue(a)
		}
		vm.push(ArrayValue(args))

	case ir.OpExec:
		// Pop all args, then the command
		argc := int(inst.Arg.Int)
		cmdArgs := make([]string, argc)
		for i := argc - 1; i >= 0; i-- {
			cmdArgs[i] = toString(vm.pop())
		}
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		err := cmd.Run()
		if err != nil {
			vm.push(IntValue(-1))
		} else {
			vm.push(IntValue(0))
		}

	case ir.OpExecOutput:
		argc := int(inst.Arg.Int)
		cmdArgs := make([]string, argc)
		for i := argc - 1; i >= 0; i-- {
			cmdArgs[i] = toString(vm.pop())
		}
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		out, err := cmd.Output()
		if err != nil {
			vm.push(StrValue(""))
		} else {
			vm.push(StrValue(string(out)))
		}

	// --- Environment builtins ---
	case ir.OpGetEnv:
		key := vm.pop()
		vm.push(StrValue(os.Getenv(key.Str)))

	case ir.OpSetEnv:
		val := vm.pop()
		key := vm.pop()
		os.Setenv(key.Str, val.Str)

	// --- TCP builtins ---
	case ir.OpTCPListen:
		port := int(toInt(vm.pop()))
		host := vm.pop()
		ln, err := net.Listen("tcp", host.Str+":"+strconv.Itoa(port))
		if err != nil {
			vm.push(IntValue(-1))
			break
		}
		listener, ok := ln.(*net.TCPListener)
		if !ok {
			_ = ln.Close()
			vm.push(IntValue(-1))
			break
		}
		handle := vm.nextTCPHandle
		vm.nextTCPHandle++
		vm.tcpHandles[handle] = listener
		vm.push(IntValue(int64(handle)))

	case ir.OpTCPAccept:
		handle := int(toInt(vm.pop()))
		conn, ok := vm.tcpHandles[handle]
		if !ok {
			vm.push(IntValue(-1))
			break
		}
		listener, ok := conn.(*net.TCPListener)
		if !ok {
			vm.push(IntValue(-1))
			break
		}
		client, err := listener.Accept()
		if err != nil {
			vm.push(IntValue(-1))
			break
		}
		newHandle := vm.nextTCPHandle
		vm.nextTCPHandle++
		vm.tcpHandles[newHandle] = client
		vm.push(IntValue(int64(newHandle)))

	case ir.OpTCPRead:
		maxBytes := int(toInt(vm.pop()))
		handle := int(toInt(vm.pop()))
		conn, ok := vm.tcpHandles[handle]
		if !ok {
			vm.push(StrValue(""))
			break
		}
		stream, ok := conn.(net.Conn)
		if !ok {
			vm.push(StrValue(""))
			break
		}
		buf := make([]byte, maxBytes)
		n, err := stream.Read(buf)
		if err != nil {
			if err == io.EOF {
				vm.push(StrValue(""))
				break
			}
			vm.push(StrValue(""))
			break
		}
		vm.push(StrValue(string(buf[:n])))

	case ir.OpTCPWrite:
		data := vm.pop()
		handle := int(toInt(vm.pop()))
		conn, ok := vm.tcpHandles[handle]
		if !ok {
			vm.push(IntValue(-1))
			break
		}
		stream, ok := conn.(net.Conn)
		if !ok {
			vm.push(IntValue(-1))
			break
		}
		bytes := []byte(data.Str)
		n, err := stream.Write(bytes)
		if err != nil {
			vm.push(IntValue(-1))
			break
		}
		vm.push(IntValue(int64(n)))

	case ir.OpTCPClose:
		handle := int(toInt(vm.pop()))
		conn, ok := vm.tcpHandles[handle]
		if !ok {
			vm.push(BoolValue(false))
			break
		}
		delete(vm.tcpHandles, handle)
		closer, ok := conn.(io.Closer)
		if !ok {
			vm.push(BoolValue(false))
			break
		}
		err := closer.Close()
		vm.push(BoolValue(err == nil))

	// --- Misc builtins ---
	case ir.OpExit:
		code := int(toInt(vm.pop()))
		os.Exit(code)
		return NilValue(), nil

	case ir.OpPanic:
		msg := vm.pop()
		return NilValue(), fmt.Errorf("panic: %s", toString(msg))

	// --- Closure support ---
	case ir.OpMakeClosure:
		fnName := inst.Label
		fn, ok := vm.funcs[fnName]
		if !ok {
			return NilValue(), fmt.Errorf("closure function %q not found", fnName)
		}
		n := int(inst.Arg.Int)
		captures := make([]Value, n)
		for i := n - 1; i >= 0; i-- {
			captures[i] = vm.pop()
		}
		vm.push(Value{Type: ValFunc, Func: fn, Captures: captures})

	case ir.OpCallFunc:
		argc := int(inst.Arg.Int)
		// Stack: [fn, arg0, arg1, ...]. fn was pushed first, args on top.
		// Pop args first (top of stack), then fn value.
		args := make([]Value, argc)
		for i := argc - 1; i >= 0; i-- {
			args[i] = vm.pop()
		}
		fnVal := vm.pop()
		if fnVal.Type != ValFunc || fnVal.Func == nil {
			return NilValue(), fmt.Errorf("cannot call non-function value")
		}
		result, err := vm.callFunction(fnVal.Func, args, fnVal.Captures)
		if err != nil {
			return NilValue(), err
		}
		vm.push(result)

	case ir.OpMakeFuture:
		val := vm.pop()
		vm.push(Value{Type: ValFuture, Future: &FutureVal{Value: val}})

	case ir.OpAwait:
		val := vm.pop()
		if val.Type != ValFuture {
			return NilValue(), fmt.Errorf("await requires Future value, got %s", val.Type.String())
		}
		vm.push(val.Future.Value)

	case ir.OpSpawn:
		fnName := inst.Label
		fn, ok := vm.funcs[fnName]
		if !ok {
			return NilValue(), fmt.Errorf("spawn function %q not found", fnName)
		}
		_, err := vm.callFunction(fn, nil, nil)
		if err != nil {
			return NilValue(), err
		}
		vm.push(NilValue())

	default:
		return NilValue(), fmt.Errorf("unknown op: %s", inst.Op)
	}

	return NilValue(), nil
}

func (vm *VM) callFunction(fn *ir.Function, args []Value, captures []Value) (Value, error) {
	locals := make([]Value, len(fn.Locals))
	for i, l := range fn.Locals {
		if i < len(fn.Params) {
			if i < len(args) {
				locals[i] = args[i]
				continue
			}
		}
		if l.Type == ir.IRInt {
			locals[i] = IntValue(0)
		} else if l.Type == ir.IRFloat {
			locals[i] = FloatValue(0)
		} else {
			locals[i] = NilValue()
		}
	}

	// Copy captured variables into their assigned local slots
	for i := range fn.CaptureIdx {
		if i < len(captures) {
			idx := fn.CaptureIdx[i]
			if idx >= 0 && idx < len(locals) {
				locals[idx] = captures[i]
			}
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

	if result.Type == ValFunc && result.Func == nil {
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
		return strconv.FormatInt(v.Int, 10)
	case ValFloat:
		return strconv.FormatFloat(v.Float, 'g', -1, 64)
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
