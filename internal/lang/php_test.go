package lang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/progress"
)

func writePHP(t *testing.T, dir, name, code string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	return path
}

func TestPHPParser_ClassAndMethod(t *testing.T) {
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

	p := NewPHPParser()
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

func TestPHPParser_FunctionAndUsage(t *testing.T) {
	tmp := t.TempDir()
	code := `<?php
function format_phone($num) { return $num; }
$user = new User();
$user->getName();
format_phone("123");
`
	path := writePHP(t, tmp, "helpers.php", code)

	p := NewPHPParser()
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

func TestPHPParser_ProcessFilesConcurrently(t *testing.T) {
	tmp := t.TempDir()
	writePHP(t, tmp, "One.php", "<?php class One {}")
	writePHP(t, tmp, "Two.php", "<?php class Two {}")

	files := []models.FileInfo{
		{Path: filepath.Join(tmp, "One.php"), RelativePath: "One.php"},
		{Path: filepath.Join(tmp, "Two.php"), RelativePath: "Two.php"},
	}

	p := NewPHPParser()
	pb := progress.NewProgressBar(len(files), "Testing parser")
	parsed, err := p.ProcessFiles(files, pb)
	if err != nil {
		t.Fatalf("ProcessFiles error: %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 parsed files, got %d", len(parsed))
	}
}

func TestPHPParser_EnumsAndFinalClasses(t *testing.T) {
	tmp := t.TempDir()
	code := `<?php
final class FinalUser extends BaseUser implements JsonSerializable, Stringable {}

trait Loggable {}

class UsesTrait {
    use Loggable;
}

enum Status: string implements BackedEnum {
    case Draft = 'draft';
}
`
	path := writePHP(t, tmp, "EnumAndFinal.php", code)

	p := NewPHPParser()
	parsed, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	var foundFinalClass, foundEnum, foundTrait, foundUsesTrait bool
	var extendsUsage, implementsUsage, enumImplements, traitUseEdge bool

	for _, el := range parsed.Elements {
		switch el.Type {
		case "class":
			if el.Name == "FinalUser" {
				foundFinalClass = true
				if el.IsAbstract {
					t.Errorf("expected final class not to be abstract")
				}
			}
			if el.Name == "UsesTrait" {
				foundUsesTrait = true
			}
		case "enum":
			if el.Name == "Status" {
				foundEnum = true
			}
		case "trait":
			if el.Name == "Loggable" {
				foundTrait = true
			}
		}
	}

	for _, u := range parsed.Usage {
		if u.Context == "FinalUser" && u.Type == "extends" && u.Name == "BaseUser" {
			extendsUsage = true
		}
		if u.Context == "FinalUser" && u.Type == "implements" && (u.Name == "JsonSerializable" || u.Name == "Stringable") {
			implementsUsage = true
		}
		if u.Context == "Status" && u.Type == "implements" && u.Name == "BackedEnum" {
			enumImplements = true
		}
		if u.Context == "UsesTrait" && u.Type == "uses_trait" && u.Name == "Loggable" {
			traitUseEdge = true
		}
	}

	if !foundFinalClass || !foundEnum || !foundTrait || !foundUsesTrait || !extendsUsage || !implementsUsage || !enumImplements || !traitUseEdge {
		t.Errorf("expected final class, enum, trait and use with relationships, got class=%v enum=%v trait=%v usesTrait=%v extends=%v implements=%v enumImplements=%v traitUse=%v",
			foundFinalClass, foundEnum, foundTrait, foundUsesTrait, extendsUsage, implementsUsage, enumImplements, traitUseEdge)
	}
}
