// Copyright (c) 2026 Boone Studios
// SPDX-License-Identifier: MIT

package lang

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/boone-studios/tukey/internal/analyzer"
	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/internal/parser"
	"github.com/boone-studios/tukey/internal/progress"
)

func writeTS(t *testing.T, dir, name, code string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create TypeScript fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write TypeScript fixture: %v", err)
	}
	return path
}

func findTSElement(elements []models.CodeElement, elementType, name string) *models.CodeElement {
	for index := range elements {
		if elements[index].Type == elementType && elements[index].Name == name {
			return &elements[index]
		}
	}
	return nil
}

func findTSUsage(usages []models.UsageElement, usageType, name, context string) bool {
	for _, usage := range usages {
		if usage.Type == usageType && usage.Name == name && usage.Context == context {
			return true
		}
	}
	return false
}

func TestTSParser_RegistrationAndExtensions(t *testing.T) {
	p := NewTSParser()
	if p.Language() != "ts" {
		t.Fatalf("Language() = %q, want ts", p.Language())
	}
	for _, extension := range []string{".ts", ".tsx", ".mts", ".cts"} {
		found := false
		for _, supported := range p.FileExtensions() {
			if supported == extension {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing TypeScript extension %q", extension)
		}
	}
	registered, ok := parser.Get("ts")
	if !ok || registered.Language() != "ts" {
		t.Fatal("TypeScript parser was not registered")
	}
}

func TestTSParser_TypedDeclarationsAndHeritage(t *testing.T) {
	tmp := t.TempDir()
	code := `import type { Entity as BaseEntity } from './models/Entity';
import { Logger } from './Logger';

export interface Repository<T> extends BaseRepository<T>, Disposable {
    readonly ready: boolean;
    find(id: string): Promise<T>;
}

export type UserID = string;
export enum Role { Admin, User }

export abstract class UserService extends BaseService implements Repository<User>, Disposable {
    private readonly client: HttpClient;
    protected static instances = 0;

    constructor(client: HttpClient) {
        super();
        this.client = client;
    }

    public async getUser<T extends Entity>(id: string, options?: LoadOptions): Promise<User> {
        const logger = new Logger();
        return this.client.get(id);
    }

    abstract dispose(): void;
}

export function formatUser<T extends { name: string }>(user: User, prefix?: string): string {
    return prefix + user.name;
}

export const makeUser = async (id: string): Promise<User> => {
    const service = new UserService(client);
    return service.getUser(id);
};
`
	path := writeTS(t, tmp, "UserService.ts", code)
	parsed, err := NewTSParser().ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	for _, expected := range []struct {
		elementType string
		name        string
	}{
		{"interface", "Repository"},
		{"type", "UserID"},
		{"enum", "Role"},
		{"class", "UserService"},
		{"method", "getUser"},
		{"method", "dispose"},
		{"property", "client"},
		{"property", "instances"},
		{"function", "formatUser"},
		{"function", "makeUser"},
	} {
		if findTSElement(parsed.Elements, expected.elementType, expected.name) == nil {
			t.Errorf("missing %s %s; elements: %+v", expected.elementType, expected.name, parsed.Elements)
		}
	}

	class := findTSElement(parsed.Elements, "class", "UserService")
	if class == nil || !class.IsAbstract {
		t.Error("expected UserService to be abstract")
	}
	method := findTSElement(parsed.Elements, "method", "getUser")
	if method == nil {
		t.Fatal("missing getUser method")
	}
	if method.ClassName != "UserService" || method.Visibility != "public" {
		t.Errorf("unexpected getUser context: %+v", method)
	}
	if !reflect.DeepEqual(method.Parameters, []string{"id", "options"}) || method.ReturnType != "Promise<User>" {
		t.Errorf("unexpected getUser signature: %+v", method)
	}
	property := findTSElement(parsed.Elements, "property", "instances")
	if property == nil || !property.IsStatic || property.Visibility != "protected" {
		t.Errorf("unexpected static property: %+v", property)
	}
	function := findTSElement(parsed.Elements, "function", "formatUser")
	if function == nil || !reflect.DeepEqual(function.Parameters, []string{"user", "prefix"}) || function.ReturnType != "string" {
		t.Errorf("unexpected formatUser signature: %+v", function)
	}

	if got := parsed.ImportAliases["BaseEntity"]; got != filepath.Join(tmp, "models")+`\Entity` {
		t.Errorf("BaseEntity alias = %q", got)
	}
	if got := parsed.ImportAliases["Logger"]; got != tmp+`\Logger` {
		t.Errorf("Logger alias = %q", got)
	}
	for _, expected := range []struct {
		usageType string
		name      string
		context   string
	}{
		{"extends", "BaseRepository", "Repository"},
		{"extends", "Disposable", "Repository"},
		{"extends", "BaseService", "UserService"},
		{"implements", "Repository", "UserService"},
		{"implements", "Disposable", "UserService"},
		{"instantiation", "Logger", "UserService::getUser"},
		{"instantiation", "UserService", "makeUser"},
		{"method_call", "service::getUser", "makeUser"},
	} {
		if !findTSUsage(parsed.Usage, expected.usageType, expected.name, expected.context) {
			t.Errorf("missing usage %+v; usages: %+v", expected, parsed.Usage)
		}
	}
}

func TestTSParser_TSXComponents(t *testing.T) {
	tmp := t.TempDir()
	code := `import React from 'react';
import { Button as UIButton } from './Button';
import { formatUser } from './user';

type Props = { user: User };

export function UserCard({ user }: Props): JSX.Element {
    const label = formatUser(user);
    return (
        <section>
            <UIButton label={label} />
        </section>
    );
}

export const Avatar = ({ user }: Props) => <UserCard user={user} />;
`
	path := writeTS(t, tmp, "UserCard.tsx", code)
	parsed, err := NewTSParser().ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	for _, name := range []string{"UserCard", "Avatar"} {
		if findTSElement(parsed.Elements, "function", name) == nil {
			t.Errorf("missing component function %s", name)
		}
	}
	if !findTSUsage(parsed.Usage, "function_call", "formatUser", "UserCard") {
		t.Errorf("missing formatUser call: %+v", parsed.Usage)
	}
	if !findTSUsage(parsed.Usage, "jsx", "UIButton", "UserCard") {
		t.Errorf("missing UIButton JSX reference: %+v", parsed.Usage)
	}
	if !findTSUsage(parsed.Usage, "jsx", "UserCard", "Avatar") {
		t.Errorf("missing UserCard JSX reference: %+v", parsed.Usage)
	}
}

func TestTSParser_ProcessFilesConcurrently(t *testing.T) {
	tmp := t.TempDir()
	writeTS(t, tmp, "One.ts", "export class One {}")
	writeTS(t, tmp, "Two.tsx", "export const Two = () => <One />;")
	files := []models.FileInfo{
		{Path: filepath.Join(tmp, "One.ts"), RelativePath: "One.ts"},
		{Path: filepath.Join(tmp, "Two.tsx"), RelativePath: "Two.tsx"},
	}

	parsed, err := NewTSParser().ProcessFiles(files, progress.NewProgressBar(len(files), "Testing TS parser"))
	if err != nil {
		t.Fatalf("ProcessFiles error: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 parsed files, got %d", len(parsed))
	}
}

func TestTSParser_ImportReferencesBuildGraphEdges(t *testing.T) {
	tmp := t.TempDir()
	loggerPath := writeTS(t, tmp, "Logger.ts", "export class Logger {}\n")
	servicePath := writeTS(t, tmp, "Service.ts", `import { Logger } from './Logger';
export class Service {
    run(): void {
        const logger = new Logger();
    }
}
`)
	cardPath := writeTS(t, tmp, "Card.tsx", `import { Service } from './Service';
export function Card() {
    return <Service />;
}
`)

	p := NewTSParser()
	var parsedFiles []*models.ParsedFile
	for _, path := range []string{loggerPath, servicePath, cardPath} {
		parsed, err := p.ParseFile(path)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", path, err)
		}
		parsedFiles = append(parsedFiles, parsed)
	}

	graph := analyzer.NewDependencyTracker().BuildDependencyGraph(parsedFiles)
	serviceID := models.StableSymbolID("class", tmp, "", "Service")
	service := graph.Nodes[serviceID]
	if service == nil {
		t.Fatalf("missing Service node; nodes: %+v", graph.Nodes)
	}
	loggerID := models.StableSymbolID("class", tmp, "", "Logger")
	runID := models.StableSymbolID("method", tmp, "Service", "run")
	if graph.Nodes[runID] == nil || graph.Nodes[runID].Dependencies[loggerID] == nil {
		t.Errorf("Service.run should depend on imported Logger; diagnostics: %+v", graph.Diagnostics)
	}
	cardID := models.StableSymbolID("function", tmp, "", "Card")
	if graph.Nodes[cardID] == nil || graph.Nodes[cardID].Dependencies[serviceID] == nil {
		t.Errorf("Card should depend on Service through JSX; card: %+v, parsed: %+v, diagnostics: %+v", graph.Nodes[cardID], parsedFiles[2], graph.Diagnostics)
	}
}
