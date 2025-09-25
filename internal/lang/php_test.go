package lang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/progress"
)

func writePHP(t testing.TB, dir, name, code string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	return path
}

func TestPHPParserBasics(t *testing.T) {
	parser := NewPHPParser()

	if parser.Language() != "php" {
		t.Errorf("Expected language 'php', got '%s'", parser.Language())
	}

	extensions := parser.FileExtensions()
	expectedExts := []string{".php", ".phtml", ".php3", ".php4", ".php5"}
	if len(extensions) != len(expectedExts) {
		t.Errorf("Expected %d extensions, got %d", len(expectedExts), len(extensions))
	}

	for i, expected := range expectedExts {
		if i < len(extensions) && extensions[i] != expected {
			t.Errorf("Expected extension '%s', got '%s'", expected, extensions[i])
		}
	}
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

func TestParseFile_FunctionAndUsage(t *testing.T) {
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

func TestPHPParserComplexClass(t *testing.T) {
	tmp := t.TempDir()
	code := `<?php
namespace App\Http\Controllers;

use Illuminate\Http\Request;
use App\Services\UserService;

class UserController extends Controller implements UserInterface {
    private $userService;
    protected static $cache = [];
    public const DEFAULT_ROLE = 'user';
    
    public function __construct(UserService $service) {
        $this->userService = $service;
    }
    
    public function index(Request $request): JsonResponse {
        return response()->json(['users' => []]);
    }
    
    private static function validateInput($data) {
        return !empty($data);
    }
    
    protected function formatUser($user) {
        return $user->toArray();
    }
}
`
	path := writePHP(t, tmp, "UserController.php", code)

	p := NewPHPParser()
	parsed, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	// Check namespace and imports
	if parsed.Namespace != "App\\Http\\Controllers" {
		t.Errorf("Expected namespace App\\Http\\Controllers, got %q", parsed.Namespace)
	}

	expectedUses := []string{
		"Illuminate\\Http\\Request",
		"App\\Services\\UserService",
	}
	if len(parsed.Uses) != len(expectedUses) {
		t.Errorf("Expected %d use statements, got %d", len(expectedUses), len(parsed.Uses))
	}

	// Check elements
	var foundClass, foundProperty, foundConstant, foundConstructor, foundIndex, foundValidate, foundFormat bool

	for _, el := range parsed.Elements {
		switch el.Type {
		case "class":
			if el.Name == "UserController" {
				foundClass = true
				if el.IsAbstract {
					t.Errorf("Expected non-abstract class, got abstract")
				}
			}
		case "property":
			if el.Name == "userService" {
				foundProperty = true
				if el.Visibility != "private" {
					t.Errorf("Expected private property, got %s", el.Visibility)
				}
				if el.ClassName != "UserController" {
					t.Errorf("Expected property in UserController, got %s", el.ClassName)
				}
			}
			if el.Name == "cache" && el.IsStatic {
				if el.Visibility != "protected" {
					t.Errorf("Expected protected static property, got %s", el.Visibility)
				}
			}
		case "constant":
			if el.Name == "DEFAULT_ROLE" {
				foundConstant = true
				if el.Visibility != "public" {
					t.Errorf("Expected public constant, got %s", el.Visibility)
				}
			}
		case "method":
			switch el.Name {
			case "__construct":
				foundConstructor = true
				if len(el.Parameters) != 1 || el.Parameters[0] != "service" {
					t.Errorf("Constructor parameters incorrect: %+v", el.Parameters)
				}
			case "index":
				foundIndex = true
				if el.Visibility != "public" {
					t.Errorf("Expected public method, got %s", el.Visibility)
				}
				if el.ReturnType != "JsonResponse" {
					t.Errorf("Expected return type JsonResponse, got %s", el.ReturnType)
				}
			case "validateInput":
				foundValidate = true
				if el.Visibility != "private" || !el.IsStatic {
					t.Errorf("Expected private static method, got %s static=%v", el.Visibility, el.IsStatic)
				}
			case "formatUser":
				foundFormat = true
				if el.Visibility != "protected" {
					t.Errorf("Expected protected method, got %s", el.Visibility)
				}
			}
		}
	}

	if !foundClass {
		t.Error("UserController class not found")
	}
	if !foundProperty {
		t.Error("userService property not found")
	}
	if !foundConstant {
		t.Error("DEFAULT_ROLE constant not found")
	}
	if !foundConstructor {
		t.Error("__construct method not found")
	}
	if !foundIndex {
		t.Error("index method not found")
	}
	if !foundValidate {
		t.Error("validateInput method not found")
	}
	if !foundFormat {
		t.Error("formatUser method not found")
	}
}

func TestPHPParserGlobalFunctions(t *testing.T) {
	tmp := t.TempDir()
	code := `<?php

function format_phone($phone) {
    return preg_replace('/[^\d]/', '', $phone);
}

function format_address($address, $country = 'US') {
    return trim($address);
}

function validate_email($email): bool {
    return filter_var($email, FILTER_VALIDATE_EMAIL) !== false;
}

// Test usage
$phone = format_phone('555-123-4567');
$address = format_address('  123 Main St  ', 'CA');
$isValid = validate_email('test@example.com');
`
	path := writePHP(t, tmp, "helpers.php", code)

	p := NewPHPParser()
	parsed, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	// Check functions
	expectedFunctions := map[string]struct {
		paramCount int
		returnType string
	}{
		"format_phone":   {1, ""},
		"format_address": {2, ""},
		"validate_email": {1, "bool"},
	}

	foundFunctions := make(map[string]bool)
	for _, el := range parsed.Elements {
		if el.Type == "function" {
			if expected, exists := expectedFunctions[el.Name]; exists {
				foundFunctions[el.Name] = true
				if len(el.Parameters) != expected.paramCount {
					t.Errorf("Function %s expected %d parameters, got %d",
						el.Name, expected.paramCount, len(el.Parameters))
				}
				if el.ReturnType != expected.returnType {
					t.Errorf("Function %s expected return type '%s', got '%s'",
						el.Name, expected.returnType, el.ReturnType)
				}
			}
		}
	}

	for funcName := range expectedFunctions {
		if !foundFunctions[funcName] {
			t.Errorf("Function %s not found", funcName)
		}
	}

	// Check usage
	var formatPhoneUsage, formatAddressUsage, validateEmailUsage bool
	for _, usage := range parsed.Usage {
		if usage.Type == "function_call" {
			switch usage.Name {
			case "format_phone":
				formatPhoneUsage = true
			case "format_address":
				formatAddressUsage = true
			case "validate_email":
				validateEmailUsage = true
			}
		}
	}

	if !formatPhoneUsage {
		t.Error("format_phone usage not detected")
	}
	if !formatAddressUsage {
		t.Error("format_address usage not detected")
	}
	if !validateEmailUsage {
		t.Error("validate_email usage not detected")
	}
}

func TestPHPParserStaticCalls(t *testing.T) {
	tmp := t.TempDir()
	code := `<?php
namespace App\Models;

class User {
    public static function find($id) {
        return Database::query("SELECT * FROM users WHERE id = ?", [$id]);
    }
    
    public function save() {
        $validator = Validator::make($this->attributes, $this->rules);
        if ($validator->fails()) {
            throw new ValidationException();
        }
        return parent::save();
    }
}

// Usage
$user = User::find(1);
$config = Config::get('app.name');
Cache::put('user:1', $user, 3600);
`
	path := writePHP(t, tmp, "User.php", code)

	p := NewPHPParser()
	parsed, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	// Check static calls in usage
	expectedStaticCalls := []string{
		"Database::query",
		"Validator::make",
		"User::find",
		"Config::get",
		"Cache::put",
		"parent::save",
	}

	foundStaticCalls := make(map[string]bool)
	for _, usage := range parsed.Usage {
		if usage.Type == "static_call" {
			foundStaticCalls[usage.Name] = true
		}
	}

	for _, expected := range expectedStaticCalls {
		if !foundStaticCalls[expected] {
			t.Errorf("Static call %s not found", expected)
		}
	}
}

func TestPHPParserBuiltinFiltering(t *testing.T) {
	tmp := t.TempDir()
	code := `<?php

function custom_helper($data) {
    // Use built-in functions that should be filtered out
    $json = json_encode($data);
    $length = strlen($json);
    $hash = md5($json);
    
    // Use custom function that should be detected
    return format_data($json);
}

function format_data($input) {
    return trim($input);
}

// Usage
$result = custom_helper(['key' => 'value']);
echo $result;
`
	path := writePHP(t, tmp, "test.php", code)

	p := NewPHPParser()
	parsed, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	// Should find custom function calls but not built-ins
	var foundCustomHelper, foundFormatData, foundBuiltin bool
	for _, usage := range parsed.Usage {
		if usage.Type == "function_call" {
			switch usage.Name {
			case "custom_helper":
				foundCustomHelper = true
			case "format_data":
				foundFormatData = true
			case "json_encode", "strlen", "md5", "trim", "echo":
				foundBuiltin = true
			}
		}
	}

	if !foundCustomHelper {
		t.Error("Expected to find custom_helper call")
	}
	if !foundFormatData {
		t.Error("Expected to find format_data call")
	}
	if foundBuiltin {
		t.Error("Should not find built-in function calls (json_encode, strlen, etc.)")
	}
}

func TestPHPParserNamespaces(t *testing.T) {
	tmp := t.TempDir()
	code := `<?php
namespace App\Services\Email;

use PHPMailer\PHPMailer\PHPMailer;
use App\Models\User as UserModel;
use Illuminate\Support\Facades\Log;

class EmailService {
    private $mailer;
    
    public function __construct() {
        $this->mailer = new PHPMailer();
    }
    
    public function sendWelcomeEmail(UserModel $user) {
        Log::info("Sending welcome email to " . $user->email);
        return $this->mailer->send();
    }
}
`
	path := writePHP(t, tmp, "EmailService.php", code)

	p := NewPHPParser()
	parsed, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	// Check namespace
	if parsed.Namespace != "App\\Services\\Email" {
		t.Errorf("Expected namespace App\\Services\\Email, got %s", parsed.Namespace)
	}

	// Check use statements
	expectedUses := []string{
		"PHPMailer\\PHPMailer\\PHPMailer",
		"App\\Models\\User",
		"Illuminate\\Support\\Facades\\Log",
	}

	if len(parsed.Uses) != len(expectedUses) {
		t.Errorf("Expected %d use statements, got %d", len(expectedUses), len(parsed.Uses))
	}

	for i, expected := range expectedUses {
		if i < len(parsed.Uses) && parsed.Uses[i] != expected {
			t.Errorf("Expected use statement '%s', got '%s'", expected, parsed.Uses[i])
		}
	}
}

func TestPHPParserComplexParameters(t *testing.T) {
	tmp := t.TempDir()
	code := `<?php

function complexFunction($simple, array $array, ?string $nullable = null, callable $callback = null, ...$variadic) {
    return true;
}

class TestClass {
    public function methodWithDefaults($required, $optional = 'default', $number = 42, $bool = true) {
        // implementation
    }
}
`
	path := writePHP(t, tmp, "complex.php", code)

	p := NewPHPParser()
	parsed, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	// Check complex function parameters
	var complexFunc, methodWithDefaults *models.CodeElement
	for i := range parsed.Elements {
		el := &parsed.Elements[i]
		if el.Type == "function" && el.Name == "complexFunction" {
			complexFunc = el
		}
		if el.Type == "method" && el.Name == "methodWithDefaults" {
			methodWithDefaults = el
		}
	}

	if complexFunc == nil {
		t.Fatal("complexFunction not found")
	}

	expectedParams := []string{"simple", "array", "nullable", "callback", "variadic"}
	if len(complexFunc.Parameters) != len(expectedParams) {
		t.Errorf("Expected %d parameters, got %d", len(expectedParams), len(complexFunc.Parameters))
	}

	for i, expected := range expectedParams {
		if i < len(complexFunc.Parameters) && complexFunc.Parameters[i] != expected {
			t.Errorf("Expected parameter '%s', got '%s'", expected, complexFunc.Parameters[i])
		}
	}

	if methodWithDefaults == nil {
		t.Fatal("methodWithDefaults not found")
	}

	expectedMethodParams := []string{"required", "optional", "number", "bool"}
	if len(methodWithDefaults.Parameters) != len(expectedMethodParams) {
		t.Errorf("Expected %d method parameters, got %d", len(expectedMethodParams), len(methodWithDefaults.Parameters))
	}
}

func TestPHPParserTraitsAndInterfaces(t *testing.T) {
	tmp := t.TempDir()
	code := `<?php
namespace App\Contracts;

interface UserRepositoryInterface {
    public function find($id);
    public function create(array $data): User;
    public function update($id, array $data): bool;
}

trait Timestampable {
    public function updateTimestamps() {
        $this->updated_at = date('Y-m-d H:i:s');
    }
}
`
	path := writePHP(t, tmp, "contracts.php", code)

	p := NewPHPParser()
	parsed, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	// Note: Current parser doesn't specifically handle interfaces/traits differently
	// but should still parse methods within them
	var foundUpdateTimestamps bool
	for _, el := range parsed.Elements {
		if el.Type == "method" && el.Name == "updateTimestamps" {
			foundUpdateTimestamps = true
		}
	}

	if !foundUpdateTimestamps {
		t.Error("updateTimestamps method from trait not found")
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

func TestPHPParserErrorHandling(t *testing.T) {
	tmp := t.TempDir()
	// Create a file with some edge cases that shouldn't crash the parser
	code := `<?php
// Incomplete class
class IncompleteClass {
    public function methodWithoutBraces()
    
    // Missing visibility
    function noVisibility() {}
    
    // Weird spacing
    public    static     function    weirdSpacing   (  $param  )   :   string  {
        return '';
    }
}

// Function with complex signature
function complexSignature(
    array $data,
    ?callable $callback = null,
    string $format = 'json'
): ?array {
    return null;
}
`
	path := writePHP(t, tmp, "edge_cases.php", code)

	p := NewPHPParser()
	parsed, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile should handle edge cases gracefully: %v", err)
	}

	// Should still find some elements despite edge cases
	if len(parsed.Elements) == 0 {
		t.Error("Expected to find some elements despite edge cases")
	}
}

// Benchmark test for PHP parser
func BenchmarkPHPParser(b *testing.B) {
	tmp := b.TempDir()
	code := `<?php
namespace App\Models;

use Illuminate\Database\Eloquent\Model;

class User extends Model {
    protected $fillable = ['name', 'email', 'password'];
    
    public function posts() {
        return $this->hasMany(Post::class);
    }
    
    public function getFullNameAttribute() {
        return $this->first_name . ' ' . $this->last_name;
    }
}

function helper_function($data) {
    return json_encode($data);
}
`
	path := writePHP(b, tmp, "benchmark.php", code)

	parser := NewPHPParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseFile(path)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Integration test with Laravel-style code
func TestPHPParserLaravelIntegration(t *testing.T) {
	tmp := t.TempDir()

	// Create multiple PHP files mimicking a Laravel structure
	files := map[string]string{
		"User.php": `<?php
namespace App\Models;

use Illuminate\Foundation\Auth\User as Authenticatable;

class User extends Authenticatable {
    protected $fillable = ['name', 'email', 'password'];
    
    public function posts() {
        return $this->hasMany(Post::class);
    }
}`,
		"UserController.php": `<?php
namespace App\Http\Controllers;

use App\Models\User;
use Illuminate\Http\Request;

class UserController extends Controller {
    public function index() {
        $users = User::all();
        return view('users.index', compact('users'));
    }
    
    public function store(Request $request) {
        $user = User::create($request->validated());
        return redirect()->route('users.index');
    }
}`,
		"helpers.php": `<?php

function format_user_name($user) {
    return ucwords($user->name);
}

function get_user_avatar($user, $size = 'medium') {
    return asset('avatars/' . $user->id . '_' . $size . '.jpg');
}`,
	}

	var fileInfos []models.FileInfo
	for filename, content := range files {
		filePath := filepath.Join(tmp, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", filename, err)
		}

		fileInfos = append(fileInfos, models.FileInfo{
			Path:         filePath,
			RelativePath: filename,
			Size:         int64(len(content)),
		})
	}

	parser := NewPHPParser()
	progressBar := progress.NewProgressBar(len(fileInfos), "Testing")

	results, err := parser.ProcessFiles(fileInfos, progressBar)
	if err != nil {
		t.Fatalf("ProcessFiles failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 parsed files, got %d", len(results))
	}

	// Verify specific elements were found across files
	var foundUserClass, foundUserController, foundFormatFunction, foundGetAvatarFunction bool
	var totalMethods, totalFunctions int

	for _, result := range results {
		for _, element := range result.Elements {
			switch {
			case element.Type == "class" && element.Name == "User":
				foundUserClass = true
			case element.Type == "class" && element.Name == "UserController":
				foundUserController = true
			case element.Type == "function" && element.Name == "format_user_name":
				foundFormatFunction = true
			case element.Type == "function" && element.Name == "get_user_avatar":
				foundGetAvatarFunction = true
			case element.Type == "method":
				totalMethods++
			case element.Type == "function":
				totalFunctions++
			}
		}
	}

	if !foundUserClass {
		t.Error("User class not found")
	}
	if !foundUserController {
		t.Error("UserController class not found")
	}
	if !foundFormatFunction {
		t.Error("format_user_name function not found")
	}
	if !foundGetAvatarFunction {
		t.Error("get_user_avatar function not found")
	}

	// Should find methods and functions
	if totalMethods == 0 {
		t.Error("No methods found across all files")
	}
	if totalFunctions == 0 {
		t.Error("No functions found across all files")
	}
}
