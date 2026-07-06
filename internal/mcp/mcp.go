package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/doko/search-api/internal/search"
)

const ProtocolVersion = "2026-07-28"

type Searcher interface {
	Search(query string) ([]search.Result, error)
	SearchFrom(query, provider string) ([]search.Result, error)
	SearchImage(query string) ([]search.ImageResult, error)
	SearchImageFrom(query, provider string) ([]search.ImageResult, error)
}

type Handler struct {
	searcher Searcher
}

func NewHandler(searcher Searcher) *Handler {
	return &Handler{searcher: searcher}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMCPError(w, nil, -32700, "parse error: invalid JSON")
		return
	}

	headerVersion := r.Header.Get("MCP-Protocol-Version")
	bodyVersion := extractProtocolVersion(req.Params)
	if headerVersion != "" && bodyVersion != "" && headerVersion != bodyVersion {
		writeMCPError(w, req.ID, -32020, "header/body protocol version mismatch")
		return
	}

	log.Printf("mcp request: method=%s id=%v", req.Method, req.ID)

	switch req.Method {
	case "initialize":
		h.handleInitialize(w, &req)
	case "notifications/initialized":
		w.WriteHeader(http.StatusAccepted)
	case "notifications/cancelled":
		w.WriteHeader(http.StatusAccepted)
	case "tools/list":
		h.handleToolsList(w, &req)
	case "tools/call":
		h.handleToolsCall(w, &req)
	default:
		writeMCPError(w, req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (h *Handler) handleInitialize(w http.ResponseWriter, req *JSONRPCRequest) {
	clientVersion := extractProtocolVersion(req.Params)
	if clientVersion == "" {
		clientVersion = ProtocolVersion
	}

	result := InitializeResult{
		ProtocolVersion: clientVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{ListChanged: false},
		},
		ServerInfo: ServerInfo{
			Name:    "search-api",
			Version: "1.0.0",
		},
	}
	writeMCPResult(w, req.ID, result)
}

func (h *Handler) handleToolsList(w http.ResponseWriter, req *JSONRPCRequest) {
	tools := []Tool{
		{
			Name:        "search_web",
			Description: "Search the web using DuckDuckGo, Brave, Tavily, or Firecrawl. Returns a list of search results with title, URL, and snippet for each result.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropertySchema{
					"query": {
						Type:        "string",
						Description: "The search query string",
					},
					"provider": {
						Type:        "string",
						Description: "Optional: search provider (duckduckgo, brave, tavily, firecrawl). Default uses ordered fallback.",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "search_images",
			Description: "Search for images using Brave or Tavily. Returns a list of image results with title, URL, and image_url for each result.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropertySchema{
					"query": {
						Type:        "string",
						Description: "The search query string",
					},
					"provider": {
						Type:        "string",
						Description: "Optional: image search provider (brave, tavily). Default uses ordered fallback.",
					},
				},
				Required: []string{"query"},
			},
		},
	}
	writeMCPResult(w, req.ID, map[string]any{"tools": tools})
}

func (h *Handler) handleToolsCall(w http.ResponseWriter, req *JSONRPCRequest) {
	params, err := decodeParams[CallParams](req.Params)
	if err != nil {
		writeMCPError(w, req.ID, -32602, "invalid params")
		return
	}

	switch params.Name {
	case "search_web":
		h.handleSearchWeb(w, req, params)
	case "search_images":
		h.handleSearchImages(w, req, params)
	default:
		writeMCPError(w, req.ID, -32602, fmt.Sprintf("unknown tool: %s", params.Name))
	}
}

func (h *Handler) handleSearchWeb(w http.ResponseWriter, req *JSONRPCRequest, params CallParams) {
	query, ok := params.Arguments["query"].(string)
	log.Printf("search_web query=%q provider=%v", query, params.Arguments["provider"])
	if !ok || strings.TrimSpace(query) == "" {
		writeMCPResult(w, req.ID, CallResult{
			Content: []Content{{Type: "text", Text: "Error: 'query' parameter is required and must be a string"}},
			IsError: true,
		})
		return
	}

	provider, _ := params.Arguments["provider"].(string)

	var results []search.Result
	var err error
	if provider != "" {
		results, err = h.searcher.SearchFrom(query, provider)
	} else {
		results, err = h.searcher.Search(query)
	}
	if err != nil {
		log.Printf("search error: %v", err)
		writeMCPResult(w, req.ID, CallResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Error: search failed: %s", err.Error())}},
			IsError: true,
		})
		return
	}

	if len(results) == 0 {
		writeMCPResult(w, req.ID, CallResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("No results found for '%s'", query)}},
		})
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Search results for '%s'\n\n", query))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("## %d. %s\n", i+1, r.Title))
		sb.WriteString(fmt.Sprintf("**URL:** %s\n", r.URL))
		sb.WriteString(fmt.Sprintf("%s\n\n", r.Snippet))
	}

	writeMCPResult(w, req.ID, CallResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
	})
}

func (h *Handler) handleSearchImages(w http.ResponseWriter, req *JSONRPCRequest, params CallParams) {
	query, ok := params.Arguments["query"].(string)
	log.Printf("search_images query=%q provider=%v", query, params.Arguments["provider"])
	if !ok || strings.TrimSpace(query) == "" {
		writeMCPResult(w, req.ID, CallResult{
			Content: []Content{{Type: "text", Text: "Error: 'query' parameter is required and must be a string"}},
			IsError: true,
		})
		return
	}

	provider, _ := params.Arguments["provider"].(string)

	var results []search.ImageResult
	var err error
	if provider != "" {
		results, err = h.searcher.SearchImageFrom(query, provider)
	} else {
		results, err = h.searcher.SearchImage(query)
	}
	if err != nil {
		log.Printf("image search error: %v", err)
		writeMCPResult(w, req.ID, CallResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Error: image search failed: %s", err.Error())}},
			IsError: true,
		})
		return
	}

	if len(results) == 0 {
		writeMCPResult(w, req.ID, CallResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("No image results found for '%s'", query)}},
		})
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Image results for '%s'\n\n", query))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("## %d. %s\n", i+1, r.Title))
		if r.URL != "" {
			sb.WriteString(fmt.Sprintf("**URL:** %s\n", r.URL))
		}
		sb.WriteString(fmt.Sprintf("**Image URL:** %s\n", r.ImageURL))
	}

	writeMCPResult(w, req.ID, CallResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
	})
}

func extractProtocolVersion(params any) string {
	m, ok := params.(map[string]any)
	if !ok {
		return ""
	}
	v, _ := m["protocolVersion"].(string)
	return v
}

func decodeParams[T any](raw any) (T, error) {
	var zero T
	data, err := json.Marshal(raw)
	if err != nil {
		return zero, err
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return zero, err
	}
	return v, nil
}

func writeMCPResult(w http.ResponseWriter, id any, result any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func writeMCPError(w http.ResponseWriter, id any, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: msg,
		},
	})
}
