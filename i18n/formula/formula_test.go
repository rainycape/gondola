package formula

import (
	"fmt"
	"testing"
)

type Test struct {
	Expr  string
	N     int
	Tests map[int]int
}

var formulas = []*Test{
	// Asian
	{"0", 1, map[int]int{0: 0, 1: 0, 2: 0, 100: 0}},
	// Romanic
	{"n != 1", 2, map[int]int{0: 1, 1: 0, 2: 1, 100: 1}},
	// Brazilian Portuguese
	{"n > 1", 2, map[int]int{0: 0, 1: 0, 2: 1, 100: 1}},
	// Latvian
	{"n%10==1 && n%100!=11 ? 0 : n != 0 ? 1 : 2", 3, map[int]int{1: 0, 21: 0, 11: 1, 0: 2}},
	// Irish
	{"n==1 ? 0 : n==2 ? 1 : 2", 3, map[int]int{1: 0, 2: 1, 3: 2, 4: 2, 300: 2}},
	// Romanian
	{"n==1 ? 0 : (n==0 || (n%100 > 0 && n%100 < 20)) ? 1 : 2", 3, map[int]int{1: 0, 0: 1, 10: 1, 20: 2, 30: 2}},
	// Lithuanian
	{"n%10==1 && n%100!=11 ? 0 : n%10>=2 && (n%100<10 || n%100>=20) ? 1 : 2", 3, map[int]int{0: 2, 1: 0, 2: 1, 3: 1, 11: 2, 12: 2, 15: 2, 22: 1}},
	// Russian
	{"n%10==1 && n%100!=11 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2", 3, map[int]int{1: 0, 11: 2, 14: 2, 34: 1}},
	// Czech
	{"(n==1) ? 0 : (n>=2 && n<=4) ? 1 : 2", 3, map[int]int{1: 0, 2: 1, 3: 1, 4: 1, 10: 2, 5: 2}},
	// Polish
	{"n==1 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2", 4, map[int]int{1: 0, 0: 2, 5: 2, 8: 2, 12: 2, 15: 2, 22: 1, 103: 1}},
	// Slovenian
	{"n%100==1 ? 0 : n%100==2 ? 1 : n%100==3 || n%100==4 ? 2 : 3", 4, map[int]int{0: 3, 1: 0, 101: 0, 2: 1, 3: 2, 4: 2, 204: 2}},
	// Some made up formulas
	{"n + 7 < 10 ? 1 : 0", 2, map[int]int{0: 1, 2: 1, 3: 0}},
	{"n - 5 > 0", 2, map[int]int{0: 0, 5: 0, 6: 1}},
	{"n / 10 > 0 ? 1 : 0", 2, map[int]int{0: 0, 9: 0, 10: 1}},
	{"n * 3 >= 9", 2, map[int]int{0: 0, 2: 0, 3: 1, 4: 1}},
}

func compile(code []byte, optimize bool, jit bool) (Formula, error) {
	p, err := vmCompile(code)
	if err != nil {
		return nil, err
	}
	if optimize {
		p = vmOptimize(p)
	}
	if jit {
		return vmJit(p)
	}
	return makeVmFunc(p), nil
}

func compiler(optimize bool, jit bool) func([]byte) (Formula, error) {
	return func(code []byte) (Formula, error) {
		return compile(code, optimize, jit)
	}
}

func testCompile(t *testing.T, comp func([]byte) (Formula, error)) {
	for _, v := range formulas {
		t.Logf("Compiling formula %q", v.Expr)
		fn, err := comp([]byte(v.Expr))
		if err != nil {
			t.Error(err)
			continue
		}
		for k, val := range v.Tests {
			x := fn(k)
			t.Logf("(%s)(%d) = %d", v.Expr, k, x)
			if x != val {
				t.Errorf("Bad result for %q with value %d. Want %d, got %d.", v.Expr, k, val, x)
			}
		}
	}
}

func TestVm(t *testing.T) {
	testCompile(t, compiler(false, false))
}

func TestVmOptimized(t *testing.T) {
	testCompile(t, compiler(true, false))
}

func TestJit(t *testing.T) {
	if _, err := vmJit(nil); err != errJitNotSupported {
		testCompile(t, compiler(true, true))
	}
}

func TestBytecode(t *testing.T) {
	for _, v := range formulas {
		t.Logf("Compiling formula %q", v.Expr)
		code, err := vmCompile([]byte(v.Expr))
		if err != nil {
			t.Error(err)
			continue
		}
		t.Log("Bytecode")
		for ii, i := range code {
			t.Logf("%d:%s\t%d", ii, i.opCode.String(), i.value)
		}
		optimized := vmOptimize(code)
		t.Log("Optimized")
		for ii, i := range optimized {
			t.Logf("%d:%s\t%d", ii, i.opCode.String(), i.value)
		}
	}
}

func benchmarkCompile(b *testing.B, fn func([]byte) (Formula, error)) {
	for ii := 0; ii < b.N; ii++ {
		for _, v := range formulas {
			_, err := fn([]byte(v.Expr))
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkCompileVm(b *testing.B) {
	benchmarkCompile(b, compileVmFormula)
}

func BenchmarkFindCompiled(b *testing.B) {
	forms := make([]string, len(formulas))
	for ii, v := range formulas {
		forms[ii] = fmt.Sprintf("nplurals=%d; plural=(%s);", v.N, v.Expr)
	}
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		for jj := range formulas {
			_, _, err := Make(forms[jj])
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func benchmarkFormulas(b *testing.B, fns []Formula) {
	b.ResetTimer()
	for ii := 0; ii < b.N; ii++ {
		for jj, v := range formulas {
			fn := fns[jj]
			for k, val := range v.Tests {
				x := fn(k)
				if x != val {
					b.Errorf("Bad result for %q with value %d. Want %d, got %d.", v.Expr, k, val, x)
				}
			}
		}
	}
}

func compileAndBenchmark(b *testing.B, fn func([]byte) (Formula, error)) {
	fns := make([]Formula, len(formulas))
	var err error
	for ii, v := range formulas {
		fns[ii], err = fn([]byte(v.Expr))
		if err != nil {
			b.Fatal(err)
		}
	}
	benchmarkFormulas(b, fns)
}

func BenchmarkVmInterpreted(b *testing.B) {
	compileAndBenchmark(b, compiler(false, false))
}

func BenchmarkVmInterpretedOptimized(b *testing.B) {
	compileAndBenchmark(b, compiler(true, false))
}

func BenchmarkJit(b *testing.B) {
	compileAndBenchmark(b, compiler(true, true))
}

func BenchmarkCompiled(b *testing.B) {
	fns := make([]Formula, len(formulas))
	var err error
	for ii, v := range formulas {
		text := fmt.Sprintf("nplurals=%d; plural=(%s);", v.N, v.Expr)
		fns[ii], _, err = Make(text)
		if err != nil {
			b.Fatal(err)
		}
	}
	benchmarkFormulas(b, fns)
}
