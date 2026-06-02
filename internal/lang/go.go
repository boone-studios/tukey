// Copyright (c) 2026 Boone Studios
// SPDX-License-Identifier: MIT

package lang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/boone-studios/tukey/internal/models"
	tparser "github.com/boone-studios/tukey/internal/parser"
	"github.com/boone-studios/tukey/internal/progress"
)

// GoParser handles parsing of Go files using standard AST library
type GoParser struct{}

// NewGoParser creates a new Go parser
func NewGoParser() *GoParser {
	return &GoParser{}
}

// Language returns the language name for this parser
func (p *GoParser) Language() string {
	return "go"
}

// FileExtensions returns the file extensions supported by this parser
func (p *GoParser) FileExtensions() []string {
	return []string{".go"}
}

var goBuiltins = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"float32": true, "float64": true, "complex64": true, "complex128": true,
	"byte": true, "rune": true, "string": true, "bool": true, "error": true,
	"any": true, "comparable": true, "nil": true, "true": true, "false": true, "iota": true,
	"append": true, "cap": true, "close": true, "complex": true, "copy": true,
	"delete": true, "imag": true, "len": true, "make": true, "new": true,
	"panic": true, "print": true, "println": true, "real": true, "recover": true,
}

// ParseFile parses a single Go file and extracts all elements and usages
func (p *GoParser) ParseFile(filePath string, moduleName string, relativePath string) (*models.ParsedFile, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(relativePath)
	var pkgPath string
	if moduleName == "" {
		if dir == "." || dir == "" {
			pkgPath = node.Name.Name
		} else {
			pkgPath = filepath.ToSlash(dir)
		}
	} else {
		if dir == "." || dir == "" {
			pkgPath = moduleName
		} else {
			pkgPath = moduleName + "/" + filepath.ToSlash(dir)
		}
	}

	parsed := &models.ParsedFile{
		Path:      filePath,
		Namespace: pkgPath,
		Elements:  []models.CodeElement{},
		Usage:     []models.UsageElement{},
		Uses:      []string{},
	}

	// 1. Extract uses and build imports map
	importsMap := make(map[string]string)
	for _, spec := range node.Imports {
		path := strings.Trim(spec.Path.Value, "\"")
		parsed.Uses = append(parsed.Uses, path)

		var alias string
		if spec.Name != nil {
			alias = spec.Name.Name
		} else {
			alias = getDefaultPackageName(path)
		}
		if alias != "" {
			importsMap[alias] = path
		}
	}

	// 2. Extract defined elements
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						line := fset.Position(ts.Pos()).Line
						var isAbstract bool
						elType := "class" // Struct is mapped to class

						if _, isStruct := ts.Type.(*ast.StructType); isStruct {
							elType = "class"
						} else if _, isInterface := ts.Type.(*ast.InterfaceType); isInterface {
							elType = "interface"
							isAbstract = true
						}

						element := models.CodeElement{
							Type:       elType,
							Name:       ts.Name.Name,
							Namespace:  pkgPath,
							Line:       line,
							File:       filePath,
							IsAbstract: isAbstract,
						}
						parsed.Elements = append(parsed.Elements, element)

						// For structs, also extract fields as properties
						if st, ok := ts.Type.(*ast.StructType); ok && st.Fields != nil {
							for _, field := range st.Fields.List {
								if len(field.Names) > 0 {
									fieldLine := fset.Position(field.Pos()).Line
									for _, name := range field.Names {
										visibility := "private"
										if ast.IsExported(name.Name) {
											visibility = "public"
										}
										propElement := models.CodeElement{
											Type:       "property",
											Name:       name.Name,
											Namespace:  pkgPath,
											ClassName:  ts.Name.Name,
											Line:       fieldLine,
											File:       filePath,
											Visibility: visibility,
										}
										parsed.Elements = append(parsed.Elements, propElement)
									}
								}
							}
						}
					}
				}
			case token.CONST:
				for _, spec := range d.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						line := fset.Position(vs.Pos()).Line
						for _, name := range vs.Names {
							visibility := "private"
							if ast.IsExported(name.Name) {
								visibility = "public"
							}
							element := models.CodeElement{
								Type:       "constant",
								Name:       name.Name,
								Namespace:  pkgPath,
								Line:       line,
								File:       filePath,
								Visibility: visibility,
							}
							parsed.Elements = append(parsed.Elements, element)
						}
					}
				}
			}

		case *ast.FuncDecl:
			line := fset.Position(d.Pos()).Line
			var params []string
			if d.Type.Params != nil {
				for _, f := range d.Type.Params.List {
					if len(f.Names) == 0 {
						params = append(params, "_")
					} else {
						for _, name := range f.Names {
							params = append(params, name.Name)
						}
					}
				}
			}

			var returnType string
			if d.Type.Results != nil && len(d.Type.Results.List) > 0 {
				returnType = getTypeNameString(d.Type.Results.List[0].Type)
			}

			if d.Recv == nil {
				// Standalone function
				element := models.CodeElement{
					Type:       "function",
					Name:       d.Name.Name,
					Namespace:  pkgPath,
					Line:       line,
					File:       filePath,
					Parameters: params,
					ReturnType: returnType,
				}
				parsed.Elements = append(parsed.Elements, element)
			} else {
				// Method
				className := getReceiverTypeName(d.Recv)
				element := models.CodeElement{
					Type:       "method",
					Name:       d.Name.Name,
					Namespace:  pkgPath,
					ClassName:  className,
					Line:       line,
					File:       filePath,
					Parameters: params,
					ReturnType: returnType,
				}
				parsed.Elements = append(parsed.Elements, element)
			}
		}
	}

	// 3. Extract usages using walker
	walker := &goASTWalker{
		fset:           fset,
		parsed:         parsed,
		importsMap:     importsMap,
		currentPkgPath: pkgPath,
		localTypes:     make(map[string]string),
	}
	ast.Walk(walker, node)

	return parsed, nil
}

// ProcessFiles parses multiple Go files concurrently
func (p *GoParser) ProcessFiles(files []models.FileInfo, progressBar *progress.ProgressBar) ([]*models.ParsedFile, error) {
	var parsedFiles []*models.ParsedFile
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)

	var moduleName string
	if len(files) > 0 {
		rootPath := strings.TrimSuffix(files[0].Path, files[0].RelativePath)
		moduleName = getModuleName(rootPath)
	}

	for _, file := range files {
		wg.Add(1)
		go func(f models.FileInfo) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			parsed, err := p.ParseFile(f.Path, moduleName, f.RelativePath)
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

// Helpers

func getModuleName(rootPath string) string {
	data, err := os.ReadFile(filepath.Join(rootPath, "go.mod"))
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func getDefaultPackageName(importPath string) string {
	parts := strings.Split(importPath, "/")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if len(parts) > 1 && (strings.HasPrefix(last, "v") && isNumeric(last[1:])) {
		last = parts[len(parts)-2]
	}
	if idx := strings.Index(last, "."); idx != -1 {
		last = last[:idx]
	}
	return last
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

func getReceiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	return getBaseTypeName(t)
}

func getBaseTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return getBaseTypeName(t.X)
	case *ast.IndexExpr:
		return getBaseTypeName(t.X)
	case *ast.IndexListExpr:
		return getBaseTypeName(t.X)
	}
	return ""
}

func getTypeNameString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return getTypeNameString(t.X)
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
	case *ast.IndexExpr:
		return getTypeNameString(t.X)
	case *ast.IndexListExpr:
		return getTypeNameString(t.X)
	}
	return ""
}

func getFuncName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
	}
	return ""
}

func inferType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.CompositeLit:
		return getTypeNameString(t.Type)
	case *ast.UnaryExpr:
		if t.Op == token.AND {
			return inferType(t.X)
		}
	case *ast.CallExpr:
		funcName := getFuncName(t.Fun)
		if funcName != "" {
			idx := strings.LastIndex(funcName, "New")
			if idx != -1 && idx+3 < len(funcName) {
				typeName := funcName[idx+3:]
				if pkgIdx := strings.LastIndex(funcName, "."); pkgIdx != -1 {
					pkg := funcName[:pkgIdx]
					if strings.Contains(funcName[pkgIdx:], "New") {
						return pkg + "." + typeName
					}
				}
				return typeName
			}
		}
	}
	return ""
}

// goASTWalker implements ast.Visitor for extracting dependencies
type goASTWalker struct {
	fset           *token.FileSet
	parsed         *models.ParsedFile
	importsMap     map[string]string
	currentPkgPath string
	currentContext string
	localTypes     map[string]string
}

func (w *goASTWalker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return w
	}

	switch n := node.(type) {
	case *ast.FuncDecl:
		oldContext := w.currentContext
		oldLocalTypes := w.localTypes

		w.currentContext = n.Name.Name
		w.localTypes = make(map[string]string)

		if n.Recv != nil && len(n.Recv.List) > 0 {
			recvName := ""
			if len(n.Recv.List[0].Names) > 0 {
				recvName = n.Recv.List[0].Names[0].Name
			}
			recvType := getBaseTypeName(n.Recv.List[0].Type)
			if recvName != "" && recvType != "" {
				w.localTypes[recvName] = recvType
			}
		}

		if n.Type.Params != nil {
			for _, field := range n.Type.Params.List {
				paramType := getTypeNameString(field.Type)
				if paramType != "" {
					for _, name := range field.Names {
						w.localTypes[name.Name] = paramType
					}
				}
			}
		}

		if n.Body != nil {
			ast.Walk(w, n.Body)
		}

		w.currentContext = oldContext
		w.localTypes = oldLocalTypes
		return nil

	case *ast.TypeSpec:
		if _, ok := n.Type.(*ast.StructType); ok {
			oldContext := w.currentContext
			w.currentContext = n.Name.Name
			ast.Walk(w, n.Type)
			w.currentContext = oldContext
			return nil
		}

	case *ast.AssignStmt:
		if n.Tok == token.DEFINE {
			for i, lhs := range n.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					if i < len(n.Rhs) {
						rhsType := inferType(n.Rhs[i])
						if rhsType != "" {
							w.localTypes[ident.Name] = rhsType
						}
					}
				}
			}
		}

	case *ast.DeclStmt:
		if genDecl, ok := n.Decl.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
			for _, spec := range genDecl.Specs {
				if valSpec, ok := spec.(*ast.ValueSpec); ok {
					if valSpec.Type != nil {
						typeName := getTypeNameString(valSpec.Type)
						if typeName != "" {
							for _, name := range valSpec.Names {
								w.localTypes[name.Name] = typeName
							}
						}
					}
				}
			}
		}

	case *ast.CallExpr:
		w.handleCallExpr(n)

	case *ast.SelectorExpr:
		w.handleSelectorExpr(n, false)
		return nil

	case *ast.Ident:
		w.handleIdent(n, false)
	}

	return w
}

func (w *goASTWalker) handleCallExpr(n *ast.CallExpr) {
	for _, arg := range n.Args {
		ast.Walk(w, arg)
	}

	switch fun := n.Fun.(type) {
	case *ast.Ident:
		if !goBuiltins[fun.Name] {
			refType := "function"
			if fun.Obj != nil && fun.Obj.Kind == ast.Typ {
				refType = "class"
			}
			w.recordUsage(refType, w.currentPkgPath+"\\"+fun.Name, fun.Pos())
		}
	case *ast.SelectorExpr:
		w.handleSelectorExpr(fun, true)
	default:
		ast.Walk(w, n.Fun)
	}
}

func (w *goASTWalker) handleSelectorExpr(n *ast.SelectorExpr, isCall bool) {
	if ident, ok := n.X.(*ast.Ident); ok {
		if importPath, ok := w.importsMap[ident.Name]; ok {
			refType := "class"
			if isCall {
				refType = "function"
			}
			w.recordUsage(refType, importPath+"\\"+n.Sel.Name, n.Sel.Pos())
			return
		}

		if typeName, ok := w.localTypes[ident.Name]; ok {
			var targetPkg, structName string
			if idx := strings.Index(typeName, "."); idx != -1 {
				pkgPrefix := typeName[:idx]
				structName = typeName[idx+1:]
				if imp, exists := w.importsMap[pkgPrefix]; exists {
					targetPkg = imp
				} else {
					targetPkg = pkgPrefix
				}
			} else {
				targetPkg = w.currentPkgPath
				structName = typeName
			}

			if isCall {
				w.recordUsage("method_call", targetPkg+"\\"+structName+"::"+n.Sel.Name, n.Sel.Pos())
			} else {
				w.recordUsage("property", targetPkg+"\\"+structName+"::"+n.Sel.Name, n.Sel.Pos())
			}
			return
		}
	}

	ast.Walk(w, n.X)

	refType := "property"
	if isCall {
		refType = "method_call"
	}
	w.recordUsage(refType, n.Sel.Name, n.Sel.Pos())
}

func (w *goASTWalker) handleIdent(n *ast.Ident, isCall bool) {
	if goBuiltins[n.Name] {
		return
	}

	if _, ok := w.importsMap[n.Name]; ok {
		return
	}

	if n.Obj != nil {
		if n.Obj.Kind == ast.Typ || n.Obj.Kind == ast.Fun || n.Obj.Kind == ast.Con {
			refType := "class"
			if n.Obj.Kind == ast.Fun {
				refType = "function"
			} else if n.Obj.Kind == ast.Con {
				refType = "constant"
			}
			w.recordUsage(refType, w.currentPkgPath+"\\"+n.Name, n.Pos())
		}
	} else {
		w.recordUsage("class", w.currentPkgPath+"\\"+n.Name, n.Pos())
	}
}

func (w *goASTWalker) recordUsage(refType, targetName string, pos token.Pos) {
	line := w.fset.Position(pos).Line
	w.parsed.Usage = append(w.parsed.Usage, models.UsageElement{
		Type:    refType,
		Name:    targetName,
		Context: w.currentContext,
		Line:    line,
	})
}

func init() {
	tparser.Register(NewGoParser())
}
