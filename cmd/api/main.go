package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/doko/search-api/internal/mcp"
	"github.com/doko/search-api/internal/search"
	"github.com/joho/godotenv"
)

var apiToken string

var searchClient *search.Client

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	godotenv.Load()

	apiToken = getEnv("API_TOKEN", "dev-secret-token")

	searchClient = search.NewClient(
		getEnv("BRAVE_API_KEY", ""),
		getEnv("TAVILY_API_KEY", ""),
		getEnv("FIRECRAWL_API_KEY", ""),
		getEnv("SEARCH_PROVIDER", "duckduckgo,brave,tavily,firecrawl"),
	)

	mcpHandler := mcp.NewHandler(searchClient)

	mux := http.NewServeMux()
	mux.Handle("/mcp", auth(cors(mcpHandler)))
	mux.Handle("/search", auth(cors(http.HandlerFunc(handleSearch))))
	mux.Handle("/health", auth(cors(http.HandlerFunc(handleHealth))))

	addr := ":8080"
	log.Printf("search api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query parameter 'q' is required"})
		return
	}

	provider := strings.TrimSpace(r.URL.Query().Get("provider"))

	var results []search.Result
	var err error
	if provider != "" {
		results, err = searchClient.SearchFrom(q, provider)
	} else {
		results, err = searchClient.Search(q)
	}
	if err != nil {
		log.Printf("search error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "search failed"})
		return
	}

	if results == nil {
		results = []search.Result{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"query":   q,
		"results": results,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			return
		}
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token != apiToken {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, MCP-Protocol-Version, Mcp-Method, Mcp-Name")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
