package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Tavily struct {
	apiKey string
	http   *http.Client
}

func (t *Tavily) Name() string { return "tavily" }

func (t *Tavily) Search(query string) ([]Result, error) {
	if t.http == nil {
		t.http = &http.Client{Timeout: 15 * time.Second}
	}

	body := map[string]any{
		"api_key":    t.apiKey,
		"query":      query,
		"max_results": 10,
		"include_images": true,
	}

	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", "https://api.tavily.com/search", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("tavily create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tavily request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily unexpected status: %d", resp.StatusCode)
	}

	var res struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("tavily decode: %w", err)
	}

	var results []Result
	for _, r := range res.Results {
		if r.Title == "" || r.URL == "" {
			continue
		}
		results = append(results, Result{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
		})
	}

	return results, nil
}

func (t *Tavily) SearchImage(query string) ([]ImageResult, error) {
	if t.http == nil {
		t.http = &http.Client{Timeout: 15 * time.Second}
	}

	body := map[string]any{
		"api_key":    t.apiKey,
		"query":      query,
		"max_results": 10,
		"include_images": true,
	}

	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", "https://api.tavily.com/search", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("tavily image create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tavily image request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily image unexpected status: %d", resp.StatusCode)
	}

	var res struct {
		Images []string `json:"images"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("tavily image decode: %w", err)
	}

	var results []ImageResult
	for _, img := range res.Images {
		if img == "" {
			continue
		}
		results = append(results, ImageResult{
			ImageURL: img,
		})
	}

	return results, nil
}
