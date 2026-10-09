# Legacy interpreter

The original Turtle interpreter, kept for reference. The language today
runs on the rewrite in `cmd/turtle` (see the main [README](../README.md)
and [`docs/architecture.md`](../docs/architecture.md)).

| File | What it is |
|---|---|
| `Turtle_interpreter.go` | the original interpreter, one ~6,800-line file that parsed each statement by re-scanning raw strings |
| `Files/files.go` | its file helpers (`[read]`, `[write]`, ...) |
| `test.trt` | an example script from that time |

The code is unchanged except for one line: the import path of `Files`,
now `Turtle/legacy/Files`.

## Building it

It uses cgo (`system()` from C for `sys`), so it needs a C compiler:

```sh
go build -o turtle-legacy ./legacy
cd legacy && ../turtle-legacy test.trt
```

CI builds, vets and tests `legacy/Files/` but leaves out the cgo package,
whose C-compiler warnings GitHub reports as failures.

Surface syntax is the same in both interpreters.
