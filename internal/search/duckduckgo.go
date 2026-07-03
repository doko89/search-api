package search

import (
	"fmt"
	"net/http"
	"net/url"
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
