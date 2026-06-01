// Copyright (c) 2026 Boone Studios
// SPDX-License-Identifier: MIT

package lang

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/parser"
	"github.com/boone-studios/tukey/internal/progress"
)

// JSParser handles parsing of JavaScript/ES6/Node files
type JSParser struct {
	importPattern      *regexp.Regexp
	requirePattern     *regexp.Regexp
	classPattern       *regexp.Regexp
	functionPattern    *regexp.Regexp
	arrowFuncPattern   *regexp.Regexp
	methodPattern      *regexp.Regexp
	newInstancePattern *regexp.Regexp
	callPattern        *regexp.Regexp
}

// NewJSParser creates a new JS parser with compiled regex patterns
func NewJSParser() *JSParser {
	return &JSParser{
		// import { User } from './models/User.js'; or import Mailer from 'mailer';
		importPattern: regexp.MustCompile(`\bimport\s+(?:[^;]*from\s+)?['"]([^'"]+)['"]`),

		// const Mailer = require('mailer');
		requirePattern: regexp.MustCompile(`\brequire\s*\(\s*['"]([^'"]+)['"]\s*\)`),

		// class UserService extends BaseService {
		classPattern: regexp.MustCompile(`^\s*(?:export\s+(?:default\s+)?)?class\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?:extends\s+([A-Za-z_][A-Za-z0-9_]*))?\s*\{?`),

		// function formatPhone(num) {
		functionPattern: regexp.MustCompile(`^\s*(?:export\s+(?:default\s+)?)?function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)`),

		// const getStatus = (user) => {
		arrowFuncPattern: regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:\([^)]*\)|[A-Za-z_][A-Za-z0-9_]*)\s*=>`),

		// async getUser(id) { or getUser(id) { or static create(data) {
		methodPattern: regexp.MustCompile(`^\s*(?:static\s+)?(?:async\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*\{?`),

		// new UserService()
		newInstancePattern: regexp.MustCompile(`new\s+([A-Za-z_][A-Za-z0-9_]*)`),

		// user.getName() or UserService.staticMethod()
		callPattern: regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
	}
}

// ParseFile parses a single JavaScript file
func (p *JSParser) ParseFile(filePath string) (*models.ParsedFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Use directory path relative to project root as the module namespace
	namespace := p.getModuleNamespace(filePath)

	parsed := &models.ParsedFile{
		Path:      filePath,
		Namespace: namespace,
		Elements:  []models.CodeElement{},
		Usage:     []models.UsageElement{},
		Uses:      []string{},
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	inClass := ""
	inFunction := ""
	braceDepth := 0
	inMultiLineComment := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Robust Multi-line comment (/* ... */) state tracking
		if inMultiLineComment {
			if idx := strings.Index(trimmedLine, "*/"); idx != -1 {
				inMultiLineComment = false
				trimmedLine = strings.TrimSpace(trimmedLine[idx+2:])
			} else {
				continue
			}
		}
		if strings.HasPrefix(trimmedLine, "/*") {
			if idx := strings.Index(trimmedLine[2:], "*/"); idx != -1 {
				trimmedLine = strings.TrimSpace(trimmedLine[2+idx+2:])
			} else {
				inMultiLineComment = true
				continue
			}
		}

		// Skip comments and empty lines
		if strings.HasPrefix(trimmedLine, "//") || trimmedLine == "" {
			continue
		}

		// Strip string literals and comments to compute brace depth accurately
		cleanLine := stripCommentsAndStringsJS(line)
		braceDepth += strings.Count(cleanLine, "{") - strings.Count(cleanLine, "}")

		// Parse ES6 imports and CommonJS requires
		if inClass == "" {
			if matches := p.importPattern.FindStringSubmatch(line); matches != nil {
				parsed.Uses = append(parsed.Uses, matches[1])
			} else if matches := p.requirePattern.FindStringSubmatch(line); matches != nil {
				parsed.Uses = append(parsed.Uses, matches[1])
			}
		}

		// Parse class declaration
		if matches := p.classPattern.FindStringSubmatch(line); matches != nil {
			inClass = matches[1]
			element := models.CodeElement{
				Type:      "class",
				Name:      matches[1],
				Namespace: namespace,
				Line:      lineNum,
				File:      filePath,
			}
			parsed.Elements = append(parsed.Elements, element)

			if matches[2] != "" { // Extends base class
				parsed.Usage = append(parsed.Usage, models.UsageElement{
					Type:    "extends",
					Name:    matches[2],
					Context: inClass,
					Line:    lineNum,
				})
			}
		}

		// Parse method declaration inside class context
		if inClass != "" {
			// Skip class declaration line itself
			if !strings.Contains(line, "class "+inClass) {
				if matches := p.methodPattern.FindStringSubmatch(line); matches != nil {
					element := models.CodeElement{
						Type:      "method",
						Name:      matches[1],
						Namespace: namespace,
						ClassName: inClass,
						Line:      lineNum,
						File:      filePath,
					}
					parsed.Elements = append(parsed.Elements, element)
					inFunction = matches[1]
				}
			}
		}

		// Parse standalone functions and arrow functions outside class context
		if inClass == "" {
			if matches := p.functionPattern.FindStringSubmatch(line); matches != nil {
				element := models.CodeElement{
					Type:      "function",
					Name:      matches[1],
					Namespace: namespace,
					Line:      lineNum,
					File:      filePath,
				}
				parsed.Elements = append(parsed.Elements, element)
				inFunction = matches[1]
			} else if matches := p.arrowFuncPattern.FindStringSubmatch(line); matches != nil {
				element := models.CodeElement{
					Type:      "function",
					Name:      matches[1],
					Namespace: namespace,
					Line:      lineNum,
					File:      filePath,
				}
				parsed.Elements = append(parsed.Elements, element)
				inFunction = matches[1]
			}
		}

		// Parse instantiations & calls usage
		p.parseUsage(line, lineNum, inFunction, inClass, parsed)

		// Reset context when exiting class/function blocks
		if braceDepth == 0 {
			inClass = ""
			inFunction = ""
		}
	}

	return parsed, scanner.Err()
}

func (p *JSParser) parseUsage(line string, lineNum int, inFunction, inClass string, parsed *models.ParsedFile) {
	context := inFunction
	if context == "" {
		context = inClass
	}

	// Instantiations: new Service()
	newMatches := p.newInstancePattern.FindAllStringSubmatch(line, -1)
	for _, match := range newMatches {
		parsed.Usage = append(parsed.Usage, models.UsageElement{
			Type:    "instantiation",
			Name:    match[1],
			Context: context,
			Line:    lineNum,
		})
	}

	// Calls: user.getName() or UserService.staticMethod()
	callMatches := p.callPattern.FindAllStringSubmatch(line, -1)
	for _, match := range callMatches {
		targetObj := match[1]
		methodName := match[2]

		if targetObj == "this" {
			continue // Skip self-referencing instance calls
		}

		// Capture static/method dependency references
		parsed.Usage = append(parsed.Usage, models.UsageElement{
			Type:    "method_call",
			Name:    targetObj + "::" + methodName,
			Context: context,
			Line:    lineNum,
		})
	}
}

// stripCommentsAndStringsJS strips comments, template literals, and strings
func stripCommentsAndStringsJS(line string) string {
	var result strings.Builder
	inDoubleQuote := false
	inSingleQuote := false
	inTemplateLiteral := false
	escaped := false

	for i := 0; i < len(line); i++ {
		char := line[i]

		if escaped {
			escaped = false
			continue
		}
		if char == '\\' && (inDoubleQuote || inSingleQuote || inTemplateLiteral) {
			escaped = true
			continue
		}

		if char == '"' && !inSingleQuote && !inTemplateLiteral {
			inDoubleQuote = !inDoubleQuote
			continue
		}
		if char == '\'' && !inDoubleQuote && !inTemplateLiteral {
			inSingleQuote = !inSingleQuote
			continue
		}
		if char == '`' && !inDoubleQuote && !inSingleQuote {
			inTemplateLiteral = !inTemplateLiteral
			continue
		}

		if inDoubleQuote || inSingleQuote || inTemplateLiteral {
			continue
		}

		if char == '/' && i+1 < len(line) {
			nextChar := line[i+1]
			if nextChar == '/' || nextChar == '*' {
				break
			}
		}

		result.WriteByte(char)
	}

	return result.String()
}

func (p *JSParser) getModuleNamespace(filePath string) string {
	// e.g., convert "src/services/UserService.js" to "src/services" as ESModule namespace
	idx := strings.LastIndex(filePath, "/")
	if idx == -1 {
		return ""
	}
	return filePath[:idx]
}

// ProcessFiles parses multiple JavaScript files concurrently
func (p *JSParser) ProcessFiles(files []models.FileInfo, progressBar *progress.ProgressBar) ([]*models.ParsedFile, error) {
	var parsedFiles []*models.ParsedFile
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)

	for _, file := range files {
		wg.Add(1)
		go func(f models.FileInfo) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			parsed, err := p.ParseFile(f.Path)
			mu.Lock()
			defer mu.Unlock()

			if err == nil {
				parsedFiles = append(parsedFiles, parsed)
			}
			progressBar.Update(1)
		}(file)
	}

	wg.Wait()
	progressBar.Finish()
	return parsedFiles, nil
}

// Language returns the language name for this parser
func (p *JSParser) Language() string {
	return "js"
}

// FileExtensions returns the file extensions supported by this parser
func (p *JSParser) FileExtensions() []string {
	return []string{".js", ".jsx", ".mjs", ".cjs"}
}

func init() {
	parser.Register(NewJSParser())
}
