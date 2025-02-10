package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webcrawler/config"
	"webcrawler/internal/crawler"
	"webcrawler/internal/summarizer"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.SetOutput(os.Stdout)

	seedURL := flag.String("url", "", "The seed URL to start crawling from")
	configPath := flag.String("config", "", "Path to configuration file")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	if *seedURL == "" {
		log.Fatal("Please provide a seed URL using the -url flag")
	}

	log.Printf("Starting crawler with URL: %s\n", *seedURL)
	log.Printf("Using config file: %s\n", *configPath)
	if *verbose {
		log.Println("Verbose logging enabled")
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	crawlerConfig := &crawler.Config{
		MaxDepth:   cfg.MaxDepth,
		RateLimit:  time.Second / time.Duration(cfg.RateLimit),
		MaxWorkers: cfg.MaxWorkers,
	}

	log.Printf("Crawler config: MaxDepth=%d, RateLimit=%v, MaxWorkers=%d\n",
		crawlerConfig.MaxDepth, crawlerConfig.RateLimit, crawlerConfig.MaxWorkers)

	ollamaSummarizer := summarizer.NewOllamaSummarizer("http://localhost:11434", "mistral")

	crawler, err := crawler.New(crawlerConfig, ollamaSummarizer)
	if err != nil {
		log.Fatalf("Failed to create crawler: %v", err)
	}

	log.Println("\nStarting crawl process...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\nReceived shutdown signal. Cancelling operations...")
		cancel()
	}()

	results, err := crawler.Crawl(ctx, *seedURL)
	if err != nil {
		log.Fatalf("Failed to start crawler: %v", err)
	}

	log.Println("Crawler started successfully, waiting for results...")

	for result := range results {
		if result.Error != nil {
			log.Printf("Error crawling %s: %v\n", result.URL, result.Error)
			continue
		}

		log.Printf("\nProcessed URL: %s (depth: %d)\n", result.URL, result.Depth)
		if result.Summary != "" {
			log.Printf("Summary: %s\n", result.Summary)
		}
		if *verbose {
			log.Printf("Content length: %d bytes\n", len(result.Content))
		}
	}

	log.Println("\nCrawling completed!")
}
