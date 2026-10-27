package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_is_camel_case(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"poll_and_execute", false},
		{"new_daemon", false},
		{"daemon", false},
		{"Run", false},
		{"main", false},
		{"Script_name", false},
		{"MAX_RETRY", false},
		{"pollAndExecute", true},
		{"NewClient", true},
		{"httpClient", true},
		{"shouldExecute", true},
	}
	for _, c := range cases {
		if got := is_camel_case(c.name); got != c.want {
			t.Errorf("is_camel_case(%q)=%v want %v", c.name, got, c.want)
		}
	}
}

func Test_name_is_allowed(t *testing.T) {
	if !name_is_allowed("main") {
		t.Fatal("main must be allowed")
	}
	if !name_is_allowed("Error") {
		t.Fatal("Error must be allowed")
	}
	if !name_is_allowed("ServeHTTP") {
		t.Fatal("ServeHTTP must be allowed")
	}
	if !name_is_allowed("Test_should_execute") {
		t.Fatal("Test_should_execute must be allowed")
	}
	if name_is_allowed("TestShouldExecute") {
		t.Fatal("TestShouldExecute must be rejected")
	}
	if name_is_allowed("pollAndExecute") {
		t.Fatal("pollAndExecute must be rejected")
	}
}

func Test_check_file_flags_camel_case(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	src := "package p\nfunc pollAndExecute() {}\nfunc poll_and_execute() {}\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	found, err := check_file(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("findings=%v, want exactly one", found)
	}
	if !strings.Contains(found[0], "pollAndExecute") {
		t.Fatalf("finding=%q, want pollAndExecute", found[0])
	}
}

func Test_check_file_allows_snake_case(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	src := "package p\nfunc new_client() {}\nfunc (c *T) load_state() {}\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	found, err := check_file(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("findings=%v, want none", found)
	}
}
