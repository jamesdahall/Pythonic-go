# heartbeat example

A miniature of the EntMon daemon, written to the Pythonic Go standard.

- `new_client` constructor returns `(*client, error)`
- `should_record` returns a string status (`"record"` / `"already_succeeded"`)
- JSON on disk uses `snake_case` tags and `Pascal_Snake` exported fields
- The loop is sequential: poll, sleep, repeat
- Paragraph comments on every type and method

```bash
# from the repo root
go test ./examples/heartbeat
go run ./cmd/heartbeat -once https://example.com
```

This is teaching code. It is not a production health checker.
