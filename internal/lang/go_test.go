// Copyright (c) 2026 Boone Studios
// SPDX-License-Identifier: MIT

package lang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/progress"
)

func writeGo(t *testing.T, dir, name, code string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write Go fixture: %v", err)
	}
	return path
}

func TestGoParser_StructAndMethod(t *testing.T) {
	tmp := t.TempDir()
	code := `package main

import (
	"fmt"
	"github.com/boone-studios/tukey/internal/models"
)

type MyStruct struct {
	Field1 string
	field2 int
}

type MyInterface interface {
	DoSomething() error
}

const MyConst = "constant_val"

func (s *MyStruct) MyMethod(a int) string {
	fmt.Println(s.Field1)
	return "hello"
}

func MyFunction() {
	s := &MyStruct{Field1: "test"}
	s.MyMethod(123)
}
`
	path := writeGo(t, tmp, "mystruct.go", code)

	p := NewGoParser()
	parsed, err := p.ParseFile(path, "github.com/boone-studios/tukey", "mystruct.go")
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	// Verify Namespace
	expectedNamespace := "github.com/boone-studios/tukey"
	if parsed.Namespace != expectedNamespace {
		t.Errorf("expected namespace %q, got %q", expectedNamespace, parsed.Namespace)
	}

	// Verify imports/uses
	if len(parsed.Uses) != 2 {
		t.Fatalf("expected 2 uses, got %d: %+v", len(parsed.Uses), parsed.Uses)
	}
	if parsed.Uses[0] != "fmt" || parsed.Uses[1] != "github.com/boone-studios/tukey/internal/models" {
		t.Errorf("unexpected imports: %+v", parsed.Uses)
	}

	// Verify elements
	var foundStruct, foundInterface, foundMethod, foundFunction, foundConst, foundProp bool
	for _, el := range parsed.Elements {
		switch el.Type {
		case "class":
			if el.Name == "MyStruct" {
				foundStruct = true
			}
		case "interface":
			if el.Name == "MyInterface" {
				foundInterface = true
			}
		case "method":
			if el.Name == "MyMethod" {
				foundMethod = true
				if el.ClassName != "MyStruct" {
					t.Errorf("expected class name to be MyStruct, got %q", el.ClassName)
				}
			}
		case "function":
			if el.Name == "MyFunction" {
				foundFunction = true
			}
		case "constant":
			if el.Name == "MyConst" {
				foundConst = true
			}
		case "property":
			if el.Name == "Field1" {
				foundProp = true
				if el.ClassName != "MyStruct" {
					t.Errorf("expected field1 class to be MyStruct, got %q", el.ClassName)
				}
			}
		}
	}

	if !foundStruct || !foundInterface || !foundMethod || !foundFunction || !foundConst || !foundProp {
		t.Errorf("missing expected elements: struct=%v interface=%v method=%v function=%v const=%v prop=%v",
			foundStruct, foundInterface, foundMethod, foundFunction, foundConst, foundProp)
	}

	// Verify usage extraction
	var foundMethodCall, foundFuncCall bool
	for _, u := range parsed.Usage {
		// MyMethod call on s (which was inferred to MyStruct)
		if u.Type == "method_call" && u.Name == "github.com/boone-studios/tukey\\MyStruct::MyMethod" {
			foundMethodCall = true
		}
		// fmt.Println call
		if u.Type == "function" && u.Name == "fmt\\Println" {
			foundFuncCall = true
		}
	}

	if !foundMethodCall {
		t.Errorf("expected to find method call, got parsed.Usage: %+v", parsed.Usage)
	}
	if !foundFuncCall {
		t.Errorf("expected to find fmt.Println function call, got parsed.Usage: %+v", parsed.Usage)
	}
}

func TestGoParser_ProcessFilesConcurrently(t *testing.T) {
	tmp := t.TempDir()
	writeGo(t, tmp, "one.go", "package one\ntype One struct{}")
	writeGo(t, tmp, "two.go", "package two\ntype Two struct{}")

	files := []models.FileInfo{
		{Path: filepath.Join(tmp, "one.go"), RelativePath: "one.go"},
		{Path: filepath.Join(tmp, "two.go"), RelativePath: "two.go"},
	}

	p := NewGoParser()
	pb := progress.NewProgressBar(len(files), "Testing Go parser")
	parsed, err := p.ProcessFiles(files, pb)
	if err != nil {
		t.Fatalf("ProcessFiles error: %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 parsed files, got %d", len(parsed))
	}
}
