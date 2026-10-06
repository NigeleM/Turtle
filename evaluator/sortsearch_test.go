package evaluator

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Turtle/object"
)

func intList(ns []int64) *object.List {
	l := &object.List{}
	for _, n := range ns {
		l.Elements = append(l.Elements, &object.Integer{Value: n})
	}
	return l
}

// pairList is list [list [key, original position], ...], so a sort by
// key (position 0) shows whether equal keys kept their order.
func pairList(ns []int64) *object.List {
	l := &object.List{}
	for i, n := range ns {
		l.Elements = append(l.Elements, intList([]int64{n, int64(i)}))
	}
	return l
}

// TestSortAlgorithmsAgree: on random lists (small and large ranges,
// many repeats, negatives, empty and one item), every algorithm gives
// min_sort's answer; the stable ones keep equal keys in their order.
func TestSortAlgorithmsAgree(t *testing.T) {
	it := New(t.TempDir())
	r := rand.New(rand.NewSource(1))
	stable := map[string]bool{"bubble_sort": true, "insertion_sort": true, "merge_sort": true, "counting_sort": true, "radix_sort": true}
	for round := 0; round < 300; round++ {
		n := r.Intn(60)
		if round%50 == 0 {
			n = 2000
		}
		spread := []int64{3, 100, 1 << 40}[round%3]
		ns := make([]int64, n)
		for i := range ns {
			ns[i] = r.Int63n(2*spread) - spread
		}
		key := &object.Integer{Value: 0}
		want := it.callSort("min_sort", []object.Object{pairList(ns), key})
		for name := range classicSorts {
			if name == "counting_sort" && spread > 1000 {
				continue
			}
			got := it.callSort(name, []object.Object{pairList(ns), key})
			if stable[name] {
				if !object.Equal(got, want) {
					t.Fatalf("%s %v:\n got %s\nwant %s", name, ns, got.Inspect(), want.Inspect())
				}
				continue
			}
			// Unstable: the keys must match, in order.
			gk := it.callSort("min_sort", []object.Object{got, key})
			if !object.Equal(firsts(got), firsts(want)) || !object.Equal(firsts(gk), firsts(want)) {
				t.Fatalf("%s %v: %s", name, ns, got.Inspect())
			}
		}
		down := it.callSort("max_sort", []object.Object{intList(ns)}).(*object.List)
		for i := 1; i < len(down.Elements); i++ {
			if compareValues("t", down.Elements[i-1], down.Elements[i]) < 0 {
				t.Fatalf("max_sort out of order: %s", down.Inspect())
			}
		}
	}
}

func firsts(l object.Object) *object.List {
	out := &object.List{}
	for _, e := range l.(*object.List).Elements {
		out.Elements = append(out.Elements, e.(*object.List).Elements[0])
	}
	return out
}

// TestSearchAlgorithmsAgree: on random sorted lists with repeats, every
// search gives the first position of the value, or -1, the same as a
// linear search; insert_position is where it would go.
func TestSearchAlgorithmsAgree(t *testing.T) {
	it := New(t.TempDir())
	r := rand.New(rand.NewSource(2))
	names := []string{"binary_search", "jump_search", "exponential_search", "interpolation_search", "ternary_search"}
	for round := 0; round < 500; round++ {
		n := r.Intn(80)
		ns := make([]int64, n)
		for i := range ns {
			ns[i] = r.Int63n(40) - 10
		}
		sorted := it.callSort("min_sort", []object.Object{intList(ns)})
		for want := int64(-12); want <= 32; want++ {
			v := &object.Integer{Value: want}
			lin := it.callSearch("linear_search", []object.Object{sorted, v}).(*object.Integer).Value
			for _, name := range names {
				got := it.callSearch(name, []object.Object{sorted, v}).(*object.Integer).Value
				if got != lin {
					t.Fatalf("%s for %d in %s: got %d, want %d", name, want, sorted.Inspect(), got, lin)
				}
			}
			pos := it.callSearch("insert_position", []object.Object{sorted, v}).(*object.Integer).Value
			els := sorted.(*object.List).Elements
			if pos < 0 || pos > int64(len(els)) ||
				pos > 0 && compareValues("t", els[pos-1], v) >= 0 ||
				pos < int64(len(els)) && compareValues("t", els[pos], v) < 0 {
				t.Fatalf("insert_position %d in %s: %d", want, sorted.Inspect(), pos)
			}
		}
	}
}

// TestStdlibExamples runs each program in testdata/stdlib; each checks
// its own results and ends with "failures: 0".
func TestStdlibExamples(t *testing.T) {
	src, err := filepath.Abs("../testdata/stdlib")
	if err != nil {
		t.Fatal(err)
	}
	programs, _ := filepath.Glob(filepath.Join(src, "*.t"))
	if len(programs) == 0 {
		t.Fatal("no programs in testdata/stdlib")
	}
	for _, prog := range programs {
		t.Run(filepath.Base(prog), func(t *testing.T) {
			code, err := os.ReadFile(prog)
			if err != nil {
				t.Fatal(err)
			}
			out, runErr := runFull(t, src, string(code), "", nil)
			if runErr != nil || !strings.Contains(out, "failures: 0") {
				t.Fatalf("failed (err %v):\n%s", runErr, out)
			}
		})
	}
}
