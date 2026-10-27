# Pythonic Go

A standard for writing Go that a Python programmer can read without translating.

Go is a great language for daemons, CLIs, and small services. Its default style
(camelCase, terse names, sparse comments) is optimized for people who already
live in Go. This standard is optimized for people who live in Python and need
Go for a binary.

It is the house style used in the EntMon daemon (`client/go` in
[entmon-mgnt](https://github.com/jamesdahall/entmon-mgnt)), written down so the
next Go file looks like the last one.

```go
func (d *daemon) should_execute(name, etag string) string {
	var prior []execution_record
	for _, e := range d.executions {
		if e.Script_name == name {
			prior = append(prior, e)
		}
	}
	if len(prior) == 0 {
		return "execute"
	}
	// ...
}
```

Same shape as Python: `snake_case`, a docstring-length comment above the
function, a linear loop, a string status instead of an iota enum.

## The one-page version

1. **snake_case** for functions, methods, variables, and unexported types.
2. **Pascal_Snake** when Go requires an exported name (`Script_name`).
3. **Comment like a Python docstring.** Say why, and what the next block does.
4. **Constructors are `new_foo`.** They return `(*foo, error)`.
5. **One struct, many methods.** No interface until there are two implementations.
6. **No goroutines until you have a measured reason.** A loop and `time.Sleep` is fine.
7. **stdlib first.** A `go.mod` with no `require` lines is a feature.
8. **Fail fast on config, fail quiet on transients.** Document the quiet path.
9. **`gofmt` is required. `ST1003` (mixedCaps) is disabled.** snake_case is intentional.
10. **Test functions are `Test_snake_case`.** Go requires the `Test` prefix; keep the rest snake.

Full rules: [STANDARD.md](STANDARD.md).

## Why this exists

The EntMon client was rewritten from Python to Go so it could ship as a single
static binary. The Go was written to stay readable to the people who wrote the
Python: same names, same control flow, paragraph comments, no cleverness.

That is a legitimate style. It is not Effective Go, and it is not a slight
against Effective Go. It is a choice: **readability for Python-fluent operators
over familiarity for career Gophers.**

## What's in this repo

| Path | What |
|---|---|
| [STANDARD.md](STANDARD.md) | The rules, with examples drawn from the EntMon daemon |
| [examples/heartbeat](examples/heartbeat) | A small complete program in this style |
| [tools/check_names](tools/check_names) | Flags camelCase function names (with an allowlist for `main`, `Error`, `ServeHTTP`, …) |
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to change the standard |

## Check a tree

```bash
go run ./tools/check_names ./examples
```

Exit 0 means every function and method name is snake_case (or on the allowlist).
Exit 1 prints the offenders.

```bash
go test ./...
gofmt -l .
```

## What this is not

- Not a new language. It is Go, `gofmt`'d, compiling with the stock toolchain.
- Not a Python interpreter, a Python FFI, or a way to call Python from Go.
- Not a replacement for the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
  when you are contributing to someone else's Go project. Use their style there.

## License

MIT. Use it, fork it, argue with it.
