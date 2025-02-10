package crawler

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"webcrawler/internal/parser"
	"webcrawler/internal/summarizer"
)

type Crawler struct {
	config     *Config
	visited    sync.Map
	limiter    *time.Ticker
	httpClient *http.Client
	summarizer *summarizer.OllamaSummarizer
}

type Config struct {
	MaxDepth    int           `json:"max_depth"`
	RateLimit   time.Duration `json:"rate_limit"`
	MaxWorkers  int           `json:"max_workers"`
	AllowedHost string        `json:"allowed_host"`
}

type Result struct {
	URL     string
	Content string
	Links   []string
	Depth   int
	Summary string
	Error   error
}

func New(config *Config, summarizer *summarizer.OllamaSummarizer) (*Crawler, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %v", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	return &Crawler{
		config:     config,
		limiter:    time.NewTicker(config.RateLimit),
		httpClient: client,
		summarizer: summarizer,
	}, nil
}

func (c *Crawler) Crawl(ctx context.Context, seedURL string) (<-chan Result, error) {
	parsedURL, err := url.Parse(seedURL)
	if err != nil {
		return nil, fmt.Errorf("invalid seed URL: %v", err)
	}

	if !parsedURL.IsAbs() {
		return nil, fmt.Errorf("seed URL must be absolute")
	}

	log.Printf("DEBUG: Starting Crawl function with seed URL: %s\n", seedURL)

	jobs := make(chan string, c.config.MaxWorkers)
	results := make(chan Result, c.config.MaxWorkers)

	var wg sync.WaitGroup
	log.Printf("DEBUG: Starting %d worker goroutines\n", c.config.MaxWorkers)
	for i := 0; i < c.config.MaxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case url, ok := <-jobs:
					if !ok {
						return
					}
					log.Printf("DEBUG: Worker %d processing URL: %s\n", workerID, url)
					result := c.crawlURL(ctx, url, 0)
					select {
					case <-ctx.Done():
						return
					case results <- result:
					}
				}
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	jobs <- seedURL
	close(jobs)

	return results, nil
}

func (c *Crawler) crawlURL(ctx context.Context, urlStr string, depth int) Result {
	result := Result{
		URL:   urlStr,
		Depth: depth,
	}

	if depth >= c.config.MaxDepth {
		return result
	}

	log.Printf("DEBUG: Waiting for rate limiter before fetching %s\n", urlStr)
	select {
	case <-ctx.Done():
		result.Error = ctx.Err()
		return result
	case <-c.limiter.C:
	}

	log.Printf("DEBUG: Fetching URL: %s\n", urlStr)

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %v", err)
		return result
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch URL: %v", err)
		return result
	}
	defer resp.Body.Close()

	log.Printf("DEBUG: Response received for %s - Status: %s, Headers: %v\n", urlStr, resp.Status, resp.Header)
	log.Printf("DEBUG: Final URL after redirects: %s\n", resp.Request.URL.String())

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
		return result
	}

	contentType := resp.Header.Get("Content-Type")
	log.Printf("DEBUG: Content-Type for %s: %s\n", urlStr, contentType)

	if !c.isAllowedHost(resp.Request.URL.String()) {
		result.Error = fmt.Errorf("non-allowed host: %s", resp.Request.URL.String())
		return result
	}

	if !strings.Contains(strings.ToLower(contentType), "text/html") {
		result.Error = fmt.Errorf("non-HTML content type: %s", contentType)
		return result
	}

	log.Printf("DEBUG: Starting to parse content from %s using Playwright\n", urlStr)
	parseResult, err := parser.ParseWithPlaywright(urlStr)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse content: %v", err)
		return result
	}

	var links []string
	for _, link := range parseResult.Links {
		parsedLink, err := url.Parse(link)
		if err != nil {
			log.Printf("WARNING: Failed to parse link %s: %v\n", link, err)
			continue
		}

		baseURL := resp.Request.URL
		if !parsedLink.IsAbs() {
			parsedLink = baseURL.ResolveReference(parsedLink)
		}

		cleanedLink := parsedLink.String()
		cleanedLink = strings.TrimRight(cleanedLink, "/") // Remove trailing slash for consistency

		if _, visited := c.visited.LoadOrStore(cleanedLink, true); !visited {
			links = append(links, cleanedLink)
		}
	}

	log.Printf("DEBUG: Found %d links in %s\n", len(links), urlStr)

	if parseResult.Text != "" {
		log.Printf("DEBUG: Starting summary generation for %s\n", urlStr)
		summary, err := c.summarizer.Summarize(parseResult.Text)
		if err != nil {
			log.Printf("ERROR: Failed to generate summary for %s: %v\n", urlStr, err)
		} else {
			log.Printf("DEBUG: Successfully generated summary for %s (%d chars)\n", urlStr, len(summary))
			result.Summary = summary
		}
	} else {
		log.Printf("WARNING: No content to summarize for %s\n", urlStr)
	}

	result.Content = parseResult.Text
	result.Links = links
	return result
}

func (c *Crawler) isAllowedHost(urlStr string) bool {
	if c.config.AllowedHost == "" {
		return true
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	log.Printf("DEBUG: Host of URL: %s\n", parsedURL.Host)

	return strings.Contains(parsedURL.Host, c.config.AllowedHost)
}

func waitForAuthentication(authURL string) bool {
	fmt.Printf("\nAuthentication required. Please authenticate in your browser.\n")
	fmt.Printf("Press Enter when you're done, or type 'cancel' to abort: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return !strings.Contains(strings.ToLower(input), "cancel")
}
