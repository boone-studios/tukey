// Copyright (c) 2026 Boone Studios
// SPDX-License-Identifier: MIT

package lang

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/parser"
	"github.com/boone-studios/tukey/internal/progress"
)

// TSParser handles parsing of TypeScript and TSX files.
type TSParser struct {
	importPattern        *regexp.Regexp
	exportFromPattern    *regexp.Regexp
	requirePattern       *regexp.Regexp
	dynamicImportPattern *regexp.Regexp
	classPattern         *regexp.Regexp
	interfacePattern     *regexp.Regexp
	typePattern          *regexp.Regexp
	enumPattern          *regexp.Regexp
	functionPattern      *regexp.Regexp
	arrowFuncPattern     *regexp.Regexp
	constPattern         *regexp.Regexp
	methodPattern        *regexp.Regexp
	propertyPattern      *regexp.Regexp
	extendsPattern       *regexp.Regexp
	implementsPattern    *regexp.Regexp
	newInstancePattern   *regexp.Regexp
	callPattern          *regexp.Regexp
	functionCallPattern  *regexp.Regexp
	jsxPattern           *regexp.Regexp
}

// NewTSParser creates a new TypeScript parser with compiled regex patterns.
func NewTSParser() *TSParser {
	identifier := `[A-Za-z_$][A-Za-z0-9_$]*`
	modifiers := `(?:(?:public|private|protected|static|abstract|async|readonly|declare|override|get|set|accessor)\s+)*`

	return &TSParser{
		importPattern:        regexp.MustCompile(`^\s*import\s+(?:type\s+)?(?:(.+?)\s+from\s+)?['"]([^'"]+)['"]`),
		exportFromPattern:    regexp.MustCompile(`^\s*export\s+(?:type\s+)?(?:\*|\{[^}]*\})\s+from\s+['"]([^'"]+)['"]`),
		requirePattern:       regexp.MustCompile(`\brequire\s*\(\s*['"]([^'"]+)['"]\s*\)`),
		dynamicImportPattern: regexp.MustCompile(`\bimport\s*\(\s*['"]([^'"]+)['"]\s*\)`),
		classPattern:         regexp.MustCompile(`^\s*(?:export\s+(?:default\s+)?)?(?:declare\s+)?(abstract\s+)?class\s+(` + identifier + `)([^\{]*)`),
		interfacePattern:     regexp.MustCompile(`^\s*(?:export\s+(?:default\s+)?)?(?:declare\s+)?interface\s+(` + identifier + `)([^\{]*)`),
		typePattern:          regexp.MustCompile(`^\s*(?:export\s+)?(?:declare\s+)?type\s+(` + identifier + `)(?:\s*<[^=]*>)?\s*=`),
		enumPattern:          regexp.MustCompile(`^\s*(?:export\s+(?:default\s+)?)?(?:declare\s+)?(?:const\s+)?enum\s+(` + identifier + `)`),
		functionPattern:      regexp.MustCompile(`^\s*(?:export\s+(?:default\s+)?)?(?:declare\s+)?(?:async\s+)?function\s*\*?\s*(` + identifier + `)\s*(?:<[^(]*>)?\s*\(([^)]*)\)\s*(?::\s*([^{;]+))?`),
		arrowFuncPattern:     regexp.MustCompile(`^\s*(?:export\s+(?:default\s+)?)?(?:const|let|var)\s+(` + identifier + `)\s*(?::\s*[^=]+)?=\s*(?:async\s+)?(?:<[^>]*>\s*)?(?:\(([^)]*)\)|(` + identifier + `))\s*(?::\s*([^=]+?))?\s*=>`),
		constPattern:         regexp.MustCompile(`^\s*(?:export\s+)?const\s+(` + identifier + `)\s*(?::\s*[^=]+)?\s*=`),
		methodPattern:        regexp.MustCompile(`^\s*(` + modifiers + `)(` + identifier + `)\s*\??\s*(?:<[^(]*>)?\s*\(([^)]*)\)\s*(?::\s*([^{;]+))?\s*(\{|;)?`),
		propertyPattern:      regexp.MustCompile(`^\s*((?:(?:public|private|protected|static|abstract|readonly|declare|override|accessor)\s+)*)(` + identifier + `)\s*[!?]?\s*(?::\s*[^=;]+)?\s*(?:=\s*[^;]+)?;?\s*$`),
		extendsPattern:       regexp.MustCompile(`\bextends\s+(` + identifier + `(?:\.` + identifier + `)*)`),
		implementsPattern:    regexp.MustCompile(`\bimplements\s+(.+)$`),
		newInstancePattern:   regexp.MustCompile(`\bnew\s+(` + identifier + `(?:\.` + identifier + `)*)`),
		callPattern:          regexp.MustCompile(`\b(` + identifier + `)\.(` + identifier + `)\s*(?:<[^;(){}]*>)?\s*\(`),
		functionCallPattern:  regexp.MustCompile(`\b(` + identifier + `)\s*(?:<[^;(){}]*>)?\s*\(`),
		jsxPattern:           regexp.MustCompile(`<([A-Z][A-Za-z0-9_$]*(?:\.[A-Z][A-Za-z0-9_$]*)*)\b`),
	}
}

type tsScope struct {
	name    string
	kind    string
	depth   int
	pending bool
}

// ParseFile parses a single TypeScript or TSX file.
func (p *TSParser) ParseFile(filePath string) (*models.ParsedFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	namespace := filepath.Dir(filePath)
	if namespace == "." {
		namespace = ""
	}
	parsed := &models.ParsedFile{
		Path:          filePath,
		Namespace:     namespace,
		Elements:      []models.CodeElement{},
		Usage:         []models.UsageElement{},
		Uses:          []string{},
		ImportAliases: make(map[string]string),
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNum := 0
	braceDepth := 0
	var container *tsScope
	var function *tsScope
	inMultiLineComment := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

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
		if strings.HasPrefix(trimmedLine, "//") || trimmedLine == "" {
			continue
		}

		cleanLine := stripCommentsAndStringsJS(line)
		depthBefore := braceDepth
		braceDepth += strings.Count(cleanLine, "{") - strings.Count(cleanLine, "}")
		hasOpeningBrace := strings.Contains(cleanLine, "{")

		if container != nil && container.pending && hasOpeningBrace {
			container.depth = braceDepth
			container.pending = false
		}
		if function != nil && function.pending && hasOpeningBrace {
			function.depth = braceDepth
			function.pending = false
		}

		if container == nil {
			p.parseImports(line, filePath, parsed)
		}

		declaredName := ""
		declaration := false
		temporaryFunction := false

		if container == nil && function == nil {
			if matches := p.classPattern.FindStringSubmatch(line); matches != nil {
				container = &tsScope{name: matches[2], kind: "class", depth: -1, pending: !hasOpeningBrace}
				if hasOpeningBrace {
					container.depth = depthBefore + 1
				}
				parsed.Elements = append(parsed.Elements, models.CodeElement{
					Type:       "class",
					Name:       matches[2],
					Namespace:  namespace,
					Line:       lineNum,
					File:       filePath,
					IsAbstract: strings.TrimSpace(matches[1]) != "",
				})
				p.parseHeritage(matches[3], container.name, lineNum, parsed)
				declaredName = matches[2]
				declaration = true
			} else if matches := p.interfacePattern.FindStringSubmatch(line); matches != nil {
				container = &tsScope{name: matches[1], kind: "interface", depth: -1, pending: !hasOpeningBrace}
				if hasOpeningBrace {
					container.depth = depthBefore + 1
				}
				parsed.Elements = append(parsed.Elements, models.CodeElement{
					Type:       "interface",
					Name:       matches[1],
					Namespace:  namespace,
					Line:       lineNum,
					File:       filePath,
					IsAbstract: true,
				})
				p.parseInterfaceHeritage(matches[2], container.name, lineNum, parsed)
				declaredName = matches[1]
				declaration = true
			} else if matches := p.enumPattern.FindStringSubmatch(line); matches != nil {
				container = &tsScope{name: matches[1], kind: "enum", depth: -1, pending: !hasOpeningBrace}
				if hasOpeningBrace {
					container.depth = depthBefore + 1
				}
				parsed.Elements = append(parsed.Elements, models.CodeElement{
					Type:      "enum",
					Name:      matches[1],
					Namespace: namespace,
					Line:      lineNum,
					File:      filePath,
				})
				declaredName = matches[1]
				declaration = true
			} else if matches := p.typePattern.FindStringSubmatch(line); matches != nil {
				parsed.Elements = append(parsed.Elements, models.CodeElement{
					Type:      "type",
					Name:      matches[1],
					Namespace: namespace,
					Line:      lineNum,
					File:      filePath,
				})
				declaredName = matches[1]
				declaration = true
			}
		}

		className := ""
		if container != nil {
			className = container.name
		}

		if !declaration && container != nil && function == nil && container.kind != "enum" {
			if matches := p.methodPattern.FindStringSubmatch(line); matches != nil && !tsControlKeyword(matches[2]) {
				modifiers := matches[1]
				parsed.Elements = append(parsed.Elements, models.CodeElement{
					Type:       "method",
					Name:       matches[2],
					Namespace:  namespace,
					ClassName:  className,
					Visibility: tsVisibility(modifiers),
					IsStatic:   hasTSModifier(modifiers, "static"),
					IsAbstract: hasTSModifier(modifiers, "abstract") || container.kind == "interface",
					Line:       lineNum,
					File:       filePath,
					Parameters: parseTSParameters(matches[3]),
					ReturnType: strings.TrimSpace(matches[4]),
				})
				declaredName = matches[2]
				declaration = true
				if hasOpeningBrace {
					function = &tsScope{name: matches[2], depth: depthBefore + 1}
				}
			} else if matches := p.propertyPattern.FindStringSubmatch(line); matches != nil {
				parsed.Elements = append(parsed.Elements, models.CodeElement{
					Type:       "property",
					Name:       matches[2],
					Namespace:  namespace,
					ClassName:  className,
					Visibility: tsVisibility(matches[1]),
					IsStatic:   hasTSModifier(matches[1], "static"),
					Line:       lineNum,
					File:       filePath,
				})
				declaredName = matches[2]
				declaration = true
			}
		}

		if !declaration && container == nil {
			if matches := p.functionPattern.FindStringSubmatch(line); matches != nil {
				parsed.Elements = append(parsed.Elements, models.CodeElement{
					Type:       "function",
					Name:       matches[1],
					Namespace:  namespace,
					Line:       lineNum,
					File:       filePath,
					Parameters: parseTSParameters(matches[2]),
					ReturnType: strings.TrimSpace(matches[3]),
				})
				declaredName = matches[1]
				declaration = true
				if hasOpeningBrace {
					function = &tsScope{name: matches[1], depth: depthBefore + 1}
				} else if !strings.Contains(cleanLine, ";") {
					function = &tsScope{name: matches[1], depth: -1, pending: true}
				}
			} else if matches := p.arrowFuncPattern.FindStringSubmatch(line); matches != nil {
				parameters := matches[2]
				if parameters == "" {
					parameters = matches[3]
				}
				parsed.Elements = append(parsed.Elements, models.CodeElement{
					Type:       "function",
					Name:       matches[1],
					Namespace:  namespace,
					Line:       lineNum,
					File:       filePath,
					Parameters: parseTSParameters(parameters),
					ReturnType: strings.TrimSpace(matches[4]),
				})
				declaredName = matches[1]
				declaration = true
				if hasOpeningBrace {
					function = &tsScope{name: matches[1], depth: depthBefore + 1}
				} else {
					function = &tsScope{name: matches[1], depth: -1}
					temporaryFunction = true
				}
			} else if matches := p.constPattern.FindStringSubmatch(line); matches != nil {
				parsed.Elements = append(parsed.Elements, models.CodeElement{
					Type:      "constant",
					Name:      matches[1],
					Namespace: namespace,
					Line:      lineNum,
					File:      filePath,
				})
				declaredName = matches[1]
			}
		}

		context := className
		if function != nil {
			context = function.name
			if className != "" {
				context = className + "::" + function.name
			}
		}
		p.parseUsage(cleanLine, lineNum, context, declaredName, parsed)

		if temporaryFunction {
			function = nil
		}
		if function != nil && function.depth > 0 && braceDepth < function.depth {
			function = nil
		}
		if container != nil && container.depth > 0 && braceDepth < container.depth {
			container = nil
			function = nil
		}
	}

	return parsed, scanner.Err()
}

func (p *TSParser) parseImports(line, filePath string, parsed *models.ParsedFile) {
	if matches := p.importPattern.FindStringSubmatch(line); matches != nil {
		parsed.Uses = append(parsed.Uses, matches[2])
		p.recordImportAliases(matches[1], matches[2], filePath, parsed)
		return
	}
	if matches := p.exportFromPattern.FindStringSubmatch(line); matches != nil {
		parsed.Uses = append(parsed.Uses, matches[1])
		return
	}
	if matches := p.requirePattern.FindStringSubmatch(line); matches != nil {
		parsed.Uses = append(parsed.Uses, matches[1])
	}
	for _, matches := range p.dynamicImportPattern.FindAllStringSubmatch(line, -1) {
		parsed.Uses = append(parsed.Uses, matches[1])
	}
}

func tsImportNamespace(filePath, importPath string) string {
	resolvedPath := filepath.Clean(filepath.Join(filepath.Dir(filePath), filepath.FromSlash(importPath)))
	if info, err := os.Stat(resolvedPath); err == nil && info.IsDir() {
		return resolvedPath
	}
	if filepath.Ext(resolvedPath) != "" {
		return filepath.Dir(resolvedPath)
	}
	for _, extension := range []string{".ts", ".tsx", ".mts", ".cts", ".js", ".jsx"} {
		if info, err := os.Stat(resolvedPath + extension); err == nil && !info.IsDir() {
			return filepath.Dir(resolvedPath)
		}
	}
	return filepath.Dir(resolvedPath)
}

func (p *TSParser) recordImportAliases(clause, importPath, filePath string, parsed *models.ParsedFile) {
	clause = strings.TrimSpace(clause)
	if clause == "" || !strings.HasPrefix(importPath, ".") {
		return
	}

	importNamespace := tsImportNamespace(filePath, importPath)
	record := func(localName, exportedName string) {
		localName = strings.TrimSpace(localName)
		if localName != "" {
			parsed.ImportAliases[localName] = importNamespace + `\` + exportedName
		}
	}

	if start := strings.Index(clause, "{"); start != -1 {
		if end := strings.Index(clause[start+1:], "}"); end != -1 {
			for _, item := range strings.Split(clause[start+1:start+1+end], ",") {
				item = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(item), "type "))
				if item == "" {
					continue
				}
				parts := strings.Fields(item)
				if len(parts) >= 3 && parts[1] == "as" {
					record(parts[2], parts[0])
				} else {
					record(parts[0], parts[0])
				}
			}
		}
		clause = strings.TrimSpace(clause[:start])
		clause = strings.TrimSpace(strings.TrimSuffix(clause, ","))
	}
	if strings.HasPrefix(clause, "*") {
		parts := strings.Fields(clause)
		if len(parts) >= 3 && parts[1] == "as" {
			parsed.ImportAliases[parts[2]] = importNamespace
		}
		return
	}
	if clause != "" {
		record(clause, clause)
	}
}

func (p *TSParser) parseHeritage(heritage, context string, lineNum int, parsed *models.ParsedFile) {
	if matches := p.extendsPattern.FindStringSubmatch(heritage); matches != nil {
		parsed.Usage = append(parsed.Usage, models.UsageElement{
			Type:    "extends",
			Name:    normalizeTSQualifiedName(matches[1]),
			Context: context,
			Line:    lineNum,
		})
	}
	if matches := p.implementsPattern.FindStringSubmatch(heritage); matches != nil {
		for _, name := range splitTopLevelTS(matches[1], ',') {
			name = strings.TrimSpace(name)
			if idx := strings.Index(name, "<"); idx != -1 {
				name = name[:idx]
			}
			if name != "" {
				parsed.Usage = append(parsed.Usage, models.UsageElement{
					Type:    "implements",
					Name:    normalizeTSQualifiedName(name),
					Context: context,
					Line:    lineNum,
				})
			}
		}
	}
}

func (p *TSParser) parseInterfaceHeritage(heritage, context string, lineNum int, parsed *models.ParsedFile) {
	if matches := p.extendsPattern.FindStringSubmatch(heritage); matches != nil {
		start := strings.Index(heritage, matches[1])
		if start == -1 {
			return
		}
		for _, name := range splitTopLevelTS(heritage[start:], ',') {
			name = strings.TrimSpace(name)
			if idx := strings.Index(name, "<"); idx != -1 {
				name = name[:idx]
			}
			if name != "" {
				parsed.Usage = append(parsed.Usage, models.UsageElement{
					Type:    "extends",
					Name:    normalizeTSQualifiedName(name),
					Context: context,
					Line:    lineNum,
				})
			}
		}
	}
}

func (p *TSParser) parseUsage(line string, lineNum int, context, declaredName string, parsed *models.ParsedFile) {
	for _, match := range p.newInstancePattern.FindAllStringSubmatch(line, -1) {
		parsed.Usage = append(parsed.Usage, models.UsageElement{
			Type:    "instantiation",
			Name:    normalizeTSQualifiedName(match[1]),
			Context: context,
			Line:    lineNum,
		})
	}

	for _, match := range p.callPattern.FindAllStringSubmatch(line, -1) {
		if match[1] == "this" || match[1] == "super" {
			continue
		}
		parsed.Usage = append(parsed.Usage, models.UsageElement{
			Type:    "method_call",
			Name:    match[1] + "::" + match[2],
			Context: context,
			Line:    lineNum,
		})
	}

	for _, index := range p.functionCallPattern.FindAllStringSubmatchIndex(line, -1) {
		name := line[index[2]:index[3]]
		if name == declaredName || tsControlKeyword(name) || name == "require" || name == "import" || name == "function" {
			continue
		}
		prefix := strings.TrimSpace(line[:index[0]])
		if strings.HasSuffix(prefix, ".") || strings.HasSuffix(prefix, "new") || strings.HasSuffix(prefix, "function") {
			continue
		}
		parsed.Usage = append(parsed.Usage, models.UsageElement{
			Type:    "function_call",
			Name:    name,
			Context: context,
			Line:    lineNum,
		})
	}

	for _, match := range p.jsxPattern.FindAllStringSubmatch(line, -1) {
		parsed.Usage = append(parsed.Usage, models.UsageElement{
			Type:    "jsx",
			Name:    normalizeTSQualifiedName(match[1]),
			Context: context,
			Line:    lineNum,
		})
	}
}

func parseTSParameters(parameters string) []string {
	var result []string
	for _, parameter := range splitTopLevelTS(parameters, ',') {
		parameter = strings.TrimSpace(parameter)
		if parameter == "" {
			continue
		}
		if idx := strings.Index(parameter, "="); idx != -1 {
			parameter = strings.TrimSpace(parameter[:idx])
		}
		for _, modifier := range []string{"public ", "private ", "protected ", "readonly ", "override "} {
			parameter = strings.TrimPrefix(parameter, modifier)
		}
		parameter = strings.TrimPrefix(parameter, "...")
		if idx := strings.Index(parameter, ":"); idx != -1 {
			parameter = strings.TrimSpace(parameter[:idx])
		}
		parameter = strings.TrimSuffix(strings.TrimSpace(parameter), "?")
		if parameter == "" {
			continue
		}
		if !regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`).MatchString(parameter) {
			parameter = "_"
		}
		result = append(result, parameter)
	}
	return result
}

func splitTopLevelTS(value string, separator rune) []string {
	var result []string
	start := 0
	angleDepth, parenDepth, bracketDepth, braceDepth := 0, 0, 0, 0
	for index, char := range value {
		switch char {
		case '<':
			angleDepth++
		case '>':
			if angleDepth > 0 {
				angleDepth--
			}
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		default:
			if char == separator && angleDepth == 0 && parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
				result = append(result, value[start:index])
				start = index + len(string(char))
			}
		}
	}
	result = append(result, value[start:])
	return result
}

func tsVisibility(modifiers string) string {
	for _, visibility := range []string{"public", "private", "protected"} {
		if hasTSModifier(modifiers, visibility) {
			return visibility
		}
	}
	return "public"
}

func hasTSModifier(modifiers, modifier string) bool {
	for _, field := range strings.Fields(modifiers) {
		if field == modifier {
			return true
		}
	}
	return false
}

func tsControlKeyword(name string) bool {
	switch name {
	case "if", "for", "while", "switch", "catch", "with", "return", "throw", "new", "typeof", "instanceof", "await", "super":
		return true
	default:
		return false
	}
}

func normalizeTSQualifiedName(name string) string {
	return strings.ReplaceAll(strings.TrimSpace(name), ".", `\`)
}

// ProcessFiles parses multiple TypeScript files concurrently.
func (p *TSParser) ProcessFiles(files []models.FileInfo, progressBar *progress.ProgressBar) ([]*models.ParsedFile, error) {
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

			if err != nil {
				parsed = failedParsedFile(f, err)
			}
			parsedFiles = append(parsedFiles, parsed)
			progressBar.Update(1)
		}(file)
	}

	wg.Wait()
	progressBar.Finish()
	return parsedFiles, nil
}

// Language returns the language name for this parser.
func (p *TSParser) Language() string {
	return "ts"
}

// FileExtensions returns the file extensions supported by this parser.
func (p *TSParser) FileExtensions() []string {
	return []string{".ts", ".tsx", ".mts", ".cts"}
}

func init() {
	parser.Register(NewTSParser())
}
