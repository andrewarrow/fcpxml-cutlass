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

// generateUID creates a consistent UID from a video ID using MD5 hash
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

// ConvertSecondsToFCPDuration converts seconds to frame-aligned FCP duration
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

// WriteToFile marshals the FCPXML struct to a file
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

// AddVideo adds a video asset and asset-clip to the FCPXML structure
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
	resourceCount := len(fcpxml.Resources.Assets) + len(fcpxml.Resources.Formats) + len(fcpxml.Resources.Effects) + len(fcpxml.Resources.Media)
	assetID := fmt.Sprintf("r%d", resourceCount+1)
	formatID := fmt.Sprintf("r%d", resourceCount+2)

	// Generate bookmark (fallback to empty string if Swift unavailable)
	bookmark, _ := generateBookmark(absPath)

	// Create asset format (based on simple_video1.fcpxml)
	assetFormat := Format{
		ID:            formatID,
		FrameDuration: "20/600s",  // 1/30 second frame duration
		Width:         "1620",
		Height:        "1080",
		ColorSpace:    "1-1-1 (Rec. 709)",
	}

	// Use a default duration of 10 seconds, properly frame-aligned
	defaultDurationSeconds := 10.0
	frameDuration := ConvertSecondsToFCPDuration(defaultDurationSeconds)
	
	// Create asset
	asset := Asset{
		ID:            assetID,
		Name:          videoName,
		UID:           uid,
		Start:         "0s",
		Duration:      frameDuration,
		HasVideo:      "1",
		Format:        formatID,
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

	// Add resources
	fcpxml.Resources.Assets = append(fcpxml.Resources.Assets, asset)
	fcpxml.Resources.Formats = append(fcpxml.Resources.Formats, assetFormat)

	// Add asset-clip to the spine if there's a sequence
	if len(fcpxml.Library.Events) > 0 && len(fcpxml.Library.Events[0].Projects) > 0 && len(fcpxml.Library.Events[0].Projects[0].Sequences) > 0 {
		sequence := &fcpxml.Library.Events[0].Projects[0].Sequences[0]
		
		// Create asset-clip with frame-aligned duration
		clipDuration := ConvertSecondsToFCPDuration(defaultDurationSeconds)
		
		assetClip := AssetClip{
			Ref:       assetID,
			Offset:    "0s",
			Name:      videoName,
			Duration:  clipDuration,
			Format:    formatID,
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
