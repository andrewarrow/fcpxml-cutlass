package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type VTTSegment struct {
	StartTime time.Duration
	EndTime   time.Duration
	Text      string
}

type Clip struct {
	StartTime time.Duration
	EndTime   time.Duration
	Duration  time.Duration
	Text      string
	ClipNum   int
}

type FCPXML struct {
	XMLName  xml.Name `xml:"fcpxml"`
	Version  string   `xml:"version,attr"`
	Resources Resources `xml:"resources"`
	Library  Library  `xml:"library"`
}

type Resources struct {
	Formats []Format `xml:"format"`
	Assets  []Asset  `xml:"asset"`
    Effects []Effect `xml:"effect,omitempty"`
}

// Effect represents a Motion or standard FCP title effect referenced by <title ref="…"> elements.
type Effect struct {
    ID   string `xml:"id,attr"`
    Name string `xml:"name,attr"`
    UID  string `xml:"uid,attr,omitempty"`
}

type Format struct {
	ID            string `xml:"id,attr"`
	Name          string `xml:"name,attr"`
	FrameDuration string `xml:"frameDuration,attr"`
	Width         string `xml:"width,attr"`
	Height        string `xml:"height,attr"`
	ColorSpace    string `xml:"colorSpace,attr"`
}

type Asset struct {
	ID           string    `xml:"id,attr"`
	Name         string    `xml:"name,attr"`
	UID          string    `xml:"uid,attr"`
	Start        string    `xml:"start,attr"`
	HasVideo     string    `xml:"hasVideo,attr"`
	Format       string    `xml:"format,attr"`
	HasAudio     string    `xml:"hasAudio,attr"`
	AudioSources string    `xml:"audioSources,attr"`
	AudioChannels string   `xml:"audioChannels,attr"`
	Duration     string    `xml:"duration,attr"`
	MediaRep     MediaRep  `xml:"media-rep"`
}

type MediaRep struct {
	Kind string `xml:"kind,attr"`
	Sig  string `xml:"sig,attr"`
	Src  string `xml:"src,attr"`
}

type Library struct {
	Events []Event `xml:"event"`
}

type Event struct {
	Name     string    `xml:"name,attr"`
	Projects []Project `xml:"project"`
}

type Project struct {
	Name      string   `xml:"name,attr"`
	Sequences []Sequence `xml:"sequence"`
}

type Sequence struct {
	Format      string `xml:"format,attr"`
	Duration    string `xml:"duration,attr"`
	TCStart     string `xml:"tcStart,attr"`
	TCFormat    string `xml:"tcFormat,attr"`
	AudioLayout string `xml:"audioLayout,attr"`
	AudioRate   string `xml:"audioRate,attr"`
	Spine       Spine  `xml:"spine"`
}

type Spine struct {
	XMLName xml.Name `xml:"spine"`
	Content string   `xml:",innerxml"`
}

type AssetClip struct {
	Ref       string `xml:"ref,attr"`
	Offset    string `xml:"offset,attr"`
	Name      string `xml:"name,attr"`
	Start     string `xml:"start,attr,omitempty"`
	Duration  string `xml:"duration,attr"`
	Format    string `xml:"format,attr"`
	TCFormat  string `xml:"tcFormat,attr"`
	AudioRole string `xml:"audioRole,attr,omitempty"`
}

type Gap struct {
	Name     string  `xml:"name,attr"`
	Offset   string  `xml:"offset,attr"`
	Duration string  `xml:"duration,attr"`
	Titles   []Title `xml:"title"`
}

type Title struct {
    Ref           string         `xml:"ref,attr"`
	Lane         string         `xml:"lane,attr"`
	Offset       string         `xml:"offset,attr"`
	Name         string         `xml:"name,attr"`
	Duration     string         `xml:"duration,attr"`
    Start        string         `xml:"start,attr,omitempty"`
	Params       []Param        `xml:"param"`
	Text         TitleText      `xml:"text"`
	TextStyleDef TextStyleDef   `xml:"text-style-def"`
}

type Param struct {
	Name  string `xml:"name,attr"`
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

type TitleText struct {
	TextStyle TextStyleRef `xml:"text-style"`
}

type TextStyleRef struct {
	Ref  string `xml:"ref,attr"`
	Text string `xml:",chardata"`
}

type TextStyleDef struct {
	ID        string    `xml:"id,attr"`
	TextStyle TextStyle `xml:"text-style"`
}

type TextStyle struct {
	Font      string `xml:"font,attr"`
	FontSize  string `xml:"fontSize,attr"`
	FontFace  string `xml:"fontFace,attr"`
	FontColor string `xml:"fontColor,attr"`
	Alignment string `xml:"alignment,attr"`
}

func parseVTTTime(timeStr string) (time.Duration, error) {
	// Parse format like "00:00:02.350"
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format: %s", timeStr)
	}
	
	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	secondsParts := strings.Split(parts[2], ".")
	seconds, _ := strconv.Atoi(secondsParts[0])
	milliseconds := 0
	if len(secondsParts) > 1 {
		// Pad or truncate to 3 digits
		msStr := secondsParts[1]
		if len(msStr) > 3 {
			msStr = msStr[:3]
		} else {
			for len(msStr) < 3 {
				msStr += "0"
			}
		}
		milliseconds, _ = strconv.Atoi(msStr)
	}
	
	return time.Duration(hours)*time.Hour + 
		   time.Duration(minutes)*time.Minute + 
		   time.Duration(seconds)*time.Second + 
		   time.Duration(milliseconds)*time.Millisecond, nil
}

func parseVTTFile(vttPath string) ([]VTTSegment, error) {
	file, err := os.Open(vttPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var segments []VTTSegment
	scanner := bufio.NewScanner(file)
	
	// Regex to match timestamp lines like "00:00:00.160 --> 00:00:02.350"
	timeRegex := regexp.MustCompile(`(\d{2}:\d{2}:\d{2}\.\d{3})\s+-->\s+(\d{2}:\d{2}:\d{2}\.\d{3})`)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if timeRegex.MatchString(line) {
			matches := timeRegex.FindStringSubmatch(line)
			if len(matches) >= 3 {
				startTime, err1 := parseVTTTime(matches[1])
				endTime, err2 := parseVTTTime(matches[2])
				
				if err1 == nil && err2 == nil {
					// Read the next line for text content
					var textLines []string
					for scanner.Scan() {
						textLine := strings.TrimSpace(scanner.Text())
						if textLine == "" {
							break
						}
						// Clean up VTT formatting tags
						cleanText := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(textLine, "")
						cleanText = regexp.MustCompile(`<\d{2}:\d{2}:\d{2}\.\d{3}>.*?</c>`).ReplaceAllString(cleanText, "")
						if cleanText != "" {
							textLines = append(textLines, cleanText)
						}
					}
					
					if len(textLines) > 0 {
						segments = append(segments, VTTSegment{
							StartTime: startTime,
							EndTime:   endTime,
							Text:      strings.Join(textLines, " "),
						})
					}
				}
			}
		}
	}
	
	return segments, scanner.Err()
}

func segmentIntoClips(segments []VTTSegment, minDuration, maxDuration time.Duration) []Clip {
	var clips []Clip
	clipNum := 1
	
	// Sort segments by start time
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].StartTime < segments[j].StartTime
	})
	
	i := 0
	for i < len(segments) {
		clipStart := segments[i].StartTime
		var clipTexts []string
		clipTexts = append(clipTexts, segments[i].Text)
		
		// Extend clip duration by adding consecutive segments
		j := i + 1
		for j < len(segments) {
			currentDuration := segments[j-1].EndTime - clipStart
			
			// If adding this segment would exceed max duration, stop
			if currentDuration >= maxDuration {
				break
			}
			
			// If we have minimum duration and there's a natural break, stop
			if currentDuration >= minDuration {
				// Look for sentence endings or pauses
				lastText := segments[j-1].Text
				if strings.HasSuffix(lastText, ".") || strings.HasSuffix(lastText, "!") || strings.HasSuffix(lastText, "?") {
					break
				}
			}
			
			clipTexts = append(clipTexts, segments[j].Text)
			j++
		}
		
		clipEnd := segments[j-1].EndTime
		clipDuration := clipEnd - clipStart
		
		// Ensure minimum duration
		if clipDuration < minDuration && j < len(segments) {
			clipEnd = clipStart + minDuration
		}
		
		clips = append(clips, Clip{
			StartTime: clipStart,
			EndTime:   clipEnd,
			Duration:  clipEnd - clipStart,
			Text:      strings.Join(clipTexts, " "),
			ClipNum:   clipNum,
		})
		
		clipNum++
		i = j
	}
	
	return clips
}

func formatDurationForFCPXML(d time.Duration) string {
	// Convert to frame-aligned format for 30fps video
	// 30000 frames per second with 1001/30000s frame duration
	totalFrames := int64(d.Seconds() * 30000 / 1001)
	// Ensure frame alignment
	return fmt.Sprintf("%d/30000s", totalFrames*1001)
}

func buildClipFCPXML(clips []Clip, videoPath string) (FCPXML, error) {
	absVideoPath, err := filepath.Abs(videoPath)
	if err != nil {
		return FCPXML{}, err
	}
	
	videoName := filepath.Base(absVideoPath)
	nameWithoutExt := strings.TrimSuffix(videoName, filepath.Ext(videoName))
	
	// Calculate total duration - textblock, clip, textblock, clip pattern
	var totalDuration time.Duration
	for _, clip := range clips {
		totalDuration += clip.Duration + 10*time.Second // Add 10s for textblock
	}
	totalDuration += 10*time.Second // Add final textblock
	
	var spineContent strings.Builder
	currentOffset := time.Duration(0)
	
	for i, clip := range clips {
		// Textblock gap before each clip
		escapedText := strings.ReplaceAll(clip.Text, "&", "&amp;")
		escapedText = strings.ReplaceAll(escapedText, "<", "&lt;")
		escapedText = strings.ReplaceAll(escapedText, ">", "&gt;")
		escapedText = strings.ReplaceAll(escapedText, "\"", "&quot;")
		escapedText = strings.ReplaceAll(escapedText, "'", "&#39;")
		
		spineContent.WriteString(fmt.Sprintf(`
			<gap name="Gap" offset="%s" duration="%s">
				<title ref="r2" lane="1" offset="%s" name="Graphic Text Block" start="%s" duration="%s">
					<text>
						<text-style ref="ts%d">%s</text-style>
					</text>
					<text-style-def id="ts%d">
						<text-style font="Helvetica Neue" fontSize="176.8" fontColor="1 1 1 1"/>
					</text-style-def>
				</title>
			</gap>`,
			formatDurationForFCPXML(currentOffset),
			formatDurationForFCPXML(10 * time.Second),
			formatDurationForFCPXML(360 * time.Millisecond),
			formatDurationForFCPXML(360 * time.Millisecond),
			formatDurationForFCPXML(10*time.Second - 133*time.Millisecond),
			i+1, escapedText, i+1))
		
		currentOffset += 10 * time.Second
		
		// Video clip
		spineContent.WriteString(fmt.Sprintf(`
			<asset-clip ref="r3" offset="%s" name="%s Clip %d" start="%s" duration="%s" tcFormat="NDF" audioRole="dialogue"/>`,
			formatDurationForFCPXML(currentOffset),
			nameWithoutExt, clip.ClipNum,
			formatDurationForFCPXML(clip.StartTime),
			formatDurationForFCPXML(clip.Duration)))
		
		currentOffset += clip.Duration
	}
	
	return FCPXML{
		Version: "1.11",
        Resources: Resources{
            Formats: []Format{
                {
                    ID:            "r1",
                    Name:          "FFVideoFormat1080p30",
                    FrameDuration: "1001/30000s",
                    Width:         "1920",
                    Height:        "1080",
                    ColorSpace:    "1-1-1 (Rec. 709)",
                },
            },
            Assets: []Asset{
                {
                    ID:           "r3",
                    Name:         nameWithoutExt,
                    UID:          absVideoPath,
                    Start:        "0s",
                    HasVideo:     "1",
                    Format:       "r1",
                    HasAudio:     "1",
                    AudioSources: "1",
                    AudioChannels: "2",
                    Duration:     formatDurationForFCPXML(totalDuration),
                    MediaRep: MediaRep{
                        Kind: "original-media",
                        Sig:  absVideoPath,
                        Src:  "file://" + absVideoPath,
                    },
                },
            },
            Effects: []Effect{
                {
                    ID:   "r2",
                    Name: "Graphic Text Block",
                    UID:  ".../Titles.localized/Basic Text.localized/Graphic Text Block.localized/Graphic Text Block.moti",
                },
            },
        },
		Library: Library{
			Events: []Event{
				{
					Name: "Auto Generated Clips",
					Projects: []Project{
						{
							Name: nameWithoutExt + " Clips",
							Sequences: []Sequence{
								{
									Format:      "r1",
									Duration:    formatDurationForFCPXML(totalDuration),
									TCStart:     "0s",
									TCFormat:    "NDF",
									AudioLayout: "stereo",
									AudioRate:   "48k",
									Spine: Spine{
										Content: spineContent.String(),
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func generateClipFCPXML(clips []Clip, videoPath, outputPath string) error {
	fcpxml, err := buildClipFCPXML(clips, videoPath)
	if err != nil {
		return err
	}

	output, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		return err
	}

	xmlContent := xml.Header + "<!DOCTYPE fcpxml>\n" + string(output)
	return os.WriteFile(outputPath, []byte(xmlContent), 0644)
}

func breakIntoLogicalParts(youtubeID string) error {
	vttPath := fmt.Sprintf("%s.en.vtt", youtubeID)
	videoPath := fmt.Sprintf("%s.mov", youtubeID)
	outputPath := fmt.Sprintf("%s_clips.fcpxml", youtubeID)
	
	// Check if files exist
	if _, err := os.Stat(vttPath); os.IsNotExist(err) {
		return fmt.Errorf("VTT file not found: %s", vttPath)
	}
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return fmt.Errorf("video file not found: %s", videoPath)
	}
	
	// Parse VTT file
	fmt.Printf("Parsing VTT file: %s\n", vttPath)
	segments, err := parseVTTFile(vttPath)
	if err != nil {
		return fmt.Errorf("error parsing VTT file: %v", err)
	}
	
	fmt.Printf("Found %d VTT segments\n", len(segments))
	
	// Segment into logical clips (6-18 seconds)
	minDuration := 6 * time.Second
	maxDuration := 18 * time.Second
	clips := segmentIntoClips(segments, minDuration, maxDuration)
	
	fmt.Printf("Generated %d clips\n", len(clips))
	for i, clip := range clips {
		fmt.Printf("Clip %d: %v - %v (%.1fs) - %s\n", 
			i+1, clip.StartTime, clip.EndTime, clip.Duration.Seconds(), 
			clip.Text[:min(50, len(clip.Text))])
	}
	
	// Generate FCPXML
	fmt.Printf("Generating FCPXML: %s\n", outputPath)
	err = generateClipFCPXML(clips, videoPath, outputPath)
	if err != nil {
		return fmt.Errorf("error generating FCPXML: %v", err)
	}
	
	fmt.Printf("Successfully generated %s with %d clips\n", outputPath, len(clips))
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
