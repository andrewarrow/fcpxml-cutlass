package hackernews

import (
	"cutlass/browser"
	"cutlass/build2/api"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// HandleHackerNewsStep1Command processes step 1: get articles and take screenshots
func HandleHackerNewsStep1Command(args []string) {
	fmt.Println("Processing Hacker News articles - Step 1: Articles and Screenshots...")

	// Create data directory if it doesn't exist
	if err := browser.EnsureDataDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating data directory: %v\n", err)
		return
	}

	// Create browser session
	session, err := browser.NewBrowserSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating browser session: %v\n", err)
		return
	}
	defer session.Close()

	// Determine which Hacker News page to load
	url := "https://news.ycombinator.com/"
	pageName := "homepage"
	if len(args) > 0 && args[0] == "newest" {
		url = "https://news.ycombinator.com/newest"
		pageName = "newest page"
	}

	// Navigate to Hacker News and fetch all articles
	fmt.Printf("Loading Hacker News %s...\n", pageName)
	if err := session.NavigateAndWait(url); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading Hacker News: %v\n", err)
		return
	}

	// Get all articles from the selected page
	fmt.Println("Fetching all articles from Hacker News...")
	articles, err := getAllHNArticles(session)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting articles: %v\n", err)
		return
	}

	fmt.Printf("Found %d articles to process\n", len(articles))

	// Write articles to text file
	if err := writeArticlesToFile(articles); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing articles to file: %v\n", err)
		return
	}

	// Process each article for step 1 (articles and screenshots only)
	for i, article := range articles {
		processHNArticleStep1(session, article, i)
	}

	fmt.Println("Step 1 completed. Run 'cutlass download hn-step-2' to generate audio files.")
}

// HandleHackerNewsStep2Command processes step 2: generate audio files
func HandleHackerNewsStep2Command(args []string) {
	fmt.Println("Processing Hacker News articles - Step 2: Audio Generation...")

	// Read articles from file
	articles, err := readArticlesFromFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading articles from file: %v\n", err)
		fmt.Println("Please run 'cutlass download hn-step-1' first.")
		return
	}

	fmt.Printf("Found %d articles to process for audio generation\n", len(articles))

	// Create or get project builder for hn.fcpxml
	hnProjectFile := "data/hn.fcpxml"
	pb, err := api.NewProjectBuilder(hnProjectFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating project builder: %v\n", err)
		return
	}

	// Track cumulative time for timecode generation
	var cumulativeSeconds float64 = 0
	var timecodeEntries []string

	// Process each article for step 2 (audio generation and FCPXML)
	for i, article := range articles {
		duration := processHNArticleStep2(article, i, pb)
		if duration > 0 {
			// Format current timecode as MM:SS
			timecode := formatTimecode(cumulativeSeconds)
			entry := fmt.Sprintf("%s (%s)[%s]", timecode, article.Title, article.URL)
			timecodeEntries = append(timecodeEntries, entry)
			
			// Add duration to cumulative time
			cumulativeSeconds += duration
		}
	}

	// Save the project
	err = pb.Save()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error saving project: %v\n", err)
		return
	}

	// Write timecode entries to file
	if len(timecodeEntries) > 0 {
		err = writeTimecodesToFile(timecodeEntries)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing timecodes to file: %v\n", err)
		}
	}

	fmt.Println("Step 2 completed. All Hacker News articles processed.")
}

// processHNArticleStep1 processes a single HN article for step 1 (articles and screenshots)
func processHNArticleStep1(session *browser.BrowserSession, article *HNArticle, index int) {
	// Create a fresh browser session for this article
	freshSession, err := browser.NewBrowserSession()
	if err != nil {
		fmt.Printf("Warning: Could not create fresh browser session for article %d: %v\n", index+1, err)
		// Fall back to using the existing session
		freshSession = session
	} else {
		defer freshSession.Close()
		// Use the fresh session for all operations
		session = freshSession
	}
	fmt.Printf("Processing article %d: %s\n", index+1, article.Title)
	fmt.Printf("Article URL: %s\n", article.URL)

	// Append URL to hnlist.txt
	if err := appendToHNList(article.URL); err != nil {
		fmt.Printf("Warning: Could not append URL to hnlist.txt: %v\n", err)
	} else {
		fmt.Printf("URL appended to data/hnlist.txt\n")
	}

	// Print article summary if available
	if article.Summary != "" {
		fmt.Printf("\n%s\n\n", article.Summary)
	}

	// Create filename-safe version of title with index
	filenameTitle := fmt.Sprintf("%d_%s", index+1, sanitizeFilename(article.Title))

	// Navigate to the article URL and take a screenshot
	fmt.Printf("Navigating to article URL for screenshot: %s\n", article.URL)

	if err := session.NavigateAndWait(article.URL); err != nil {
		fmt.Printf("Warning: Could not navigate to article URL: %v\n", err)
	} else {
		// Create final filename for screenshot
		finalFilename := filepath.Join("data", fmt.Sprintf("hn_%s.png", filenameTitle))

		// Take screenshot of the page (visible area only)
		fmt.Println("Taking screenshot of article page...")
		screenshot, err := session.Page.Screenshot(false, nil)
		if err != nil {
			fmt.Printf("Warning: Could not take screenshot: %v\n", err)
		} else {
			// Save screenshot to file
			err = os.WriteFile(finalFilename, screenshot, 0644)
			if err != nil {
				fmt.Printf("Warning: Could not save screenshot: %v\n", err)
			} else {
				fmt.Printf("Screenshot saved: %s\n", finalFilename)
			}
		}
	}

	fmt.Printf("Completed processing article %d (Step 1)\n\n", index+1)
}

// processHNArticleStep2 processes a single HN article for step 2 (audio generation)
// Returns the duration of the generated audio file in seconds (0 if failed)
func processHNArticleStep2(article *HNArticle, index int, pb *api.ProjectBuilder) float64 {
	fmt.Printf("Processing article %d for audio: %s\n", index+1, article.Title)

	// Create filename-safe version of title with index
	filenameTitle := fmt.Sprintf("%d_%s", index+1, sanitizeFilename(article.Title))

	// Check if thumbnail exists
	thumbnailPath := filepath.Join("data", fmt.Sprintf("hn_%s.png", filenameTitle))
	if _, err := os.Stat(thumbnailPath); os.IsNotExist(err) {
		fmt.Printf("Warning: Thumbnail not found for article %d, skipping audio generation\n", index+1)
		return 0
	}

	// Generate speech from article title using chatterbox
	speechText := article.Title

	fmt.Println("Generating speech from article title...")
	audioFilename := filepath.Join("data", fmt.Sprintf("hn_%s.wav", filenameTitle))

	chatterboxCmd := exec.Command("/opt/miniconda3/envs/chatterbox/bin/python3",
		"/Users/aa/os/chatterbox/chatterbox/main.py",
		speechText,
		audioFilename)

	chatterboxOutput, err := chatterboxCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Warning: Could not generate speech: %v\n", err)
		fmt.Printf("Chatterbox output: %s\n", string(chatterboxOutput))
		return 0
	} else {
		fmt.Printf("Speech generated: %s\n", audioFilename)

		// Get audio duration
		duration, err := getAudioDurationSeconds(audioFilename)
		if err != nil {
			fmt.Printf("Warning: Could not get audio duration: %v\n", err)
			duration = 0
		}

		// Add video/image with audio and text to project using build2 API
		err = pb.AddClipSafe(api.ClipConfig{
			VideoFile: thumbnailPath,
			AudioFile: audioFilename,
			Text:      article.Title,
		})
		if err != nil {
			fmt.Printf("Warning: Could not add clip to project: %v\n", err)
			return 0
		} else {
			fmt.Printf("Clip added to hn.fcpxml\n")
		}

		fmt.Printf("Completed processing article %d (Step 2)\n\n", index+1)
		return duration
	}
}

// HNArticle represents a Hacker News article
type HNArticle struct {
	Title   string
	URL     string
	Summary string
}

// getAllHNArticles gets all articles from Hacker News homepage
func getAllHNArticles(session *browser.BrowserSession) ([]*HNArticle, error) {
	// Get all title elements
	titleElements, err := session.Page.Elements("span.titleline a")
	if err != nil || len(titleElements) == 0 {
		return nil, fmt.Errorf("could not find title elements")
	}

	var articles []*HNArticle
	for i, element := range titleElements {
		title, err := element.Text()
		if err != nil {
			fmt.Printf("Warning: Could not get title text for article %d: %v\n", i, err)
			continue
		}

		articleURL, err := element.Attribute("href")
		if err != nil || articleURL == nil {
			fmt.Printf("Warning: Could not get article URL for article %d\n", i)
			continue
		}

		// Skip relative URLs (not actual article links)
		url := *articleURL
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			continue
		}

		// Try to get article summary by looking for comment count or points
		summary := ""

		articles = append(articles, &HNArticle{
			Title:   title,
			URL:     url,
			Summary: summary,
		})
	}

	if len(articles) == 0 {
		return nil, fmt.Errorf("no valid articles found")
	}

	return articles, nil
}

// appendToHNList appends URL to hnlist.txt
func appendToHNList(url string) error {
	hnListPath := filepath.Join("data", "hnlist.txt")
	file, err := os.OpenFile(hnListPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening hnlist.txt: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(url + "\n")
	if err != nil {
		return fmt.Errorf("error writing to hnlist.txt: %v", err)
	}

	return nil
}

// writeArticlesToFile writes articles to a text file for step 2
func writeArticlesToFile(articles []*HNArticle) error {
	articlesPath := filepath.Join("data", "hn_articles.txt")
	file, err := os.Create(articlesPath)
	if err != nil {
		return fmt.Errorf("error creating articles file: %v", err)
	}
	defer file.Close()

	for i, article := range articles {
		line := fmt.Sprintf("%d|%s|%s\n", i+1, article.Title, article.URL)
		_, err = file.WriteString(line)
		if err != nil {
			return fmt.Errorf("error writing article to file: %v", err)
		}
	}

	fmt.Printf("Articles written to %s\n", articlesPath)
	return nil
}

// readArticlesFromFile reads articles from text file for step 2
func readArticlesFromFile() ([]*HNArticle, error) {
	articlesPath := filepath.Join("data", "hn_articles.txt")
	content, err := os.ReadFile(articlesPath)
	if err != nil {
		return nil, fmt.Errorf("error reading articles file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	var articles []*HNArticle

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}

		articles = append(articles, &HNArticle{
			Title: parts[1],
			URL:   parts[2],
		})
	}

	return articles, nil
}

// sanitizeFilename creates a filename-safe version of a string
func sanitizeFilename(title string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\-\s]`)
	safe := reg.ReplaceAllString(title, "")
	safe = strings.ReplaceAll(strings.TrimSpace(safe), " ", "_")
	safe = strings.ToLower(safe)

	if len(safe) > 100 {
		safe = safe[:100]
	}

	return safe
}

// getAudioDurationSeconds uses ffprobe to get audio duration in seconds
func getAudioDurationSeconds(audioPath string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", audioPath)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	
	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	
	err = json.Unmarshal(output, &result)
	if err != nil {
		return 0, err
	}
	
	// Parse duration as float seconds
	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return 0, err
	}
	
	return duration, nil
}

// formatTimecode converts seconds to MM:SS format
func formatTimecode(seconds float64) string {
	totalSeconds := int(seconds)
	minutes := totalSeconds / 60
	secs := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}

// writeTimecodesToFile writes timecode entries to hn_timecodes.txt
func writeTimecodesToFile(entries []string) error {
	timecodesPath := filepath.Join("data", "hn_timecodes.txt")
	file, err := os.Create(timecodesPath)
	if err != nil {
		return fmt.Errorf("error creating timecodes file: %v", err)
	}
	defer file.Close()

	for _, entry := range entries {
		_, err = file.WriteString(entry + "\n")
		if err != nil {
			return fmt.Errorf("error writing timecode entry: %v", err)
		}
	}

	fmt.Printf("Timecodes written to %s\n", timecodesPath)
	return nil
}
