// check_names reports function and method names that use camelCase.
// Pythonic Go uses snake_case; mixedCaps is a style violation, not a
// compiler error, so this tool is the gate.
//
//	go run ./tools/check_names ./examples ./tools
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// allowed_exact is names the language or a stdlib interface requires.
// These are the only identifiers that may be camelCase or a single
// exported English word that is not snake_case.
var allowed_exact = map[string]bool{
	"main": true,
	"init": true,

	// error / stringer / fmt
	"Error":    true,
	"String":   true,
	"GoString": true,
	"Format":   true,

	// io
	"Read":       true,
	"Write":      true,
	"Close":      true,
	"Flush":      true,
	"Reset":      true,
	"Seek":       true,
	"ReadAt":     true,
	"WriteAt":    true,
	"ReadFrom":   true,
	"WriteTo":    true,
	"UnreadByte": true,
	"ReadByte":   true,
	"WriteByte":  true,
	"ReadRune":   true,
	"WriteRune":  true,

	// http
	"ServeHTTP":      true,
	"Serve":          true,
	"ListenAndServe": true,

	// sort.Interface
	"Len":  true,
	"Less": true,
	"Swap": true,

	// encoding
	"MarshalJSON":     true,
	"UnmarshalJSON":   true,
	"MarshalText":     true,
	"UnmarshalText":   true,
	"MarshalBinary":   true,
	"UnmarshalBinary": true,
	"MarshalXML":      true,
	"UnmarshalXML":    true,

	// errors
	"Unwrap": true,
	"Is":     true,
	"As":     true,

	// net.Error
	"Timeout":   true,
	"Temporary": true,

	// database/sql
	"Scan":  true,
	"Value": true,

	// flag.Value
	"Set": true,
	"Get": true,

	// testing helpers sometimes required as this spelling
	"Run": true, // testing.M.Run, and our own one-word export
}

// skip_dir_names are not walked. Generated and third-party code is not
// held to this standard.
var skip_dir_names = map[string]bool{
	".git":         true,
	"vendor":       true,
	"testdata":     true,
	"node_modules": true,
}

func main() {
	roots := os.Args[1:]
	if len(roots) == 0 {
		roots = []string{"."}
	}

	var findings []string
	for _, root := range roots {
		found, err := check_tree(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "check_names: %v\n", err)
			os.Exit(2)
		}
		findings = append(findings, found...)
	}

	if len(findings) == 0 {
		fmt.Println("ok: all function names are snake_case")
		return
	}
	for _, f := range findings {
		fmt.Println(f)
	}
	os.Exit(1)
}

func check_tree(root string) ([]string, error) {
	var findings []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skip_dir_names[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, ".pb.go") {
			return nil
		}
		found, parse_err := check_file(path)
		if parse_err != nil {
			return parse_err
		}
		findings = append(findings, found...)
		return nil
	})
	return findings, err
}

func check_file(path string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	var findings []string
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			return true
		}
		name := fn.Name.Name
		if name_is_allowed(name) {
			return true
		}
		if is_camel_case(name) {
			pos := fset.Position(fn.Name.Pos())
			findings = append(findings, fmt.Sprintf("%s:%d: %s is camelCase; use snake_case", pos.Filename, pos.Line, name))
		}
		return true
	})
	return findings, nil
}

func name_is_allowed(name string) bool {
	if allowed_exact[name] {
		return true
	}
	// go test requires Test/Benchmark/Example/Fuzz + a capital letter.
	// The rest of the name is still checked for camelCase.
	for _, prefix := range []string{"Test", "Benchmark", "Example", "Fuzz"} {
		if strings.HasPrefix(name, prefix) {
			rest := strings.TrimPrefix(name, prefix)
			if rest == "" {
				return true
			}
			// Test_load_state: rest is _load_state — not camelCase.
			return !is_camel_case(rest)
		}
	}
	return false
}

// is_camel_case is true when a lowercase letter is followed later by an
// uppercase letter: pollAndExecute, NewClient, httpClient. It is false
// for snake_case (poll_and_execute), Pascal_Snake (Script_name), single
// words (Run, daemon), and SCREAMING_SNAKE (MAX_RETRY).
func is_camel_case(name string) bool {
	seen_lower := false
	for _, r := range name {
		if unicode.IsLower(r) {
			seen_lower = true
			continue
		}
		if unicode.IsUpper(r) && seen_lower {
			return true
		}
	}
	return false
}
