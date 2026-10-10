package evaluator

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"Turtle/object"
)

// Memory: how much a value takes, how much the program holds, and how
// much a step allocated (diagnose, hypothesis and turtle test show it).
// Turtle frees a value by itself once nothing refers to it any more; Go
// collects such values in batches, so freememory[] collects now and gives
// the memory back to the system, between big steps of a program.

// allocatedBytes is all the memory the program has allocated so far: the
// difference across a step is what the step allocated, whatever has been
// freed since.
func allocatedBytes() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.TotalAlloc
}

// heldBytes is the memory the program's values hold now, after a
// collection, so it doesn't count values already let go.
func heldBytes() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// memoryLine is a report's memory line: what the step allocated.
func memoryLine(since uint64) string {
	return fmt.Sprintf("  %-9s %s allocated", "memory", fmtBytes(allocatedBytes()-since))
}

// fmtBytes writes a number of bytes for people: 512 B, 3.4 KB, 31.5 MB.
func fmtBytes(n uint64) string {
	const k = 1000 // as limits are written: 5000000 is 5 MB
	switch {
	case n < k:
		return fmt.Sprintf("%d B", n)
	case n < k*k:
		return fmt.Sprintf("%.1f KB", float64(n)/k)
	case n < k*k*k:
		return fmt.Sprintf("%.1f MB", float64(n)/(k*k))
	}
	return fmt.Sprintf("%.2f GB", float64(n)/(k*k*k))
}

// sizeOf is about how many bytes v takes, with everything in it; a value
// in it twice (or shared with another) counts once.
func sizeOf(v object.Object) uint64 {
	seen := map[object.Object]bool{}
	var size func(v object.Object) uint64
	size = func(v object.Object) uint64 {
		if v == nil || seen[v] {
			return 0
		}
		seen[v] = true
		const word, slot = 8, 16 // a number; a place in a list that holds any value
		switch x := v.(type) {
		case *object.Integer, *object.Float, *object.Boolean, *object.None:
			return word
		case *object.String:
			return 2*word + uint64(len(x.Value))
		case *object.List:
			n := 3*word + slot*uint64(cap(x.Elements))
			for _, e := range x.Elements {
				n += size(e)
			}
			return n
		case *object.Set:
			n := 3*word + slot*uint64(cap(x.Elements))
			for _, e := range x.Elements {
				n += size(e)
			}
			return n
		case *object.Map:
			entries := x.Entries()
			n := 6*word + uint64(len(entries))*(2*slot+2*word) // keys, values and the index
			for _, e := range entries {
				n += size(e.Key) + size(e.Val)
			}
			return n
		case *object.Assembly:
			n := 2*word + slot*uint64(len(x.Values))
			for _, f := range x.Values {
				n += size(f)
			}
			return n
		case *object.Matrix:
			return 4*word + word*uint64(len(x.Data))
		}
		return 4 * word // a function, date, database, ...: its own record only
	}
	return size(v)
}

// callMemory is the system library's memory functions.
func callMemory(name string, args []object.Object) (object.Object, bool) {
	switch name {
	case "memory":
		requireFuncArgs(name, args, 0)
		return object.Int(int64(heldBytes())), true
	case "sizeof":
		requireFuncArgs(name, args, 1)
		return object.Int(int64(sizeOf(args[0]))), true
	case "freememory":
		requireFuncArgs(name, args, 0)
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		before := m.HeapAlloc
		debug.FreeOSMemory() // collects, then gives what's free back to the system
		runtime.ReadMemStats(&m)
		freed := int64(0)
		if before > m.HeapAlloc {
			freed = int64(before - m.HeapAlloc)
		}
		return object.Int(freed), true
	}
	return nil, false
}
