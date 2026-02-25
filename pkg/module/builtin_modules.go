package module

var builtinModuleExports = map[string][]string{
	"net": {
		"tcp_listen",
		"tcp_accept",
		"tcp_read",
		"tcp_write",
		"tcp_close",
	},
	"web": {
		"tcp_listen",
		"tcp_accept",
		"tcp_read",
		"tcp_write",
		"tcp_close",
	},
}

func IsBuiltinModule(name string) bool {
	_, ok := builtinModuleExports[name]
	return ok
}

func BuiltinModuleExports(name string) (map[string]bool, bool) {
	names, ok := builtinModuleExports[name]
	if !ok {
		return nil, false
	}

	exports := make(map[string]bool, len(names))
	for _, n := range names {
		exports[n] = true
	}
	return exports, true
}
