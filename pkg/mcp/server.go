// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/boone-studios/tukey/pkg/query"
)

// JSONRPCRequest represents an incoming JSON-RPC 2.0 message
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError is a standard JSON-RPC 2.0 error object
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSON-RPC Error Codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// Server handles the MCP JSON-RPC protocol over standard I/O (stdio)
type Server struct {
	engine *query.Engine
	in     io.Reader
	out    io.Writer
}

// NewServer initializes a new Tukey MCP Server instance.
func NewServer(engine *query.Engine, in io.Reader, out io.Writer) *Server {
	return &Server{
		engine: engine,
		in:     in,
		out:    out,
	}
}

// Start runs the server, listening to input stream and writing responses
func (s *Server) Start() error {
	scanner := bufio.NewScanner(s.in)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		resp := s.handleRequest(line)
		if resp != nil {
			respBytes, err := json.Marshal(resp)
			if err != nil {
				continue
			}
			s.out.Write(respBytes)
			s.out.Write([]byte("\n"))
		}
	}
	return scanner.Err()
}

func (s *Server) handleRequest(reqBytes []byte) *JSONRPCResponse {
	var req JSONRPCRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &RPCError{
				Code:    ParseError,
				Message: "Parse error: invalid JSON",
			},
		}
	}

	if req.JSONRPC != "2.0" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    InvalidRequest,
				Message: "Invalid Request: missing jsonrpc field set to '2.0'",
			},
		}
	}

	result, errObj := s.dispatch(req.Method, req.Params)
	if errObj != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   errObj,
		}
	}

	// Notifications do not return responses... :)
	if result == nil && req.Method == "notifications/initialized" {
		return nil
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *Server) dispatch(method string, params json.RawMessage) (interface{}, *RPCError) {
	switch method {
	case "initialize":
		return InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{},
			},
			ServerInfo: Implementation{
				Name:    "tukey-mcp",
				Version: "0.3.0",
			},
		}, nil

	case "notifications/initialized":
		return nil, nil

	case "tools/list":
		return ListToolsResult{
			Tools: []Tool{
				{
					Name:        "tukey_find_symbol",
					Description: "Find code elements (classes, methods, functions) whose name, namespace, or ID matches a search term.",
					InputSchema: InputSchema{
						Type: "object",
						Properties: map[string]Property{
							"term": {
								Type:        "string",
								Description: "The symbol name or substring to search for (e.g., 'GatewayFactory' or 'PaymentService::pay').",
							},
						},
						Required: []string{"term"},
					},
				},
				{
					Name:        "tukey_get_callers",
					Description: "Show all code elements that call or reference a given symbol (incoming edges).",
					InputSchema: InputSchema{
						Type: "object",
						Properties: map[string]Property{
							"symbol": {
								Type:        "string",
								Description: "The name, namespaced name, or exact ID of the target symbol.",
							},
						},
						Required: []string{"symbol"},
					},
				},
				{
					Name:        "tukey_get_dependents",
					Description: "Show all code elements that the named symbol directly depends on (outgoing edges).",
					InputSchema: InputSchema{
						Type: "object",
						Properties: map[string]Property{
							"symbol": {
								Type:        "string",
								Description: "The name, namespaced name, or exact ID of the target symbol.",
							},
						},
						Required: []string{"symbol"},
					},
				},
				{
					Name:        "tukey_find_orphans",
					Description: "List all orphaned code elements (candidates for dead/unused code) that have zero dependencies and zero dependents.",
					InputSchema: InputSchema{
						Type: "object",
					},
				},
				{
					Name:        "tukey_get_localized_context",
					Description: "Get a minimized localized context graph around a target symbol, returning its direct dependencies and dependents up to a specified depth limit.",
					InputSchema: InputSchema{
						Type: "object",
						Properties: map[string]Property{
							"symbol": {
								Type:        "string",
								Description: "The name, namespaced name, or exact ID of the target symbol.",
							},
							"depth": {
								Type:        "integer",
								Description: "The maximum depth of traversal (defaults to 1).",
							},
						},
						Required: []string{"symbol"},
					},
				},
			},
		}, nil

	case "tools/call":
		var callParams struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(params, &callParams); err != nil {
			return nil, &RPCError{
				Code:    InvalidParams,
				Message: "Invalid params for tools/call",
			}
		}

		resText, isErr := s.handleToolCall(callParams.Name, callParams.Arguments)
		return CallToolResult{
			Content: []Content{
				{
					Type: "text",
					Text: resText,
				},
			},
			IsError: isErr,
		}, nil

	default:
		return nil, &RPCError{
			Code:    MethodNotFound,
			Message: fmt.Sprintf("Method not found: %s", method),
		}
	}
}

func (s *Server) handleToolCall(name string, args json.RawMessage) (string, bool) {
	switch name {
	case "tukey_find_symbol":
		var arguments struct {
			Term string `json:"term"`
		}
		if err := json.Unmarshal(args, &arguments); err != nil {
			return "Error: failed to parse arguments: missing 'term' string.", true
		}
		if arguments.Term == "" {
			return "Error: 'term' argument must be a non-empty string.", true
		}

		result := s.engine.Find(arguments.Term)
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Sprintf("Error encoding result: %v", err), true
		}
		return string(data), false

	case "tukey_get_callers":
		var arguments struct {
			Symbol string `json:"symbol"`
		}
		if err := json.Unmarshal(args, &arguments); err != nil {
			return "Error: failed to parse arguments: missing 'symbol' string.", true
		}
		if arguments.Symbol == "" {
			return "Error: 'symbol' argument must be a non-empty string.", true
		}

		result := s.engine.Callers(arguments.Symbol)
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Sprintf("Error encoding result: %v", err), true
		}
		return string(data), false

	case "tukey_get_dependents":
		var arguments struct {
			Symbol string `json:"symbol"`
		}
		if err := json.Unmarshal(args, &arguments); err != nil {
			return "Error: failed to parse arguments: missing 'symbol' string.", true
		}
		if arguments.Symbol == "" {
			return "Error: 'symbol' argument must be a non-empty string.", true
		}

		result := s.engine.Dependents(arguments.Symbol)
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Sprintf("Error encoding result: %v", err), true
		}
		return string(data), false

	case "tukey_find_orphans":
		result := s.engine.Orphans()
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Sprintf("Error encoding result: %v", err), true
		}
		return string(data), false

	case "tukey_get_localized_context":
		var arguments struct {
			Symbol string `json:"symbol"`
			Depth  *int   `json:"depth"`
		}
		if err := json.Unmarshal(args, &arguments); err != nil {
			return "Error: failed to parse arguments: missing 'symbol' string.", true
		}
		if arguments.Symbol == "" {
			return "Error: 'symbol' argument must be a non-empty string.", true
		}

		depth := 1
		if arguments.Depth != nil {
			depth = *arguments.Depth
		}

		result := s.engine.LocalizedContext(arguments.Symbol, depth)
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Sprintf("Error encoding result: %v", err), true
		}
		return string(data), false

	default:
		return fmt.Sprintf("Error: unknown tool name: %s", name), true
	}
}

// MCP Protocol Message Types
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
}

type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged *bool `json:"listChanged,omitempty"`
}

type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
