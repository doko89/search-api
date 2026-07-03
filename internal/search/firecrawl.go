package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Firecrawl struct {
	apiKey string
	http   *http.Client
}

func (f *Firecrawl) Name() string { return "firecrawl" }

func (f *Firecrawl) Search(query string) ([]Result, error) {
	if f.http == nil {
		f.http = &http.Client{Timeout: 15 * time.Second}
	}

	body := map[string]any{
		"query": query,
		"limit": 10,
	}

	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", "https://api.firecrawl.dev/v1/search", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("firecrawl create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.apiKey)

	resp, err := f.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("firecrawl request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("firecrawl unexpected status: %d", resp.StatusCode)
	}

	var res struct {
		Success bool `json:"success"`
		Data    []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("firecrawl decode: %w", err)
	}

	if !res.Success {
		return nil, fmt.Errorf("firecrawl returned unsuccessful response")
	}

	var results []Result
	for _, r := range res.Data {
		if r.Title == "" || r.URL == "" {
			continue
		}
		results = append(results, Result{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Description,
		})
	}

	return results, nil
}
