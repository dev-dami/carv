package vm

import (
	"testing"
	"os"
	"runtime/pprof"
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
	f, _ := os.Create("/tmp/cpu.pprof")
	pprof.StartCPUProfile(f)
	
	for run := 0; run < 20; run++ {
		l := lexer.New(source)
		p := parser.New(l)
		prog := p.ParseProgram()
		checker := types.NewChecker()
		checker.Check(prog)
		lowerer := ir.NewLowerer(checker.TypeInfo())
		mod := lowerer.Lower(prog)
		v := New(mod)
		v.Run()
	}
	
	pprof.StopCPUProfile()
	f.Close()
}
