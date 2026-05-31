package vm

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/dev-dami/carv/pkg/ir"
	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/parser"
	"github.com/dev-dami/carv/pkg/types"
)

func benchVM(b *testing.B, source string) {
	b.Helper()
	l := lexer.New(source)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		b.Fatalf("parse errors: %v", p.Errors())
	}
	checker := types.NewChecker()
	if !checker.Check(prog) {
		b.Fatalf("type errors: %v", checker.Errors())
	}
	lowerer := ir.NewLowerer(checker.TypeInfo())
	mod := lowerer.Lower(prog)
	v := New(mod)
	var buf bytes.Buffer
	v.SetOutput(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm2 := New(mod)
		vm2.SetOutput(&buf)
		_, err := vm2.Run()
		if err != nil {
			b.Fatalf("vm error: %v", err)
		}
	}
}

func BenchmarkEmpty(b *testing.B) {
	benchVM(b, `fn main() {}`)
}

func BenchmarkIntArithmetic(b *testing.B) {
	benchVM(b, `
fn main() {
    mut sum = 0;
    mut i = 0;
    while i < 100000 {
        sum = sum + i;
        i = i + 1;
    }
    println(sum);
}
`)
}

func BenchmarkFloatArithmetic(b *testing.B) {
	benchVM(b, `
fn main() {
    mut sum = 0.0;
    mut i = 0;
    while i < 100000 {
        sum = sum + 1.5;
        i = i + 1;
    }
    println(sum);
}
`)
}

func BenchmarkFuncCall(b *testing.B) {
	benchVM(b, `
fn add(a: int, b: int) -> int {
    return a + b;
}
fn main() {
    mut result = 0;
    mut i = 0;
    while i < 100000 {
        result = add(result, i);
        i = i + 1;
    }
    println(result);
}
`)
}

func BenchmarkStringConcat(b *testing.B) {
	benchVM(b, `
fn main() {
    mut s = "";
    mut i = 0;
    while i < 1000 {
        s = s + "x";
        i = i + 1;
    }
    println(len(s));
}
`)
}

func BenchmarkArrayOps(b *testing.B) {
	benchVM(b, `
fn main() {
    mut arr = [];
    mut i = 0;
    while i < 10000 {
        arr = push(arr, i);
        i = i + 1;
    }
    println(len(arr));
}
`)
}

func BenchmarkMapOps(b *testing.B) {
	benchVM(b, `
fn main() {
    mut m = {};
    mut i = 0;
    while i < 5000 {
        set(m, fmt(i), i);
        i = i + 1;
    }
    println(len(keys(m)));
}
`)
}

func BenchmarkMixed(b *testing.B) {
	benchVM(b, `
fn fib(n: int) -> int {
    if n < 2 {
        return n;
    }
    return fib(n - 1) + fib(n - 2);
}
fn main() {
    println(fib(20));
}
`)
}

// Run the benchmark suite and print detailed results
func BenchmarkAll(b *testing.B) {
	sources := []struct {
		name string
		code string
	}{
		{"Empty", `fn main() {}`},
		{"IntLoop_100k", `
fn main() {
    mut sum = 0;
    mut i = 0;
    while i < 100000 {
        sum = sum + i;
        i = i + 1;
    }
    println(sum);
}
`},
		{"FloatLoop_100k", `
fn main() {
    mut sum = 0.0;
    mut i = 0;
    while i < 100000 {
        sum = sum + 1.5;
        i = i + 1;
    }
    println(sum);
}
`},
		{"FnCall_100k", `
fn add(a: int, b: int) -> int {
    return a + b;
}
fn main() {
    mut result = 0;
    mut i = 0;
    while i < 100000 {
        result = add(result, i);
        i = i + 1;
    }
    println(result);
}
`},
		{"StringConcat_1k", `
fn main() {
    mut s = "";
    mut i = 0;
    while i < 1000 {
        s = s + "x";
        i = i + 1;
    }
    println(len(s));
}
`},
		{"ArrayPush_10k", `
fn main() {
    mut arr = [];
    mut i = 0;
    while i < 10000 {
        arr = push(arr, i);
        i = i + 1;
    }
    println(len(arr));
}
`},
		{"MapSet_5k", `
fn main() {
    mut m = {};
    mut i = 0;
    while i < 5000 {
        m = set(m, string(i), i);
        i = i + 1;
    }
    println(len(keys(m)));
}
`},
		{"Fib_20", `
fn fib(n: int) -> int {
    if n < 2 {
        return n;
    }
    return fib(n - 1) + fib(n - 2);
}
fn main() {
    println(fib(20));
}
`},
	}

	for _, s := range sources {
		b.Run(s.name, func(b *testing.B) {
			benchVM(b, s.code)
		})
	}
}

// Print per-op throughput estimation
func BenchmarkThroughput(b *testing.B) {
	b.Skip("manual throughput measurement")

	source := `
fn main() {
    mut sum = 0;
    mut i = 0;
    while i < 1000000 {
        sum = sum + i;
        i = i + 1;
    }
    println(sum);
}
`
	l := lexer.New(source)
	p := parser.New(l)
	prog := p.ParseProgram()
	checker := types.NewChecker()
	checker.Check(prog)
	lowerer := ir.NewLowerer(checker.TypeInfo())
	mod := lowerer.Lower(prog)

	fn := mod.Functions["main"]
	totalInsts := len(fn.Body)

	v := New(mod)
	var buf bytes.Buffer
	v.SetOutput(&buf)

	const runs = 50
	var elapsed int64
	for i := 0; i < runs; i++ {
		vm2 := New(mod)
		vm2.SetOutput(&buf)
		if _, err := vm2.Run(); err != nil {
			b.Fatalf("run failed: %v", err)
		}
	}

	_ = totalInsts
	_ = elapsed
	fmt.Printf("Function body has %d instructions\n", totalInsts)
}
