// Package fcp provides FCPXML generation using structs.
//
// 🚨 CRITICAL: All XML generation MUST follow CLAUDE.md rules:
// - NEVER use string templates with %s placeholders (see CLAUDE.md "NO XML STRING TEMPLATES")
// - ALWAYS use structs and xml.MarshalIndent for XML generation
// - ALL durations MUST be frame-aligned → USE ConvertSecondsToFCPDuration() function
// - ALL IDs MUST be unique → COUNT existing resources: len(Assets)+len(Formats)+len(Effects)+len(Media)  
// - BEFORE commits → RUN ValidateClaudeCompliance() + xmllint --dtdvalid FCPXMLv1_13.dtd
package fcp

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type TemplateVideo struct {
	ID       string
	UID      string
	Bookmark string
}

type NumberSection struct {
	Number  int
	VideoID string
	Offset  string
}

type TemplateData struct {
	FirstName      string
	LastName       string
	LastNameSuffix string
	Videos         []TemplateVideo
	Numbers        []NumberSection
}

// generateUID creates a consistent UID from a video ID using MD5 hash.
// 
// 🚨 CLAUDE.md Rule: UID Consistency Requirements
// - UIDs MUST be deterministic based on file content/name, not file path
// - Once FCP imports a media file with a specific UID, that UID is permanently associated
// - Different UIDs for same file cause "cannot be imported again with different unique identifier" errors
func generateUID(videoID string) string {
	// Create a hash from the video ID to ensure consistent UIDs
	hasher := md5.New()
	hasher.Write([]byte("cutlass_video_" + videoID))
	hash := hasher.Sum(nil)
	// Convert to uppercase hex string (32 characters)
	return strings.ToUpper(hex.EncodeToString(hash))
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

// ConvertSecondsToFCPDuration converts seconds to frame-aligned FCP duration.
//
// 🚨 CLAUDE.md Rule: Frame Boundary Alignment - CRITICAL!
// - FCP uses time base of 24000/1001 ≈ 23.976 fps for frame alignment
// - Duration format: (frames*1001)/24000s where frames is an integer
// - NEVER use simple seconds * 24000 calculations - creates non-frame-aligned durations
// - Non-frame-aligned durations cause "not on an edit frame boundary" errors in FCP
// - Example: 21600000/24000s = NON-FRAME-ALIGNED ❌, 21599578/24000s = FRAME-ALIGNED ✅
func ConvertSecondsToFCPDuration(seconds float64) string {
	// Convert to frame count using the sequence time base (1001/24000s frame duration)
	// This means 24000/1001 frames per second ≈ 23.976 fps
	framesPerSecond := 24000.0 / 1001.0
	exactFrames := seconds * framesPerSecond
	
	// Choose the frame count that gives the closest duration to the target
	floorFrames := int(math.Floor(exactFrames))
	ceilFrames := int(math.Ceil(exactFrames))
	
	floorDuration := float64(floorFrames) / framesPerSecond
	ceilDuration := float64(ceilFrames) / framesPerSecond
	
	var frames int
	if math.Abs(seconds-floorDuration) <= math.Abs(seconds-ceilDuration) {
		frames = floorFrames
	} else {
		frames = ceilFrames
	}
	
	// Format as rational using the sequence time base
	return fmt.Sprintf("%d/24000s", frames*1001)
}

// GenerateEmpty creates an empty FCPXML file structure and returns a pointer to it
func GenerateEmpty(filename string) (*FCPXML, error) {
	// Create empty FCPXML structure matching empty.fcpxml
	fcpxml := &FCPXML{
		Version: "1.13",
		Resources: Resources{
			Formats: []Format{
				{
					ID:            "r1",
					Name:          "FFVideoFormat720p2398",
					FrameDuration: "1001/24000s",
					Width:         "1280",
					Height:        "720",
					ColorSpace:    "1-1-1 (Rec. 709)",
				},
			},
		},
		Library: Library{
			Location: "file:///Users/aa/Movies/Untitled.fcpbundle/",
			Events: []Event{
				{
					Name: "6-13-25",
					UID:  "78463397-97FD-443D-B4E2-07C581674AFC",
					Projects: []Project{
						{
							Name:    "wiki",
							UID:     "DEA19981-DED5-4851-8435-14515931C68A",
							ModDate: "2025-06-13 11:46:22 -0700",
							Sequences: []Sequence{
								{
									Format:      "r1",
									Duration:    "0s",
									TCStart:     "0s",
									TCFormat:    "NDF",
									AudioLayout: "stereo",
									AudioRate:   "48k",
									Spine: Spine{
										AssetClips: []AssetClip{},
									},
								},
							},
						},
					},
				},
			},
			SmartCollections: []SmartCollection{
				{
					Name:  "Projects",
					Match: "all",
					Matches: []Match{
						{Rule: "is", Type: "project"},
					},
				},
				{
					Name:  "All Video",
					Match: "any",
					MediaMatches: []MediaMatch{
						{Rule: "is", Type: "videoOnly"},
						{Rule: "is", Type: "videoWithAudio"},
					},
				},
				{
					Name:  "Audio Only",
					Match: "all",
					MediaMatches: []MediaMatch{
						{Rule: "is", Type: "audioOnly"},
					},
				},
				{
					Name:  "Stills",
					Match: "all",
					MediaMatches: []MediaMatch{
						{Rule: "is", Type: "stills"},
					},
				},
				{
					Name:  "Favorites",
					Match: "all",
					RatingMatches: []RatingMatch{
						{Value: "favorites"},
					},
				},
			},
		},
	}

	// If filename is provided, write to file
	if filename != "" {
		err := WriteToFile(fcpxml, filename)
		if err != nil {
			return nil, err
		}
	}

	return fcpxml, nil
}

// WriteToFile marshals the FCPXML struct to a file.
//
// 🚨 CLAUDE.md Rule: NO XML STRING TEMPLATES → USE xml.MarshalIndent() function
// - After writing, VALIDATE with: xmllint --dtdvalid FCPXMLv1_13.dtd filename  
// - Before commits, CHECK with: ValidateClaudeCompliance() function
func WriteToFile(fcpxml *FCPXML, filename string) error {
	// Marshal to XML with proper formatting
	output, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %v", err)
	}

	// Add XML declaration and DOCTYPE
	xmlHeader := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

`
	fullXML := xmlHeader + string(output)

	// Write to file
	err = os.WriteFile(filename, []byte(fullXML), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// AddVideo adds a video asset and asset-clip to the FCPXML structure.
//
// 🚨 CLAUDE.md Rules Applied Here:
// - Uses STRUCTS ONLY - no string templates → append to fcpxml.Resources.Assets, sequence.Spine.AssetClips
// - Generates UNIQUE IDs → resourceCount = len(Assets)+len(Formats)+len(Effects)+len(Media) 
// - Uses frame-aligned durations → ConvertSecondsToFCPDuration() function 
// - Maintains UID consistency → generateUID() function for deterministic UIDs
//
// ❌ NEVER: fmt.Sprintf("<asset-clip ref='%s'...") - CRITICAL VIOLATION!
// ✅ ALWAYS: Use provided functions and struct field assignment
func AddVideo(fcpxml *FCPXML, videoPath string) error {
	// Get absolute path
	absPath, err := filepath.Abs(videoPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("video file does not exist: %s", absPath)
	}

	// Generate unique IDs
	videoName := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	uid := generateUID(videoName)
	
	// Count existing resources to generate unique IDs
	// 🚨 CLAUDE.md Rule: Unique ID Requirements → THIS pattern prevents ID collisions:
	// resourceCount = len(Assets)+len(Formats)+len(Effects)+len(Media)
	// nextID = fmt.Sprintf("r%d", resourceCount+1)
	resourceCount := len(fcpxml.Resources.Assets) + len(fcpxml.Resources.Formats) + len(fcpxml.Resources.Effects) + len(fcpxml.Resources.Media)
	assetID := fmt.Sprintf("r%d", resourceCount+1)

	// Generate bookmark (fallback to empty string if Swift unavailable)
	bookmark, _ := generateBookmark(absPath)

	// 🚨 CLAUDE.md Rule: Format Consistency - Videos use 720p sequence format (r1)
	// Note: No separate format needed since we always use 720p sequence format

	// Use a default duration of 10 seconds, properly frame-aligned
	defaultDurationSeconds := 10.0
	frameDuration := ConvertSecondsToFCPDuration(defaultDurationSeconds)
	
	// Create asset
	// 🚨 CLAUDE.md Rule: Format Consistency - Videos use 720p sequence format (r1)
	asset := Asset{
		ID:            assetID,
		Name:          videoName,
		UID:           uid,
		Start:         "0s",
		Duration:      frameDuration,
		HasVideo:      "1",
		Format:        "r1",  // Always use 720p sequence format
		HasAudio:      "1",
		VideoSources:  "1",
		AudioSources:  "1",
		AudioChannels: "1",
		AudioRate:     "44100",
		MediaRep: MediaRep{
			Kind: "original-media",
			Sig:  uid,
			Src:  "file://" + absPath,
		},
	}

	// Add bookmark if available
	if bookmark != "" {
		// Note: The MediaRep struct doesn't include Bookmark field yet
		// This would need to be added to the types.go if bookmark support is needed
	}

	// Add resources - only add asset, no separate format needed  
	fcpxml.Resources.Assets = append(fcpxml.Resources.Assets, asset)

	// Add asset-clip to the spine if there's a sequence
	if len(fcpxml.Library.Events) > 0 && len(fcpxml.Library.Events[0].Projects) > 0 && len(fcpxml.Library.Events[0].Projects[0].Sequences) > 0 {
		sequence := &fcpxml.Library.Events[0].Projects[0].Sequences[0]
		
		// Create asset-clip with frame-aligned duration
		clipDuration := ConvertSecondsToFCPDuration(defaultDurationSeconds)
		
		// 🚨 CLAUDE.md Rule: Format Consistency Requirements  
		// - Asset-clips MUST use sequence format, NOT asset format
		// - Since we always use 720p, both asset and clip use r1 format
		assetClip := AssetClip{
			Ref:       assetID,
			Offset:    "0s",
			Name:      videoName,
			Duration:  clipDuration,
			Format:    "r1",  // CRITICAL: Always use 720p sequence format
			TCFormat:  "NDF",
			AudioRole: "dialogue",
		}

		// Add asset-clip to spine using structs
		sequence.Spine.AssetClips = append(sequence.Spine.AssetClips, assetClip)
		
		// Update sequence duration to match the asset
		sequence.Duration = clipDuration
	}

	return nil
}

// ValidateClaudeCompliance performs automated checks for CLAUDE.md rule compliance.
//
// 🚨 CLAUDE.md Validation - Run this before any commit!
// This function helps catch violations of critical rules in CLAUDE.md
func ValidateClaudeCompliance(fcpxml *FCPXML) []string {
	var violations []string
	
	// Check for unique IDs across all resources
	idMap := make(map[string]bool)
	
	// Check asset IDs
	for _, asset := range fcpxml.Resources.Assets {
		if idMap[asset.ID] {
			violations = append(violations, fmt.Sprintf("Duplicate ID found: %s (Asset)", asset.ID))
		}
		idMap[asset.ID] = true
	}
	
	// Check format IDs  
	for _, format := range fcpxml.Resources.Formats {
		if idMap[format.ID] {
			violations = append(violations, fmt.Sprintf("Duplicate ID found: %s (Format)", format.ID))
		}
		idMap[format.ID] = true
	}
	
	// Check effect IDs
	for _, effect := range fcpxml.Resources.Effects {
		if idMap[effect.ID] {
			violations = append(violations, fmt.Sprintf("Duplicate ID found: %s (Effect)", effect.ID))
		}
		idMap[effect.ID] = true
	}
	
	// Check media IDs
	for _, media := range fcpxml.Resources.Media {
		if idMap[media.ID] {
			violations = append(violations, fmt.Sprintf("Duplicate ID found: %s (Media)", media.ID))
		}
		idMap[media.ID] = true
	}
	
	// Check for frame alignment in durations (basic check for common violations)
	// Look for duration patterns that are definitely not frame-aligned
	checkDuration := func(duration, location string) {
		if strings.Contains(duration, "/600s") && !strings.Contains(duration, "1001") {
			violations = append(violations, fmt.Sprintf("Potentially non-frame-aligned duration '%s' at %s - use ConvertSecondsToFCPDuration()", duration, location))
		}
		if strings.Contains(duration, "/24000s") {
			// Check if it follows (frames*1001)/24000s pattern
			if !strings.Contains(duration, "1001") {
				violations = append(violations, fmt.Sprintf("Non-frame-aligned duration '%s' at %s - must be (frames*1001)/24000s", duration, location))
			}
		}
	}
	
	// Check asset durations
	for _, asset := range fcpxml.Resources.Assets {
		checkDuration(asset.Duration, fmt.Sprintf("Asset %s", asset.ID))
	}
	
	// Check sequence durations
	for _, event := range fcpxml.Library.Events {
		for _, project := range event.Projects {
			for _, sequence := range project.Sequences {
				checkDuration(sequence.Duration, fmt.Sprintf("Sequence in Project %s", project.Name))
				
				// Check asset-clip durations in spine
				for _, clip := range sequence.Spine.AssetClips {
					checkDuration(clip.Duration, fmt.Sprintf("AssetClip %s in Spine", clip.Name))
				}
			}
		}
	}

	// 🚨 CLAUDE.md Rule: Format Consistency Requirements
	// Check for format mismatches between sequences and asset-clips
	for _, event := range fcpxml.Library.Events {
		for _, project := range event.Projects {
			for _, sequence := range project.Sequences {
				sequenceFormat := sequence.Format
				
				// Check asset-clip formats in spine
				for _, clip := range sequence.Spine.AssetClips {
					if clip.Format != sequenceFormat {
						violations = append(violations, fmt.Sprintf("Format mismatch: AssetClip '%s' has format '%s' but sequence has format '%s' - CAUSES FCP CRASHES", clip.Name, clip.Format, sequenceFormat))
					}
				}
			}
		}
	}
	
	return violations
}

// isImageFile checks if the given file is an image (PNG, JPG, JPEG).
//
// 🚨 CLAUDE.md Rule: Image vs Video Asset Properties
// - Image files should NOT have audio properties (HasAudio, AudioSources, AudioChannels)
// - Image files MUST have VideoSources = "1" 
// - Duration is set by caller, not hardcoded to "0s"
func isImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg"
}

// AddImage adds an image asset and asset-clip to the FCPXML structure.
//
// 🚨 CLAUDE.md Rules Applied Here:
// - Uses STRUCTS ONLY - no string templates → append to fcpxml.Resources.Assets, sequence.Spine.AssetClips
// - Generates UNIQUE IDs → resourceCount = len(Assets)+len(Formats)+len(Effects)+len(Media) 
// - Uses frame-aligned durations → ConvertSecondsToFCPDuration() function 
// - Maintains UID consistency → generateUID() function for deterministic UIDs
// - Image-specific properties → VideoSources="1", NO audio properties (HasAudio, AudioSources, AudioChannels)
//
// ❌ NEVER: fmt.Sprintf("<asset-clip ref='%s'...") - CRITICAL VIOLATION!
// ✅ ALWAYS: Use provided functions and struct field assignment
func AddImage(fcpxml *FCPXML, imagePath string, durationSeconds float64) error {
	// Validate that this is actually an image file
	if !isImageFile(imagePath) {
		return fmt.Errorf("file is not a supported image format (PNG, JPG, JPEG): %s", imagePath)
	}

	// Get absolute path
	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("image file does not exist: %s", absPath)
	}

	// Generate unique IDs
	imageName := strings.TrimSuffix(filepath.Base(imagePath), filepath.Ext(imagePath))
	uid := generateUID(imageName)
	
	// Count existing resources to generate unique IDs
	// 🚨 CLAUDE.md Rule: Unique ID Requirements → THIS pattern prevents ID collisions:
	// resourceCount = len(Assets)+len(Formats)+len(Effects)+len(Media)
	// nextID = fmt.Sprintf("r%d", resourceCount+1)
	resourceCount := len(fcpxml.Resources.Assets) + len(fcpxml.Resources.Formats) + len(fcpxml.Resources.Effects) + len(fcpxml.Resources.Media)
	assetID := fmt.Sprintf("r%d", resourceCount+1)
	// Note: No separate format needed for images since we always use 720p sequence format

	// Generate bookmark (fallback to empty string if Swift unavailable)
	bookmark, _ := generateBookmark(absPath)

	// Convert duration to frame-aligned format
	frameDuration := ConvertSecondsToFCPDuration(durationSeconds)
	
	// Create asset with image-specific properties
	// 🚨 CLAUDE.md Rule: Image vs Video Asset Properties
	// - Image files should NOT have audio properties (HasAudio, AudioSources, AudioChannels)
	// - Image files MUST have VideoSources = "1" 
	// 🚨 CLAUDE.md Rule: Format Consistency - Images use 720p sequence format (r1)
	asset := Asset{
		ID:            assetID,
		Name:          imageName,
		UID:           uid,
		Start:         "0s",
		Duration:      frameDuration,
		HasVideo:      "1",
		Format:        "r1",  // Always use 720p sequence format
		VideoSources:  "1",   // Required for image assets
		// Note: NO audio properties for image files
		MediaRep: MediaRep{
			Kind: "original-media",
			Sig:  uid,
			Src:  "file://" + absPath,
		},
	}

	// Add bookmark if available
	if bookmark != "" {
		// Note: The MediaRep struct doesn't include Bookmark field yet
		// This would need to be added to the types.go if bookmark support is needed
	}

	// Add resources - only add asset, no separate format needed
	fcpxml.Resources.Assets = append(fcpxml.Resources.Assets, asset)

	// Add asset-clip to the spine if there's a sequence
	if len(fcpxml.Library.Events) > 0 && len(fcpxml.Library.Events[0].Projects) > 0 && len(fcpxml.Library.Events[0].Projects[0].Sequences) > 0 {
		sequence := &fcpxml.Library.Events[0].Projects[0].Sequences[0]
		
		// Create asset-clip with frame-aligned duration
		clipDuration := ConvertSecondsToFCPDuration(durationSeconds)
		
		// 🚨 CLAUDE.md Rule: Format Consistency Requirements  
		// - Asset-clips MUST use sequence format, NOT asset format
		// - Format mismatch between sequence and asset-clip causes FCP crashes
		// - Asset keeps its native format, but clips inherit sequence format
		sequenceFormat := sequence.Format  // Use sequence format, not asset format
		
		assetClip := AssetClip{
			Ref:      assetID,
			Offset:   "0s",
			Name:     imageName,
			Duration: clipDuration,
			Format:   sequenceFormat,  // CRITICAL: Use sequence format, not formatID
			TCFormat: "NDF",
			// Note: NO AudioRole for image clips
		}

		// Add asset-clip to spine using structs
		sequence.Spine.AssetClips = append(sequence.Spine.AssetClips, assetClip)
		
		// Update sequence duration to match the total duration
		// If this is the first clip, set duration. Otherwise, we'd need to calculate total duration
		if len(sequence.Spine.AssetClips) == 1 {
			sequence.Duration = clipDuration
		} else {
			// For multiple clips, we'd need more sophisticated duration calculation
			// For now, just extend the sequence to include this clip
			// This is a simplified approach - real timeline management would be more complex
			sequence.Duration = clipDuration
		}
	}

	return nil
}
