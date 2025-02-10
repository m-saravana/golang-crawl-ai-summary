# Web Crawler with Summarization

A web crawler written in Go that crawls web pages and generates summaries using the Ollama AI model. The crawler is designed to be configurable, and using locally running ollama server and Mistral model for summarization. Used Playwright for browser automation to interact with web pages and extract content.

## Features

- Concurrent web crawling with configurable worker pools
- Rate limiting to prevent overwhelming target websites
- Depth-limited crawling for limiting links inside a web page
- AI-powered content summarization using Ollama
- Configurable via JSON file or environment variables
- Graceful shutdown handling
- Verbose logging option
- Error handling for network issues, invalid URLs, rate limiting, context cancellation, and summarization failures.


## Prerequisites

- Go 1.21 or higher
- Ollama running locally (default: http://localhost:11434)
- Mistral model installed in Ollama

## Installation

Clone the repository:
```bash
git clone https://m-saravana-golang-crawl-ai-summary.git
cd WebCrawlerWithSummarization
go mod tidy
```

## Usage
```bash
go run cmd/crawler/main.go -url <starting-url> [-config <path-to-config>] [-verbose]
go run cmd/crawler/main.go -url <https://saravananm.netlify.app/blog/rag_evaluation/> -config config.json
```
-url: The starting URL to crawl (required)
-config: Path to configuration file (optional)
-verbose: Enable verbose logging (optional)

## Output:
```
Summary:

1. Key Points:
   - The author created an evaluation pipeline for their RAG app using Ragas, a tool for evaluating Language Models (LLMs).
   - They realized the importance of implementing production metrics, such as context recall, precision, and model adherence to instructions.
   - The author observed inconsistent metrics, high cost considerations, and long evaluation times when using this system.

2. Important Terms:
   - RAG app: A software application whose specifics are not provided in the text, but it seems to be an AI-based tool or service.
   - Named Entity Recognition (NER): A subtask of natural language processing that focuses on identifying and categorizing named entities in text into predefined classes such as person names, organizations, locations, medical codes, time expressions, quantities, monetary values, percentages, etc.
   - LLM: Large Language Model, a type of artificial intelligence model capable of generating human-like text.
   - Ragas: An open-source tool for evaluating LLMs developed by Exploding Gradients.
   - Unit test: A software testing method that checks whether the individual parts of an app work as expected.
   - LLMOps: Unknown, could be a term related to operations or management of large language models.
   - Perturbations: Small changes or variations made to a system for the purpose of understanding its behavior.

3. Main Takeaways:
   - The author found it essential to implement a more robust evaluation pipeline using measurable metrics for their RAG app.
   - They encountered challenges with inconsistent metrics, high costs, and long evaluation times when using Ragas.
   - Despite these challenges, the author remains optimistic about refining the process and is curious about other methods used for evaluating systems.
   
```

## Contributing
Contributions are welcome! Please feel free to submit a Pull Request.

## License
This project is licensed under the MIT License - see the LICENSE file for details.


