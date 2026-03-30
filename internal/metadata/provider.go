package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Provider struct {
	Key            string            `json:"key"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Endpoints      map[string]string `json:"endpoints"`
	Properties     ProviderProps     `json:"properties"`
	Parser         map[string]any    `json:"parser"`
	PostProcessing map[string]any    `json:"postProcessing"`
}

type ProviderProps struct {
	LibraryType    string       `json:"library_type"`
	ResultsEntity  string       `json:"results_entity"`
	SearchFallback string       `json:"search_fallback"`
	SearchArray    *SearchArray `json:"search_array"`
}

type SearchArray struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func LoadProviders(pluginsDir string) (map[string]*Provider, error) {
	providers := make(map[string]*Provider)

	err := filepath.WalkDir(pluginsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".json") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var p Provider
		if err := json.Unmarshal(data, &p); err != nil {
			return nil
		}
		if p.Type == "metadata" && p.Properties.LibraryType != "" {
			providers[p.Key] = &p
		}
		return nil
	})
	return providers, err
}
