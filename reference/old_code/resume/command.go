package resume

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func HandleResumeCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: resume file required\n")
		fmt.Fprintf(os.Stderr, "Usage: cutlass resume <resume-file>\n")
		fmt.Fprintf(os.Stderr, "Example: cutlass resume ./assets/resume.txt\n")
		os.Exit(1)
	}

	resumeFile := args[0]
	
	// Check if file exists
	if _, err := os.Stat(resumeFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: File '%s' does not exist\n", resumeFile)
		os.Exit(1)
	}

	fmt.Printf("Processing resume file: %s\n", resumeFile)
	
	domains, err := extractDomainsFromFile(resumeFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading resume file: %v\n", err)
		os.Exit(1)
	}

	if len(domains) == 0 {
		fmt.Printf("No domains found in resume file\n")
		return
	}

	fmt.Printf("Found %d domains to screenshot:\n", len(domains))
	for _, domain := range domains {
		fmt.Printf("  - %s\n", domain)
	}

	err = captureScreenshots(domains)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error capturing screenshots: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully captured screenshots for %d domains\n", len(domains))
}

func extractDomainsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var domains []string
	scanner := bufio.NewScanner(file)
	
	// Regex to match domain patterns
	domainRegex := regexp.MustCompile(`([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}`)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines
		if line == "" {
			continue
		}
		
		// Look for lines that DON'T start with two spaces (main lines with domains)
		// The lines with two spaces are descriptions underneath
		if !strings.HasPrefix(line, "  ") {
			// Find all domain matches in the line
			matches := domainRegex.FindAllString(line, -1)
			for _, match := range matches {
				// Clean up common prefixes/suffixes that might be captured
				domain := strings.TrimSuffix(strings.TrimPrefix(match, "www."), ".")
				if domain != "" && !contains(domains, domain) {
					domains = append(domains, domain)
				}
			}
		}
	}

	return domains, scanner.Err()
}

func captureScreenshots(domains []string) error {
	// Create assets directory if it doesn't exist
	assetsDir := "./assets"
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return fmt.Errorf("failed to create assets directory: %v", err)
	}

	// Launch browser
	l := launcher.New().Headless(true)
	defer l.Cleanup()
	url, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %v", err)
	}
	browser := rod.New().ControlURL(url)
	defer browser.Close()

	if err := browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %v", err)
	}

	for _, domain := range domains {
		fmt.Printf("Capturing screenshot for %s...\n", domain)
		
		// Create a new page with panic recovery
		var page *rod.Page
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("  Warning: Failed to create page for %s: %v\n", domain, r)
					page = nil
				}
			}()
			page = browser.MustPage()
		}()
		
		if page == nil {
			continue
		}
		
		// Set a reasonable timeout
		page = page.Timeout(30 * time.Second)
		
		// Navigate to the domain (try HTTPS first, fallback to HTTP)
		urls := []string{
			fmt.Sprintf("https://%s", domain),
			fmt.Sprintf("http://%s", domain),
		}
		
		var loadErr error
		for _, testURL := range urls {
			fmt.Printf("  Trying %s...\n", testURL)
			loadErr = page.Navigate(testURL)
			if loadErr == nil {
				// Wait for page to load with error handling
				loadErr = page.WaitLoad()
				if loadErr != nil {
					fmt.Printf("  Warning: Page load timeout for %s: %v\n", testURL, loadErr)
					continue
				}
				
				// Wait a bit more for dynamic content
				page.WaitRequestIdle(3*time.Second, []string{}, []string{}, nil)
				break
			}
		}
		
		if loadErr != nil {
			fmt.Printf("  Warning: Failed to load %s: %v\n", domain, loadErr)
			safeClosePage(page)
			continue
		}
		
		// Take screenshot of visible viewport only (no scrolling)
		filename := filepath.Join(assetsDir, fmt.Sprintf("%s.png", domain))
		screenshot, err := page.Screenshot(false, nil)
		if err != nil {
			fmt.Printf("  Warning: Failed to take screenshot of %s: %v\n", domain, err)
			safeClosePage(page)
			continue
		}
		
		// Save screenshot
		err = os.WriteFile(filename, screenshot, 0644)
		if err != nil {
			fmt.Printf("  Warning: Failed to save screenshot for %s: %v\n", domain, err)
			safeClosePage(page)
			continue
		}
		
		fmt.Printf("  Saved screenshot: %s\n", filename)
		safeClosePage(page)
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func safeClosePage(page *rod.Page) {
	if page == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			// Ignore panic from page close
		}
	}()
	page.Close()
}