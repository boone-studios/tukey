package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boone-studios/tukey/internal/models"
)

func writePHP(t *testing.T, dir, name, code string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	return path
}

func TestParseFile_ClassAndMethod(t *testing.T) {
	tmp := t.TempDir()
	code := `<?php
namespace App\Models;
use App\Services\Mailer;

abstract class User {
    private static $instances = [];
    public function __construct() {}
    public static function create($data): self {}
    const STATUS_ACTIVE = 'active';
}
`
	path := writePHP(t, tmp, "User.php", code)

	p := New()
	parsed, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	if parsed.Namespace != "App\\Models" {
		t.Errorf("expected namespace App\\Models, got %q", parsed.Namespace)
	}
	if len(parsed.Uses) == 0 || parsed.Uses[0] != "App\\Services\\Mailer" {
		t.Errorf("expected use statement App\\Services\\Mailer, got %+v", parsed.Uses)
	}

	var foundCreate, foundConst, foundClass bool
	for _, el := range parsed.Elements {
		switch el.Type {
		case "class":
			foundClass = true
			if !el.IsAbstract {
				t.Errorf("expected abstract class, got non-abstract")
			}
		case "method":
			if el.Name == "create" {
				foundCreate = true
				if el.ReturnType != "self" {
					t.Errorf("expected return type self, got %q", el.ReturnType)
				}
				if el.Visibility != "public" {
					t.Errorf("expected public, got %s", el.Visibility)
				}
			}
		case "constant":
			foundConst = true
		}
	}
	if !foundClass || !foundCreate || !foundConst {
		t.Errorf("missing expected elements: class=%v create=%v const=%v",
			foundClass, foundCreate, foundConst)
	}
}

func TestParseFile_FunctionAndUsage(t *testing.T) {
	tmp := t.TempDir()
	code := `<?php
function format_phone($num) { return $num; }
$user = new User();
$user->getName();
format_phone("123");
`
	path := writePHP(t, tmp, "helpers.php", code)

	p := New()
	parsed, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	var fn, usageNew, usageMethod, usageFunc bool
	for _, el := range parsed.Elements {
		if el.Type == "function" && el.Name == "format_phone" {
			fn = true
			if len(el.Parameters) != 1 || el.Parameters[0] != "num" {
				t.Errorf("function parameters parsed incorrectly: %+v", el.Parameters)
			}
		}
	}
	for _, u := range parsed.Usage {
		switch u.Type {
		case "instantiation":
			usageNew = true
		case "method_call":
			usageMethod = true
		case "function_call":
			if u.Name == "format_phone" {
				usageFunc = true
			}
		}
	}
	if !(fn && usageNew && usageMethod && usageFunc) {
		t.Errorf("expected fn=%v new=%v method=%v func=%v",
			fn, usageNew, usageMethod, usageFunc)
	}
}

func TestProcessFilesConcurrently(t *testing.T) {
	tmp := t.TempDir()
	writePHP(t, tmp, "One.php", "<?php class One {}")
	writePHP(t, tmp, "Two.php", "<?php class Two {}")

	files := []models.FileInfo{
		{Path: filepath.Join(tmp, "One.php"), RelativePath: "One.php"},
		{Path: filepath.Join(tmp, "Two.php"), RelativePath: "Two.php"},
	}

	p := New()
	parsed, err := p.ProcessFiles(files)
	if err != nil {
		t.Fatalf("ProcessFiles error: %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 parsed files, got %d", len(parsed))
	}
}
