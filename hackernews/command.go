package hackernews

import (
	"crypto/rand"
	"cutlass/browser"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// HandleHackerNewsCommand processes Hacker News articles like Wikipedia random
func HandleHackerNewsCommand(args []string) {
	fmt.Println("Processing Hacker News articles...")

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

	// Navigate to Hacker News
	fmt.Println("Loading Hacker News homepage...")
	if err := session.NavigateAndWait("https://news.ycombinator.com/"); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading Hacker News: %v\n", err)
		return
	}

	// Get first article
	article, err := getFirstHNArticle(session)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting first article: %v\n", err)
		return
	}

	fmt.Printf("Found article: %s\n", article.Title)
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

	// Create filename-safe version of title
	filenameTitle := sanitizeFilename(article.Title)

	// Navigate to Google Videos search
	searchQuery := fmt.Sprintf("https://www.google.com/search?tbm=vid&q=%s", strings.ReplaceAll(article.Title, " ", "+"))
	fmt.Printf("Searching Google Videos for: %s\n", article.Title)

	if err := session.NavigateAndWait(searchQuery); err != nil {
		fmt.Fprintf(os.Stderr, "Error navigating to Google Videos: %v\n", err)
		return
	}

	// Find and get the first video link
	videoURL, err := getFirstVideoLink(session)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding video link: %v\n", err)
		return
	}

	fmt.Printf("Found video URL: %s\n", videoURL)

	// Append video URL to youtube.txt
	if err := appendToYouTubeList(videoURL); err != nil {
		fmt.Printf("Warning: Could not append video URL to youtube.txt: %v\n", err)
	} else {
		fmt.Printf("Video URL appended to data/youtube.txt\n")
	}

	// Close browser before running external commands
	session.Close()

	// Use yt-dlp to download thumbnail
	fmt.Println("Using yt-dlp to download video thumbnail...")

	// Create final filename
	finalFilename := filepath.Join("data", fmt.Sprintf("hn_%s.png", filenameTitle))

	// Run yt-dlp command
	cmd := exec.Command("yt-dlp", "--write-thumbnail", "--skip-download", "-o", filepath.Join("data", "temp_thumbnail.%(ext)s"), videoURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running yt-dlp: %v\n", err)
		fmt.Fprintf(os.Stderr, "yt-dlp output: %s\n", string(output))
		return
	}

	fmt.Printf("yt-dlp output: %s\n", string(output))

	// Find the downloaded thumbnail file and rename it
	thumbnailFiles, err := filepath.Glob(filepath.Join("data", "temp_thumbnail.*"))
	if err != nil || len(thumbnailFiles) == 0 {
		fmt.Fprintf(os.Stderr, "Error: Could not find downloaded thumbnail file\n")
		return
	}

	// Rename the first thumbnail file to our desired name
	err = os.Rename(thumbnailFiles[0], finalFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error renaming thumbnail file: %v\n", err)
		return
	}

	fmt.Printf("Thumbnail saved: %s\n", finalFilename)

	// Generate speech from article title/summary using chatterbox
	speechText := article.Title
	if article.Summary != "" {
		speechText = article.Summary
	}

	fmt.Println("Generating speech from article content...")
	audioFilename := filepath.Join("data", fmt.Sprintf("hn_%s.wav", filenameTitle))

	// Call chatterbox CLI to generate speech
	chatterboxCmd := exec.Command("/opt/miniconda3/envs/chatterbox/bin/python3",
		"/Users/aa/os/chatterbox/dia/cli.py",
		speechText,
		"--output="+audioFilename)

	chatterboxOutput, err := chatterboxCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Warning: Could not generate speech: %v\n", err)
		fmt.Printf("Chatterbox output: %s\n", string(chatterboxOutput))
	} else {
		fmt.Printf("Speech generated: %s\n", audioFilename)

		// Generate FCPXML and append to hn.fcpxml
		if err := generateAndAppendHNFCPXML(finalFilename, audioFilename, filenameTitle, article.Title); err != nil {
			fmt.Printf("Warning: Could not generate FCPXML: %v\n", err)
		} else {
			fmt.Printf("FCPXML appended to data/hn.fcpxml\n")
		}
	}
}

// HNArticle represents a Hacker News article
type HNArticle struct {
	Title   string
	URL     string
	Summary string
}

// getFirstHNArticle gets the first article from Hacker News
func getFirstHNArticle(session *browser.BrowserSession) (*HNArticle, error) {
	// Get first title
	titleElements, err := session.Page.Elements("span.titleline a")
	if err != nil || len(titleElements) == 0 {
		return nil, fmt.Errorf("could not find title elements")
	}

	title, err := titleElements[0].Text()
	if err != nil {
		return nil, fmt.Errorf("could not get title text: %v", err)
	}

	articleURL, err := titleElements[0].Attribute("href")
	if err != nil || articleURL == nil {
		return nil, fmt.Errorf("could not get article URL")
	}

	// Handle relative URLs
	url := *articleURL
	if strings.HasPrefix(url, "item?id=") {
		url = "https://news.ycombinator.com/" + url
	}

	// Try to get article summary by looking for comment count or points
	summary := ""
	
	return &HNArticle{
		Title:   title,
		URL:     url,
		Summary: summary,
	}, nil
}

// getFirstVideoLink finds the first video link from Google search results
func getFirstVideoLink(session *browser.BrowserSession) (string, error) {
	selectors := []string{
		"div.g h3 a",
		"div[data-ved] h3 a", 
		"h3.LC20lb a",
		"a[href*='youtube.com']",
		"a[href*='watch']",
		"div.g a",
	}

	for _, selector := range selectors {
		elements, err := session.Page.Elements(selector)
		if err != nil {
			continue
		}

		if len(elements) > 0 {
			href, err := elements[0].Attribute("href")
			if err != nil || href == nil || *href == "" {
				continue
			}
			return *href, nil
		}
	}

	return "", fmt.Errorf("could not find any video links")
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

// appendToYouTubeList appends URL to youtube.txt
func appendToYouTubeList(url string) error {
	youtubeListPath := filepath.Join("data", "youtube.txt")
	file, err := os.OpenFile(youtubeListPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening youtube.txt: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(url + "\n")
	if err != nil {
		return fmt.Errorf("error writing to youtube.txt: %v", err)
	}

	return nil
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

// HNTemplateData represents data for HN FCPXML template
type HNTemplateData struct {
	ImageAssetID  string
	ImageName     string
	ImageUID      string
	ImagePath     string
	ImageBookmark string
	ImageFormatID string
	ImageWidth    string
	ImageHeight   string
	AudioAssetID  string
	AudioName     string
	AudioUID      string
	AudioPath     string
	AudioBookmark string
	AudioDuration string
	IngestDate    string
	TitleEffectID string
	Title         string
}

// generateUID creates a unique identifier
func generateUID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return strings.ToUpper(hex.EncodeToString(bytes))
}

// getAudioDuration gets audio duration using ffprobe
func getAudioDuration(audioPath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format=duration", "-of", "csv=p=0", audioPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return "", err
	}

	samples := int64(duration * 44100)
	return fmt.Sprintf("%d/44100s", samples), nil
}

// generateAndAppendHNFCPXML generates FCPXML for HN article
func generateAndAppendHNFCPXML(imagePath, audioPath, name, title string) error {
	// Get audio duration using ffprobe
	audioDuration, err := getAudioDuration(audioPath)
	if err != nil {
		return fmt.Errorf("failed to get audio duration: %v", err)
	}

	// Generate UIDs and asset IDs
	imageUID := generateUID()
	audioUID := generateUID()
	timestamp := int(time.Now().Unix())

	// Convert relative paths to absolute paths with file:// prefix
	absImagePath, err := filepath.Abs(imagePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for image: %v", err)
	}
	absAudioPath, err := filepath.Abs(audioPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for audio: %v", err)
	}

	// Create template data
	data := HNTemplateData{
		ImageAssetID:  "r" + strconv.Itoa(timestamp),
		ImageName:     name,
		ImageUID:      imageUID,
		ImagePath:     "file://" + absImagePath,
		ImageBookmark: "placeholder_bookmark",
		ImageFormatID: "r" + strconv.Itoa(timestamp+1),
		ImageWidth:    "640",
		ImageHeight:   "480",
		AudioAssetID:  "r" + strconv.Itoa(timestamp+2),
		AudioName:     name,
		AudioUID:      audioUID,
		AudioPath:     "file://" + absAudioPath,
		AudioBookmark: "placeholder_bookmark",
		AudioDuration: audioDuration,
		IngestDate:    time.Now().Format("2006-01-02 15:04:05 -0700"),
		TitleEffectID: "r" + strconv.Itoa(timestamp+3),
		Title:         title,
	}

	// Read template (reuse Wikipedia template for now)
	templateContent, err := os.ReadFile("templates/one_wiki.fcpxml")
	if err != nil {
		return fmt.Errorf("failed to read template: %v", err)
	}

	// Execute template
	tmpl, err := template.New("hn").Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	// Read or create hn.fcpxml
	hnPath := "data/hn.fcpxml"
	var hnContent []byte
	
	if _, err := os.Stat(hnPath); os.IsNotExist(err) {
		// Create new hn.fcpxml based on wiki.fcpxml
		wikiContent, err := os.ReadFile("data/wiki.fcpxml")
		if err != nil {
			return fmt.Errorf("failed to read wiki.fcpxml template: %v", err)
		}
		hnContent = wikiContent
	} else {
		hnContent, err = os.ReadFile(hnPath)
		if err != nil {
			return fmt.Errorf("failed to read hn.fcpxml: %v", err)
		}
	}

	// Follow same insertion logic as Wikipedia
	hnStr := string(hnContent)
	resourcesEnd := strings.Index(hnStr, "    </resources>")
	if resourcesEnd == -1 {
		return fmt.Errorf("could not find </resources> tag")
	}

	newHNContent := hnStr[:resourcesEnd] + result.String() + "\n" + hnStr[resourcesEnd:]

	// Write back to file
	if err := os.WriteFile(hnPath, []byte(newHNContent), 0644); err != nil {
		return fmt.Errorf("failed to write hn.fcpxml: %v", err)
	}

	return nil
}