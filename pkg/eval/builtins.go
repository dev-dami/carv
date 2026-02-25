package eval

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

var cliArgs []string
var tcpMu sync.Mutex
var tcpNextHandle int64 = 1
var tcpListeners = map[int64]net.Listener{}
var tcpConns = map[int64]net.Conn{}

func SetArgs(args []string) {
	cliArgs = args
}

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
	"parse_int": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("parse_int() takes exactly 1 argument")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("parse_int() argument must be a string")
			}
			val, err := strconv.ParseInt(str.Value, 10, 64)
			if err != nil {
				return newError("cannot parse '%s' as int", str.Value)
			}
			return &Integer{Value: val}
		},
	},
	"parse_float": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("parse_float() takes exactly 1 argument")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("parse_float() argument must be a string")
			}
			val, err := strconv.ParseFloat(str.Value, 64)
			if err != nil {
				return newError("cannot parse '%s' as float", str.Value)
			}
			return &Float{Value: val}
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
			err := os.WriteFile(path.Value, []byte(content.Value), 0o644)
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
	"ord": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("ord() takes exactly 1 argument")
			}
			switch arg := args[0].(type) {
			case *Char:
				return &Integer{Value: int64(arg.Value)}
			case *String:
				if len(arg.Value) == 0 {
					return newError("ord() requires non-empty string")
				}
				return &Integer{Value: int64(arg.Value[0])}
			default:
				return newError("ord() requires char or string, got %s", arg.Type())
			}
		},
	},
	"chr": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("chr() takes exactly 1 argument")
			}
			i, ok := args[0].(*Integer)
			if !ok {
				return newError("chr() requires integer, got %s", args[0].Type())
			}
			return &Char{Value: rune(i.Value)}
		},
	},
	"char_at": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("char_at() takes exactly 2 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("char_at() first argument must be string")
			}
			idx, ok := args[1].(*Integer)
			if !ok {
				return newError("char_at() second argument must be integer")
			}
			i := int(idx.Value)
			if i < 0 || i >= len(str.Value) {
				return NIL
			}
			return &Char{Value: rune(str.Value[i])}
		},
	},
	"contains": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("contains() takes exactly 2 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("contains() first argument must be string")
			}
			sub, ok := args[1].(*String)
			if !ok {
				return newError("contains() second argument must be string")
			}
			if strings.Contains(str.Value, sub.Value) {
				return TRUE
			}
			return FALSE
		},
	},
	"starts_with": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("starts_with() takes exactly 2 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("starts_with() first argument must be string")
			}
			prefix, ok := args[1].(*String)
			if !ok {
				return newError("starts_with() second argument must be string")
			}
			if strings.HasPrefix(str.Value, prefix.Value) {
				return TRUE
			}
			return FALSE
		},
	},
	"ends_with": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("ends_with() takes exactly 2 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("ends_with() first argument must be string")
			}
			suffix, ok := args[1].(*String)
			if !ok {
				return newError("ends_with() second argument must be string")
			}
			if strings.HasSuffix(str.Value, suffix.Value) {
				return TRUE
			}
			return FALSE
		},
	},
	"replace": {
		Fn: func(args ...Object) Object {
			if len(args) != 3 {
				return newError("replace() takes exactly 3 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("replace() first argument must be string")
			}
			old, ok := args[1].(*String)
			if !ok {
				return newError("replace() second argument must be string")
			}
			new, ok := args[2].(*String)
			if !ok {
				return newError("replace() third argument must be string")
			}
			return &String{Value: strings.ReplaceAll(str.Value, old.Value, new.Value)}
		},
	},
	"index_of": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("index_of() takes exactly 2 arguments")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("index_of() first argument must be string")
			}
			sub, ok := args[1].(*String)
			if !ok {
				return newError("index_of() second argument must be string")
			}
			return &Integer{Value: int64(strings.Index(str.Value, sub.Value))}
		},
	},
	"to_upper": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("to_upper() takes exactly 1 argument")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("to_upper() argument must be string")
			}
			return &String{Value: strings.ToUpper(str.Value)}
		},
	},
	"to_lower": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("to_lower() takes exactly 1 argument")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("to_lower() argument must be string")
			}
			return &String{Value: strings.ToLower(str.Value)}
		},
	},
	"exit": {
		Fn: func(args ...Object) Object {
			code := 0
			if len(args) > 0 {
				if i, ok := args[0].(*Integer); ok {
					code = int(i.Value)
				}
			}
			os.Exit(code)
			return NIL
		},
	},
	"panic": {
		Fn: func(args ...Object) Object {
			msg := "panic"
			if len(args) > 0 {
				msg = args[0].Inspect()
			}
			return newError("panic: %s", msg)
		},
	},
	"type_of": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("type_of() takes exactly 1 argument")
			}
			return &String{Value: string(args[0].Type())}
		},
	},
	"keys": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("keys() takes exactly 1 argument")
			}
			m, ok := args[0].(*Map)
			if !ok {
				return newError("keys() requires a map, got %s", args[0].Type())
			}
			keys := make([]Object, 0, len(m.Pairs))
			for _, pair := range m.Pairs {
				keys = append(keys, pair.Key)
			}
			return &Array{Elements: keys}
		},
	},
	"values": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("values() takes exactly 1 argument")
			}
			m, ok := args[0].(*Map)
			if !ok {
				return newError("values() requires a map, got %s", args[0].Type())
			}
			values := make([]Object, 0, len(m.Pairs))
			for _, pair := range m.Pairs {
				values = append(values, pair.Value)
			}
			return &Array{Elements: values}
		},
	},
	"has_key": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("has_key() takes exactly 2 arguments")
			}
			m, ok := args[0].(*Map)
			if !ok {
				return newError("has_key() first argument must be a map")
			}
			key, ok := args[1].(Hashable)
			if !ok {
				return newError("has_key() second argument must be hashable")
			}
			_, exists := m.Pairs[key.HashKey()]
			if exists {
				return TRUE
			}
			return FALSE
		},
	},
	"delete": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("delete() takes exactly 2 arguments")
			}
			m, ok := args[0].(*Map)
			if !ok {
				return newError("delete() first argument must be a map")
			}
			key, ok := args[1].(Hashable)
			if !ok {
				return newError("delete() second argument must be hashable")
			}
			newPairs := make(map[HashKey]MapPair)
			for k, v := range m.Pairs {
				if k != key.HashKey() {
					newPairs[k] = v
				}
			}
			return &Map{Pairs: newPairs}
		},
	},
	"set": {
		Fn: func(args ...Object) Object {
			if len(args) != 3 {
				return newError("set() takes exactly 3 arguments")
			}
			m, ok := args[0].(*Map)
			if !ok {
				return newError("set() first argument must be a map")
			}
			key, ok := args[1].(Hashable)
			if !ok {
				return newError("set() second argument must be hashable")
			}
			newPairs := make(map[HashKey]MapPair)
			for k, v := range m.Pairs {
				newPairs[k] = v
			}
			newPairs[key.HashKey()] = MapPair{Key: args[1], Value: args[2]}
			return &Map{Pairs: newPairs}
		},
	},
	"args": {
		Fn: func(args ...Object) Object {
			if len(args) != 0 {
				return newError("args() takes no arguments")
			}
			elements := make([]Object, len(cliArgs))
			for i, arg := range cliArgs {
				elements[i] = &String{Value: arg}
			}
			return &Array{Elements: elements}
		},
	},
	"tcp_listen": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("tcp_listen() takes exactly 2 arguments")
			}
			host, ok := args[0].(*String)
			if !ok {
				return newError("tcp_listen() first argument must be a string")
			}
			port, ok := args[1].(*Integer)
			if !ok {
				return newError("tcp_listen() second argument must be an integer")
			}

			addr := fmt.Sprintf("%s:%d", host.Value, port.Value)
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return newError("tcp_listen: %s", err.Error())
			}

			handle := tcpAllocHandle()
			tcpMu.Lock()
			tcpListeners[handle] = ln
			tcpMu.Unlock()
			return &Integer{Value: handle}
		},
	},
	"tcp_accept": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("tcp_accept() takes exactly 1 argument")
			}
			listenerHandle, ok := args[0].(*Integer)
			if !ok {
				return newError("tcp_accept() argument must be an integer listener handle")
			}

			tcpMu.Lock()
			ln, exists := tcpListeners[listenerHandle.Value]
			tcpMu.Unlock()
			if !exists {
				return newError("tcp_accept: invalid listener handle %d", listenerHandle.Value)
			}

			conn, err := ln.Accept()
			if err != nil {
				return newError("tcp_accept: %s", err.Error())
			}

			connHandle := tcpAllocHandle()
			tcpMu.Lock()
			tcpConns[connHandle] = conn
			tcpMu.Unlock()
			return &Integer{Value: connHandle}
		},
	},
	"tcp_read": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("tcp_read() takes exactly 2 arguments")
			}
			connHandle, ok := args[0].(*Integer)
			if !ok {
				return newError("tcp_read() first argument must be an integer connection handle")
			}
			maxBytes, ok := args[1].(*Integer)
			if !ok {
				return newError("tcp_read() second argument must be an integer")
			}
			if maxBytes.Value <= 0 {
				return &String{Value: ""}
			}

			tcpMu.Lock()
			conn, exists := tcpConns[connHandle.Value]
			tcpMu.Unlock()
			if !exists {
				return newError("tcp_read: invalid connection handle %d", connHandle.Value)
			}

			buf := make([]byte, maxBytes.Value)
			n, err := conn.Read(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return &String{Value: ""}
				}
				return newError("tcp_read: %s", err.Error())
			}
			return &String{Value: string(buf[:n])}
		},
	},
	"tcp_write": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("tcp_write() takes exactly 2 arguments")
			}
			connHandle, ok := args[0].(*Integer)
			if !ok {
				return newError("tcp_write() first argument must be an integer connection handle")
			}
			data, ok := args[1].(*String)
			if !ok {
				return newError("tcp_write() second argument must be a string")
			}

			tcpMu.Lock()
			conn, exists := tcpConns[connHandle.Value]
			tcpMu.Unlock()
			if !exists {
				return newError("tcp_write: invalid connection handle %d", connHandle.Value)
			}

			n, err := conn.Write([]byte(data.Value))
			if err != nil {
				return newError("tcp_write: %s", err.Error())
			}
			return &Integer{Value: int64(n)}
		},
	},
	"tcp_close": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("tcp_close() takes exactly 1 argument")
			}
			handle, ok := args[0].(*Integer)
			if !ok {
				return newError("tcp_close() argument must be an integer handle")
			}
			if err := tcpCloseHandle(handle.Value); err != nil {
				return newError("tcp_close: %s", err.Error())
			}
			return TRUE
		},
	},
	"exec": {
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return newError("exec() requires at least 1 argument")
			}
			cmdStr, ok := args[0].(*String)
			if !ok {
				return newError("exec() first argument must be a string")
			}
			cmdArgs := make([]string, len(args)-1)
			for i := 1; i < len(args); i++ {
				arg, ok := args[i].(*String)
				if !ok {
					return newError("exec() arguments must be strings")
				}
				cmdArgs[i-1] = arg.Value
			}
			cmd := exec.Command(cmdStr.Value, cmdArgs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					return &Integer{Value: int64(exitErr.ExitCode())}
				}
				return newError("exec: %s", err.Error())
			}
			return &Integer{Value: 0}
		},
	},
	"exec_output": {
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return newError("exec_output() requires at least 1 argument")
			}
			cmdStr, ok := args[0].(*String)
			if !ok {
				return newError("exec_output() first argument must be a string")
			}
			cmdArgs := make([]string, len(args)-1)
			for i := 1; i < len(args); i++ {
				arg, ok := args[i].(*String)
				if !ok {
					return newError("exec_output() arguments must be strings")
				}
				cmdArgs[i-1] = arg.Value
			}
			cmd := exec.Command(cmdStr.Value, cmdArgs...)
			output, err := cmd.Output()
			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					return &Err{Value: &String{Value: string(exitErr.Stderr)}}
				}
				return &Err{Value: &String{Value: err.Error()}}
			}
			return &Ok{Value: &String{Value: string(output)}}
		},
	},
	"mkdir": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("mkdir() takes exactly 1 argument")
			}
			path, ok := args[0].(*String)
			if !ok {
				return newError("mkdir() argument must be a string")
			}
			err := os.MkdirAll(path.Value, 0o755)
			if err != nil {
				return newError("mkdir: %s", err.Error())
			}
			return NIL
		},
	},
	"append_file": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("append_file() takes exactly 2 arguments")
			}
			path, ok := args[0].(*String)
			if !ok {
				return newError("append_file() first argument must be a string")
			}
			content, ok := args[1].(*String)
			if !ok {
				return newError("append_file() second argument must be a string")
			}
			f, err := os.OpenFile(path.Value, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				return newError("append_file: %s", err.Error())
			}
			defer f.Close()
			_, err = f.WriteString(content.Value)
			if err != nil {
				return newError("append_file: %s", err.Error())
			}
			return NIL
		},
	},
	"getenv": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("getenv() takes exactly 1 argument")
			}
			key, ok := args[0].(*String)
			if !ok {
				return newError("getenv() argument must be a string")
			}
			return &String{Value: os.Getenv(key.Value)}
		},
	},
	"setenv": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("setenv() takes exactly 2 arguments")
			}
			key, ok := args[0].(*String)
			if !ok {
				return newError("setenv() first argument must be a string")
			}
			val, ok := args[1].(*String)
			if !ok {
				return newError("setenv() second argument must be a string")
			}
			err := os.Setenv(key.Value, val.Value)
			if err != nil {
				return newError("setenv: %s", err.Error())
			}
			return NIL
		},
	},
}

func newError(format string, a ...interface{}) *Error {
	return &Error{Message: fmt.Sprintf(format, a...)}
}

func tcpAllocHandle() int64 {
	tcpMu.Lock()
	defer tcpMu.Unlock()
	handle := tcpNextHandle
	tcpNextHandle++
	return handle
}

func tcpCloseHandle(handle int64) error {
	tcpMu.Lock()
	defer tcpMu.Unlock()

	if conn, ok := tcpConns[handle]; ok {
		delete(tcpConns, handle)
		return conn.Close()
	}
	if ln, ok := tcpListeners[handle]; ok {
		delete(tcpListeners, handle)
		return ln.Close()
	}
	return fmt.Errorf("invalid handle %d", handle)
}
