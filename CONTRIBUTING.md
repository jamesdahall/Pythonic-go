# Contributing to Pythonic Go

This repo is a style standard. A change has to make Go easier to read for
someone who thinks in Python, or it does not land.

## What belongs here

- A rule that is already true of the EntMon daemon (or a clear, small extension).
- An example that compiles and has a test.
- A check_names allowlist entry for a stdlib interface method we missed.

## What does not

- Switching the default to camelCase.
- Adding a framework, a logger, or an HTTP router as a dependency of the examples.
- "More idiomatic Go" that trades a linear function for an interface, a
  goroutine, or a generic helper used once.
- Reformatting STANDARD.md for tone without changing a rule.

## How to send a change

1. Edit STANDARD.md and, if the rule is checkable, extend `tools/check_names`.
2. Add or update an example under `examples/` so the rule has a compiling home.
3. Run:

```bash
gofmt -w .
go test ./...
go run ./tools/check_names ./examples ./tools
```

4. Open a pull request that states which rule changed and why a Python-fluent
   reader is better off.

## Naming in this repo

Follow the standard you are editing. `Test_load_state` not `TestLoadState`.
`new_client` not `NewClient`. `check_names` not `checkNames`.
