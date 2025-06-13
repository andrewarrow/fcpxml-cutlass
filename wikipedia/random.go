package wikipedia

import (
	"cutlass/browser"
	"cutlass/build2/api"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-rod/rod"
)

func HandleWikipediaRandomCommand(args []string) {
	fmt.Println("Fetching random Wikipedia article...")

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

	// Navigate to Wikipedia random page
	fmt.Println("Loading random Wikipedia page...")
	if err := session.NavigateAndWait("https://en.wikipedia.org/wiki/Special:Random"); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading Wikipedia: %v\n", err)
		return
	}

	// Extract title from the page
	titleElement, err := session.Page.Element("h1.firstHeading")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding title element: %v\n", err)
		return
	}

	title, err := titleElement.Text()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting title: %v\n", err)
		return
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
		fmt.Fprintf(os.Stderr, "Error navigating to Google Videos: %v\n", err)
		return
	}

	// Find and click the first video link
	fmt.Println("Looking for first video link...")

	// Debug: Print page title to confirm we're on the right page
	pageTitle, _ := session.Page.Eval("document.title")
	fmt.Printf("Debug: Current page title: %v\n", pageTitle)

	// Get the video URL using improved link selection
	videoURL, err := getFirstVideoLinkWikipedia(session)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding video link: %v\n", err)
		return
	}

	if videoURL != "" {
		fmt.Printf("Selected video URL: %s\n", videoURL)

		// Append video URL to youtube.txt
		if err := appendToYouTubeList(videoURL); err != nil {
			fmt.Printf("Warning: Could not append video URL to youtube.txt: %v\n", err)
		} else {
			fmt.Printf("Video URL appended to data/youtube.txt\n")
		}

		// Close the browser since we no longer need it
		session.Close()

		// Use yt-dlp to download thumbnail
		fmt.Println("Using yt-dlp to download video thumbnail...")

		// Create final filename
		finalFilename := filepath.Join("data", fmt.Sprintf("wiki_%s.png", filenameTitle))

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

				// Generate FCPXML using build2 system
				wikiProjectFile := "data/wiki.fcpxml"
				if err := generateWithBuild2(finalFilename, audioFilename, title, wikiProjectFile); err != nil {
					fmt.Printf("Warning: Could not generate FCPXML: %v\n", err)
				} else {
					fmt.Printf("FCPXML updated using build2 system: %s\n", wikiProjectFile)
				}
			}
		}

	} else {
		fmt.Fprintf(os.Stderr, "Error: Link href is empty\n")
		session.Close()
		return
	}
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

