// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/progress"
)

// Parser handles parsing of PHP files
type Parser struct {
	// Regex patterns for different PHP constructs
	namespacePattern      *regexp.Regexp
	usePattern            *regexp.Regexp
	classPattern          *regexp.Regexp
	functionPattern       *regexp.Regexp
	methodPattern         *regexp.Regexp
	propertyPattern       *regexp.Regexp
	constantPattern       *regexp.Regexp
	staticCallPattern     *regexp.Regexp
	methodCallPattern     *regexp.Regexp
	newInstancePattern    *regexp.Regexp
	globalFunctionPattern *regexp.Regexp
}

// New creates a new PHP parser with compiled regex patterns
func New() *Parser {
	return &Parser{
		// Namespace: namespace App\Models;
		namespacePattern: regexp.MustCompile(`^\s*namespace\s+([A-Za-z_\\][A-Za-z0-9_\\]*)\s*;`),

		// Use statements: use App\Models\User;
		usePattern: regexp.MustCompile(`^\s*use\s+([A-Za-z_\\][A-Za-z0-9_\\]*)\s*(?:as\s+([A-Za-z_][A-Za-z0-9_]*))?\s*;`),

		// Class: class User extends Model implements UserInterface
		classPattern: regexp.MustCompile(`^\s*(abstract\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?:extends\s+([A-Za-z_][A-Za-z0-9_]*))?\s*(?:implements\s+([A-Za-z0-9_,\s]+))?\s*\{?`),

		// Function: function getUserById($id): User
		functionPattern: regexp.MustCompile(`^\s*function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*(?::\s*([A-Za-z_\\][A-Za-z0-9_\\]*))?\s*\{?`),

		// Method: public static function create($data): self
		methodPattern: regexp.MustCompile(`^\s*(public|private|protected)?\s*(static\s+)?(abstract\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*(?::\s*([A-Za-z_\\][A-Za-z0-9_\\]*))?\s*\{?`),

		// Property: private $name; protected static $instances = [];
		propertyPattern: regexp.MustCompile(`^\s*(public|private|protected)\s+(static\s+)?\$([A-Za-z_][A-Za-z0-9_]*)`),

		// Constant: const STATUS_ACTIVE = 'active';
		constantPattern: regexp.MustCompile(`^\s*(public|private|protected\s+)?const\s+([A-Z_][A-Z0-9_]*)\s*=`),

		// Static calls: User::find($id), self::$instance
		staticCallPattern: regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)::(\$?[A-Za-z_][A-Za-z0-9_]*)`),

		// Method calls: $user->getName(), $this->property
		methodCallPattern: regexp.MustCompile(`\$[A-Za-z_][A-Za-z0-9_]*->(\$?[A-Za-z_][A-Za-z0-9_]*)`),

		// New instances: new User(), new \App\Models\User()
		newInstancePattern: regexp.MustCompile(`new\s+([A-Za-z_\\][A-Za-z0-9_\\]*)`),

		// Global function calls: format_phone($phone), validate_email($email)
		globalFunctionPattern: regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`),
	}
}

// ParseFile analyzes a single PHP file and extracts all elements
func (p *Parser) ParseFile(filePath string) (*models.ParsedFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	parsed := &models.ParsedFile{
		Path:     filePath,
		Elements: []models.CodeElement{},
		Usage:    []models.UsageElement{},
		Uses:     []string{},
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	inClass := ""
	inFunction := ""
	braceDepth := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(trimmedLine, "//") || strings.HasPrefix(trimmedLine, "#") ||
			strings.HasPrefix(trimmedLine, "/*") || trimmedLine == "" {
			continue
		}

		// Track brace depth to know when we exit classes/functions
		braceDepth += strings.Count(line, "{") - strings.Count(line, "}")

		// Parse namespace
		if matches := p.namespacePattern.FindStringSubmatch(line); matches != nil {
			parsed.Namespace = matches[1]
		}

		// Parse use statements
		if matches := p.usePattern.FindStringSubmatch(line); matches != nil {
			parsed.Uses = append(parsed.Uses, matches[1])
		}

		// Parse class declaration
		if matches := p.classPattern.FindStringSubmatch(line); matches != nil {
			inClass = matches[2]
			element := models.CodeElement{
				Type:       "class",
				Name:       matches[2],
				Namespace:  parsed.Namespace,
				Line:       lineNum,
				File:       filePath,
				IsAbstract: strings.Contains(matches[1], "abstract"),
			}
			parsed.Elements = append(parsed.Elements, element)
		}

		// Parse method declaration (inside class)
		if inClass != "" {
			if matches := p.methodPattern.FindStringSubmatch(line); matches != nil {
				visibility := "public" // Default visibility
				if matches[1] != "" {
					visibility = matches[1]
				}

				element := models.CodeElement{
					Type:       "method",
					Name:       matches[4],
					Namespace:  parsed.Namespace,
					ClassName:  inClass,
					Visibility: visibility,
					IsStatic:   strings.Contains(matches[2], "static"),
					IsAbstract: strings.Contains(matches[3], "abstract"),
					Line:       lineNum,
					File:       filePath,
					Parameters: parseParameters(matches[5]),
					ReturnType: matches[6],
				}
				parsed.Elements = append(parsed.Elements, element)
				inFunction = matches[4]
			}
		}

		// Parse standalone function declaration
		if inClass == "" {
			if matches := p.functionPattern.FindStringSubmatch(line); matches != nil {
				element := models.CodeElement{
					Type:       "function",
					Name:       matches[1],
					Namespace:  parsed.Namespace,
					Line:       lineNum,
					File:       filePath,
					Parameters: parseParameters(matches[2]),
					ReturnType: matches[3],
				}
				parsed.Elements = append(parsed.Elements, element)
				inFunction = matches[1]
			}
		}

		// Parse property declaration
		if inClass != "" {
			if matches := p.propertyPattern.FindStringSubmatch(line); matches != nil {
				element := models.CodeElement{
					Type:       "property",
					Name:       matches[3],
					Namespace:  parsed.Namespace,
					ClassName:  inClass,
					Visibility: matches[1],
					IsStatic:   strings.Contains(matches[2], "static"),
					Line:       lineNum,
					File:       filePath,
				}
				parsed.Elements = append(parsed.Elements, element)
			}
		}

		// Parse constant declaration
		if matches := p.constantPattern.FindStringSubmatch(line); matches != nil {
			visibility := "public" // Default for constants
			if matches[1] != "" {
				visibility = strings.TrimSpace(matches[1])
			}

			element := models.CodeElement{
				Type:       "constant",
				Name:       matches[2],
				Namespace:  parsed.Namespace,
				ClassName:  inClass,
				Visibility: visibility,
				Line:       lineNum,
				File:       filePath,
			}
			parsed.Elements = append(parsed.Elements, element)
		}

		// Parse usage patterns
		p.parseUsage(line, lineNum, inFunction, inClass, parsed)

		// Reset context when exiting classes/functions
		if braceDepth == 0 {
			inClass = ""
			inFunction = ""
		}
	}

	return parsed, scanner.Err()
}

// parseUsage finds references to external code elements
func (p *Parser) parseUsage(line string, lineNum int, inFunction, inClass string, parsed *models.ParsedFile) {
	context := inFunction
	if context == "" {
		context = inClass
	}

	// Find static calls
	staticMatches := p.staticCallPattern.FindAllStringSubmatch(line, -1)
	for i := 0; i < len(staticMatches); i++ {
		match := staticMatches[i]
		usage := models.UsageElement{
			Type:     "static_call",
			Name:     match[1] + "::" + match[2],
			Context:  context,
			Line:     lineNum,
			IsStatic: true,
		}
		parsed.Usage = append(parsed.Usage, usage)
	}

	// Find method calls
	methodMatches := p.methodCallPattern.FindAllStringSubmatch(line, -1)
	for i := 0; i < len(methodMatches); i++ {
		match := methodMatches[i]
		usage := models.UsageElement{
			Type:    "method_call",
			Name:    match[1],
			Context: context,
			Line:    lineNum,
		}
		parsed.Usage = append(parsed.Usage, usage)
	}

	// Find new instances
	newMatches := p.newInstancePattern.FindAllStringSubmatch(line, -1)
	for i := 0; i < len(newMatches); i++ {
		match := newMatches[i]
		usage := models.UsageElement{
			Type:    "instantiation",
			Name:    match[1],
			Context: context,
			Line:    lineNum,
		}
		parsed.Usage = append(parsed.Usage, usage)
	}

	// Find global function calls
	globalMatches := p.globalFunctionPattern.FindAllStringSubmatch(line, -1)
	for i := 0; i < len(globalMatches); i++ {
		match := globalMatches[i]
		funcName := match[1]

		// Skip if this looks like a method call or static call
		if strings.Contains(line, "->") || strings.Contains(line, "::") {
			continue
		}

		// Skip PHP built-in functions and common keywords
		if p.isBuiltinFunction(funcName) {
			continue
		}

		// Skip if this is a method/class definition line
		if strings.Contains(line, "function "+funcName) ||
			strings.Contains(line, "class "+funcName) {
			continue
		}

		usage := models.UsageElement{
			Type:    "function_call",
			Name:    funcName,
			Context: context,
			Line:    lineNum,
		}
		parsed.Usage = append(parsed.Usage, usage)
	}
}

// isBuiltinFunction checks if a function name is a PHP built-in
func (p *Parser) isBuiltinFunction(funcName string) bool {
	builtins := map[string]bool{
		// Common PHP built-ins that we want to ignore
		"array": true, "count": true, "isset": true, "empty": true,
		"strlen": true, "substr": true, "strpos": true, "str_replace": true,
		"preg_match": true, "preg_replace": true, "explode": true, "implode": true,
		"trim": true, "ltrim": true, "rtrim": true, "strtolower": true, "strtoupper": true,
		"ucfirst": true, "ucwords": true, "sprintf": true, "printf": true,
		"file_get_contents": true, "file_put_contents": true, "fopen": true, "fclose": true,
		"json_encode": true, "json_decode": true, "serialize": true, "unserialize": true,
		"md5": true, "sha1": true, "hash": true, "base64_encode": true, "base64_decode": true,
		"time": true, "date": true, "strtotime": true, "mktime": true,
		"rand": true, "mt_rand": true, "shuffle": true, "array_merge": true, "array_keys": true,
		"array_values": true, "array_filter": true, "array_map": true, "sort": true,
		"var_dump": true, "print_r": true, "die": true, "exit": true, "echo": true, "print": true,
		"include": true, "require": true, "include_once": true, "require_once": true,
		"defined": true, "define": true, "constant": true, "get_class": true, "is_array": true,
		"is_string": true, "is_numeric": true, "is_null": true, "is_object": true,
		"call_user_func": true, "call_user_func_array": true, "func_get_args": true,
		// Common Laravel helpers (these might be custom, but very common)
		"config": true, "env": true, "app": true, "view": true, "route": true, "url": true,
		"asset": true, "redirect": true, "back": true, "old": true, "session": true,
		"auth": true, "bcrypt": true, "collect": true, "dd": true, "dump": true,
		// Control structures and keywords (false positives)
		"if": true, "else": true, "elseif": true, "endif": true, "for": true, "foreach": true,
		"while": true, "do": true, "switch": true, "case": true, "default": true,
		"try": true, "catch": true, "finally": true, "throw": true, "return": true,
	}

	return builtins[strings.ToLower(funcName)]
}

// parseParameters extracts parameter names from function signature
func parseParameters(paramStr string) []string {
	if paramStr == "" {
		return []string{}
	}

	params := strings.Split(paramStr, ",")
	var result []string

	for _, param := range params {
		param = strings.TrimSpace(param)
		// Extract parameter name (after $ sign)
		if idx := strings.Index(param, "$"); idx != -1 {
			paramName := param[idx+1:]
			// Remove default value if present
			if eqIdx := strings.Index(paramName, "="); eqIdx != -1 {
				paramName = paramName[:eqIdx]
			}
			result = append(result, strings.TrimSpace(paramName))
		}
	}

	return result
}

// ProcessFiles parses multiple PHP files concurrently
func (p *Parser) ProcessFiles(files []models.FileInfo) ([]*models.ParsedFile, error) {
	return p.ProcessFilesWithProgress(files, nil)
}

// ProcessFilesWithProgress parses multiple PHP files concurrently with progress tracking
func (p *Parser) ProcessFilesWithProgress(files []models.FileInfo, progressBar *progress.ProgressBar) ([]*models.ParsedFile, error) {
	var parsedFiles []*models.ParsedFile
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Channel to limit concurrent parsing
	semaphore := make(chan struct{}, 10) // Max 10 concurrent parsers

	for _, file := range files {
		wg.Add(1)
		go func(f models.FileInfo) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			parsed, err := p.ParseFile(f.Path)
			if err != nil {
				fmt.Printf("⚠️  Error parsing %s: %v\n", f.RelativePath, err)
				if progressBar != nil {
					progressBar.Update(1)
				}
				return
			}

			mu.Lock()
			parsedFiles = append(parsedFiles, parsed)
			if progressBar != nil {
				progressBar.Update(1)
			}
			mu.Unlock()
		}(file)
	}

	wg.Wait()

	if progressBar != nil {
		progressBar.Finish()
	}

	return parsedFiles, nil
}
