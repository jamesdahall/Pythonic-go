.PHONY: test check fmt

test:
	go test ./...

check:
	go run ./tools/check_names ./examples ./tools ./cmd

fmt:
	gofmt -w .
