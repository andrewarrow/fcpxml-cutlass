package wikipedia

import (
	"cutlass/browser"
	"cutlass/build2/api"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

type WikiFile struct {
	Name     string
	ModTime  time.Time
	BaseName string
}

func getExistingWikiFiles() ([]WikiFile, error) {
	var files []WikiFile
	
	// Get all wiki_*.wav files
	wavFiles, err := filepath.Glob("data/wiki_*.wav")
	if err != nil {
		return nil, err
	}
	
	for _, file := range wavFiles {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		
		// Extract base name (remove wiki_ prefix and .wav suffix)
		baseName := filepath.Base(file)
		baseName = strings.TrimPrefix(baseName, "wiki_")
		baseName = strings.TrimSuffix(baseName, ".wav")
		
		files = append(files, WikiFile{
			Name:     file,
			ModTime:  info.ModTime(),
			BaseName: baseName,
		})
	}
	
	// Sort by modification time
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.Before(files[j].ModTime)
	})
	
	return files, nil
}

func HandleWikipediaRandomCommand(args []string, max int) {
	fmt.Printf("Fetching random Wikipedia articles (max: %d)...\n", max)

	// Create data directory if it doesn't exist
	if err := browser.EnsureDataDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating data directory: %v\n", err)
		return
	}
	
	// Count existing files
	existingFiles, err := getExistingWikiFiles()
	if err != nil {
		fmt.Printf("Warning: Could not count existing files: %v\n", err)
		existingFiles = []WikiFile{}
	}
	
	currentCount := len(existingFiles)
	fmt.Printf("Found %d existing wiki files\n", currentCount)
	
	if currentCount >= max {
		fmt.Printf("Already have %d files (max: %d). Nothing to do.\n", currentCount, max)
		return
	}

	// Load existing timecodes and cumulative time
	existingEntries, cumulativeSeconds, err := loadExistingTimecodes()
	if err != nil {
		fmt.Printf("Warning: Could not load existing timecodes: %v\n", err)
		existingEntries = []string{}
		cumulativeSeconds = 0
	}
	
	fmt.Printf("Starting at cumulative time: %s\n", formatTimecode(cumulativeSeconds))

	// Create browser session
	session, err := browser.NewBrowserSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating browser session: %v\n", err)
		return
	}
	defer session.Close()
	
	// Process articles until we reach max
	for currentCount < max {
		fmt.Printf("\n=== Processing article %d of %d ===\n", currentCount+1, max)
		
		if err := processOneArticle(session, &existingEntries, &cumulativeSeconds); err != nil {
			fmt.Printf("Error processing article: %v\n", err)
			continue
		}
		
		currentCount++
	}
	
	fmt.Printf("\nCompleted processing %d articles!\n", max)
}

func processOneArticle(session *browser.BrowserSession, existingEntries *[]string, cumulativeSeconds *float64) error {

	// Navigate to Wikipedia random page
	fmt.Println("Loading random Wikipedia page...")
	if err := session.NavigateAndWait("https://en.wikipedia.org/wiki/Special:Random"); err != nil {
		return fmt.Errorf("error loading Wikipedia: %v", err)
	}

	// Extract title from the page
	titleElement, err := session.Page.Element("h1.firstHeading")
	if err != nil {
		return fmt.Errorf("error finding title element: %v", err)
	}

	title, err := titleElement.Text()
	if err != nil {
		return fmt.Errorf("error extracting title: %v", err)
	}

	fmt.Printf("Found article: %s\n", title)

	// Get current page URL
	pageInfo, err := session.Page.Info()
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
	firstParagraph, err := extractFirstParagraph(session.Page)
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

	if err := session.NavigateAndWait(searchQuery); err != nil {
		return fmt.Errorf("error navigating to Google Videos: %v", err)
	}

	// Find and click the first video link
	fmt.Println("Looking for first video link...")

	// Debug: Print page title to confirm we're on the right page
	pageTitle, _ := session.Page.Eval("document.title")
	fmt.Printf("Debug: Current page title: %v\n", pageTitle)

	// Get the video URL using improved link selection
	videoURL, err := getFirstVideoLinkWikipedia(session)
	if err != nil {
		return fmt.Errorf("error finding video link: %v", err)
	}

	if videoURL != "" {
		fmt.Printf("Selected video URL: %s\n", videoURL)

		// Append video URL to youtube.txt
		if err := appendToYouTubeList(videoURL); err != nil {
			fmt.Printf("Warning: Could not append video URL to youtube.txt: %v\n", err)
		} else {
			fmt.Printf("Video URL appended to data/youtube.txt\n")
		}

		// Don't close the session here - it will be reused for the next article

		// Use yt-dlp to download thumbnail
		fmt.Println("Using yt-dlp to download video thumbnail...")

		// Create final filename
		finalFilename := filepath.Join("data", fmt.Sprintf("wiki_%s.png", filenameTitle))

		// Run yt-dlp command
		cmd := exec.Command("yt-dlp", "--write-thumbnail", "--skip-download", "-o", filepath.Join("data", "temp_thumbnail.%(ext)s"), videoURL)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error running yt-dlp: %v, output: %s", err, string(output))
		}

		fmt.Printf("yt-dlp output: %s\n", string(output))

		// Find the downloaded thumbnail file and rename it
		thumbnailFiles, err := filepath.Glob(filepath.Join("data", "temp_thumbnail.*"))
		if err != nil || len(thumbnailFiles) == 0 {
			return fmt.Errorf("could not find downloaded thumbnail file")
		}

		// Rename the first thumbnail file to our desired name
		err = os.Rename(thumbnailFiles[0], finalFilename)
		if err != nil {
			return fmt.Errorf("error renaming thumbnail file: %v", err)
		}

		fmt.Printf("Thumbnail saved: %s\n", finalFilename)

		// Record timecode BEFORE processing (start time of this clip)
		currentTimecode := formatTimecode(*cumulativeSeconds)
		fmt.Printf("Current article will start at: %s\n", currentTimecode)

		// Generate speech from first paragraph using chatterbox
		if firstParagraph != "" {
			fmt.Println("Generating speech from first paragraph...")
			audioFilename := filepath.Join("data", fmt.Sprintf("wiki_%s.wav", filenameTitle))

			// Call chatterbox CLI to generate speech
			chatterboxCmd := exec.Command("/opt/miniconda3/envs/chatterbox/bin/python3",
				"/Users/aa/os/chatterbox/chatterbox/main.py",
				firstParagraph,
				audioFilename)

			chatterboxOutput, err := chatterboxCmd.CombinedOutput()
			if err != nil {
				fmt.Printf("Warning: Could not generate speech: %v\n", err)
				fmt.Printf("Chatterbox output: %s\n", string(chatterboxOutput))
			} else {
				fmt.Printf("Speech generated: %s\n", audioFilename)

				// Get audio duration for timecode tracking
				duration, err := getAudioDurationSeconds(audioFilename)
				if err != nil {
					fmt.Printf("Warning: Could not get audio duration: %v\n", err)
					duration = 10.0 // Default duration if cannot get actual duration
				}
				fmt.Printf("Audio duration: %.2f seconds\n", duration)

				// Generate FCPXML using build2 system
				wikiProjectFile := "data/wiki.fcpxml"
				if err := generateWithBuild2(finalFilename, audioFilename, title, wikiProjectFile); err != nil {
					fmt.Printf("Warning: Could not generate FCPXML: %v\n", err)
				} else {
					fmt.Printf("FCPXML updated using build2 system: %s\n", wikiProjectFile)

					// Add new timecode entry
					var newEntry string
					if session.Page != nil {
						pageInfo, _ := session.Page.Info()
						if pageInfo != nil {
							newEntry = fmt.Sprintf("%s (%s)[%s]", currentTimecode, title, pageInfo.URL)
						} else {
							newEntry = fmt.Sprintf("%s (%s)[%s]", currentTimecode, title, "https://en.wikipedia.org/wiki/Special:Random")
						}
					} else {
						newEntry = fmt.Sprintf("%s (%s)[%s]", currentTimecode, title, "https://en.wikipedia.org/wiki/Special:Random")
					}
					
					// Extract video ID and create YouTube URL
					videoID := extractVideoID(videoURL)
					youtubeEntry := fmt.Sprintf("%s (%s)[%s]", currentTimecode, title, fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID))
					
					*existingEntries = append(*existingEntries, newEntry)
					*existingEntries = append(*existingEntries, youtubeEntry)

					// Update cumulative time for next clip
					*cumulativeSeconds += duration

					// Write updated timecodes to file
					if err := writeWikiTimecodesToFile(*existingEntries); err != nil {
						fmt.Printf("Warning: Could not write timecodes: %v\n", err)
					}

					fmt.Printf("Next article will start at: %s\n", formatTimecode(*cumulativeSeconds))
				}
			}
		}

	} else {
		return fmt.Errorf("link href is empty")
	}
	
	return nil
}

// getFirstVideoLinkWikipedia finds the first video link from Google search results, preferring watch URLs over channel URLs
func getFirstVideoLinkWikipedia(session *browser.BrowserSession) (string, error) {
	selectors := []string{
		"div.g h3 a",
		"div[data-ved] h3 a",
		"h3.LC20lb a",
		"a[href*='youtube.com']",
		"a[href*='watch']",
		"div.g a",
	}

	var allLinks []string
	
	// Collect all potential links first
	for _, selector := range selectors {
		fmt.Printf("Debug: Trying selector: %s\n", selector)
		elements, err := session.Page.Elements(selector)
		if err != nil {
			fmt.Printf("Debug: Error with selector %s: %v\n", selector, err)
			continue
		}
		fmt.Printf("Debug: Found %d elements with selector %s\n", len(elements), selector)

		for _, element := range elements {
			href, err := element.Attribute("href")
			if err != nil || href == nil || *href == "" {
				continue
			}
			allLinks = append(allLinks, *href)
		}
	}

	if len(allLinks) == 0 {
		// Debug: Print page HTML snippet to see structure
		bodyHTML, _ := session.Page.Eval("document.body.innerHTML.substring(0, 1000)")
		fmt.Printf("Debug: Page HTML snippet: %v\n", bodyHTML)
		return "", fmt.Errorf("could not find any video links with any selector")
	}

	// First pass: look for YouTube watch URLs (actual videos)
	for _, link := range allLinks {
		if strings.Contains(link, "youtube.com") && strings.Contains(link, "/watch?v=") {
			fmt.Printf("Debug: Found preferred watch URL: %s\n", link)
			return link, nil
		}
	}

	// Second pass: look for other YouTube watch URLs
	for _, link := range allLinks {
		if strings.Contains(link, "youtube.com") && strings.Contains(link, "watch") {
			fmt.Printf("Debug: Found watch URL: %s\n", link)
			return link, nil
		}
	}

	// Third pass: any YouTube URL except channels
	for _, link := range allLinks {
		if strings.Contains(link, "youtube.com") && !strings.Contains(link, "/channel/") && !strings.Contains(link, "/c/") && !strings.Contains(link, "/@") {
			fmt.Printf("Debug: Found non-channel YouTube URL: %s\n", link)
			return link, nil
		}
	}

	// Fourth pass: any link (fallback, including channels if that's all we have)
	fmt.Printf("Debug: Using fallback URL: %s\n", allLinks[0])
	return allLinks[0], nil
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

// extractSentences splits text into sentences using common sentence endings
func extractSentences(text string) []string {
	// Simple sentence detection using periods, exclamation marks, and question marks
	// followed by space and capital letter or end of string
	sentences := []string{}
	
	// Split on sentence boundaries but keep the punctuation
	words := strings.Fields(text)
	if len(words) == 0 {
		return sentences
	}

	var currentSentence strings.Builder
	
	for i, word := range words {
		currentSentence.WriteString(word)
		
		// Check if word ends with sentence-ending punctuation
		if strings.HasSuffix(word, ".") || strings.HasSuffix(word, "!") || strings.HasSuffix(word, "?") {
			// Look ahead to see if next word starts with capital (indicates new sentence)
			// or if this is the last word
			if i == len(words)-1 || (i < len(words)-1 && isCapitalized(words[i+1])) {
				sentences = append(sentences, strings.TrimSpace(currentSentence.String()))
				currentSentence.Reset()
				continue
			}
		}
		
		// Add space if not the last word
		if i < len(words)-1 {
			currentSentence.WriteString(" ")
		}
	}
	
	// Add any remaining text as the last sentence
	if currentSentence.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(currentSentence.String()))
	}
	
	return sentences
}

// isCapitalized checks if a word starts with a capital letter
func isCapitalized(word string) bool {
	if len(word) == 0 {
		return false
	}
	first := rune(word[0])
	return first >= 'A' && first <= 'Z'
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

	// Collect text from paragraphs and limit to 1-2 sentences for 3-9 seconds of audio
	var combinedText strings.Builder

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

		// Extract 1-2 sentences from the combined text
		sentences := extractSentences(combinedText.String())
		if len(sentences) >= 1 {
			// Return first sentence if it's reasonable length, otherwise first 2 sentences
			firstSentence := sentences[0]
			if len(sentences) == 1 || len(firstSentence) >= 50 {
				return firstSentence, nil
			} else if len(sentences) >= 2 {
				return firstSentence + " " + sentences[1], nil
			}
			return firstSentence, nil
		}
	}

	if combinedText.Len() == 0 {
		return "", fmt.Errorf("no non-empty paragraphs found")
	}

	// Fallback: if no proper sentences found, return first 100 characters
	text := combinedText.String()
	if len(text) > 100 {
		// Find last space within 100 chars to avoid breaking mid-word
		lastSpace := strings.LastIndex(text[:100], " ")
		if lastSpace > 0 {
			return text[:lastSpace], nil
		}
		return text[:100], nil
	}

	return text, nil
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



func generateWithBuild2(imagePath, audioPath, title, projectFile string) error {
	// Create or load project using build2 API
	pb, err := api.NewProjectBuilder(projectFile)
	if err != nil {
		return fmt.Errorf("failed to create project builder: %v", err)
	}
	
	// Add the clip with image, audio, and text overlay
	err = pb.AddClipSafe(api.ClipConfig{
		VideoFile: imagePath,
		AudioFile: audioPath,
		Text:      title,
	})
	if err != nil {
		return fmt.Errorf("failed to add clip to project: %v", err)
	}
	
	// Save the project
	err = pb.Save()
	if err != nil {
		return fmt.Errorf("failed to save project: %v", err)
	}
	
	return nil
}

// getAudioDurationSeconds gets the duration of an audio file using ffprobe
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

// writeWikiTimecodesToFile writes timecode entries to wiki_timecodes.txt
func writeWikiTimecodesToFile(entries []string) error {
	timecodesPath := filepath.Join("data", "wiki_timecodes.txt")
	file, err := os.Create(timecodesPath)
	if err != nil {
		return fmt.Errorf("error creating wiki timecodes file: %v", err)
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

// loadExistingTimecodes loads existing timecode entries and calculates cumulative duration
func loadExistingTimecodes() ([]string, float64, error) {
	timecodesPath := filepath.Join("data", "wiki_timecodes.txt")
	
	// Check if file exists
	if _, err := os.Stat(timecodesPath); os.IsNotExist(err) {
		return []string{}, 0, nil
	}

	file, err := os.Open(timecodesPath)
	if err != nil {
		return nil, 0, fmt.Errorf("error opening existing timecodes file: %v", err)
	}
	defer file.Close()

	var entries []string
	var cumulativeDuration float64

	// Read existing entries
	buf := make([]byte, 1024*1024) // 1MB buffer
	n, _ := file.Read(buf)
	content := string(buf[:n])
	
	lines := strings.Split(strings.TrimSpace(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			entries = append(entries, line)
		}
	}

	// Calculate cumulative duration by examining all existing wiki audio files
	audioFiles, err := filepath.Glob(filepath.Join("data", "wiki_*.wav"))
	if err == nil {
		for _, audioFile := range audioFiles {
			duration, err := getAudioDurationSeconds(audioFile)
			if err == nil {
				cumulativeDuration += duration
			}
		}
	}

	return entries, cumulativeDuration, nil
}

// extractVideoID extracts the video ID from a YouTube URL
func extractVideoID(url string) string {
	// Handle standard YouTube URLs like https://www.youtube.com/watch?v=VIDEO_ID
	if strings.Contains(url, "youtube.com/watch?v=") {
		parts := strings.Split(url, "v=")
		if len(parts) > 1 {
			// Get everything after v= and before any additional parameters
			videoID := strings.Split(parts[1], "&")[0]
			return videoID
		}
	}
	
	// Handle shortened YouTube URLs like https://youtu.be/VIDEO_ID
	if strings.Contains(url, "youtu.be/") {
		parts := strings.Split(url, "youtu.be/")
		if len(parts) > 1 {
			// Get everything after youtu.be/ and before any additional parameters
			videoID := strings.Split(parts[1], "?")[0]
			return videoID
		}
	}
	
	// If we can't extract a video ID, return a default
	return "nGsnoAiVWvc"
}

