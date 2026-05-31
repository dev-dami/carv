package vm

import (
	"os"
	"runtime/pprof"
	"testing"

	"github.com/dev-dami/carv/pkg/ir"
	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/parser"
	"github.com/dev-dami/carv/pkg/types"
)

func TestProfileArrayPush(t *testing.T) {
	source := `
fn main() {
    mut arr = [];
    mut i = 0;
    while i < 5000 {
        arr = push(arr, i);
        i = i + 1;
    }
    println(len(arr));
}
`
	f, err := os.Create("/tmp/cpu.pprof")
	if err != nil {
		t.Fatalf("create profile file: %v", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		_ = f.Close()
		t.Fatalf("start cpu profile: %v", err)
	}
	
	for run := 0; run < 20; run++ {
		l := lexer.New(source)
		p := parser.New(l)
		prog := p.ParseProgram()
		checker := types.NewChecker()
		checker.Check(prog)
		lowerer := ir.NewLowerer(checker.TypeInfo())
		mod := lowerer.Lower(prog)
		v := New(mod)
		if _, err := v.Run(); err != nil {
			pprof.StopCPUProfile()
			_ = f.Close()
			t.Fatalf("run failed: %v", err)
		}
	}

	pprof.StopCPUProfile()
	if err := f.Close(); err != nil {
		t.Fatalf("close profile file: %v", err)
	}
}
