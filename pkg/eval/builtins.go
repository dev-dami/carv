package eval

import (
	"fmt"
	"os"
	"strings"
)

var builtins = map[string]*Builtin{
	"print": {
		Fn: func(args ...Object) Object {
			var out []string
			for _, arg := range args {
				out = append(out, arg.Inspect())
			}
			fmt.Println(strings.Join(out, " "))
			return NIL
		},
	},
	"println": {
		Fn: func(args ...Object) Object {
			var out []string
			for _, arg := range args {
				out = append(out, arg.Inspect())
			}
			fmt.Println(strings.Join(out, " "))
			return NIL
		},
	},
	"len": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("len() takes exactly 1 argument")
			}
			switch arg := args[0].(type) {
			case *String:
				return &Integer{Value: int64(len(arg.Value))}
			case *Array:
				return &Integer{Value: int64(len(arg.Elements))}
			default:
				return newError("len() not supported for %s", arg.Type())
			}
		},
	},
	"str": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("str() takes exactly 1 argument")
			}
			return &String{Value: args[0].Inspect()}
		},
	},
	"int": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("int() takes exactly 1 argument")
			}
			switch arg := args[0].(type) {
			case *Integer:
				return arg
			case *Float:
				return &Integer{Value: int64(arg.Value)}
			case *Boolean:
				if arg.Value {
					return &Integer{Value: 1}
				}
				return &Integer{Value: 0}
			default:
				return newError("cannot convert %s to int", arg.Type())
			}
		},
	},
	"float": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("float() takes exactly 1 argument")
			}
			switch arg := args[0].(type) {
			case *Float:
				return arg
			case *Integer:
				return &Float{Value: float64(arg.Value)}
			default:
				return newError("cannot convert %s to float", arg.Type())
			}
		},
	},
	"push": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("push() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return newError("first argument to push must be array")
			}
			newElements := make([]Object, len(arr.Elements)+1)
			copy(newElements, arr.Elements)
			newElements[len(arr.Elements)] = args[1]
			return &Array{Elements: newElements}
		},
	},
	"head": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("head() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return newError("argument to head must be array")
			}
			if len(arr.Elements) == 0 {
				return NIL
			}
			return arr.Elements[0]
		},
	},
	"tail": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("tail() takes exactly 1 argument")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return newError("argument to tail must be array")
			}
			if len(arr.Elements) == 0 {
				return &Array{Elements: []Object{}}
			}
			newElements := make([]Object, len(arr.Elements)-1)
			copy(newElements, arr.Elements[1:])
			return &Array{Elements: newElements}
		},
	},
	"read_file": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("read_file() takes exactly 1 argument")
			}
			path, ok := args[0].(*String)
			if !ok {
				return newError("read_file() argument must be a string")
			}
			content, err := os.ReadFile(path.Value)
			if err != nil {
				return newError("read_file: %s", err.Error())
			}
			return &String{Value: string(content)}
		},
	},
	"write_file": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("write_file() takes exactly 2 arguments")
			}
			path, ok := args[0].(*String)
			if !ok {
				return newError("write_file() first argument must be a string")
			}
			content, ok := args[1].(*String)
			if !ok {
				return newError("write_file() second argument must be a string")
			}
			err := os.WriteFile(path.Value, []byte(content.Value), 0644)
			if err != nil {
				return newError("write_file: %s", err.Error())
			}
			return NIL
		},
	},
	"file_exists": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("file_exists() takes exactly 1 argument")
			}
			path, ok := args[0].(*String)
			if !ok {
				return newError("file_exists() argument must be a string")
			}
			_, err := os.Stat(path.Value)
			if err != nil {
				return FALSE
			}
			return TRUE
		},
	},
	"split": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("split() takes exactly 2 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("split() first argument must be a string")
			}
			sep, ok := args[1].(*String)
			if !ok {
				return newError("split() second argument must be a string")
			}
			parts := strings.Split(str.Value, sep.Value)
			elements := make([]Object, len(parts))
			for i, p := range parts {
				elements[i] = &String{Value: p}
			}
			return &Array{Elements: elements}
		},
	},
	"join": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("join() takes exactly 2 arguments")
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return newError("join() first argument must be an array")
			}
			sep, ok := args[1].(*String)
			if !ok {
				return newError("join() second argument must be a string")
			}
			parts := make([]string, len(arr.Elements))
			for i, elem := range arr.Elements {
				parts[i] = elem.Inspect()
			}
			return &String{Value: strings.Join(parts, sep.Value)}
		},
	},
	"trim": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("trim() takes exactly 1 argument")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("trim() argument must be a string")
			}
			return &String{Value: strings.TrimSpace(str.Value)}
		},
	},
	"substr": {
		Fn: func(args ...Object) Object {
			if len(args) < 2 || len(args) > 3 {
				return newError("substr() takes 2 or 3 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("substr() first argument must be a string")
			}
			start, ok := args[1].(*Integer)
			if !ok {
				return newError("substr() second argument must be an integer")
			}
			startIdx := int(start.Value)
			if startIdx < 0 {
				startIdx = 0
			}
			if startIdx >= len(str.Value) {
				return &String{Value: ""}
			}
			if len(args) == 2 {
				return &String{Value: str.Value[startIdx:]}
			}
			end, ok := args[2].(*Integer)
			if !ok {
				return newError("substr() third argument must be an integer")
			}
			endIdx := int(end.Value)
			if endIdx > len(str.Value) {
				endIdx = len(str.Value)
			}
			if endIdx <= startIdx {
				return &String{Value: ""}
			}
			return &String{Value: str.Value[startIdx:endIdx]}
		},
	},
}

func newError(format string, a ...interface{}) *Error {
	return &Error{Message: fmt.Sprintf(format, a...)}
}
