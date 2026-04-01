package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var audibleRegionTLD = map[string]string{
	"us": ".com",
	"uk": ".co.uk",
	"ca": ".ca",
	"au": ".com.au",
	"de": ".de",
	"fr": ".fr",
	"it": ".it",
	"es": ".es",
	"jp": ".co.jp",
	"in": ".in",
	"br": ".com.br",
}

var audibleLocale = map[string]string{
	"us": "en-US",
	"uk": "en-GB",
	"ca": "en-CA",
	"au": "en-AU",
	"de": "de-DE",
	"fr": "fr-FR",
	"it": "it-IT",
	"es": "es-ES",
	"jp": "ja-JP",
	"in": "en-IN",
	"br": "pt-BR",
}

var audibleHTTPClient = &http.Client{Timeout: 30 * time.Second}

func audibleFetch(query, region string) ([]map[string]any, error) {
	region = strings.ToLower(strings.TrimSpace(region))
	if region == "" {
		region = "us"
	}

	tld, ok := audibleRegionTLD[region]
	if !ok {
		log.Printf("[Audible] unknown region %q, falling back to us", region)
		tld = ".com"
	}
	locale := audibleLocale[region]
	if locale == "" {
		locale = "en-US"
	}

	params := url.Values{}
	params.Set("title", query)
	params.Set("num_results", "10")
	params.Set("response_groups", "product_desc,contributors,product_attrs,media,category_ladders")

	endpoint := fmt.Sprintf("https://api.audible%s/1.0/catalog/products/?%s", tld, params.Encode())
	log.Printf("[Audible] searching title=%q region=%q url=%s", query, region, endpoint)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Audible/671 CFNetwork/1240.0.4 Darwin/20.6.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", locale)

	resp, err := audibleHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("audible request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Audible] HTTP %d for title=%q: %s", resp.StatusCode, query, string(body))
		return nil, fmt.Errorf("audible HTTP %d", resp.StatusCode)
	}

	var data struct {
		Products []map[string]any `json:"products"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("audible unmarshal: %w", err)
	}
	log.Printf("[Audible] got %d results for title=%q", len(data.Products), query)

	return data.Products, nil
}

// AudibleSearch returns the best matching result for a title, preferring an
// exact title match and falling back to the first result.
func AudibleSearch(title, region string) (map[string]any, error) {
	products, err := audibleFetch(title, region)
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, nil
	}

	product := products[0]
	titleLower := strings.ToLower(title)
	for _, p := range products {
		if t, ok := p["title"].(string); ok && strings.ToLower(t) == titleLower {
			product = p
			break
		}
	}

	return normalizeAudibleProduct(product), nil
}

// AudibleSearchAll returns all results for a query, normalized.
func AudibleSearchAll(query, region string) ([]map[string]any, error) {
	products, err := audibleFetch(query, region)
	if err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0, len(products))
	for _, p := range products {
		results = append(results, normalizeAudibleProduct(p))
	}
	return results, nil
}

func normalizeAudibleProduct(p map[string]any) map[string]any {
	out := map[string]any{
		"metadata_id":     p["asin"],
		"title":           p["title"],
		"description":     p["publisher_summary"],
		"publisher":       p["publisher_name"],
		"release_date":    p["release_date"],
		"runtime_minutes": p["runtime_length_min"],
	}

	if images, ok := p["product_images"].(map[string]any); ok {
		out["coverImage"] = images["500"]
	}

	if authors, ok := p["authors"].([]any); ok {
		names := make([]any, 0, len(authors))
		for _, a := range authors {
			if m, ok := a.(map[string]any); ok {
				if name, ok := m["name"]; ok {
					names = append(names, name)
				}
			}
		}
		out["authors"] = names
	}

	if narrators, ok := p["narrators"].([]any); ok {
		names := make([]any, 0, len(narrators))
		for _, n := range narrators {
			if m, ok := n.(map[string]any); ok {
				if name, ok := m["name"]; ok {
					names = append(names, name)
				}
			}
		}
		out["narrators"] = names
	}

	if ladders, ok := p["category_ladders"].([]any); ok {
		genres := make([]any, 0)
		seen := map[string]bool{}
		for _, ladder := range ladders {
			if l, ok := ladder.(map[string]any); ok {
				if rungs, ok := l["ladder"].([]any); ok {
					for _, rung := range rungs {
						if r, ok := rung.(map[string]any); ok {
							if name, ok := r["name"].(string); ok && !seen[name] {
								seen[name] = true
								genres = append(genres, name)
							}
						}
					}
				}
			}
		}
		out["genres"] = genres
	}

	return out
}
