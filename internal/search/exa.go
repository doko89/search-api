package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Exa struct {
	apiKey string
	http   *http.Client
}

func (e *Exa) Name() string { return "exa" }

func (e *Exa) Search(query string) ([]Result, error) {
	if e.http == nil {
		e.http = &http.Client{Timeout: 15 * time.Second}
	}

	body := map[string]any{
		"query":     query,
		"type":      "auto",
		"numResults": 10,
		"contents": map[string]any{
			"highlights": true,
		},
	}

	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", "https://api.exa.ai/search", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("exa create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", e.apiKey)

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exa request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exa unexpected status: %d", resp.StatusCode)
	}

	var res struct {
		Results []struct {
			Title       string   `json:"title"`
			URL         string   `json:"url"`
			Highlights  []string `json:"highlights"`
			Text        string   `json:"text"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("exa decode: %w", err)
	}

	var results []Result
	for _, r := range res.Results {
		if r.Title == "" || r.URL == "" {
			continue
		}
		snippet := ""
		if len(r.Highlights) > 0 && r.Highlights[0] != "" {
			snippet = r.Highlights[0]
		} else if r.Text != "" {
			if len(r.Text) > 300 {
				snippet = r.Text[:300] + "..."
			} else {
				snippet = r.Text
			}
		}
		results = append(results, Result{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: snippet,
		})
	}

	return results, nil
}
