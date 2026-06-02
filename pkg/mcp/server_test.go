// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/boone-studios/tukey/internal/models"
	"github.com/boone-studios/tukey/pkg/query"
)

// buildTestEngine returns an Engine backed by a synthetic graph for unit tests.
func buildTestEngine() *query.Engine {
	gateway := &models.DependencyNode{
		ID:        "class:App\\Factories\\GatewayFactory:15",
		Name:      "GatewayFactory",
		Type:      "class",
		File:      "app/Factories/GatewayFactory.php",
		Namespace: "App\\Factories",
		Line:      15,
		Score:     12,
		Dependencies: map[string]*models.DependencyRef{
			"class:App\\Http\\Client:8": {
				TargetID:   "class:App\\Http\\Client:8",
				TargetName: "HttpClient",
				Type:       "instantiation",
				Count:      1,
				Lines:      []int{20},
			},
		},
		Dependents: map[string]*models.DependencyRef{
			"class:App\\Services\\PaymentService:5": {
				TargetID:   "class:App\\Services\\PaymentService:5",
				TargetName: "PaymentService",
				Type:       "instantiation",
				Count:      2,
				Lines:      []int{30, 45},
			},
		},
	}
	httpClient := &models.DependencyNode{
		ID:           "class:App\\Http\\Client:8",
		Name:         "HttpClient",
		Type:         "class",
		File:         "app/Http/Client.php",
		Namespace:    "App\\Http",
		Line:         8,
		Score:        5,
		Dependencies: map[string]*models.DependencyRef{},
		Dependents: map[string]*models.DependencyRef{
			"class:App\\Factories\\GatewayFactory:15": {
				TargetID:   "class:App\\Factories\\GatewayFactory:15",
				TargetName: "GatewayFactory",
				Type:       "instantiation",
				Count:      1,
				Lines:      []int{20},
			},
		},
	}
	payment := &models.DependencyNode{
		ID:        "class:App\\Services\\PaymentService:5",
		Name:      "PaymentService",
		Type:      "class",
		File:      "app/Services/PaymentService.php",
		Namespace: "App\\Services",
		Line:      5,
		Score:     18,
		Dependencies: map[string]*models.DependencyRef{
			"class:App\\Factories\\GatewayFactory:15": {
				TargetID:   "class:App\\Factories\\GatewayFactory:15",
				TargetName: "GatewayFactory",
				Type:       "instantiation",
				Count:      2,
				Lines:      []int{30, 45},
			},
		},
		Dependents: map[string]*models.DependencyRef{},
	}
	orphan := &models.DependencyNode{
		ID:           "function:deadHelper:99",
		Name:         "deadHelper",
		Type:         "function",
		File:         "app/helpers.php",
		Line:         99,
		Score:        3,
		Dependencies: map[string]*models.DependencyRef{},
		Dependents:   map[string]*models.DependencyRef{},
	}

	graph := &models.DependencyGraph{
		Nodes: map[string]*models.DependencyNode{
			gateway.ID:    gateway,
			httpClient.ID: httpClient,
			payment.ID:    payment,
			orphan.ID:     orphan,
		},
		TotalNodes: 4,
		TotalEdges: 3,
		Orphans:    []*models.DependencyNode{orphan},
	}

	// Envelope for loading
	type analysisEnvelope struct {
		Graph *models.DependencyGraph `json:"graph"`
	}
	env := analysisEnvelope{Graph: graph}
	data, err := json.Marshal(env)
	if err != nil {
		panic(err)
	}

	return parseEngineFromJSON(data)
}

func parseEngineFromJSON(jsonData []byte) *query.Engine {
	importFile := "temp_test_analysis.json"
	writeErr := osWriteFile(importFile, jsonData, 0644)
	if writeErr != nil {
		panic(writeErr)
	}
	defer osRemove(importFile)

	engine, err := query.Load(importFile)
	if err != nil {
		panic(err)
	}
	return engine
}

func osWriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func osRemove(name string) error {
	return os.Remove(name)
}

func TestMCPServer_Initialize(t *testing.T) {
	engine := buildTestEngine()
	in := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"initialize","id":1}` + "\n")
	var out bytes.Buffer

	srv := NewServer(engine, in, &out)
	err := srv.Start()
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("jsonrpc field: got %q want %q", resp.JSONRPC, "2.0")
	}

	if resp.ID.(float64) != 1 {
		t.Errorf("id field: got %v want 1", resp.ID)
	}

	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Result is not a map: %T", resp.Result)
	}

	if resultMap["protocolVersion"] != "2024-11-05" {
		t.Errorf("protocolVersion: got %v want 2024-11-05", resultMap["protocolVersion"])
	}
}

func TestMCPServer_ToolsList(t *testing.T) {
	engine := buildTestEngine()
	in := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"tools/list","id":42}` + "\n")
	var out bytes.Buffer

	srv := NewServer(engine, in, &out)
	err := srv.Start()
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Result is not a map: %T", resp.Result)
	}

	tools, ok := resultMap["tools"].([]interface{})
	if !ok {
		t.Fatalf("tools is not a slice: %T", resultMap["tools"])
	}

	if len(tools) != 5 {
		t.Errorf("expected 5 tools, got %d", len(tools))
	}
}

func TestMCPServer_ToolCalls(t *testing.T) {
	engine := buildTestEngine()

	tests := []struct {
		name       string
		request    string
		wantResult string
	}{
		{
			name:       "Find Symbol exact",
			request:    `{"jsonrpc":"2.0","method":"tools/call","id":10,"params":{"name":"tukey_find_symbol","arguments":{"term":"GatewayFactory"}}}` + "\n",
			wantResult: `"query": "find"`,
		},
		{
			name:       "Find Symbol substring",
			request:    `{"jsonrpc":"2.0","method":"tools/call","id":11,"params":{"name":"tukey_find_symbol","arguments":{"term":"gateway"}}}` + "\n",
			wantResult: `"GatewayFactory"`,
		},
		{
			name:       "Get Callers",
			request:    `{"jsonrpc":"2.0","method":"tools/call","id":12,"params":{"name":"tukey_get_callers","arguments":{"symbol":"GatewayFactory"}}}` + "\n",
			wantResult: `"PaymentService"`,
		},
		{
			name:       "Get Dependents",
			request:    `{"jsonrpc":"2.0","method":"tools/call","id":13,"params":{"name":"tukey_get_dependents","arguments":{"symbol":"GatewayFactory"}}}` + "\n",
			wantResult: `"HttpClient"`,
		},
		{
			name:       "Find Orphans",
			request:    `{"jsonrpc":"2.0","method":"tools/call","id":14,"params":{"name":"tukey_find_orphans","arguments":{}}}` + "\n",
			wantResult: `"deadHelper"`,
		},
		{
			name:       "Get Localized Context",
			request:    `{"jsonrpc":"2.0","method":"tools/call","id":15,"params":{"name":"tukey_get_localized_context","arguments":{"symbol":"GatewayFactory","depth":1}}}` + "\n",
			wantResult: `"HttpClient"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := bytes.NewBufferString(tt.request)
			var out bytes.Buffer

			srv := NewServer(engine, in, &out)
			err := srv.Start()
			if err != nil {
				t.Fatalf("Start returned error: %v", err)
			}

			var resp JSONRPCResponse
			if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if resp.Error != nil {
				t.Fatalf("Received RPC error: %s", resp.Error.Message)
			}

			resultMap, ok := resp.Result.(map[string]interface{})
			if !ok {
				t.Fatalf("Result is not a map: %T", resp.Result)
			}

			content, ok := resultMap["content"].([]interface{})
			if !ok {
				t.Fatalf("content is not a slice: %T", resultMap["content"])
			}

			if len(content) == 0 {
				t.Fatal("content slice is empty")
			}

			textMap := content[0].(map[string]interface{})
			text := textMap["text"].(string)

			if !strings.Contains(text, tt.wantResult) {
				t.Errorf("Result text: got %q want substring %q", text, tt.wantResult)
			}
		})
	}
}

func TestMCPServer_Errors(t *testing.T) {
	engine := buildTestEngine()

	tests := []struct {
		name     string
		request  string
		wantCode int
	}{
		{
			name:     "Invalid JSON",
			request:  `{invalid-json` + "\n",
			wantCode: ParseError,
		},
		{
			name:     "Invalid Request Version",
			request:  `{"jsonrpc":"1.0","method":"initialize","id":1}` + "\n",
			wantCode: InvalidRequest,
		},
		{
			name:     "Method Not Found",
			request:  `{"jsonrpc":"2.0","method":"nonexistent_method","id":2}` + "\n",
			wantCode: MethodNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := bytes.NewBufferString(tt.request)
			var out bytes.Buffer

			srv := NewServer(engine, in, &out)
			err := srv.Start()
			if err != nil {
				t.Fatalf("Start returned error: %v", err)
			}

			var resp JSONRPCResponse
			if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if resp.Error == nil {
				t.Fatal("Expected RPC error, got nil")
			}

			if resp.Error.Code != tt.wantCode {
				t.Errorf("Error code: got %d want %d (%s)", resp.Error.Code, tt.wantCode, resp.Error.Message)
			}
		})
	}
}
