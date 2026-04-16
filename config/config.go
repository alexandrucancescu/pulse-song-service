package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Endpoint represents a single HTTP endpoint where file content will be posted.
type Endpoint struct {
	URL     string            `json:"url"`
	PostKey string            `json:"postKey"`
	Headers map[string]string `json:"headers"`
}

// Config holds the application configuration loaded from config.json.
type Config struct {
	File      string     `json:"file"`
	Endpoints []Endpoint `json:"endpoints"`
}

// Load reads config.json from the given directory, validates it, and returns the Config.
// On Windows (production), pass the executable's directory.
// During development, pass the project directory.
func Load(dir string) (*Config, error) {
	path := filepath.Join(dir, "config.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON in config file: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// validate checks that all required fields are present and makes sense.
func validate(cfg *Config) error {
	if cfg.File == "" {
		return errors.New("\"file\" is required")
	}

	if len(cfg.Endpoints) == 0 {
		return errors.New("\"endpoints\" must contain at least one entry")
	}

	for i, ep := range cfg.Endpoints {
		if ep.URL == "" {
			return fmt.Errorf("endpoint #%d: \"url\" is required", i+1)
		}

		// Default postKey to "title" if not provided, matching the Node.js behavior.
		if ep.PostKey == "" {
			cfg.Endpoints[i].PostKey = "title"
		}
	}

	return nil
}
