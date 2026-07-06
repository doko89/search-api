package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Brave struct {
	apiKey string
	http   *http.Client
}

func (b *Brave) Name() string { return "brave" }

func (b *Brave) Search(query string) ([]Result, error) {
	if b.http == nil {
		b.http = &http.Client{Timeout: 10 * time.Second}
	}

	u := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=10", url.QueryEscape(query))

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("brave create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", b.apiKey)

	resp, err := b.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("brave request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brave unexpected status: %d", resp.StatusCode)
	}

	var body struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("brave decode: %w", err)
	}

	var results []Result
	for _, r := range body.Web.Results {
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

func (b *Brave) SearchImage(query string) ([]ImageResult, error) {
	if b.http == nil {
		b.http = &http.Client{Timeout: 10 * time.Second}
	}

	u := fmt.Sprintf("https://api.search.brave.com/res/v1/images/search?q=%s&count=10", url.QueryEscape(query))

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("brave image create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", b.apiKey)

	resp, err := b.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("brave image request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brave image unexpected status: %d", resp.StatusCode)
	}

	var body struct {
		Results []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
			Properties struct {
				URL string `json:"url"`
			} `json:"properties"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("brave image decode: %w", err)
	}

	var results []ImageResult
	for _, r := range body.Results {
		imgURL := r.Properties.URL
		if imgURL == "" {
			continue
		}
		results = append(results, ImageResult{
			Title:    r.Title,
			URL:      r.URL,
			ImageURL: imgURL,
		})
	}

	return results, nil
}
