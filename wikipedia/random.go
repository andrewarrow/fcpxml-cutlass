package wikipedia

import (
	"fmt"
	"os"
	"os/exec"
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

	// Get current page URL
	pageInfo, err := page.Info()
	if err != nil {
		fmt.Printf("Warning: Could not get page URL: %v\n", err)
	} else {
		fmt.Printf("Wikipedia URL: %s\n", pageInfo.URL)
		
		// Append URL to wikilist.txt
		if err := appendToWikiList(pageInfo.URL); err != nil {
			fmt.Printf("Warning: Could not append URL to wikilist.txt: %v\n", err)
		} else {
			fmt.Printf("URL appended to data/wikilist.txt\n")
		}
	}

	// Extract first paragraph
	firstParagraph, err := extractFirstParagraph(page)
	if err != nil {
		fmt.Printf("Warning: Could not extract first paragraph: %v\n", err)
	} else {
		fmt.Printf("\n%s\n\n", firstParagraph)
	}

	// Create filename-safe version of title
	filenameTitle := sanitizeFilename(title)

	// Navigate to Google Videos search
	searchQuery := fmt.Sprintf("https://www.google.com/search?tbm=vid&q=%s", strings.ReplaceAll(title, " ", "+"))
	fmt.Printf("Searching Google Videos for: %s\n", title)

	err = page.Navigate(searchQuery)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error navigating to Google Videos: %v\n", err)
		os.Exit(1)
	}

	// Wait for Google Videos to load
	err = page.WaitLoad()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error waiting for Google Videos to load: %v\n", err)
		os.Exit(1)
	}

	// Wait for videos to load
	page.WaitRequestIdle(5*time.Second, []string{}, []string{}, nil)

	// Find and click the first video link
	fmt.Println("Looking for first video link...")

	// Debug: Print page title to confirm we're on the right page
	pageTitle, _ := page.Eval("document.title")
	fmt.Printf("Debug: Current page title: %v\n", pageTitle)

	// Debug: Try multiple selectors to find video links
	selectors := []string{
		"div.g h3 a",
		"div[data-ved] h3 a",
		"h3.LC20lb a",
		"a[href*='youtube.com']",
		"a[href*='watch']",
		"div.g a",
	}

	var firstVideoLink *rod.Element
	for _, selector := range selectors {
		fmt.Printf("Debug: Trying selector: %s\n", selector)
		elements, err := page.Elements(selector)
		if err != nil {
			fmt.Printf("Debug: Error with selector %s: %v\n", selector, err)
			continue
		}
		fmt.Printf("Debug: Found %d elements with selector %s\n", len(elements), selector)

		if len(elements) > 0 {
			firstVideoLink = elements[0]
			fmt.Printf("Debug: Using first element from selector: %s\n", selector)
			break
		}
	}

	if firstVideoLink == nil {
		// Debug: Print page HTML snippet to see structure
		bodyHTML, _ := page.Eval("document.body.innerHTML.substring(0, 1000)")
		fmt.Printf("Debug: Page HTML snippet: %v\n", bodyHTML)
		fmt.Fprintf(os.Stderr, "Error: Could not find any video links with any selector\n")
		os.Exit(1)
	}

	// Debug: Print the actual link URL before clicking
	linkHref, err := firstVideoLink.Attribute("href")
	if err != nil {
		fmt.Printf("Debug: Could not get href attribute: %v\n", err)
	} else {
		fmt.Printf("Debug: About to click link: %s\n", *linkHref)
	}

	// Get the video URL
	if linkHref != nil && *linkHref != "" {
		videoURL := *linkHref
		fmt.Printf("Found video URL: %s\n", videoURL)
		
		// Append video URL to youtube.txt
		if err := appendToYouTubeList(videoURL); err != nil {
			fmt.Printf("Warning: Could not append video URL to youtube.txt: %v\n", err)
		} else {
			fmt.Printf("Video URL appended to data/youtube.txt\n")
		}

		// Close the browser since we no longer need it
		page.Close()
		browser.Close()
		l.Cleanup()

		// Use yt-dlp to download thumbnail
		fmt.Println("Using yt-dlp to download video thumbnail...")

		// Create final filename
		finalFilename := filepath.Join(dataDir, fmt.Sprintf("wiki_%s.png", filenameTitle))

		// Run yt-dlp command
		cmd := exec.Command("yt-dlp", "--write-thumbnail", "--skip-download", "-o", filepath.Join(dataDir, "temp_thumbnail.%(ext)s"), videoURL)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running yt-dlp: %v\n", err)
			fmt.Fprintf(os.Stderr, "yt-dlp output: %s\n", string(output))
			os.Exit(1)
		}

		fmt.Printf("yt-dlp output: %s\n", string(output))

		// Find the downloaded thumbnail file and rename it
		thumbnailFiles, err := filepath.Glob(filepath.Join(dataDir, "temp_thumbnail.*"))
		if err != nil || len(thumbnailFiles) == 0 {
			fmt.Fprintf(os.Stderr, "Error: Could not find downloaded thumbnail file\n")
			os.Exit(1)
		}

		// Rename the first thumbnail file to our desired name
		err = os.Rename(thumbnailFiles[0], finalFilename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error renaming thumbnail file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Thumbnail saved: %s\n", finalFilename)

		// Generate speech from first paragraph using chatterbox
		if firstParagraph != "" {
			fmt.Println("Generating speech from first paragraph...")
			audioFilename := filepath.Join(dataDir, fmt.Sprintf("wiki_%s.wav", filenameTitle))

			// Call chatterbox CLI to generate speech
			chatterboxCmd := exec.Command("/opt/miniconda3/envs/chatterbox/bin/python3",
				"/Users/aa/os/chatterbox/dia/cli.py",
				firstParagraph,
				"--output="+audioFilename)

			chatterboxOutput, err := chatterboxCmd.CombinedOutput()
			if err != nil {
				fmt.Printf("Warning: Could not generate speech: %v\n", err)
				fmt.Printf("Chatterbox output: %s\n", string(chatterboxOutput))
			} else {
				fmt.Printf("Speech generated: %s\n", audioFilename)
			}
		}

	} else {
		fmt.Fprintf(os.Stderr, "Error: Link href is empty\n")
		page.Close()
		browser.Close()
		l.Cleanup()
		os.Exit(1)
	}
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

func extractFirstParagraph(page *rod.Page) (string, error) {
	// Try to find the first paragraph in the Wikipedia article content
	// Wikipedia articles typically have the first paragraph in #mw-content-text .mw-parser-output > p
	paragraphs, err := page.Elements("#mw-content-text .mw-parser-output > p")
	if err != nil {
		return "", fmt.Errorf("could not find paragraphs: %v", err)
	}

	if len(paragraphs) == 0 {
		return "", fmt.Errorf("no paragraphs found")
	}

	// Collect text from paragraphs until we have at least 90 words
	var combinedText strings.Builder
	wordCount := 0

	for _, p := range paragraphs {
		text, err := p.Text()
		if err != nil {
			continue
		}

		// Skip empty paragraphs or ones that are just whitespace
		trimmed := strings.TrimSpace(text)
		if len(trimmed) == 0 {
			continue
		}

		// Add paragraph text
		if combinedText.Len() > 0 {
			combinedText.WriteString(" ")
		}
		combinedText.WriteString(trimmed)

		// Count words in this paragraph
		words := strings.Fields(trimmed)
		wordCount += len(words)

		// If we have at least 90 words, we're done
		if wordCount >= 9 {
			break
		}
	}

	if combinedText.Len() == 0 {
		return "", fmt.Errorf("no non-empty paragraphs found")
	}

	return combinedText.String(), nil
}

func appendToWikiList(url string) error {
	wikiListPath := filepath.Join("data", "wikilist.txt")
	
	// Open file in append mode, create if doesn't exist
	file, err := os.OpenFile(wikiListPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening wikilist.txt: %v", err)
	}
	defer file.Close()
	
	// Append URL with newline
	_, err = file.WriteString(url + "\n")
	if err != nil {
		return fmt.Errorf("error writing to wikilist.txt: %v", err)
	}
	
	return nil
}

func appendToYouTubeList(url string) error {
	youtubeListPath := filepath.Join("data", "youtube.txt")
	
	// Open file in append mode, create if doesn't exist
	file, err := os.OpenFile(youtubeListPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening youtube.txt: %v", err)
	}
	defer file.Close()
	
	// Append URL with newline
	_, err = file.WriteString(url + "\n")
	if err != nil {
		return fmt.Errorf("error writing to youtube.txt: %v", err)
	}
	
	return nil
}
