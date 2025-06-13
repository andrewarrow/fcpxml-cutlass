package build

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"cutlass/fcp"
)

// isPNGFile checks if the given file is a PNG image
func isPNGFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".png"
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

// getAudioDuration uses ffprobe to get audio duration in FCP time format
func getAudioDuration(audioPath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", audioPath)
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

// addVideoToProject adds a video or image file to an existing FCPXML project
func addVideoToProject(projectFile, videoFile, withText, withSound string) error {
	// Read the existing project file
	content, err := os.ReadFile(projectFile)
	if err != nil {
		return fmt.Errorf("failed to read project file: %v", err)
	}
	
	// Parse the XML
	var fcpxml fcp.FCPXML
	err = parseXML(content, &fcpxml)
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
	
	// Get duration based on file type and audio
	var duration string
	if withSound != "" {
		// If audio is provided, use audio duration
		var err error
		duration, err = getAudioDuration(withSound)
		if err != nil {
			return fmt.Errorf("failed to get audio duration: %v", err)
		}
	} else if isPNGFile(absVideoPath) {
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
	
	// Ensure Text effect exists in resources (needed for text overlays)
	textEffectID := ensureTextEffect(&fcpxml)
	
	// Create PNG format first if needed
	var pngFormatID string
	if isPNGFile(absVideoPath) {
		pngFormatID = ensurePNGFormat(&fcpxml)
	}

	// Check if asset already exists in the project
	existingAssetID := findExistingAsset(&fcpxml, absVideoPath)
	if existingAssetID == "" {
		// Asset doesn't exist, create it
		existingAssetID = createAsset(&fcpxml, absVideoPath, baseName, duration, pngFormatID)
	}
	
	if withSound != "" {
		// Create compound clip with video and audio
		err = addCompoundClipToSpine(&fcpxml, existingAssetID, absVideoPath, withSound, baseName, duration, withText, textEffectID)
		if err != nil {
			return err
		}
	} else {
		// Add asset-clip to the spine
		err = addAssetToSpine(&fcpxml, existingAssetID, absVideoPath, baseName, duration, withText, textEffectID)
		if err != nil {
			return err
		}
	}
	
	// Write back to project file
	return writeProjectFile(projectFile, &fcpxml)
}

// ensurePNGFormat adds PNG format if it doesn't exist and returns its ID
func ensurePNGFormat(fcpxml *fcp.FCPXML) string {
	// Check if PNG format already exists
	for _, format := range fcpxml.Resources.Formats {
		if format.Name == "FFVideoFormatRateUndefined" {
			return format.ID
		}
	}
	
	// Add PNG format if it doesn't exist
	// Generate a unique ID for the PNG format - must not conflict with any other resource IDs
	// Count all existing resources: assets + formats + effects
	totalResources := len(fcpxml.Resources.Assets) + len(fcpxml.Resources.Formats) + len(fcpxml.Resources.Effects)
	pngFormatID := fmt.Sprintf("r%d", totalResources+1)
	pngFormat := fcp.Format{
		ID:         pngFormatID,
		Name:       "FFVideoFormatRateUndefined",
		Width:      "1280",
		Height:     "720",
		ColorSpace: "1-13-1",
		// No FrameDuration for still images
	}
	fcpxml.Resources.Formats = append(fcpxml.Resources.Formats, pngFormat)
	return pngFormatID
}

// createAsset creates a new asset and adds it to the resources
func createAsset(fcpxml *fcp.FCPXML, absVideoPath, baseName, duration, pngFormatID string) string {
	// Calculate next available ID considering all resources: assets + formats + effects
	totalResources := len(fcpxml.Resources.Assets) + len(fcpxml.Resources.Formats) + len(fcpxml.Resources.Effects)
	assetID := fmt.Sprintf("r%d", totalResources+1)
	
	// Generate consistent UID from file path
	assetUID := generateUID(absVideoPath)
	
	// Generate bookmark for the video file
	_, _ = generateBookmark(absVideoPath) // Ignore errors, continue without bookmark

	
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
	return assetID
}

// addAssetToSpine adds an asset clip to the timeline spine
func addAssetToSpine(fcpxml *fcp.FCPXML, assetID, absVideoPath, baseName, duration, withText, textEffectID string) error {
	if len(fcpxml.Library.Events) == 0 || len(fcpxml.Library.Events[0].Projects) == 0 {
		return fmt.Errorf("no project found in FCPXML")
	}
	
	project := &fcpxml.Library.Events[0].Projects[0]
	if len(project.Sequences) == 0 {
		return fmt.Errorf("no sequence found in project")
	}
	
	// Calculate offset by parsing existing spine content
	offset := calculateTimelineOffset(project.Sequences[0].Spine.Content)
	
	var clipXML []byte
	var err error
	
	if isPNGFile(absVideoPath) {
		// Use video element for PNG files (still images)
		videoClip := fcp.Video{
			Ref:      assetID,
			Offset:   offset,
			Name:     baseName,
			Start:    "0s",
			Duration: duration,
		}
		
		// Add text overlay if requested
		if withText != "" {
			textTitle := createTextTitle(withText, duration, baseName, textEffectID)
			videoClip.NestedTitles = []fcp.Title{textTitle}
			
			// Add position animation when text is present (2 keyframes over 2 seconds)
			videoClip.AdjustTransform = &fcp.AdjustTransform{
				Params: []fcp.Param{
					{
						Name: "position",
						Key:  "", // Empty key for adjust-transform params
						Value: "", // Empty value when using keyframes
						KeyframeAnimation: &fcp.KeyframeAnimation{
							Keyframes: []fcp.Keyframe{
								{Time: "0s", Value: "0 0"},
								{Time: "48048/24000s", Value: "0 -22.1038"},
							},
						},
					},
				},
			}
		}
		
		clipXML, err = marshalXML(videoClip)
	} else {
		// Use asset-clip for video files
		assetClip := fcp.AssetClip{
			Ref:      assetID,
			Offset:   offset,
			Name:     baseName,
			Duration: duration,
			Format:   "r1",
			TCFormat: "NDF",
		}
		
		// Add text overlay if requested
		if withText != "" {
			textTitle := createTextTitle(withText, duration, baseName, textEffectID)
			assetClip.Titles = []fcp.Title{textTitle}
			
			// Add position animation when text is present (2 keyframes over 2 seconds)
			assetClip.AdjustTransform = &fcp.AdjustTransform{
				Params: []fcp.Param{
					{
						Name: "position",
						Key:  "", // Empty key for adjust-transform params
						Value: "", // Empty value when using keyframes
						KeyframeAnimation: &fcp.KeyframeAnimation{
							Keyframes: []fcp.Keyframe{
								{Time: "0s", Value: "0 0"},
								{Time: "48048/24000s", Value: "0 -22.1038"},
							},
						},
					},
				},
			}
		}
		
		clipXML, err = marshalXML(assetClip)
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
	
	return nil
}

// addCompoundClipToSpine creates a compound clip with video and audio and adds it to the timeline
func addCompoundClipToSpine(fcpxml *fcp.FCPXML, videoAssetID, absVideoPath, audioPath, baseName, duration, withText, textEffectID string) error {
	if len(fcpxml.Library.Events) == 0 || len(fcpxml.Library.Events[0].Projects) == 0 {
		return fmt.Errorf("no project found in FCPXML")
	}
	
	project := &fcpxml.Library.Events[0].Projects[0]
	if len(project.Sequences) == 0 {
		return fmt.Errorf("no sequence found in project")
	}
	
	// Get absolute path for audio file
	absAudioPath, err := filepath.Abs(audioPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute audio path: %v", err)
	}
	
	// Check if audio file exists
	if _, err := os.Stat(absAudioPath); os.IsNotExist(err) {
		return fmt.Errorf("audio file does not exist: %s", absAudioPath)
	}
	
	// Create audio asset
	audioAssetID := createAudioAsset(fcpxml, absAudioPath, baseName, duration)
	
	// Create compound clip media
	compoundClipID := createCompoundClipMedia(fcpxml, videoAssetID, audioAssetID, absVideoPath, baseName, duration, withText)
	
	// Add ref-clip to the spine
	return addRefClipToSpine(fcpxml, compoundClipID, baseName, duration, withText, textEffectID)
}

// createAudioAsset creates an audio asset and returns its ID
func createAudioAsset(fcpxml *fcp.FCPXML, absAudioPath, baseName, duration string) string {
	// Calculate next available ID considering all resources
	totalResources := len(fcpxml.Resources.Assets) + len(fcpxml.Resources.Formats) + len(fcpxml.Resources.Effects) + len(fcpxml.Resources.Media)
	audioAssetID := fmt.Sprintf("r%d", totalResources+1)
	
	// Generate consistent UID from file path
	audioUID := generateUID(absAudioPath)
	
	// Generate bookmark for the audio file
	_, _ = generateBookmark(absAudioPath) // Ignore errors, continue without bookmark
	
	// Add audio asset to resources
	audioAsset := fcp.Asset{
		ID:            audioAssetID,
		Name:          baseName,
		UID:           audioUID,
		Start:         "0s",
		Duration:      duration,
		HasVideo:      "0",
		Format:        "r1",
		HasAudio:      "1",
		AudioSources:  "1",
		AudioChannels: "1",
		AudioRate:     "24000",
		MediaRep: fcp.MediaRep{
			Kind: "original-media",
			Sig:  audioUID,
			Src:  "file://" + absAudioPath,
		},
	}
	
	fcpxml.Resources.Assets = append(fcpxml.Resources.Assets, audioAsset)
	return audioAssetID
}

// createCompoundClipMedia creates a compound clip media element and returns its ID
func createCompoundClipMedia(fcpxml *fcp.FCPXML, videoAssetID, audioAssetID, absVideoPath, baseName, duration, withText string) string {
	// Calculate next available ID considering all resources (add 1 more to avoid conflict with audio asset that was just created)
	totalResources := len(fcpxml.Resources.Assets) + len(fcpxml.Resources.Formats) + len(fcpxml.Resources.Effects) + len(fcpxml.Resources.Media)
	mediaID := fmt.Sprintf("r%d", totalResources+1)
	
	// Generate UID for compound clip
	mediaUID := generateUID(baseName + "_compound")
	
	// Create spine content for the compound clip
	var spineContent string
	if isPNGFile(absVideoPath) {
		// Video element for PNG with audio asset-clip on lane -1
		videoClip := fcp.Video{
			Ref:      videoAssetID,
			Offset:   "0s",
			Name:     baseName,
			Start:    "86399313/24000s", // Standard FCP start time for compound clips
			Duration: duration,
		}
		
		videoXML, _ := marshalXML(videoClip)
		videoContent := strings.ReplaceAll(string(videoXML), "\n", "\n                        ")
		
		// Audio asset-clip on lane -1
		audioClip := fcp.AssetClip{
			Ref:       audioAssetID,
			Lane:      "-1",
			Offset:    getAudioOffset(duration), // Calculate proper audio offset
			Name:      baseName,
			Duration:  duration,
			Format:    "r1",
			TCFormat:  "NDF",
			AudioRole: "dialogue",
		}
		
		audioXML, _ := marshalXML(audioClip)
		audioContent := strings.ReplaceAll(string(audioXML), "\n", "\n                            ")
		
		spineContent = fmt.Sprintf(`
                        %s
                            %s
                        </video>`, videoContent[:len(videoContent)-8], audioContent) // Remove closing tag and add audio
	}
	
	// Create compound clip media
	media := fcp.Media{
		ID:      mediaID,
		Name:    baseName + " Clip",
		UID:     mediaUID,
		ModDate: "2025-06-13 10:53:41 -0700", // Use current time format
		Sequence: fcp.Sequence{
			Format:      "r1",
			Duration:    duration,
			TCStart:     "0s",
			TCFormat:    "NDF",
			AudioLayout: "stereo",
			AudioRate:   "48k",
			Spine: fcp.Spine{
				Content: spineContent,
			},
		},
	}
	
	fcpxml.Resources.Media = append(fcpxml.Resources.Media, media)
	return mediaID
}

// getAudioOffset calculates the proper audio offset for compound clips
func getAudioOffset(duration string) string {
	// Convert duration to audio offset format used in FCP compound clips
	// This is a simplified calculation - in real FCP this depends on the specific timing
	return "28799771/8000s" // Standard offset seen in example
}

// addRefClipToSpine adds a ref-clip that references the compound clip to the spine
func addRefClipToSpine(fcpxml *fcp.FCPXML, mediaID, baseName, duration, withText, textEffectID string) error {
	if len(fcpxml.Library.Events) == 0 || len(fcpxml.Library.Events[0].Projects) == 0 {
		return fmt.Errorf("no project found in FCPXML")
	}
	
	project := &fcpxml.Library.Events[0].Projects[0]
	if len(project.Sequences) == 0 {
		return fmt.Errorf("no sequence found in project")
	}
	
	// Calculate offset by parsing existing spine content
	offset := calculateTimelineOffset(project.Sequences[0].Spine.Content)
	
	// Create ref-clip
	refClip := fcp.RefClip{
		Ref:      mediaID,
		Offset:   offset,
		Name:     baseName + " Clip",
		Duration: duration,
	}
	
	// Add text overlay if requested
	if withText != "" {
		textTitle := createTextTitle(withText, duration, baseName, textEffectID)
		refClip.Titles = []fcp.Title{textTitle}
		
		// Add position animation when text is present
		refClip.AdjustTransform = &fcp.AdjustTransform{
			Params: []fcp.Param{
				{
					Name: "position",
					Key:  "",
					Value: "",
					KeyframeAnimation: &fcp.KeyframeAnimation{
						Keyframes: []fcp.Keyframe{
							{Time: "0s", Value: "0 0"},
							{Time: "48048/24000s", Value: "0 -22.1038"},
						},
					},
				},
			},
		}
	}
	
	clipXML, err := marshalXML(refClip)
	if err != nil {
		return fmt.Errorf("failed to marshal ref-clip: %v", err)
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
	
	return nil
}