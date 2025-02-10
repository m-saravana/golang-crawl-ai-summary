package summarizer

import "fmt"

// Type represents the type of summarizer to use
type Type string

const (
	// TypeOllama represents the Ollama summarizer
	TypeOllama Type = "ollama"
)

// Config holds configuration for summarizer creation
type Config struct {
	Type Type
	// Ollama specific config
	OllamaURL   string
	OllamaModel string
}

// Factory creates summarizers based on configuration
type Factory struct {
	config Config
}

// NewFactory creates a new summarizer factory
func NewFactory(config Config) *Factory {
	return &Factory{
		config: config,
	}
}

// CreateSummarizer creates a summarizer based on the configuration
func (f *Factory) CreateSummarizer() (Summarizer, error) {
	switch f.config.Type {
	case TypeOllama:
		return NewOllamaSummarizer(f.config.OllamaURL, f.config.OllamaModel), nil
	default:
		return nil, fmt.Errorf("unsupported summarizer type: %s", f.config.Type)
	}
}
