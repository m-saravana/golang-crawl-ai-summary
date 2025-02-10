package config

import (
	"encoding/json"
	"fmt"
	"os"

	"webcrawler/internal/summarizer"
)

// Config holds the application configuration
type Config struct {
	// Crawler configuration
	MaxDepth   int     `json:"maxDepth"`
	RateLimit  float64 `json:"rateLimit"`
	MaxWorkers int     `json:"maxWorkers"`

	// Summarizer configuration
	SummarizerType string `json:"summarizerType"` // "ollama"
	OllamaURL      string `json:"ollamaUrl"`
	OllamaModel    string `json:"ollamaModel"`
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	// Default configuration
	config := &Config{
		MaxDepth:       2,
		RateLimit:      1.0,
		MaxWorkers:     5,
		SummarizerType: "ollama",
		OllamaURL:      "http://localhost:11434",
		OllamaModel:    "mistral",
	}

	// If config file exists, load it
	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else {
			defer file.Close()
			if err := json.NewDecoder(file).Decode(config); err != nil {
				return nil, err
			}
		}
	}

	// Override with environment variables if they exist
	if envMaxDepth := os.Getenv("CRAWLER_MAX_DEPTH"); envMaxDepth != "" {
		var depth int
		if _, err := fmt.Sscan(envMaxDepth, &depth); err == nil {
			config.MaxDepth = depth
		}
	}

	if envSummarizerType := os.Getenv("SUMMARIZER_TYPE"); envSummarizerType != "" {
		config.SummarizerType = envSummarizerType
	}

	if envOllamaURL := os.Getenv("OLLAMA_URL"); envOllamaURL != "" {
		config.OllamaURL = envOllamaURL
	}

	if envOllamaModel := os.Getenv("OLLAMA_MODEL"); envOllamaModel != "" {
		config.OllamaModel = envOllamaModel
	}

	return config, nil
}

// CreateSummarizer creates a summarizer based on the configuration
func (c *Config) CreateSummarizer() (summarizer.Summarizer, error) {
	config := summarizer.Config{
		Type:        summarizer.Type(c.SummarizerType),
		OllamaURL:   c.OllamaURL,
		OllamaModel: c.OllamaModel,
	}

	factory := summarizer.NewFactory(config)
	return factory.CreateSummarizer()
}
