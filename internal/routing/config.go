package routing

import (
	"fmt"
	"maps"
	"os"

	"github.com/goccy/go-yaml"
)

type File struct {
	Backends []Backend `yaml:"backends"`
	Routes   []Route   `yaml:"routes"`
	Defaults Default   `yaml:"defaults"`
}

type Backend struct {
	Name       string            `yaml:"name"` // required value
	URL        string            `yaml:"url"`  // required value
	Timeout    string            `yaml:"timeout"`
	HealthPath string            `yaml:"health_path"`
	Retries    int               `yaml:"retries"`
	Headers    map[string]string `yaml:"headers"`
}

type Route struct {
	Method  string `yaml:"method"`
	Path    string `yaml:"path"`
	Backend string `yaml:"backend"`
}

type Default struct {
	Timeout    string            `yaml:"timeout"`
	HealthPath string            `yaml:"health_path"`
	Retries    int               `yaml:"retries"`
	Headers    map[string]string `yaml:"headers"`
}

// LoadFile loads the config.yaml on the given path
// Merges Defaults into the Backends and validates the options set in config
func LoadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	f := File{}
	errUn := yaml.Unmarshal(data, &f)
	if errUn != nil {
		return nil, fmt.Errorf("parse config: %w", errUn)
	}

	// Loop through backends to fill default values
	for i, b := range f.Backends {
		// early error if name/url are missing (required)
		if b.Name == "" {
			return nil, fmt.Errorf("backend index at %d: missing name", i)
		}
		if b.URL == "" {
			return nil, fmt.Errorf("backend index at %q: missing url", b.Name)
		}

		if b.Timeout == "" {
			f.Backends[i].Timeout = f.Defaults.Timeout
		}
		if b.HealthPath == "" {
			f.Backends[i].HealthPath = f.Defaults.HealthPath
		}
		if b.Retries == 0 {
			f.Backends[i].Retries = f.Defaults.Retries
		}

		switch {
		case f.Defaults.Headers == nil:
			continue
		case b.Headers == nil:
			// create map size defaults
			merged := make(map[string]string, len(f.Defaults.Headers))
			// copy defaults to merged
			maps.Copy(merged, f.Defaults.Headers)
			// assign merged to backend headers
			f.Backends[i].Headers = merged
		default:
			merged := make(map[string]string, len(f.Defaults.Headers))
			maps.Copy(merged, f.Defaults.Headers)
			maps.Copy(merged, b.Headers)
			f.Backends[i].Headers = merged
		}
	}
	return &f, nil
}
