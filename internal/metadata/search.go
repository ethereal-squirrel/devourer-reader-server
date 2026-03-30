package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"strings"
)

func Search(providers map[string]*Provider, providerKey, by, value, apiKey string) (map[string]any, error) {
	p, ok := providers[providerKey]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", providerKey)
	}

	endpointTpl, ok := p.Endpoints[by]
	if !ok {
		return nil, fmt.Errorf("provider %s has no endpoint for %q", providerKey, by)
	}

	url := strings.ReplaceAll(endpointTpl, "{{query}}", neturl.QueryEscape(value))
	if apiKey != "" {
		url = strings.ReplaceAll(url, "{{apiKey}}", neturl.QueryEscape(apiKey))
	}

	log.Printf("[Metadata] GET %s", url)

	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Metadata] HTTP %d response: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("metadata HTTP %d", resp.StatusCode)
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	results := getNestedValue(raw, p.Properties.ResultsEntity)
	if results == nil {
		return nil, nil
	}

	arr, ok := results.([]any)
	if !ok || len(arr) == 0 {
		return nil, nil
	}

	selected := selectResult(arr, value, p)
	if selected == nil {
		return nil, nil
	}

	return parseMetadata(selected, p.Parser, p.PostProcessing), nil
}

func selectResult(results []any, query string, p *Provider) map[string]any {
	query = strings.ToLower(query)

	for _, r := range results {
		item, ok := r.(map[string]any)
		if !ok {
			continue
		}

		if p.Properties.SearchArray != nil {
			arr := getNestedValue(item, p.Properties.SearchArray.Key)
			if arrSlice, ok := arr.([]any); ok {
				for _, elem := range arrSlice {
					if m, ok := elem.(map[string]any); ok {
						if v, ok := m["title"].(string); ok && strings.ToLower(v) == query {
							return item
						}
					}
				}
			}
		} else if p.Properties.SearchFallback != "" {
			if v, ok := item[p.Properties.SearchFallback].(string); ok {
				if strings.ToLower(v) == query {
					return item
				}
			}
		}
	}

	if first, ok := results[0].(map[string]any); ok {
		return first
	}
	return nil
}

func parseMetadata(data map[string]any, parser map[string]any, postProcessing map[string]any) map[string]any {
	out := make(map[string]any, len(parser))

	for key, spec := range parser {
		switch v := spec.(type) {
		case nil:
			out[key] = nil
		case string:
			out[key] = getNestedValue(data, v)
		case map[string]any:
			keyStr, _ := v["key"].(string)
			valStr, _ := v["value"].(string)
			if keyStr == "static" {
				out[key] = v["value"]
			} else {
				nested := getNestedValue(data, keyStr)
				if arr, ok := nested.([]any); ok && valStr != "" {
					mapped := make([]any, 0, len(arr))
					for _, elem := range arr {
						if m, ok := elem.(map[string]any); ok {
							mapped = append(mapped, getNestedValue(m, valStr))
						}
					}
					out[key] = mapped
				} else {
					out[key] = nested
				}
			}
		}
	}

	for key, cfg := range postProcessing {
		cfgMap, ok := cfg.(map[string]any)
		if !ok {
			continue
		}
		action, _ := cfgMap["action"].(string)
		switch action {
		case "convertToArray":
			if v, exists := out[key]; exists && v != nil {
				if _, isArr := v.([]any); !isArr {
					out[key] = []any{v}
				}
			}
		case "convertToIndustryIdentifier":
			if v, exists := out[key]; exists && v != nil {
				ids, _ := out["identifiers"].([]any)
				ids = append(ids, map[string]any{"type": key, "value": v})
				out["identifiers"] = ids
			}
		}
	}

	return out
}

func getNestedValue(obj map[string]any, path string) any {
	if path == "" {
		return nil
	}
	parts := strings.SplitN(path, ".", 2)
	val, ok := obj[parts[0]]
	if !ok {
		return nil
	}
	if len(parts) == 1 {
		return val
	}
	if nested, ok := val.(map[string]any); ok {
		return getNestedValue(nested, parts[1])
	}
	return nil
}
