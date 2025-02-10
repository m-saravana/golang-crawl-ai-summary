package parser

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/playwright-community/playwright-go"
)

type ParseResult struct {
	Text  string
	Links []string
}

var (
	pw      *playwright.Playwright
	browser playwright.Browser
	once    sync.Once
	initErr error
)

func initPlaywright() error {
	once.Do(func() {
		runOpts := &playwright.RunOptions{
			SkipInstallBrowsers: false,
		}
		pw, err := playwright.Run(runOpts)
		if err != nil {
			initErr = fmt.Errorf("failed to start playwright: %v", err)
			return
		}

		launchOpts := playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(true),
			Args: []string{
				"--disable-gpu",
				"--no-sandbox",
				"--disable-setuid-sandbox",
				"--disable-web-security",
				"--disable-features=IsolateOrigins,site-per-process",
			},
		}
		browser, err = pw.Chromium.Launch(launchOpts)
		if err != nil {
			initErr = fmt.Errorf("failed to launch browser: %v", err)
			if pw != nil {
				pw.Stop()
			}
			return
		}
	})
	return initErr
}

func ParseWithPlaywright(url string) (ParseResult, error) {
	if err := initPlaywright(); err != nil {
		return ParseResult{}, fmt.Errorf("failed to initialize playwright: %v", err)
	}

	contextOpts := playwright.BrowserNewContextOptions{
		JavaScriptEnabled: playwright.Bool(true),
		UserAgent:         playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		ExtraHttpHeaders: map[string]string{
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"Accept-Language": "en-US,en;q=0.5",
		},
	}
	context, err := browser.NewContext(contextOpts)
	if err != nil {
		return ParseResult{}, fmt.Errorf("failed to create browser context: %v", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		return ParseResult{}, fmt.Errorf("failed to create page: %v", err)
	}

	page.SetDefaultTimeout(45000) // 45 seconds
	page.SetDefaultNavigationTimeout(45000)

	log.Printf("DEBUG: Navigating to URL: %s\n", url)
	if _, err := page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000),
	}); err != nil {
		return ParseResult{}, fmt.Errorf("failed to navigate to URL: %v", err)
	}

	log.Printf("DEBUG: Page loaded, waiting for content to be visible...")

	log.Printf("DEBUG: Trying direct content extraction...")
	contentHandle, err := page.EvaluateHandle(`() => {
		try {
			// Try to find the main content container
			const selectors = [
				'article',
				'main article',
				'.blog-content',
				'.post-content',
				'main',
				'.content',
				'#content',
				'body'
			];

			let content = null;
			for (const selector of selectors) {
				content = document.querySelector(selector);
				if (content) {
					console.log('Found content using selector:', selector);
					break;
				}
			}

			if (!content) {
				console.warn('No content element found');
				return '';
			}

			// Create a copy of the content to manipulate
			const clone = content.cloneNode(true);

			// Remove non-content elements
			[
				'script',
				'style',
				'pre',
				'code',
				'nav',
				'footer',
				'header',
				'aside',
				'#skip-to-main',
				'.skip-to-main',
				'.navigation',
				'.nav-menu',
				'.menu',
				'.sidebar',
				'.table-of-contents',
				'.social-share',
				'.share-buttons',
				'.comments',
				'.comment-section',
				'.site-header',
				'.site-footer',
				'.site-navigation',
				'.breadcrumbs'
			].forEach(selector => {
				const elements = clone.querySelectorAll(selector);
				console.log('Removing', elements.length, selector, 'elements');
				elements.forEach(el => el.remove());
			});

			// Get text content and clean it up
			let text = clone.textContent;

			// Clean up the text in multiple steps
			text = text.replace(/\s+/g, ' ');  // Replace multiple whitespace with single space
			text = text.replace(/^\s+|\s+$/g, '');  // Trim whitespace
			text = text.replace(/Skip to (?:main )?content/gi, '');  // Remove "Skip to content" text
			text = text.replace(/Home\s*Blogs\s*Thoughts/g, '');  // Remove navigation text
			text = text.replace(/\s*- RAG evaluation/g, '');  // Remove title duplication
			text = text.replace(/rag_evaluation\/?/g, '');  // Remove URL fragments
			text = text.replace(/\s{3,}/g, '\n\n');  // Replace 3+ spaces with newlines
			text = text.trim();

			// Add some structure back
			text = text.split(/\n{2,}/)  // Split on multiple newlines
				.filter(para => para.trim().length > 0)  // Remove empty paragraphs
				.map(para => para.trim())  // Trim each paragraph
				.join('\n\n');  // Join with double newlines

			console.log('Successfully extracted content:', text.substring(0, 100) + '...');
			return text;
		} catch (error) {
			console.error('Error extracting content:', error);
			return '';
		}
	}`)
	if err != nil {
		return ParseResult{}, fmt.Errorf("failed to extract content: %v", err)
	}
	defer contentHandle.Dispose()

	content, err := contentHandle.JSONValue()
	if err != nil {
		return ParseResult{}, fmt.Errorf("failed to get content value: %v", err)
	}

	log.Printf("DEBUG: Extracting links...")
	linksHandle, err := page.EvaluateHandle(`() => {
		try {
			const links = document.querySelectorAll('a[href]');
			console.log('Found', links.length, 'links');
			const extractedLinks = Array.from(links)
				.map(link => link.href)
				.filter(href => href && (href.startsWith('http://') || href.startsWith('https://')))
				.filter(href => !href.includes('javascript:'))
				.filter((href, index, self) => self.indexOf(href) === index); // Remove duplicates
			console.log('After filtering:', extractedLinks.length, 'links remain');
			return extractedLinks;
		} catch (error) {
			console.error('Error extracting links:', error);
			return [];
		}
	}`)
	if err != nil {
		return ParseResult{}, fmt.Errorf("failed to extract links: %v", err)
	}
	defer linksHandle.Dispose()

	links, err := linksHandle.JSONValue()
	if err != nil {
		return ParseResult{}, fmt.Errorf("failed to get links value: %v", err)
	}

	contentStr := ""
	if content != nil {
		contentStr = strings.TrimSpace(content.(string))
	}

	var linksList []string
	if linksArr, ok := links.([]interface{}); ok {
		for _, link := range linksArr {
			if linkStr, ok := link.(string); ok {
				linksList = append(linksList, linkStr)
			}
		}
	}

	log.Printf("DEBUG: Extracted %d bytes of content and %d links\n", len(contentStr), len(linksList))
	if len(contentStr) > 0 {
		log.Printf("DEBUG: First 100 chars of content: %s\n", contentStr[:min(100, len(contentStr))])
	}

	return ParseResult{
		Text:  contentStr,
		Links: linksList,
	}, nil
}

func Cleanup() {
	if browser != nil {
		if err := browser.Close(); err != nil {
			log.Printf("ERROR: Failed to close browser: %v", err)
		}
	}
	if pw != nil {
		if err := pw.Stop(); err != nil {
			log.Printf("ERROR: Failed to stop playwright: %v", err)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
