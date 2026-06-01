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

func writeJS(t *testing.T, dir, name, code string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write JS fixture: %v", err)
	}
	return path
}

func TestJSParser_ClassAndMethod(t *testing.T) {
	tmp := t.TempDir()
	code := `import { BaseService } from './BaseService';
const config = require('./config');

export default class UserService extends BaseService {
    constructor() {
        super();
    }

    async getUser(id) {
        return this.client.get(id);
    }

    static create(data) {
        return new UserService();
    }
}
`
	path := writeJS(t, tmp, "UserService.js", code)

	p := NewJSParser()
	parsed, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	// Verify imports/uses
	if len(parsed.Uses) != 2 {
		t.Fatalf("expected 2 uses, got %d: %+v", len(parsed.Uses), parsed.Uses)
	}
	if parsed.Uses[0] != "./BaseService" || parsed.Uses[1] != "./config" {
		t.Errorf("unexpected imports: %+v", parsed.Uses)
	}

	// Verify elements
	var foundClass, foundMethodGetUser, foundMethodCreate bool
	for _, el := range parsed.Elements {
		switch el.Type {
		case "class":
			if el.Name == "UserService" {
				foundClass = true
			}
		case "method":
			if el.Name == "getUser" {
				foundMethodGetUser = true
				if el.ClassName != "UserService" {
					t.Errorf("expected getUser class context to be UserService, got %q", el.ClassName)
				}
			}
			if el.Name == "create" {
				foundMethodCreate = true
				if el.ClassName != "UserService" {
					t.Errorf("expected create class context to be UserService, got %q", el.ClassName)
				}
			}
		}
	}

	if !foundClass || !foundMethodGetUser || !foundMethodCreate {
		t.Errorf("missing expected elements: class=%v getUser=%v create=%v",
			foundClass, foundMethodGetUser, foundMethodCreate)
	}

	// Verify usage extraction
	var foundExtends, foundInstantiation bool
	for _, u := range parsed.Usage {
		if u.Type == "extends" && u.Name == "BaseService" && u.Context == "UserService" {
			foundExtends = true
		}
		if u.Type == "instantiation" && u.Name == "UserService" && u.Context == "create" {
			foundInstantiation = true
		}
	}
	if !foundExtends || !foundInstantiation {
		t.Errorf("expected extends=%v instantiation=%v", foundExtends, foundInstantiation)
	}
}

func TestJSParser_FunctionsAndBraceDepth(t *testing.T) {
	tmp := t.TempDir()
	code := `// Some comments containing {} braces
/*
  Another comment block with braces { }
 */
function formatPhone(num) {
    const x = "string containing braces: { }";
    return num;
}

const getStatus = (user) => {
    return 'active';
};
`
	path := writeJS(t, tmp, "utils.js", code)

	p := NewJSParser()
	parsed, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	var foundFormatPhone, foundGetStatus bool
	for _, el := range parsed.Elements {
		if el.Type == "function" {
			if el.Name == "formatPhone" {
				foundFormatPhone = true
			}
			if el.Name == "getStatus" {
				foundGetStatus = true
			}
		}
	}

	if !foundFormatPhone || !foundGetStatus {
		t.Errorf("expected formatPhone=%v getStatus=%v", foundFormatPhone, foundGetStatus)
	}
}

func TestJSParser_ProcessFilesConcurrently(t *testing.T) {
	tmp := t.TempDir()
	writeJS(t, tmp, "One.js", "class One {}")
	writeJS(t, tmp, "Two.js", "class Two {}")

	files := []models.FileInfo{
		{Path: filepath.Join(tmp, "One.js"), RelativePath: "One.js"},
		{Path: filepath.Join(tmp, "Two.js"), RelativePath: "Two.js"},
	}

	p := NewJSParser()
	pb := progress.NewProgressBar(len(files), "Testing JS parser")
	parsed, err := p.ProcessFiles(files, pb)
	if err != nil {
		t.Fatalf("ProcessFiles error: %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 parsed files, got %d", len(parsed))
	}
}
