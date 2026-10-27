# The Pythonic Go Standard

Write Go so a Python programmer can read it at a glance.

This standard was extracted from the EntMon daemon (`client/go/main.go` and
`client/go/daemon.go` in [entmon-mgnt](https://github.com/jamesdahall/entmon-mgnt)).
Those two files are the reference implementation. When a rule here and that
code disagree, the code wins until this document is updated.

## 0. What we keep from Go

Go is the language. We do not invent syntax.

- `gofmt` is mandatory. Tabs, gofmt grouping, gofmt import order.
- Errors are values. Return them. Wrap with `fmt.Errorf("...: %w", err)`.
- `defer` closes bodies, cancels contexts, and unlocks.
- Modules, `go test`, and the stdlib are the toolchain.
- Types, compiles, and race detector stay. We are not writing Python.

What we change is **how we name, comment, and structure** the Go we write.

## 1. snake_case

Functions, methods, variables, constants, and unexported types use
`snake_case`.

```go
func new_daemon(api_url, api_key string, poll_interval, script_timeout int) (*daemon, error)

func (d *daemon) poll_and_execute()

var dangerous_patterns = []*regexp.Regexp{ /* ... */ }

type execution_record struct { /* ... */ }
```

This is the whole point. `pollAndExecute` reads as Go. `poll_and_execute`
reads as the Python it was ported from.

Package names stay short and conventional (`main`, `http`, `json`). Do not
name a package `pythonic_go`.

### Exported names: Pascal_Snake

Go only exports identifiers that start with a capital letter. json, gob, and
reflect skip unexported fields. When a name must be exported, capitalize the
first letter and leave the underscores.

```go
type execution_record struct {
	Script_name string `json:"script_name"`
	Executed_at string `json:"executed_at"`
	Status      string `json:"status"`
	Exit_code   int    `json:"exit_code"`
	Etag        string `json:"etag"`
}
```

One-word exported fields stay one word (`Name`, `Status`, `Etag`). Multi-word
fields keep the underscore (`Script_name`, `Exit_code`, `Instance_id`).

JSON tags are always `snake_case`, matching the Python and the wire format.

### Test names

`go test` requires `Test`, `Benchmark`, `Example`, or `Fuzz` plus a capital
letter. Write `Test_should_execute`, not `TestShouldExecute`.

```go
func Test_should_execute_retries_when_etag_changes(t *testing.T) { /* ... */ }
```

### What is allowed to be camelCase

Only names the language or an interface forces:

- `main`, `init`
- stdlib interface methods: `Error`, `String`, `ServeHTTP`, `Read`, `Write`,
  `Close`, `Len`, `Less`, `Swap`, `Unwrap`, `MarshalJSON`, …
- The `Test` / `Benchmark` / `Example` / `Fuzz` prefix itself

`tools/check_names` flags everything else.

### Disable mixedCaps lint

`staticcheck` ST1003 and `revive` var-naming will complain. They are wrong
for this codebase. Turn them off:

```yaml
# .golangci.yml
linters:
  disable:
    - stylecheck
issues:
  exclude-rules:
    - text: "ST1003:"
      linters: [staticcheck]
    - text: "var-naming:"
      linters: [revive]
```

Do not disable `gofmt`, `govet`, or `errcheck`. Those still earn their keep.

## 2. Comment like a Python docstring

Every exported-or-not type, constructor, and non-trivial function gets a
paragraph comment above it. The comment says **why it exists** and **what
the caller should know**, not a restatement of the signature.

```go
// should_execute is the idempotency gate. It inspects the local execution
// history for a script and returns one of four decisions:
//
//   - "execute"           — never run before; run it now
//   - "already_succeeded" — ran and succeeded at least once; never run again
//   - "stay_failed"       — ran and failed; content unchanged (same ETag)
//   - "retry"             — ran and failed; content changed (different ETag)
func (d *daemon) should_execute(name, etag string) string {
```

Inside a function, a short comment introduces each block the way a Python
inline comment does. A reader should be able to skim comments and know the
shape of the function without decoding every line.

Comments are for operators and Python-fluent teammates, not for a compiler.
Write full sentences. Name the failure mode you are avoiding.

Do not write comments that only repeat the code (`// increment i`). Do not
leave a function of more than a few lines uncommented because "the code is
the documentation." In this style, the comments *are* the onboarding.

## 3. One obvious way

| Thing | The way |
|---|---|
| Constructor | `new_foo(...) (*foo, error)` |
| Method receiver | pointer, named after the type: `func (d *daemon)` |
| Package-level helper | `snake_case` function, no method, when it does not need state |
| Config error | return `error`, caller prints and `os.Exit(1)` |
| Transient error | log, return, try again next loop |
| Status | a string (`"success"`, `"failed"`, `"execute"`) until you have a real type |
| Logging | `fmt.Printf` / `fmt.Fprintf(os.Stderr, ...)` until you need structured logs |
| Time | `time.Duration` at the boundary, `int` seconds in config |
| JSON | `encoding/json`, `snake_case` tags, `MarshalIndent` for files humans read |

Do not add a second constructor (`MustNew`, `NewFromContext`) until the first
one is in the way. Do not add a logger interface for a daemon that prints to
the journal.

## 4. One struct, many methods

The EntMon daemon is one `daemon` struct created in `new_daemon` and passed
by pointer for the life of the process. That is the model.

- Put runtime state on the struct.
- Put behavior on methods of that struct.
- Do not introduce an interface until there are two implementations you
  actually compile.
- Do not split a 50-line method into five unexported helpers that are only
  called once. Linear is better than nested, and better than scatter.

A package-level function is right when it has no reason to see the struct
(`validate_content` in the daemon scans a string and returns `(bool, string)`).

## 5. Linear control flow

Flat is better than nested. Return early. Keep the happy path at the bottom
of the function, the way a Python function `return`s on the error cases.

```go
func (d *daemon) load_state() error {
	data, err := os.ReadFile(d.state_file)
	if os.IsNotExist(err) {
		return d.save_state()
	}
	if err != nil {
		return err
	}
	// ... happy path
}
```

`if err != nil { return err }` is Go and it is also Pythonic: errors are not
silent, and they do not indent the rest of the function.

A `switch` on a string status is better than a boolean maze.

## 6. Fail fast on config, fail quiet on transients

This pair is load-bearing in the daemon and it is a rule.

**Config** (missing `api_url`, unreadable state directory that is not
"file not found"): return an error. `main` prints it and exits. Do not start
a loop that cannot do useful work.

**Transients** (network blip, API 5xx, download failed): log a warning and
return. The next poll retries. Do not record a failed execution for a
download that never happened — that would burn the retry slot.

**Corrupt but local** (JSON state file half-written): warn, continue with
empty state. Re-running an idempotent setup script is better than refusing
to start.

Every quiet path has a comment that says why it is quiet.

Bad baked-in integers (`-ldflags` poll interval that is not a number) keep
the documented default. A daemon that crashes because an ldflag was typo'd
is worse than one that polls every 60 seconds.

## 7. stdlib first

The EntMon `go.mod` has no `require` lines. That is the target.

Use `net/http`, `encoding/json`, `os/exec`, `os`, `path/filepath`, `regexp`,
`sort`, `strings`, `time`, `context`, `flag`.

Add a module when the stdlib genuinely cannot do the job (a cloud SDK, a
Windows service wrapper). Do not add a module for logging, HTTP routing,
or assertion libraries. `if got != want { t.Fatalf(...) }` is enough.

## 8. Concurrency is opt-in

The daemon's main loop is:

```go
func (d *daemon) run() {
	interval := time.Duration(d.poll_interval) * time.Second
	for {
		d.poll_and_execute()
		time.Sleep(interval)
	}
}
```

No goroutines, no channels, no wait groups. That is intentional. A fleet
small enough to poll on a fixed interval does not need a worker pool.

Add a goroutine when you have a measured reason (a listener and a worker,
a context-cancelled shutdown, a parallel fetch you actually profiled). Do
not add one because "Go is about concurrency."

`context.WithTimeout` around `exec.CommandContext` is the right kind of
concurrency: it is a deadline, it is visible, and it kills the child.

## 9. Errors, not panics; tuples, not exceptions

Return `(T, error)`. In `main`, print and exit.

A Python-shaped tuple is fine when the second value is not an error:

```go
func validate_content(content string) (bool, string) {
	for _, re := range dangerous_patterns {
		if re.MatchString(content) {
			return false, "Script blocked: contains dangerous pattern"
		}
	}
	return true, ""
}
```

Do not panic for operator-facing failures. Do not use named result
parameters to hide a `return`. Named results are allowed when they make a
three-value return readable (`scripts, pending_ticket, err`).

Wrap with `%w` when the caller might `errors.Is` / `errors.As`. Otherwise
`fmt.Errorf("list-scripts returned %d: %s", code, body)` is enough.

## 10. Strings for small state machines

The daemon uses strings for execution status and for the idempotency
decision. That is a Python habit and it is allowed.

```go
switch d.should_execute(s.Name, s.Etag) {
case "execute", "retry":
	to_execute = append(to_execute, s)
case "stay_failed":
	fmt.Printf("ERROR: '%s' previously failed — skipping\n", s.Name)
}
```

Promote a string to a named type when it is used in more than two
functions or when a typo would compile and do the wrong thing. Do not
start with `iota`. `iota` is for bit flags and for people who already
know which number is which.

## 11. Persistence and JSON

Human-readable JSON on disk. `json.MarshalIndent(s, "", "  ")`.
`omitempty` on fields that are meaningless off EC2.

Write state as soon as it changes, not at the end of the loop. A crash
between "script ran" and "state saved" is how you re-run a script; a
crash after save is how you skip it. Choose the one that matches the
idempotency story, and comment it.

Create directories with `os.MkdirAll`. It is idempotent. Call it every
time you write.

## 12. HTTP

One `*http.Client` on the struct, with a timeout. A second client is
allowed when the timeout must be different (the daemon uses 2s for IMDS
and 30s for the API).

Set headers explicitly. If a gateway requires lowercase header names,
assign the map directly (`req.Header["x-host-id"] = []string{...}`) and
comment why `Header.Set` is wrong here.

Always `defer resp.Body.Close()`. Read the body on non-200 so the error
message includes it.

## 13. Tests

- File next to the code: `heartbeat_test.go` beside `heartbeat.go`.
- Name: `Test_what_it_covers`.
- Table-driven tests are fine when the table is easier to read than the
  loop it replaces. They are not required.
- No testify, no ginkgo, no `assert` package.
- Cover the decision function (`should_execute`, `should_record`) before
  you cover the HTTP client.

A test that needs the network is not a unit test. Skip it or stub the
round tripper.

## 14. What we do not take from Python

- Exceptions. No `panic`/`recover` as control flow.
- Default arguments and kwargs. Every parameter is positional and present.
- Inheritance. There is no base class. Compose a struct.
- Dynamic types. `any` is a last resort, not a dict.
- List comprehensions. Write the `for` loop. It is the same number of
  lines and it has room for a comment.
- `*args`. A slice parameter is fine; a variadic helper used once is not.

## 15. What we do not take from typical Go

- camelCase as the default.
- "A comment that restates the code is a smell" used as a reason to
  write no comments.
- An interface for a single implementation.
- A goroutine in `main` because the program might need to stop someday.
- A logging facade.
- Tiny functions whose only job is to have a name.

When you contribute to kubernetes, or cobra, or anyone else's Go, use
their style. This standard is for *our* Go: operators' daemons, internal
CLIs, and small services written by people who also write Python.

## 16. Mapping from the Zen of Python

| Zen | In this Go |
|---|---|
| Readability counts | snake_case, paragraph comments, linear functions |
| Explicit is better than implicit | no hidden goroutines, no `init()` magic, no default-true flags |
| Simple is better than complex | stdlib, one struct, a loop |
| Complex is better than complicated | a long `execute_script` with comments beats a pile of helpers |
| Flat is better than nested | early return |
| Sparse is better than dense | blank lines between blocks; comments take space on purpose |
| Errors should never pass silently | unless the comment says this one is a transient |
| There should be one obvious way | the table in §3 |
| If the implementation is hard to explain, it's a bad idea | the comment is the test |
| Namespaces are one honking great idea | packages stay boring |

## 17. Checklist before you merge

- [ ] `gofmt -l .` is empty
- [ ] `go test ./...` passes
- [ ] `go run ./tools/check_names .` passes (or the new name is on the allowlist for a real reason)
- [ ] Every new function has a paragraph comment
- [ ] JSON tags are snake_case
- [ ] No new module without a sentence in the PR saying why stdlib cannot do it
- [ ] No new goroutine without a sentence saying what it waits on and how it stops

## Appendix A — EntMon identifiers this was mined from

These names are the corpus. New code in that daemon should look like them.

**Constructors and entrypoints:** `new_daemon`, `main`, `run`, `poll_and_execute`

**Methods:** `detect_os`, `get_host_id`, `load_state`, `save_state`,
`check_for_clone`, `should_execute`, `script_timeout`, `add_headers`,
`list_scripts`, `fetch_script`, `execute_script`, `report_execution`

**Types:** `daemon`, `execution_record`, `persisted_state`, `script_info`

**Fields (exported for JSON):** `Script_name`, `Executed_at`, `Exit_code`,
`Instance_id`

**Package-level:** `dangerous_patterns`, `script_timeout_re`,
`validate_content`, `builtin_api_url`, `builtin_api_key`,
`builtin_poll_interval`, `builtin_script_timeout`

**String statuses:** `"execute"`, `"already_succeeded"`, `"stay_failed"`,
`"retry"`, `"success"`, `"failed"`

A complete miniature of the same shape lives in
[examples/heartbeat](examples/heartbeat).
