package wikipedia

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func HandleWikipediaRandomCommand(args []string) {
	fmt.Println("Fetching random Wikipedia article...")

	// Create data directory if it doesn't exist
	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating data directory: %v\n", err)
		os.Exit(1)
	}

	// Launch browser
	l := launcher.New().Headless(true)
	defer l.Cleanup()
	url, err := l.Launch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error launching browser: %v\n", err)
		os.Exit(1)
	}
	browser := rod.New().ControlURL(url)
	defer browser.Close()

	if err := browser.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to browser: %v\n", err)
		os.Exit(1)
	}

	// Create page with panic recovery
	var page *rod.Page
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Error creating page: %v\n", r)
				os.Exit(1)
			}
		}()
		page = browser.MustPage()
	}()

	// Set timeout
	page = page.Timeout(30 * time.Second)

	// Navigate to Wikipedia random page
	fmt.Println("Loading random Wikipedia page...")
	err = page.Navigate("https://en.wikipedia.org/wiki/Special:Random")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error navigating to Wikipedia: %v\n", err)
		os.Exit(1)
	}

	// Wait for page to load
	err = page.WaitLoad()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error waiting for page load: %v\n", err)
		os.Exit(1)
	}

	// Wait for dynamic content
	page.WaitRequestIdle(3*time.Second, []string{}, []string{}, nil)

	// Extract title from the page
	titleElement, err := page.Element("h1.firstHeading")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding title element: %v\n", err)
		os.Exit(1)
	}
	
	title, err := titleElement.Text()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting title: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found article: %s\n", title)

	// Create filename-safe version of title
	filenameTitle := sanitizeFilename(title)

	// Navigate to Google Images search
	searchQuery := fmt.Sprintf("https://www.google.com/search?tbm=isch&q=%s", strings.ReplaceAll(title, " ", "+"))
	fmt.Printf("Searching Google Images for: %s\n", title)
	
	err = page.Navigate(searchQuery)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error navigating to Google Images: %v\n", err)
		os.Exit(1)
	}

	// Wait for Google Images to load
	err = page.WaitLoad()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error waiting for Google Images to load: %v\n", err)
		os.Exit(1)
	}

	// Wait for images to load
	page.WaitRequestIdle(5*time.Second, []string{}, []string{}, nil)

	// Take screenshot
	filename := filepath.Join(dataDir, fmt.Sprintf("wiki_%s.png", filenameTitle))
	screenshot, err := page.Screenshot(false, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error taking screenshot: %v\n", err)
		os.Exit(1)
	}

	// Save screenshot
	err = os.WriteFile(filename, screenshot, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error saving screenshot: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Screenshot saved: %s\n", filename)
	page.Close()
}

func sanitizeFilename(title string) string {
	// Remove or replace characters that aren't safe for filenames
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\-\s]`)
	safe := reg.ReplaceAllString(title, "")
	
	// Replace spaces with underscores and convert to lowercase
	safe = strings.ReplaceAll(strings.TrimSpace(safe), " ", "_")
	safe = strings.ToLower(safe)
	
	// Limit length to avoid filesystem issues
	if len(safe) > 100 {
		safe = safe[:100]
	}
	
	return safe
}