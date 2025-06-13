package wikipedia

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
		elements, err := session.Page.Elements(selector)
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
		bodyHTML, _ := session.Page.Eval("document.body.innerHTML.substring(0, 1000)")
		fmt.Printf("Debug: Page HTML snippet: %v\n", bodyHTML)
		fmt.Fprintf(os.Stderr, "Error: Could not find any video links with any selector\n")
		return
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

				// Generate FCPXML and append to wiki.fcpxml
				if err := generateAndAppendFCPXML(finalFilename, audioFilename, filenameTitle, title); err != nil {
					fmt.Printf("Warning: Could not generate FCPXML: %v\n", err)
				} else {
					fmt.Printf("FCPXML appended to data/wiki.fcpxml\n")
				}
			}
		}

	} else {
		fmt.Fprintf(os.Stderr, "Error: Link href is empty\n")
		session.Close()
		return
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

type WikiTemplateData struct {
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
	VideoDuration string
	IngestDate    string
	TitleEffectID string
	Title         string
}

func generateUID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return strings.ToUpper(hex.EncodeToString(bytes))
}

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

	// Convert to FCPXML format (samples/rate)
	samples := int64(duration * 44100)
	return fmt.Sprintf("%d/44100s", samples), nil
}

func generateAndAppendFCPXML(imagePath, audioPath, name, title string) error {
	// Get audio duration using ffprobe
	audioDuration, err := getAudioDuration(audioPath)
	if err != nil {
		return fmt.Errorf("failed to get audio duration: %v", err)
	}

	// Convert audio duration to 24000s format for video duration
	videoDuration, err := convertAudioDurationToVideo(audioDuration)
	if err != nil {
		return fmt.Errorf("failed to convert audio duration: %v", err)
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
	data := WikiTemplateData{
		ImageAssetID:  "r" + strconv.Itoa(timestamp),
		ImageName:     name,
		ImageUID:      imageUID,
		ImagePath:     absImagePath,
		ImageBookmark: "placeholder_bookmark",
		ImageFormatID: "r" + strconv.Itoa(timestamp+1),
		ImageWidth:    "640",
		ImageHeight:   "480",
		AudioAssetID:  "r" + strconv.Itoa(timestamp+2),
		AudioName:     name,
		AudioUID:      audioUID,
		AudioPath:     absAudioPath,
		AudioBookmark: "placeholder_bookmark",
		AudioDuration: audioDuration,
		VideoDuration: videoDuration,
		IngestDate:    time.Now().Format("2006-01-02 15:04:05 -0700"),
		TitleEffectID: "r" + strconv.Itoa(timestamp+3),
		Title:         title,
	}

	// Read template
	templateContent, err := os.ReadFile("templates/one_wiki.fcpxml")
	if err != nil {
		return fmt.Errorf("failed to read template: %v", err)
	}

	// Execute template
	tmpl, err := template.New("wiki").Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	// Read or create wiki.fcpxml
	wikiPath := "data/wiki.fcpxml"
	var wikiContent []byte

	if _, err := os.Stat(wikiPath); os.IsNotExist(err) {
		// Create new wiki.fcpxml based on two_wiki.fcpxml template
		templateContent, err := os.ReadFile("templates/two_wiki.fcpxml")
		if err != nil {
			return fmt.Errorf("failed to read wiki.fcpxml template: %v", err)
		}
		wikiContent = templateContent
	} else {
		wikiContent, err = os.ReadFile(wikiPath)
		if err != nil {
			return fmt.Errorf("failed to read wiki.fcpxml: %v", err)
		}
	}

	// Find insertion points
	wikiStr := string(wikiContent)

	// Insert assets before </resources>
	resourcesEnd := strings.Index(wikiStr, "    </resources>")
	if resourcesEnd == -1 {
		return fmt.Errorf("could not find </resources> tag")
	}

	newWikiContent := wikiStr[:resourcesEnd] + result.String() + "\n" + wikiStr[resourcesEnd:]

	// Now add the video clip to the timeline
	// Find the last video element in the spine to get the end offset
	lastVideoEnd := findLastVideoOffset(newWikiContent)

	// Calculate title duration (video duration minus title offset)
	titleDuration, err := subtractDuration(videoDuration, "86399313/24000s")
	if err != nil {
		return fmt.Errorf("failed to calculate title duration: %v", err)
	}

	// Create video element for timeline
	videoElement := fmt.Sprintf(`                        <video ref="%s" offset="%s" start="86399313/24000s" duration="%s">
                            <asset-clip ref="%s" lane="-1" offset="28799771/8000s" name="%s" duration="%s" format="r1" audioRole="dialogue"/>
                            <title ref="%s" lane="1" offset="86399313/24000s" name="%s - Lower Third Text &amp; Subhead" start="86486400/24000s" duration="%s">
                                <param name="Position" key="9999/10003/13260/11488/1/100/101" value="-55.875 1522.87"/>
                                <param name="Layout Method" key="9999/10003/13260/11488/2/314" value="1 (Paragraph)"/>
                                <param name="Left Margin" key="9999/10003/13260/11488/2/323" value="-1728"/>
                                <param name="Right Margin" key="9999/10003/13260/11488/2/324" value="1728"/>
                                <param name="Top Margin" key="9999/10003/13260/11488/2/325" value="-794"/>
                                <param name="Bottom Margin" key="9999/10003/13260/11488/2/326" value="-966.1"/>
                                <param name="Auto-Shrink" key="9999/10003/13260/11488/2/370" value="3 (To All Margins)"/>
                                <param name="Auto-Shrink Scale" key="9999/10003/13260/11488/2/376" value="0.74"/>
                                <param name="Opacity" key="9999/10003/13260/11488/4/13051/1000/1044" value="0"/>
                                <param name="Animate" key="9999/10003/13260/11488/4/13051/201/203" value="3 (Line)"/>
                                <param name="Spread" key="9999/10003/13260/11488/4/13051/201/204" value="5"/>
                                <param name="Speed" key="9999/10003/13260/11488/4/13051/201/208" value="6 (Custom)"/>
                                <param name="Custom Speed" key="9999/10003/13260/11488/4/13051/201/209">
                                    <keyframeAnimation>
                                        <keyframe time="0s" value="0"/>
                                        <keyframe time="10s" value="1"/>
                                    </keyframeAnimation>
                                </param>
                                <param name="Apply Speed" key="9999/10003/13260/11488/4/13051/201/211" value="2 (Per Object)"/>
                                <param name="Start Offset" key="9999/10003/13260/11488/4/13051/201/235" value="34"/>
                                <param name="Position" key="9999/10003/13260/3296674397/1/100/101" value="-61.6875 1516.64"/>
                                <param name="Layout Method" key="9999/10003/13260/3296674397/2/314" value="1 (Paragraph)"/>
                                <param name="Left Margin" key="9999/10003/13260/3296674397/2/323" value="-1728"/>
                                <param name="Right Margin" key="9999/10003/13260/3296674397/2/324" value="1728"/>
                                <param name="Top Margin" key="9999/10003/13260/3296674397/2/325" value="972"/>
                                <param name="Bottom Margin" key="9999/10003/13260/3296674397/2/326" value="-776.6"/>
                                <param name="Line Spacing" key="9999/10003/13260/3296674397/2/354/3296667315/404" value="-19"/>
                                <param name="Auto-Shrink" key="9999/10003/13260/3296674397/2/370" value="3 (To All Margins)"/>
                                <param name="Alignment" key="9999/10003/13260/3296674397/2/373" value="0 (Left) 0 (Top)"/>
                                <param name="Opacity" key="9999/10003/13260/3296674397/4/3296674797/1000/1044" value="0"/>
                                <param name="Animate" key="9999/10003/13260/3296674397/4/3296674797/201/203" value="3 (Line)"/>
                                <param name="Spread" key="9999/10003/13260/3296674397/4/3296674797/201/204" value="5"/>
                                <param name="Speed" key="9999/10003/13260/3296674397/4/3296674797/201/208" value="6 (Custom)"/>
                                <param name="Custom Speed" key="9999/10003/13260/3296674397/4/3296674797/201/209">
                                    <keyframeAnimation>
                                        <keyframe time="-71680/153600s" value="0"/>
                                        <keyframe time="1896960/153600s" value="1"/>
                                    </keyframeAnimation>
                                </param>
                                <param name="Apply Speed" key="9999/10003/13260/3296674397/4/3296674797/201/211" value="2 (Per Object)"/>
                                <text>
                                    <text-style ref="ts1_%d">%s</text-style>
                                </text>
                                <text>
                                    <text-style ref="ts2_%d">This content is adapted from Wikipedia article above, used under the Creative Commons Attribution-ShareAlike 4.0 International License.</text-style>
                                </text>
                                <text-style-def id="ts1_%d">
                                    <text-style font="Helvetica Neue" fontSize="170" fontColor="1 1 1 1" bold="1" shadowColor="0 0 0 0.75" shadowOffset="5 315" lineSpacing="-19"/>
                                </text-style-def>
                                <text-style-def id="ts2_%d">
                                    <text-style font="Helvetica Neue" fontSize="69.56" fontFace="Medium" fontColor="1 1 1 1" shadowColor="0 0 0 0.75" shadowOffset="5 315"/>
                                </text-style-def>
                            </title>
                        </video>`,
		data.ImageAssetID, lastVideoEnd, videoDuration, data.AudioAssetID+"_audio", name, videoDuration, data.TitleEffectID, title, titleDuration, timestamp, title, timestamp, timestamp, timestamp)

	// Insert video element before </spine>
	spineEnd := strings.Index(newWikiContent, "                    </spine>")
	if spineEnd == -1 {
		return fmt.Errorf("could not find </spine> tag")
	}

	finalContent := newWikiContent[:spineEnd] + videoElement + "\n" + newWikiContent[spineEnd:]

	// Update sequence duration to include new clip
	finalContent = updateSequenceDuration(finalContent, lastVideoEnd, videoDuration)

	// Write back to file
	if err := os.WriteFile(wikiPath, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write wiki.fcpxml: %v", err)
	}

	return nil
}

func findLastVideoOffset(xmlContent string) string {
	// Find the last video offset in the timeline
	// Look for pattern: offset="XXXX/24000s"
	re := regexp.MustCompile(`offset="(\d+/24000s)"`)
	matches := re.FindAllStringSubmatch(xmlContent, -1)

	if len(matches) == 0 {
		return "0s"
	}

	// Get the last match
	lastOffset := matches[len(matches)-1][1]

	// Extract numerator and add the duration to get new offset
	parts := strings.Split(lastOffset, "/")
	if len(parts) != 2 {
		return "0s"
	}

	offsetNum, _ := strconv.Atoi(parts[0])

	// Find the duration of that last video
	re2 := regexp.MustCompile(`duration="(\d+/24000s)"`)
	durMatches := re2.FindAllStringSubmatch(xmlContent, -1)

	if len(durMatches) > 0 {
		lastDur := durMatches[len(durMatches)-1][1]
		durParts := strings.Split(lastDur, "/")
		if len(durParts) == 2 {
			durNum, _ := strconv.Atoi(durParts[0])
			newOffset := offsetNum + durNum
			return fmt.Sprintf("%d/24000s", newOffset)
		}
	}

	return fmt.Sprintf("%d/24000s", offsetNum+100000) // fallback
}

func convertAudioDurationToVideo(audioDuration string) (string, error) {
	// Convert from samples/44100s to frames/24000s
	// audioDuration format: "XXXXX/44100s"
	parts := strings.Split(strings.TrimSuffix(audioDuration, "s"), "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid audio duration format: %s", audioDuration)
	}

	samples, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid samples: %v", err)
	}

	// Convert samples at 44100Hz to frames at 24000Hz (24fps)
	// duration_seconds = samples / 44100
	// frames = duration_seconds * 24000
	frames := (samples * 24000) / 44100

	// FCPXML requires frame durations to be on edit boundaries
	// Round to nearest multiple of 1001 (for 23.976fps compatibility)
	frames = ((frames + 500) / 1001) * 1001

	return fmt.Sprintf("%d/24000s", frames), nil
}

func subtractDuration(duration1, duration2 string) (string, error) {
	// Both durations should be in format "XXXXX/24000s"
	parts1 := strings.Split(strings.TrimSuffix(duration1, "s"), "/")
	parts2 := strings.Split(strings.TrimSuffix(duration2, "s"), "/")
	
	if len(parts1) != 2 || len(parts2) != 2 {
		return "", fmt.Errorf("invalid duration format: %s or %s", duration1, duration2)
	}
	
	frames1, err := strconv.ParseInt(parts1[0], 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid frames in duration1: %v", err)
	}
	
	frames2, err := strconv.ParseInt(parts2[0], 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid frames in duration2: %v", err)
	}
	
	// Subtract frames2 from frames1
	result := frames1 - frames2
	if result < 0 {
		result = 0 // Don't allow negative durations
	}
	
	return fmt.Sprintf("%d/24000s", result), nil
}

func updateSequenceDuration(xmlContent, lastOffset, videoDuration string) string {
	// Extract numbers from lastOffset and videoDuration
	offsetParts := strings.Split(strings.TrimSuffix(lastOffset, "s"), "/")
	durationParts := strings.Split(strings.TrimSuffix(videoDuration, "s"), "/")

	if len(offsetParts) == 2 && len(durationParts) == 2 {
		offsetNum, _ := strconv.Atoi(offsetParts[0])
		durNum, _ := strconv.Atoi(durationParts[0])
		newTotal := offsetNum + durNum

		// Update sequence duration
		re := regexp.MustCompile(`<sequence format="r1" duration="(\d+/24000s)"`)
		replacement := fmt.Sprintf(`<sequence format="r1" duration="%d/24000s"`, newTotal)
		return re.ReplaceAllString(xmlContent, replacement)
	}

	return xmlContent
}
