// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	if len(tools) != 7 {
		t.Errorf("expected 7 tools, got %d", len(tools))
	}
	for _, rawTool := range tools {
		tool := rawTool.(map[string]interface{})
		schema := tool["inputSchema"].(map[string]interface{})
		properties := schema["properties"].(map[string]interface{})
		if properties["limit"] == nil || properties["cursor"] == nil {
			t.Errorf("tool %s lacks pagination properties", tool["name"])
		}
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
			name:       "Get Analysis Diagnostics",
			request:    `{"jsonrpc":"2.0","method":"tools/call","id":15,"params":{"name":"tukey_get_analysis_diagnostics","arguments":{}}}` + "\n",
			wantResult: `"query": "diagnostics"`,
		},
		{
			name:       "Get Analysis Status",
			request:    `{"jsonrpc":"2.0","method":"tools/call","id":16,"params":{"name":"tukey_get_analysis_status","arguments":{}}}` + "\n",
			wantResult: `"status": "unknown"`,
		},
		{
			name:       "Get Localized Context",
			request:    `{"jsonrpc":"2.0","method":"tools/call","id":17,"params":{"name":"tukey_get_localized_context","arguments":{"symbol":"GatewayFactory","depth":1}}}` + "\n",
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

func TestMCPServer_Pagination(t *testing.T) {
	graph := &models.DependencyGraph{Nodes: make(map[string]*models.DependencyNode)}
	for i := 0; i < 60; i++ {
		node := &models.DependencyNode{
			ID: fmt.Sprintf("function:Node%02d", i), Name: fmt.Sprintf("Node%02d", i),
			Type: "function", File: "nodes.php", Line: i + 1,
			Dependencies: map[string]*models.DependencyRef{}, Dependents: map[string]*models.DependencyRef{},
		}
		graph.Nodes[node.ID] = node
	}
	data, err := json.Marshal(map[string]interface{}{"graph": graph})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(parseEngineFromJSON(data), nil, nil)

	text, isErr := server.handleToolCall("tukey_find_symbol", json.RawMessage(`{"term":"Node","limit":2}`))
	if isErr {
		t.Fatalf("unexpected tool error: %s", text)
	}
	var first query.QueryResult
	if err := json.Unmarshal([]byte(text), &first); err != nil {
		t.Fatal(err)
	}
	if first.Count != 2 || first.Page == nil || first.Page.Total != 60 || first.Page.NextCursor == "" {
		t.Fatalf("unexpected first MCP page: %+v", first)
	}

	nextArgs, _ := json.Marshal(map[string]interface{}{
		"term": "Node", "limit": 2, "cursor": first.Page.NextCursor,
	})
	text, isErr = server.handleToolCall("tukey_find_symbol", nextArgs)
	if isErr {
		t.Fatalf("unexpected continuation error: %s", text)
	}
	var second query.QueryResult
	if err := json.Unmarshal([]byte(text), &second); err != nil {
		t.Fatal(err)
	}
	if second.Count != 2 || second.Results[0].ID == first.Results[0].ID {
		t.Fatalf("cursor did not advance MCP results: %+v", second)
	}

	text, isErr = server.handleToolCall("tukey_find_symbol", json.RawMessage(`{"term":"Node","limit":-1}`))
	if !isErr || !strings.Contains(text, "limit must not be negative") {
		t.Fatalf("expected negative limit error, got %s", text)
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
