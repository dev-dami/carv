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
	"gpio": {
		"pin_mode", "digital_write", "digital_read", "analog_read", "analog_write",
	},
	"uart": {
		"uart_init", "uart_write", "uart_read", "uart_available",
	},
	"spi": {
		"spi_init", "spi_transfer", "spi_write", "spi_read",
	},
	"i2c": {
		"i2c_init", "i2c_write", "i2c_read",
	},
	"timer": {
		"timer_init", "timer_start", "timer_stop", "timer_get_count", "delay_ms", "delay_us",
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
