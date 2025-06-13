package hackernews

import (
	"crypto/rand"
	"cutlass/browser"
	"encoding/hex"
	"fmt"
	"net/url"
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

	// Navigate to Hacker News and fetch all articles
	fmt.Println("Loading Hacker News homepage...")
	if err := session.NavigateAndWait("https://news.ycombinator.com/"); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading Hacker News: %v\n", err)
		return
	}

	// Get all articles from the homepage
	fmt.Println("Fetching all articles from Hacker News...")
	articles, err := getAllHNArticles(session)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting articles: %v\n", err)
		return
	}

	fmt.Printf("Found %d articles to process\n", len(articles))

	// Process each article in sequence
	for i, article := range articles {
		processHNArticle(session, article, i)
	}
}

// processHNArticle processes a single HN article through the full pipeline
func processHNArticle(session *browser.BrowserSession, article *HNArticle, index int) {
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

	videoURL := ""
	tokens := strings.Split(article.Title, " ")
	// Navigate to Google Videos search
	for {
		searchQuery := fmt.Sprintf("https://www.google.com/search?tbm=vid&q=%s",
			url.QueryEscape(strings.Join(tokens, " ")))
		fmt.Printf("Searching Google Videos for: %s\n", article.Title)

		if err := session.NavigateAndWait(searchQuery); err != nil {
			fmt.Fprintf(os.Stderr, "Error navigating to Google Videos: %v\n", err)
			return
		}

		// Find and get the first video link
		var err error
		videoURL, err = getFirstVideoLink(session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding video link: %v\n", err)
			if len(tokens) > 2 {
				tokens = tokens[0:2]
				continue
			} else {
				fmt.Printf("Skipping video download for this article\n")
				break
			}
		}
		break
	}

	if videoURL != "" {
		fmt.Printf("Found video URL: %s\n", videoURL)

		// Append video URL to youtube.txt
		if err := appendToYouTubeList(videoURL); err != nil {
			fmt.Printf("Warning: Could not append video URL to youtube.txt: %v\n", err)
		} else {
			fmt.Printf("Video URL appended to data/youtube.txt\n")
		}

		// Use yt-dlp to download thumbnail
		fmt.Println("Using yt-dlp to download video thumbnail...")

		// Create final filename
		finalFilename := filepath.Join("data", fmt.Sprintf("hn_%s.png", filenameTitle))

		// Run yt-dlp command
		cmd := exec.Command("yt-dlp", "--write-thumbnail", "--skip-download", "-o", filepath.Join("data", "temp_thumbnail.%(ext)s"), videoURL)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Warning: Could not download thumbnail: %v\n", err)
			fmt.Printf("yt-dlp output: %s\n", string(output))
		} else {
			fmt.Printf("yt-dlp output: %s\n", string(output))

			// Find the downloaded thumbnail file and rename it
			thumbnailFiles, err := filepath.Glob(filepath.Join("data", "temp_thumbnail.*"))
			if err != nil || len(thumbnailFiles) == 0 {
				fmt.Printf("Warning: Could not find downloaded thumbnail file\n")
			} else {
				// Rename the first thumbnail file to our desired name
				err = os.Rename(thumbnailFiles[0], finalFilename)
				if err != nil {
					fmt.Printf("Warning: Could not rename thumbnail file: %v\n", err)
				} else {
					fmt.Printf("Thumbnail saved: %s\n", finalFilename)

					// Generate speech from article title using chatterbox
					speechText := article.Title

					fmt.Println("Generating speech from article title...")
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
			}
		}
	}

	fmt.Printf("Completed processing article %d\n\n", index+1)
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

		// Handle relative URLs
		url := *articleURL
		if strings.HasPrefix(url, "item?id=") {
			url = "https://news.ycombinator.com/" + url
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

	// Now add the video clip to the timeline
	// Find the last video element in the spine to get the end offset
	lastVideoEnd := findLastVideoOffset(newHNContent)

	// Convert audio duration to 24000s format for video duration
	videoDuration, err := convertAudioDurationToVideo(audioDuration)
	if err != nil {
		return fmt.Errorf("failed to convert audio duration: %v", err)
	}

	// Create video element for timeline (reuse Wikipedia logic)
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
                                <param name="Alignment" key="9999/10003/13260/3296674397/2/373" value="0 (Left) 2 (Bottom)"/>
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
                                    <text-style ref="ts2_%d">This content is adapted from Hacker News article above.</text-style>
                                </text>
                                <text-style-def id="ts1_%d">
                                    <text-style font="Helvetica Neue" fontSize="170" fontColor="1 1 1 1" bold="1" shadowColor="0 0 0 0.75" shadowOffset="5 315" lineSpacing="-19"/>
                                </text-style-def>
                                <text-style-def id="ts2_%d">
                                    <text-style font="Helvetica Neue" fontSize="69.56" fontFace="Medium" fontColor="1 1 1 1" shadowColor="0 0 0 0.75" shadowOffset="5 315"/>
                                </text-style-def>
                            </title>
                        </video>`,
		data.ImageAssetID, lastVideoEnd, videoDuration, data.AudioAssetID, name, videoDuration, data.TitleEffectID, title, videoDuration, timestamp, title, timestamp, timestamp, timestamp)

	// Insert video element before </spine>
	spineEnd := strings.Index(newHNContent, "                    </spine>")
	if spineEnd == -1 {
		return fmt.Errorf("could not find </spine> tag")
	}

	finalContent := newHNContent[:spineEnd] + videoElement + "\n" + newHNContent[spineEnd:]

	// Update sequence duration to include new clip
	finalContent = updateSequenceDuration(finalContent, lastVideoEnd, videoDuration)

	// Write back to file
	if err := os.WriteFile(hnPath, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write hn.fcpxml: %v", err)
	}

	return nil
}

// Helper functions from Wikipedia (for FCPXML timeline management)
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
