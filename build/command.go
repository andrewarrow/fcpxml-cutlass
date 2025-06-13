package build

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cutlass/fcp"
	"github.com/spf13/cobra"
)

var BuildCmd = &cobra.Command{
	Use:   "build [filename] [add-video] [media-file]",
	Short: "Build a blank FCP project or add media to existing project",
	Long:  "Create a blank Final Cut Pro project from empty.fcpxml template, or add a video/image to it",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		if !strings.HasSuffix(filename, ".fcpxml") {
			filename += ".fcpxml"
		}
		
		// Check if this is an add-video command
		if len(args) >= 3 && args[1] == "add-video" {
			mediaFile := args[2]
			
			// Get the --with-text flag value
			withText, _ := cmd.Flags().GetString("with-text")
			
			// First ensure the project exists, create if it doesn't
			if _, err := os.Stat(filename); os.IsNotExist(err) {
				err := createBlankProject(filename)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating blank project: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Created blank project: %s\n", filename)
			}
			
			// Add media to the project
			err := addVideoToProject(filename, mediaFile, withText)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error adding media to project: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Added media %s to project %s\n", mediaFile, filename)
			if withText != "" {
				fmt.Printf("Added text overlay: %s\n", withText)
			}
		} else {
			// Just create a blank project
			err := createBlankProject(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating blank project: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Created blank project: %s\n", filename)
		}
	},
}

func init() {
	BuildCmd.Flags().String("with-text", "", "Add text overlay on top of the video")
}


func createBlankProject(filename string) error {
	// Read the empty.fcpxml template
	emptyContent, err := ioutil.ReadFile("empty.fcpxml")
	if err != nil {
		return fmt.Errorf("failed to read empty.fcpxml: %v", err)
	}
	
	// Parse the XML to modify timestamps and UIDs
	var fcpxml fcp.FCPXML
	err = xml.Unmarshal(emptyContent, &fcpxml)
	if err != nil {
		return fmt.Errorf("failed to parse empty.fcpxml: %v", err)
	}
	
	// Update timestamps and generate new UIDs
	currentTime := time.Now().Format("2006-01-02 15:04:05 -0700")
	
	if len(fcpxml.Library.Events) > 0 {
		// Update event name to current date
		fcpxml.Library.Events[0].Name = time.Now().Format("1-2-06")
		
		if len(fcpxml.Library.Events[0].Projects) > 0 {
			// Update project modification date
			fcpxml.Library.Events[0].Projects[0].ModDate = currentTime
			
			// Extract base filename without extension
			baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
			fcpxml.Library.Events[0].Projects[0].Name = baseName
		}
	}
	
	// Generate the XML output with proper formatting
	output, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %v", err)
	}
	
	// Add XML declaration and DOCTYPE
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

` + string(output)
	
	// Write to output file
	err = ioutil.WriteFile(filename, []byte(xmlContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}
	
	return nil
}

// isPNGFile checks if the given file is a PNG image
func isPNGFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".png"
}

func addVideoToProject(projectFile, videoFile, withText string) error {
	// Read the existing project file
	content, err := ioutil.ReadFile(projectFile)
	if err != nil {
		return fmt.Errorf("failed to read project file: %v", err)
	}
	
	// Parse the XML
	var fcpxml fcp.FCPXML
	err = xml.Unmarshal(content, &fcpxml)
	if err != nil {
		return fmt.Errorf("failed to parse project file: %v", err)
	}
	
	// Get absolute path for the video file
	absVideoPath, err := filepath.Abs(videoFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}
	
	// Check if media file exists
	if _, err := os.Stat(absVideoPath); os.IsNotExist(err) {
		return fmt.Errorf("media file does not exist: %s", absVideoPath)
	}
	
	// Create asset ID and UID
	baseName := strings.TrimSuffix(filepath.Base(videoFile), filepath.Ext(videoFile))
	
	// Get duration based on file type
	var duration string
	if isPNGFile(absVideoPath) {
		// PNG files are set to 10 seconds
		duration = "240240/24000s" // 10 seconds at 23.976fps
	} else {
		// Get video duration for video files
		var err error
		duration, err = getVideoDuration(absVideoPath)
		if err != nil {
			duration = "240240/24000s" // Default to 10 seconds if duration detection fails
		}
	}
	
	// Create PNG format first if needed
	var pngFormatID string
	if isPNGFile(absVideoPath) {
		// Check if PNG format already exists
		pngFormatExists := false
		for _, format := range fcpxml.Resources.Formats {
			if format.Name == "FFVideoFormatRateUndefined" {
				pngFormatID = format.ID
				pngFormatExists = true
				break
			}
		}
		
		// Add PNG format if it doesn't exist
		if !pngFormatExists {
			// Generate a unique ID for the PNG format - must not conflict with any other resource IDs
			// Count all existing resources: assets + formats + effects
			totalResources := len(fcpxml.Resources.Assets) + len(fcpxml.Resources.Formats) + len(fcpxml.Resources.Effects)
			pngFormatID = fmt.Sprintf("r%d", totalResources+1)
			pngFormat := fcp.Format{
				ID:         pngFormatID,
				Name:       "FFVideoFormatRateUndefined",
				Width:      "1280",
				Height:     "720",
				ColorSpace: "1-13-1",
				// No FrameDuration for still images
			}
			fcpxml.Resources.Formats = append(fcpxml.Resources.Formats, pngFormat)
		}
	}

	// Check if asset already exists in the project
	existingAssetID := findExistingAsset(&fcpxml, absVideoPath)
	if existingAssetID == "" {
		// Asset doesn't exist, create it
		// Calculate next available ID considering all resources: assets + formats + effects
		totalResources := len(fcpxml.Resources.Assets) + len(fcpxml.Resources.Formats) + len(fcpxml.Resources.Effects)
		assetID := fmt.Sprintf("r%d", totalResources+1)
		
		// Generate consistent UID from file path
		assetUID := generateUID(absVideoPath)
		
		// Generate bookmark for the video file
		_, _ = generateBookmark(absVideoPath) // Ignore errors, continue without bookmark

		// Ensure Text effect exists in resources (needed for text overlays)
	ensureTextEffect(&fcpxml)
	
	// Add asset to resources
		var asset fcp.Asset
		if isPNGFile(absVideoPath) {
			// PNG/image asset - similar to Final Cut Pro's structure
			asset = fcp.Asset{
				ID:           assetID,
				Name:         baseName,
				UID:          assetUID,
				Start:        "0s",
				Duration:     "0s", // PNG assets use 0s duration in FCP
				HasVideo:     "1",
				Format:       pngFormatID, // Use the PNG format
				VideoSources: "1",         // Required for image assets
				MediaRep: fcp.MediaRep{
					Kind: "original-media",
					Sig:  assetUID, // Use same UID for sig
					Src:  "file://" + absVideoPath,
				},
			}
		} else {
			// Video asset
			asset = fcp.Asset{
				ID:            assetID,
				Name:          baseName,
				UID:           assetUID,
				Start:         "0s",
				HasVideo:      "1",
				Format:        "r1",
				VideoSources:  "", // Empty for video assets
				HasAudio:      "1",
				AudioSources:  "1",
				AudioChannels: "2",
				Duration:      duration,
				MediaRep: fcp.MediaRep{
					Kind: "original-media",
					Sig:  assetUID, // Use same UID for sig
					Src:  "file://" + absVideoPath,
				},
			}
		}
		
		fcpxml.Resources.Assets = append(fcpxml.Resources.Assets, asset)
		existingAssetID = assetID
	}
	
	// Add asset-clip to the spine
	if len(fcpxml.Library.Events) > 0 && len(fcpxml.Library.Events[0].Projects) > 0 {
		project := &fcpxml.Library.Events[0].Projects[0]
		if len(project.Sequences) > 0 {
			// Calculate offset by parsing existing spine content
			offset := calculateTimelineOffset(project.Sequences[0].Spine.Content)
			
			var clipXML []byte
			var err error
			
			if isPNGFile(absVideoPath) {
				// Use video element for PNG files (still images)
				videoClip := fcp.Video{
					Ref:      existingAssetID,
					Offset:   offset,
					Name:     baseName,
					Start:    "0s",
					Duration: duration,
				}
				
				// Add text overlay if requested
				if withText != "" {
					textTitle := createTextTitle(withText, duration, baseName)
					videoClip.NestedTitles = []fcp.Title{textTitle}
				}
				
				clipXML, err = xml.Marshal(videoClip)
			} else {
				// Use asset-clip for video files
				assetClip := fcp.AssetClip{
					Ref:      existingAssetID,
					Offset:   offset,
					Name:     baseName,
					Duration: duration,
					Format:   "r1",
					TCFormat: "NDF",
				}
				
				// Add text overlay if requested
				if withText != "" {
					textTitle := createTextTitle(withText, duration, baseName)
					assetClip.Titles = []fcp.Title{textTitle}
				}
				
				clipXML, err = xml.Marshal(assetClip)
			}
			
			if err != nil {
				return fmt.Errorf("failed to marshal clip: %v", err)
			}
			
			// Append to existing spine content
			indentedXML := strings.ReplaceAll(string(clipXML), "\n", "\n                        ")
			if strings.TrimSpace(project.Sequences[0].Spine.Content) == "" {
				// First clip
				project.Sequences[0].Spine.Content = "\n                        " + indentedXML + "\n                    "
			} else {
				// Append to existing clips
				project.Sequences[0].Spine.Content = strings.TrimSuffix(project.Sequences[0].Spine.Content, "\n                    ") + 
					"\n                        " + indentedXML + "\n                    "
			}
			
			// Update sequence duration to total timeline length
			totalDuration := calculateTotalDuration(project.Sequences[0].Spine.Content)
			project.Sequences[0].Duration = totalDuration
		}
	}
	
	
	// Generate the XML output
	output, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %v", err)
	}
	
	// Add XML declaration and DOCTYPE
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

` + string(output)
	
	// Write back to project file
	err = ioutil.WriteFile(projectFile, []byte(xmlContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write project file: %v", err)
	}
	
	return nil
}

// generateUID creates a consistent UID from a file path using MD5 hash
func generateUID(filePath string) string {
	// Create a hash from the file path to ensure consistent UIDs
	hasher := md5.New()
	hasher.Write([]byte("cutlass_video_" + filePath))
	hash := hasher.Sum(nil)
	// Convert to uppercase hex string and format as UID
	hexStr := strings.ToUpper(hex.EncodeToString(hash))
	return fmt.Sprintf("%s-%s-%s-%s-%s", 
		hexStr[0:8], hexStr[8:12], hexStr[12:16], hexStr[16:20], hexStr[20:32])
}

// generateBookmark creates a macOS security bookmark for a file path using Swift
func generateBookmark(filePath string) (string, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}
	
	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", absPath)
	}
	
	// Use Swift to create a security bookmark
	swiftCode := fmt.Sprintf(`
import Foundation

let url = URL(fileURLWithPath: "%s")
do {
    let bookmarkData = try url.bookmarkData(options: [.suitableForBookmarkFile])
    let base64String = bookmarkData.base64EncodedString()
    print(base64String)
} catch {
    print("ERROR: Could not create bookmark: \\(error)")
}
`, absPath)
	
	// Create temporary Swift file
	tmpFile, err := os.CreateTemp("", "bookmark*.swift")
	if err != nil {
		return "", nil
	}
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(swiftCode)
	tmpFile.Close()
	if err != nil {
		return "", nil
	}
	
	// Execute Swift script
	cmd := exec.Command("swift", tmpFile.Name())
	output, err := cmd.Output()
	if err != nil {
		// Fallback to empty bookmark - some systems may still work
		return "", nil
	}
	
	bookmark := strings.TrimSpace(string(output))
	if strings.Contains(bookmark, "ERROR") {
		return "", nil
	}
	
	return bookmark, nil
}

// findExistingAsset checks if an asset with the same file path already exists
func findExistingAsset(fcpxml *fcp.FCPXML, filePath string) string {
	for _, asset := range fcpxml.Resources.Assets {
		if asset.MediaRep.Src == "file://"+filePath {
			return asset.ID
		}
	}
	return ""
}

// calculateTimelineOffset parses existing spine content and calculates where the next clip should start
func calculateTimelineOffset(spineContent string) string {
	if strings.TrimSpace(spineContent) == "" {
		return "0s"
	}
	
	// Parse existing asset-clips to find the total timeline length
	totalDuration := calculateTotalDuration(spineContent)
	return totalDuration
}

// calculateTotalDuration parses spine content and calculates the total timeline duration
func calculateTotalDuration(spineContent string) string {
	if strings.TrimSpace(spineContent) == "" {
		return "0s"
	}
	
	// Find all duration values in both asset-clips and video elements
	totalFrames := 0
	
	// Use regex to find all duration attributes more precisely
	// This avoids double-counting when splitting by both tags
	lines := strings.Split(spineContent, "\n")
	for _, line := range lines {
		// Look for asset-clip or video elements with duration
		if (strings.Contains(line, "asset-clip") || strings.Contains(line, "<video")) && strings.Contains(line, "duration=") {
			// Extract duration value
			start := strings.Index(line, "duration=\"") + 10
			if start > 9 {
				end := strings.Index(line[start:], "\"")
				if end > 0 {
					durationStr := line[start : start+end]
					// Parse "frames/24000s" format
					if strings.HasSuffix(durationStr, "/24000s") {
						framesStr := strings.TrimSuffix(durationStr, "/24000s")
						if frames, err := strconv.Atoi(framesStr); err == nil {
							totalFrames += frames
						}
					}
				}
			}
		}
	}
	
	if totalFrames == 0 {
		return "0s"
	}
	
	return fmt.Sprintf("%d/24000s", totalFrames)
}

// getVideoDuration uses ffprobe to get video duration in FCP time format
func getVideoDuration(videoPath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", videoPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	
	err = json.Unmarshal(output, &result)
	if err != nil {
		return "", err
	}
	
	// Parse duration as float seconds
	durationFloat, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return "", err
	}
	
	// Convert to frame count using the sequence time base (1001/24000s frame duration)
	// This means 24000/1001 frames per second ≈ 23.976 fps
	framesPerSecond := 24000.0 / 1001.0
	frames := int(durationFloat * framesPerSecond)
	
	// Format as rational using the sequence time base
	return fmt.Sprintf("%d/24000s", frames*1001), nil
}

// ensureTextEffect ensures the Text effect is available in resources
func ensureTextEffect(fcpxml *fcp.FCPXML) {
	// Check if Text effect already exists
	for _, effect := range fcpxml.Resources.Effects {
		if effect.Name == "Text" {
			return // Already exists
		}
	}
	
	// Add Text effect if it doesn't exist
	textEffect := fcp.Effect{
		ID:   "r6",
		Name: "Text",
		UID:  ".../Titles.localized/Basic Text.localized/Text.localized/Text.moti",
	}
	fcpxml.Resources.Effects = append(fcpxml.Resources.Effects, textEffect)
}

// createTextTitle creates a Title struct for text overlay
func createTextTitle(text, duration, baseName string) fcp.Title {
	return fcp.Title{
		Ref:      "r6", // Reference to Text effect
		Lane:     "1",  // Lane 1 (above the video)
		Offset:   "0s",
		Name:     baseName + " - Text",
		Duration: duration,
		Start:    "86486400/24000s",
		Params: []fcp.Param{
			{Name: "Layout Method", Key: "9999/10003/13260/3296672360/2/314", Value: "1 (Paragraph)"},
			{Name: "Left Margin", Key: "9999/10003/13260/3296672360/2/323", Value: "-1730"},
			{Name: "Right Margin", Key: "9999/10003/13260/3296672360/2/324", Value: "1730"},
			{Name: "Top Margin", Key: "9999/10003/13260/3296672360/2/325", Value: "960"},
			{Name: "Bottom Margin", Key: "9999/10003/13260/3296672360/2/326", Value: "-960"},
			{Name: "Alignment", Key: "9999/10003/13260/3296672360/2/354/3296667315/401", Value: "1 (Center)"},
			{Name: "Line Spacing", Key: "9999/10003/13260/3296672360/2/354/3296667315/404", Value: "-19"},
			{Name: "Auto-Shrink", Key: "9999/10003/13260/3296672360/2/370", Value: "3 (To All Margins)"},
			{Name: "Alignment", Key: "9999/10003/13260/3296672360/2/373", Value: "0 (Left) 0 (Top)"},
			{Name: "Opacity", Key: "9999/10003/13260/3296672360/4/3296673134/1000/1044", Value: "0"},
			{Name: "Speed", Key: "9999/10003/13260/3296672360/4/3296673134/201/208", Value: "6 (Custom)"},
			{
				Name: "Custom Speed", 
				Key: "9999/10003/13260/3296672360/4/3296673134/201/209",
				KeyframeAnimation: &fcp.KeyframeAnimation{
					Keyframes: []fcp.Keyframe{
						{Time: "-469658744/1000000000s", Value: "0"},
						{Time: "12328542033/1000000000s", Value: "1"},
					},
				},
			},
			{Name: "Apply Speed", Key: "9999/10003/13260/3296672360/4/3296673134/201/211", Value: "2 (Per Object)"},
		},
		Text: &fcp.TitleText{
			TextStyle: fcp.TextStyleRef{
				Ref:  "ts1",
				Text: text,
			},
		},
		TextStyleDef: &fcp.TextStyleDef{
			ID: "ts1",
			TextStyle: fcp.TextStyle{
				Font:        "Helvetica Neue",
				FontSize:    "196",
				FontColor:   "1 1 1 1",
				Bold:        "1",
				Alignment:   "center",
				LineSpacing: "-19",
			},
		},
	}
}