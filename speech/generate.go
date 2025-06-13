package speech

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

// generateVideoUID generates a unique identifier for the video file based on its content
func generateVideoUID(videoPath string) (string, error) {
	file, err := os.Open(videoPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// Convert to uppercase hex string like FCP UIDs
	return fmt.Sprintf("%X", hash.Sum(nil)), nil
}

// isPNGImage checks if the file is a PNG image
func isPNGImage(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".png"
}

// getMediaDuration gets the duration of a media file using ffprobe for video, or returns 20 seconds for PNG images
func getMediaDuration(mediaPath string) (string, string, error) {
	if isPNGImage(mediaPath) {
		// For PNG images, use a fixed 20-second duration
		duration := 20.0
		
		// Convert to FCP asset format (frames/44100s)
		assetFrames := int64(duration * 44100)
		assetDuration := fmt.Sprintf("%d/44100s", assetFrames)

		// Convert to FCP clip format (frames/600s) aligned to frame boundaries
		// Frame duration is 20/600s, so we need to round to multiples of 20
		clipFrames := int64(duration * 600)
		// Round to nearest frame boundary (multiple of 20)
		clipFrames = (clipFrames / 20) * 20
		clipDuration := fmt.Sprintf("%d/600s", clipFrames)

		return assetDuration, clipDuration, nil
	}

	// For video files, use ffprobe
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_entries", "format=duration", mediaPath)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to run ffprobe: %v", err)
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse ffprobe output: %v", err)
	}

	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse duration: %v", err)
	}

	// Convert to FCP asset format (frames/44100s)
	assetFrames := int64(duration * 44100)
	assetDuration := fmt.Sprintf("%d/44100s", assetFrames)

	// Convert to FCP clip format (frames/600s) aligned to frame boundaries
	// Frame duration is 20/600s, so we need to round to multiples of 20
	clipFrames := int64(duration * 600)
	// Round to nearest frame boundary (multiple of 20)
	clipFrames = (clipFrames / 20) * 20
	clipDuration := fmt.Sprintf("%d/600s", clipFrames)

	return assetDuration, clipDuration, nil
}

type TextElement struct {
	Text                string
	Index               int
	Offset              string
	Duration            string
	YPosition           int
	Lane                int
	ReverseStartTimeNano string
	ReverseEndTimeNano   string
}

type SpeechData struct {
	TextElements      []TextElement
	VideoPath         string
	VideoUID          string
	VideoDuration     string
	VideoClipDuration string
	ReverseStartTime  string
	ReverseEndTime    string
	IsStillImage      bool
}

func GenerateSpeechFCPXML(inputFile, outputFile, videoFile string) error {
	// Read the text file
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %v", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read input file: %v", err)
	}

	if len(lines) == 0 {
		return fmt.Errorf("no text found in input file")
	}

	// Create text elements with staggered timing
	var textElements []TextElement
	baseOffsetFrames := 4900         // Starting offset in timebase units (4900/3000s)
	pauseDurationFrames := 6000      // 2 seconds = 6000/3000s between each text appearance
	timeBase := 3000                 // From format frameDuration="100/3000s"
	yPositionBase := 800             // Base Y position
	ySpacing := 300                  // Vertical spacing between text elements
	
	// Calculate reverse animation timing
	// Last text appears at: baseOffsetFrames + (len(lines)-1) * pauseDurationFrames
	lastTextOffsetFrames := baseOffsetFrames + ((len(lines) - 1) * pauseDurationFrames)
	pauseAfterLastText := 6000       // 2 seconds pause after last text appears
	reverseAnimationDuration := 4000 // 1.33 seconds for reverse animation
	
	reverseStartFrames := lastTextOffsetFrames + pauseAfterLastText
	reverseEndFrames := reverseStartFrames + reverseAnimationDuration
	
	reverseStartTime := fmt.Sprintf("%d/%ds", reverseStartFrames, timeBase)
	reverseEndTime := fmt.Sprintf("%d/%ds", reverseEndFrames, timeBase)
	
	// Convert to nanoseconds for text animation (matching the existing format)
	reverseStartNano := fmt.Sprintf("%d/1000000000s", (reverseStartFrames * 1000000000) / timeBase)
	reverseEndNano := fmt.Sprintf("%d/1000000000s", (reverseEndFrames * 1000000000) / timeBase)

	for i, line := range lines {
		offsetFrames := baseOffsetFrames + (i * pauseDurationFrames)
		yPos := yPositionBase - (i * ySpacing) // Stack text elements vertically
		lane := -(i + 1)                       // Assign negative lanes (-1, -2, -3, -4 for items)
		
		// Calculate duration so each title ends just before the slide-back animation
		// Duration = reverseStartFrames - offsetFrames
		durationFrames := reverseStartFrames - offsetFrames
		duration := fmt.Sprintf("%d/%d", durationFrames, timeBase)

		textElements = append(textElements, TextElement{
			Text:                 line,
			Index:                i + 1,
			Offset:               fmt.Sprintf("%d/%d", offsetFrames, timeBase),
			Duration:             duration,
			YPosition:            yPos,
			Lane:                 lane,
			ReverseStartTimeNano: reverseStartNano,
			ReverseEndTimeNano:   reverseEndNano,
		})
	}

	// Get absolute path for video file
	absVideoPath, err := filepath.Abs(videoFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for video file: %v", err)
	}

	// Generate unique UID for the media file
	videoUID, err := generateVideoUID(videoFile)
	if err != nil {
		return fmt.Errorf("failed to generate media UID: %v", err)
	}

	// Get media duration
	videoDuration, videoClipDuration, err := getMediaDuration(videoFile)
	if err != nil {
		return fmt.Errorf("failed to get media duration: %v", err)
	}

	// Create the speech data
	speechData := SpeechData{
		TextElements:      textElements,
		VideoPath:         "file://" + absVideoPath,
		VideoUID:          videoUID,
		VideoDuration:     videoDuration,
		VideoClipDuration: videoClipDuration,
		ReverseStartTime:  reverseStartTime,
		ReverseEndTime:    reverseEndTime,
		IsStillImage:      isPNGImage(videoFile),
	}

	// Read the template
	templatePath := "templates/slide.fcpxml"
	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %v", err)
	}

	// Parse and execute the template
	tmpl, err := template.New("speech").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Execute template
	if err := tmpl.Execute(outFile, speechData); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}

type ResumeSection struct {
	ImagePath string
	TextLines []string
}

type ResumeData struct {
	Sections []ResumeSection
	TotalDuration string
	TotalClipDuration string
}

func GenerateResumeFCPXML(resumeFile, outputFile string) error {
	// Parse the resume file
	sections, err := parseResumeFile(resumeFile)
	if err != nil {
		return fmt.Errorf("failed to parse resume file: %v", err)
	}

	if len(sections) == 0 {
		return fmt.Errorf("no sections found in resume file")
	}

	// Calculate total duration based on number of sections
	// Each section gets 20 seconds
	sectionDuration := 20.0
	totalDuration := float64(len(sections)) * sectionDuration
	
	// Convert to FCP asset format (frames/44100s)
	assetFrames := int64(totalDuration * 44100)
	totalAssetDuration := fmt.Sprintf("%d/44100s", assetFrames)

	// Convert to FCP clip format (frames/600s) aligned to frame boundaries
	clipFrames := int64(totalDuration * 600)
	// Round to nearest frame boundary (multiple of 20)
	clipFrames = (clipFrames / 20) * 20
	totalClipDuration := fmt.Sprintf("%d/600s", clipFrames)

	// Create the resume data
	resumeData := ResumeData{
		Sections: sections,
		TotalDuration: totalAssetDuration,
		TotalClipDuration: totalClipDuration,
	}

	// Generate FCPXML using template
	if err := generateResumeXML(resumeData, outputFile); err != nil {
		return fmt.Errorf("failed to generate XML: %v", err)
	}

	return nil
}

func parseResumeFile(resumeFile string) ([]ResumeSection, error) {
	file, err := os.Open(resumeFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var sections []ResumeSection
	var currentSection *ResumeSection
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Check if this line is a PNG filename
		if strings.HasSuffix(strings.ToLower(line), ".png") {
			// Start a new section
			if currentSection != nil {
				sections = append(sections, *currentSection)
			}
			
			// Get absolute path for the image
			imagePath := filepath.Join("assets", line)
			absImagePath, err := filepath.Abs(imagePath)
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path for image %s: %v", line, err)
			}
			
			// Check if image file exists
			if _, err := os.Stat(absImagePath); os.IsNotExist(err) {
				return nil, fmt.Errorf("image file does not exist: %s", absImagePath)
			}
			
			currentSection = &ResumeSection{
				ImagePath: "file://" + absImagePath,
				TextLines: []string{},
			}
		} else if currentSection != nil {
			// Add text line to current section (any non-PNG line)
			text := strings.TrimSpace(line)
			if text != "" {
				currentSection.TextLines = append(currentSection.TextLines, text)
			}
		}
	}

	// Add the last section
	if currentSection != nil {
		sections = append(sections, *currentSection)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return sections, nil
}

func generateResumeXML(data ResumeData, outputFile string) error {
	// Create a multi-section FCPXML template
	tmplContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

<fcpxml version="1.13">
	<resources>
		<format id="r1" name="FFVideoFormat720p30" frameDuration="100/3000s" width="1280" height="720" colorSpace="1-1-1 (Rec. 709)"/>
		{{range $sectionIndex, $section := .Sections}}
		<asset id="r{{add $sectionIndex 10}}" name="{{base $section.ImagePath}}" uid="{{printf "%08X" (add $sectionIndex 1)}}" start="0s" duration="{{$.TotalDuration}}" hasVideo="1" format="r1" hasAudio="0">
			<media-rep kind="original-media" sig="{{printf "%032X" (add $sectionIndex 1)}}" src="{{$section.ImagePath}}"></media-rep>
		</asset>
		{{end}}
		<effect id="r2" name="Text" uid=".../Titles.localized/Basic Text.localized/Text.localized/Text.moti"/>
	</resources>

	<library location="file:///Users/aa/Desktop/">
		<event name="Test Project" uid="AC90D6CC-5C26-44CA-805E-7BA143E57440">
			<project name="Resume" uid="63FC3253-EB57-4F5E-9653-0C4F64E72E40" id="r3">
				<sequence duration="{{.TotalClipDuration}}" format="r1" tcStart="0s" tcFormat="NDF" audioLayout="stereo" audioRate="48k">
					<spine>
						{{range $sectionIndex, $section := .Sections}}
						<video ref="r{{add $sectionIndex 10}}" offset="{{mul $sectionIndex 12000}}/600s" duration="12000/600s" start="0s" name="{{base $section.ImagePath}}">
							<adjust-transform>
								<param name="position">
									<param name="X" key="1">
										<keyframeAnimation>
											<keyframe time="0s" value="0"/>
											<keyframe time="4000/3000s" value="41.6667"/>
											<keyframe time="28900/3000s" value="41.6667"/>
											<keyframe time="32900/3000s" value="0"/>
										</keyframeAnimation>
									</param>
									<param name="Y" key="2">
										<keyframeAnimation>
											<keyframe time="0s" value="0" curve="linear"/>
										</keyframeAnimation>
									</param>
								</param>
							</adjust-transform>
							{{range $textIndex, $text := $section.TextLines}}
							<title ref="r2" offset="{{add (mul $textIndex 1800) 1800}}/600s" duration="{{sub 12000 (add (mul $textIndex 1800) 1800)}}/600s" name="{{$text}}" lane="{{sub 0 (add $textIndex 1)}}" start="1800/600s">
								<param name="Build In" key="9999/10000/2/101" value="0"/>
								<param name="Build Out" key="9999/10000/2/102" value="0"/>
								<param name="Position" key="9999/10003/13260/3296672360/1/100/101" value="0 {{sub 800 (mul $textIndex 300)}}"/>
								<param name="Layout Method" key="9999/10003/13260/3296672360/2/314" value="1 (Paragraph)"/>
								<param name="Left Margin" key="9999/10003/13260/3296672360/2/323" value="-1730"/>
								<param name="Right Margin" key="9999/10003/13260/3296672360/2/324" value="1730"/>
								<param name="Top Margin" key="9999/10003/13260/3296672360/2/325" value="960"/>
								<param name="Bottom Margin" key="9999/10003/13260/3296672360/2/326" value="-960"/>
								<param name="Alignment" key="9999/10003/13260/3296672360/2/354/3296667315/401" value="3 (Justify Last Line Left)"/>
								<param name="Justification" key="9999/10003/13260/3296672360/2/354/3296667315/402" value="2 (Full)"/>
								<param name="Line Spacing" key="9999/10003/13260/3296672360/2/354/3296667315/404" value="-19"/>
								<param name="Auto-Shrink" key="9999/10003/13260/3296672360/2/370" value="3 (To All Margins)"/>
								<param name="Alignment" key="9999/10003/13260/3296672360/2/373" value="0 (Left) 1 (Middle)"/>
								<param name="Opacity" key="9999/10003/13260/3296672360/4/3296673134/1000/1044" value="0"/>
								<param name="Speed" key="9999/10003/13260/3296672360/4/3296673134/201/208" value="6 (Custom)"/>
								<param name="Custom Speed" key="9999/10003/13260/3296672360/4/3296673134/201/209">
									<keyframeAnimation>
										<keyframe time="-469658744/1000000000s" value="0"/>
										<keyframe time="9633333333/1000000000s" value="1"/>
										<keyframe time="10966666666/1000000000s" value="0"/>
										<keyframe time="12328542033/1000000000s" value="1"/>
									</keyframeAnimation>
								</param>
								<param name="Apply Speed" key="9999/10003/13260/3296672360/4/3296673134/201/211" value="2 (Per Object)"/>
								<text>
									<text-style ref="ts{{add $sectionIndex 10}}{{add $textIndex 1}}">{{$text}}</text-style>
								</text>
								<text-style-def id="ts{{add $sectionIndex 10}}{{add $textIndex 1}}">
									<text-style font="Helvetica Neue" fontSize="48" fontFace="Bold" fontColor="1 1 1 1" alignment="center"/>
								</text-style-def>
							</title>
							{{end}}
						</video>
						{{end}}
					</spine>
				</sequence>
			</project>
		</event>
	</library>
</fcpxml>`

	// Parse template
	tmpl, err := template.New("resume").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"base": func(path string) string { return filepath.Base(strings.TrimPrefix(path, "file://")) },
	}).Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Execute template
	if err := tmpl.Execute(outFile, data); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}