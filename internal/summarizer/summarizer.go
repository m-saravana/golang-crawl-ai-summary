package summarizer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type ContentUnderstanding struct {
	URL            string    `json:"url"`
	SimplifiedText string    `json:"simplified_text"`
	Notes          string    `json:"notes"`
	LastModified   time.Time `json:"last_modified"`
}

type Summarizer interface {
	Summarize(text string) (string, error)
}

type OllamaSummarizer struct {
	baseURL string
	model   string
}

func NewOllamaSummarizer(baseURL, model string) *OllamaSummarizer {
	if model == "" {
		model = "mistral" // default model
	}
	return &OllamaSummarizer{
		baseURL: baseURL,
		model:   model,
	}
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

func (o *OllamaSummarizer) makeRequest(jsonData []byte) (*ollamaResponse, error) {
	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	resp, err := client.Post(fmt.Sprintf("%s/api/generate", o.baseURL), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("ollama error: %s", result.Error)
	}

	return &result, nil
}

// Summarize generates a summary of the given text using Ollama
func (o *OllamaSummarizer) Summarize(text string) (string, error) {
	// Trim and clean the text
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("empty text")
	}

	// Log input text length
	log.Printf("Input text length: %d characters\n", len(text))

	// If text is too long, take first and last parts
	const maxLen = 12000
	if len(text) > maxLen {
		firstPart := text[:maxLen/2]
		lastPart := text[len(text)-maxLen/2:]
		text = firstPart + "\n...\n" + lastPart
	}

	// Prepare the prompt for structured summary
	prompt := fmt.Sprintf(`You are a helpful AI assistant. Create a structured summary of this text with:

1. Key Points (3-4 bullet points)
2. Important Terms (3-4 terms with brief explanations)
3. Main Takeaways (2-3 points)

Text: %s

Remember to be concise and specific.`, text)

	// Make request to Ollama
	reqBody := ollamaRequest{
		Model:  o.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Make the request with retries
	var summary string
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("Attempt %d of %d to generate summary\n", attempt, maxAttempts)

		resp, err := o.makeRequest(jsonData)
		if err != nil {
			if attempt == maxAttempts {
				return "", fmt.Errorf("failed to generate summary after %d attempts: %v", maxAttempts, err)
			}
			log.Printf("Attempt %d failed: %v. Retrying...\n", attempt, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		summary = resp.Response
		break
	}

	if summary == "" {
		return "", fmt.Errorf("failed to generate summary: empty response")
	}

	return summary, nil
}
