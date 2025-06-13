package hackernews

import (
	"fmt"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// HandleHackerNewsCommand fetches and displays Hacker News article titles using Rod
func HandleHackerNewsCommand(args []string) {
	fmt.Println("Fetching Hacker News article titles...")

	// Launch browser
	l := launcher.New().Headless(true)
	defer l.Cleanup()
	url, err := l.Launch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
		return
	}
	browser := rod.New().ControlURL(url)
	defer browser.Close()

	if err := browser.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to browser: %v\n", err)
		return
	}

	// Create page with panic recovery
	var page *rod.Page
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Error creating page: %v\n", r)
				return
			}
		}()
		page = browser.MustPage()
	}()

	// Set timeout
	page = page.Timeout(30 * time.Second)

	// Navigate to Hacker News
	fmt.Println("Loading Hacker News homepage...")
	err = page.Navigate("https://news.ycombinator.com/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error navigating to Hacker News: %v\n", err)
		return
	}

	// Wait for page to load
	err = page.WaitLoad()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error waiting for page load: %v\n", err)
		return
	}

	// Wait for dynamic content
	page.WaitRequestIdle(3*time.Second, []string{}, []string{}, nil)

	// Extract article titles using Rod's DOM methods
	titles, err := extractHackerNewsTitles(page)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting titles: %v\n", err)
		return
	}

	fmt.Println("Hacker News Article Titles:")
	fmt.Println("==========================")
	for i, title := range titles {
		fmt.Printf("%d. %s\n", i+1, title)
	}
}

// extractHackerNewsTitles extracts article titles from Hacker News using Rod's DOM methods
func extractHackerNewsTitles(page *rod.Page) ([]string, error) {
	// Hacker News uses <span class="titleline"><a>Title</a></span> structure
	titleElements, err := page.Elements("span.titleline a")
	if err != nil {
		return nil, fmt.Errorf("could not find title elements: %v", err)
	}

	if len(titleElements) == 0 {
		return nil, fmt.Errorf("no title elements found")
	}

	var titles []string
	for _, element := range titleElements {
		title, err := element.Text()
		if err != nil {
			continue // Skip elements that can't be read
		}

		// Skip empty titles
		if title != "" {
			titles = append(titles, title)
		}
	}

	if len(titles) == 0 {
		return nil, fmt.Errorf("no non-empty titles found")
	}

	return titles, nil
}