//go:build gcc

package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/parser"
	"github.com/dev-dami/carv/pkg/types"
)

// compileGeneratedC verifies that the generated C source compiles cleanly with gcc.
// Only built when the "gcc" build tag is provided: go test -tags=gcc ./...
func compileGeneratedC(t *testing.T, source string) {
	t.Helper()

	tmpDir := t.TempDir()
	cFile := filepath.Join(tmpDir, "out.c")
	outBin := filepath.Join(tmpDir, "out")

	if err := os.WriteFile(cFile, []byte(source), 0o644); err != nil {
		t.Fatalf("failed to write generated C file: %v", err)
	}

	cmd := exec.Command("gcc", "-O2", "-o", outBin, cFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gcc failed to compile emitted C: %v\n%s", err, string(output))
	}
}

func TestAsyncGeneratedCCompiles(t *testing.T) {
	output := generateOutputFromSource(t, `
async fn fetch() -> int {
	return 1;
}

async fn carv_main() -> int {
	let x = await fetch();
	return x;
}
`)

	compileGeneratedC(t, output)
}

func TestTCPBuiltinsGeneratedCCompiles(t *testing.T) {
	output := generateOutputFromSource(t, `
fn main() {
	let listener = tcp_listen("127.0.0.1", 8080);
	tcp_close(listener);
}
`)

	compileGeneratedC(t, output)
}

func TestMapLiteralGeneratedCCompiles(t *testing.T) {
	output := generateOutputFromSource(t, `
let scores = {"alice": 95, "bob": 87};
println(scores);
`)
	compileGeneratedC(t, output)
}

func TestResultFunctionGeneratedCCompiles(t *testing.T) {
	output := generateOutputFromSource(t, `
fn divide(a: int, b: int) {
    if b == 0 {
        return Err("division by zero");
    }
    return Ok(a / b);
}

let result = divide(10, 2);
match result {
    Ok(v) => println(v),
    Err(e) => println(e),
};
`)
	compileGeneratedC(t, output)
}

func TestHALModulesGeneratedCCompiles(t *testing.T) {
	input := `
require "gpio" as gpio;
require "uart" as uart;
require "spi" as spi;
require "i2c" as i2c;
require "timer" as timer;
fn main() {
	gpio.pin_mode(13, 1);
	gpio.digital_write(13, true);
	let v = gpio.digital_read(13);
	let a = gpio.analog_read(0);
	gpio.analog_write(9, 128);
	println(v);
	println(a);

	let uh = uart.uart_init(1, 9600);
	let wrote = uart.uart_write(uh, "hello");
	let data = uart.uart_read(uh, 64);
	let avail = uart.uart_available(uh);
	println(wrote);
	println(avail);

	let sh = spi.spi_init(0, 1000000);
	let resp = spi.spi_transfer(sh, "ab");
	let sw = spi.spi_write(sh, "cd");
	let sr = spi.spi_read(sh, 4);
	println(resp);
	println(sw);
	println(sr);

	let ih = i2c.i2c_init(1, 80);
	let iw = i2c.i2c_write(ih, "ef");
	let ir = i2c.i2c_read(ih, 2);
	println(iw);
	println(ir);

	let th = timer.timer_init(0, 72);
	timer.timer_start(th);
	timer.timer_stop(th);
	let cnt = timer.timer_get_count(th);
	timer.delay_ms(100);
	timer.delay_us(50);
	println(cnt);
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	checker := types.NewChecker()
	if !checker.Check(program) {
		t.Fatalf("type errors: %v", checker.Errors())
	}

	gen := NewCGenerator()
	gen.SetTypeInfo(checker.TypeInfo())
	output := gen.Generate(program)

	compileGeneratedC(t, output)
}
