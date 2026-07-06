package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type DuckDuckGo struct {
	http *http.Client
}

func (d *DuckDuckGo) Name() string { return "duckduckgo" }

func (d *DuckDuckGo) Search(query string) ([]Result, error) {
	if d.http == nil {
		d.http = &http.Client{Timeout: 10 * time.Second}
	}

	u := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")

	resp, err := d.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var results []Result
	doc.Find(".result").Each(func(i int, s *goquery.Selection) {
		titleSel := s.Find(".result__title a")
		href, _ := titleSel.Attr("href")
		href = cleanRedirectURL(href)
		title := strings.TrimSpace(titleSel.Text())

		snippet := strings.TrimSpace(s.Find(".result__snippet").Text())

		if title == "" || href == "" {
			return
		}

		results = append(results, Result{
			Title:   title,
			URL:     href,
			Snippet: snippet,
		})
	})

	return results, nil
}

func (d *DuckDuckGo) SearchImage(query string) ([]ImageResult, error) {
	if d.http == nil {
		d.http = &http.Client{Timeout: 10 * time.Second}
	}

	vqd, err := d.getVQDToken(query)
	if err != nil {
		return nil, fmt.Errorf("get vqd token: %w", err)
	}

	u := fmt.Sprintf("https://duckduckgo.com/i.js?q=%s&o=json&vqd=%s", url.QueryEscape(query), url.QueryEscape(vqd))

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("create image request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Referer", fmt.Sprintf("https://duckduckgo.com/?q=%s&iax=images&ia=images", url.QueryEscape(query)))

	resp, err := d.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("image request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image unexpected status: %d", resp.StatusCode)
	}

	var body struct {
		Results []struct {
			Title     string `json:"title"`
			URL       string `json:"url"`
			Image     string `json:"image"`
			Thumbnail string `json:"thumbnail"`
			Height    int    `json:"height"`
			Width     int    `json:"width"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("image decode: %w", err)
	}

	var results []ImageResult
	for _, r := range body.Results {
		if r.Image == "" {
			continue
		}
		results = append(results, ImageResult{
			Title:    r.Title,
			URL:      r.URL,
			ImageURL: r.Image,
		})
	}

	return results, nil
}

func (d *DuckDuckGo) getVQDToken(query string) (string, error) {
	u := fmt.Sprintf("https://duckduckgo.com/?q=%s&iax=images&ia=images", url.QueryEscape(query))

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")

	resp, err := d.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("parse token html: %w", err)
	}

	var vqd string
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		content := s.Text()
		if strings.Contains(content, "vqd") {
			re := regexp.MustCompile(`vqd['":=]+\s*['"]?([a-f0-9\-]+)['"]?`)
			matches := re.FindStringSubmatch(content)
			if len(matches) >= 2 {
				vqd = matches[1]
			}
		}
	})

	if vqd == "" {
		return "", fmt.Errorf("vqd token not found")
	}

	return vqd, nil
}

func cleanRedirectURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if q := u.Query().Get("uddg"); q != "" {
		return q
	}
	return raw
}
