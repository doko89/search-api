package search

import (
	"fmt"
	"log"
	"strings"
)

type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type ImageResult struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	ImageURL string `json:"image_url"`
}

type Provider interface {
	Search(query string) ([]Result, error)
	Name() string
}

type ImageProvider interface {
	SearchImage(query string) ([]ImageResult, error)
}

type Client struct {
	providers []Provider
}

func NewClient(braveKey, tavilyKey, firecrawlKey, exaKey, providerOrder string) *Client {
	providers := buildProviders(braveKey, tavilyKey, firecrawlKey, exaKey, providerOrder)
	return &Client{providers: providers}
}

func buildProviders(braveKey, tavilyKey, firecrawlKey, exaKey, order string) []Provider {
	available := map[string]Provider{
		"duckduckgo": &DuckDuckGo{},
		"brave":      &Brave{apiKey: braveKey},
		"tavily":     &Tavily{apiKey: tavilyKey},
		"firecrawl":  &Firecrawl{apiKey: firecrawlKey},
		"exa":        &Exa{apiKey: exaKey},
	}

	names := strings.Split(order, ",")
	var providers []Provider
	for _, name := range names {
		name = strings.TrimSpace(name)
		if p, ok := available[name]; ok {
			providers = append(providers, p)
		}
	}

	if len(providers) == 0 {
		log.Println("WARNING: no valid providers configured, falling back to duckduckgo")
		providers = append(providers, &DuckDuckGo{})
	}

	return providers
}

func (c *Client) Search(query string) ([]Result, error) {
	var lastErr error
	for _, p := range c.providers {
		results, err := p.Search(query)
		if err == nil && len(results) > 0 {
			return results, nil
		}
		if err != nil {
			lastErr = err
			log.Printf("provider %s failed: %v", p.Name(), err)
		}
	}
	return nil, fmt.Errorf("all providers failed: %w", lastErr)
}

func (c *Client) SearchFrom(query, provider string) ([]Result, error) {
	if provider == "" {
		return c.Search(query)
	}
	for _, p := range c.providers {
		if p.Name() == provider {
			return p.Search(query)
		}
	}
	return nil, fmt.Errorf("unknown provider: %s", provider)
}

func (c *Client) SearchImage(query string) ([]ImageResult, error) {
	var lastErr error
	for _, p := range c.providers {
		ip, ok := p.(ImageProvider)
		if !ok {
			continue
		}
		results, err := ip.SearchImage(query)
		if err == nil && len(results) > 0 {
			return results, nil
		}
		if err != nil {
			lastErr = err
			log.Printf("image provider %s failed: %v", p.Name(), err)
		}
	}
	return nil, fmt.Errorf("all image providers failed: %w", lastErr)
}

func (c *Client) SearchImageFrom(query, provider string) ([]ImageResult, error) {
	if provider == "" {
		return c.SearchImage(query)
	}
	for _, p := range c.providers {
		if p.Name() == provider {
			ip, ok := p.(ImageProvider)
			if !ok {
				return nil, fmt.Errorf("provider %s does not support image search", provider)
			}
			return ip.SearchImage(query)
		}
	}
	return nil, fmt.Errorf("unknown provider: %s", provider)
}
