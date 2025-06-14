package timeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cutlass/build2/clips"
	"cutlass/build2/core"
	"cutlass/build2/utils"
	"cutlass/fcp"
)

// ClipConfig represents clip configuration (to avoid circular import)
type ClipConfig struct {
	VideoFile      string
	AudioFile      string
	Text           string
	CustomDuration string
}

// TimelineBuilder provides fluent API for timeline construction
type TimelineBuilder struct {
	registry    *core.ResourceRegistry
	strategy    clips.ClipStrategy
	fcpxml      *fcp.FCPXML
	elements    []clips.TimelineElement
	totalDuration string
}

// NewTimelineBuilder creates a new timeline builder
func NewTimelineBuilder(registry *core.ResourceRegistry) *TimelineBuilder {
	return &TimelineBuilder{
		registry:      registry,
		strategy:      clips.NewSmartClipStrategy(),
		fcpxml:        registry.GetFCPXML(),
		elements:      make([]clips.TimelineElement, 0),
		totalDuration: "0s",
	}
}

// AddClip adds a clip to the timeline
func (tb *TimelineBuilder) AddClip(videoFile, audioFile, text string) error {
	return tb.AddClipWithConfig(ClipConfig{
		VideoFile: videoFile,
		AudioFile: audioFile,
		Text:      text,
	})
}

// AddClipWithConfig adds a clip to the timeline using a ClipConfig
func (tb *TimelineBuilder) AddClipWithConfig(config ClipConfig) error {
	// Get absolute path for video file
	absVideoPath, err := filepath.Abs(config.VideoFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}
	
	// Check if video file exists
	if _, err := os.Stat(absVideoPath); os.IsNotExist(err) {
		return fmt.Errorf("video file does not exist: %s", absVideoPath)
	}
	
	// Get base name
	baseName := strings.TrimSuffix(filepath.Base(config.VideoFile), filepath.Ext(config.VideoFile))
	
	// Calculate duration - use custom duration if provided
	var duration string
	if config.CustomDuration != "" {
		duration = config.CustomDuration
	} else {
		duration, err = tb.calculateDuration(absVideoPath, config.AudioFile)
		if err != nil {
			return fmt.Errorf("failed to calculate duration: %v", err)
		}
	}
	
	// Calculate offset by parsing existing timeline content
	offset := tb.calculateTimelineOffset()
	
	// Create transaction for atomic resource creation
	tx := core.NewTransaction(tb.registry)
	
	// Ensure required formats and effects exist
	pngFormatID := tb.ensurePNGFormat(tx)
	textEffectID := tb.ensureTextEffect(tx)
	
	// Create video asset
	videoAssetID, err := tb.createVideoAsset(tx, absVideoPath, baseName, duration, pngFormatID)
	if err != nil {
		tx.Rollback()
		return err
	}
	
	var audioAssetID, mediaID string
	
	// Create audio asset if provided
	if config.AudioFile != "" {
		audioAssetID, err = tb.createAudioAsset(tx, config.AudioFile, baseName, duration)
		if err != nil {
			tx.Rollback()
			return err
		}
		
		// Create compound clip media if needed
		if tb.strategy.ShouldCreateCompoundClip(absVideoPath, config.AudioFile, clips.TimelineContext{}) {
			mediaID, err = tb.createCompoundClipMedia(tx, baseName, duration, videoAssetID, audioAssetID)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	
	// Commit all resources
	err = tx.Commit()
	if err != nil {
		return err
	}
	
	// Create timeline element
	clipConfig := clips.ClipConfig{
		VideoAssetID: videoAssetID,
		AudioAssetID: audioAssetID,
		MediaID:      mediaID,
		TextEffectID: textEffectID,
		BaseName:     baseName,
		Duration:     duration,
		Offset:       offset,
		Text:         config.Text,
	}
	
	element := tb.strategy.CreateOptimalClip(absVideoPath, config.AudioFile, config.Text, clipConfig)
	tb.elements = append(tb.elements, element)
	
	// Update total duration to include existing content plus new clip
	tb.updateTotalDuration(offset, duration)
	
	return nil
}

// AddVideoOnly adds a video-only clip to the timeline (for separate video track approach)
func (tb *TimelineBuilder) AddVideoOnly(videoFile, text, customDuration string) error {
	return tb.AddClipWithConfig(ClipConfig{
		VideoFile:      videoFile,
		AudioFile:      "", // No audio
		Text:           text,
		CustomDuration: customDuration,
	})
}

// AddVideoWithNestedAudio adds a video clip with nested audio clip inside (Info.fcpxml approach)
func (tb *TimelineBuilder) AddVideoWithNestedAudio(videoFile, audioFile, text, customDuration string) error {
	return tb.AddClipWithConfig(ClipConfig{
		VideoFile:      videoFile,
		AudioFile:      audioFile, // This will create nested audio
		Text:           text,
		CustomDuration: customDuration,
	})
}

// AddAudioOnly adds an audio-only clip to the timeline on lane -1 (for separate audio track approach)
func (tb *TimelineBuilder) AddAudioOnly(audioFile string, offset string) error {
	// Get absolute path for audio file
	absAudioPath, err := filepath.Abs(audioFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute audio path: %v", err)
	}
	
	// Check if audio file exists
	if _, err := os.Stat(absAudioPath); os.IsNotExist(err) {
		return fmt.Errorf("audio file does not exist: %s", absAudioPath)
	}
	
	// Get base name and audio duration
	baseName := strings.TrimSuffix(filepath.Base(audioFile), filepath.Ext(audioFile))
	duration, err := utils.GetAudioDuration(audioFile)
	if err != nil {
		return fmt.Errorf("failed to get audio duration: %v", err)
	}
	
	// Create transaction for atomic resource creation
	tx := core.NewTransaction(tb.registry)
	
	// Create audio asset
	audioAssetID, err := tb.createAudioAsset(tx, audioFile, baseName, duration)
	if err != nil {
		tx.Rollback()
		return err
	}
	
	// Commit resources
	err = tx.Commit()
	if err != nil {
		return err
	}
	
	// Create audio-only timeline element
	audioElement := &clips.AudioOnlyElement{
		AudioRef: audioAssetID,
		Offset:   offset,
		Name:     baseName,
		Duration: duration,
	}
	
	tb.elements = append(tb.elements, audioElement)
	
	// Update total duration
	tb.updateTotalDuration(offset, duration)
	
	return nil
}

// Build finalizes the timeline and updates the FCPXML
func (tb *TimelineBuilder) Build() error {
	if len(tb.fcpxml.Library.Events) == 0 || len(tb.fcpxml.Library.Events[0].Projects) == 0 {
		return fmt.Errorf("no project found in FCPXML")
	}
	
	project := &tb.fcpxml.Library.Events[0].Projects[0]
	if len(project.Sequences) == 0 {
		return fmt.Errorf("no sequence found in project")
	}
	
	// Get existing spine content
	existingContent := strings.TrimSpace(project.Sequences[0].Spine.Content)
	
	// Build new spine content from all new elements
	var newElementsContent strings.Builder
	
	for _, element := range tb.elements {
		if newElementsContent.Len() > 0 {
			newElementsContent.WriteString("\n                        ")
		}
		
		// Indent the XML properly
		xml := element.GetXML()
		indentedXML := strings.ReplaceAll(xml, "\n", "\n                        ")
		newElementsContent.WriteString(indentedXML)
	}
	
	// Combine existing and new content
	var finalContent strings.Builder
	
	if existingContent != "" {
		finalContent.WriteString("\n                        ")
		finalContent.WriteString(existingContent)
		
		if newElementsContent.Len() > 0 {
			finalContent.WriteString("\n                        ")
			finalContent.WriteString(newElementsContent.String())
		}
		
		finalContent.WriteString("\n                    ")
	} else if newElementsContent.Len() > 0 {
		finalContent.WriteString("\n                        ")
		finalContent.WriteString(newElementsContent.String())
		finalContent.WriteString("\n                    ")
	}
	
	// Update sequence content and duration
	project.Sequences[0].Spine.Content = finalContent.String()
	project.Sequences[0].Duration = tb.totalDuration
	
	return nil
}

// Helper methods

func (tb *TimelineBuilder) calculateDuration(videoPath, audioPath string) (string, error) {
	if audioPath != "" {
		// Use audio duration if audio is provided
		return utils.GetAudioDuration(audioPath)
	} else if isImageFile(videoPath) {
		// Image files default to 10 seconds
		return "240240/24000s", nil // 10 seconds at 23.976fps
	} else {
		// Get video duration
		duration, err := utils.GetVideoDuration(videoPath)
		if err != nil {
			return "240240/24000s", nil // Default to 10 seconds if detection fails
		}
		return duration, nil
	}
}

func (tb *TimelineBuilder) ensurePNGFormat(tx *core.ResourceTransaction) string {
	// Check if PNG format already exists
	for _, format := range tb.fcpxml.Resources.Formats {
		if format.Name == "FFVideoFormatRateUndefined" {
			return format.ID
		}
	}
	
	// Create PNG format
	id := tx.ReserveIDs(1)[0]
	format, _ := tx.CreateFormat(id, "FFVideoFormatRateUndefined", "1280", "720", "1-13-1")
	return format.ID
}

func (tb *TimelineBuilder) ensureTextEffect(tx *core.ResourceTransaction) string {
	// Check if Text effect already exists
	for _, effect := range tb.fcpxml.Resources.Effects {
		if effect.Name == "Text" {
			return effect.ID
		}
	}
	
	// Create Text effect
	id := tx.ReserveIDs(1)[0]
	effect, _ := tx.CreateEffect(id, "Text", ".../Titles.localized/Basic Text.localized/Text.localized/Text.moti")
	return effect.ID
}

func (tb *TimelineBuilder) createVideoAsset(tx *core.ResourceTransaction, videoPath, baseName, duration, formatID string) (string, error) {
	// Check if asset already exists
	if asset, exists := tb.registry.GetOrCreateAsset(videoPath); exists {
		return asset.ID, nil
	}
	
	// Create new asset
	id := tx.ReserveIDs(1)[0]
	asset, err := tx.CreateAsset(id, videoPath, baseName, duration, formatID)
	if err != nil {
		return "", err
	}
	
	return asset.ID, nil
}

func (tb *TimelineBuilder) createAudioAsset(tx *core.ResourceTransaction, audioPath, baseName, duration string) (string, error) {
	// Get absolute path for audio file
	absAudioPath, err := filepath.Abs(audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute audio path: %v", err)
	}
	
	// Check if audio file exists
	if _, err := os.Stat(absAudioPath); os.IsNotExist(err) {
		return "", fmt.Errorf("audio file does not exist: %s", absAudioPath)
	}
	
	// Check if asset already exists
	if asset, exists := tb.registry.GetOrCreateAsset(absAudioPath); exists {
		return asset.ID, nil
	}
	
	// Create new audio asset
	id := tx.ReserveIDs(1)[0]
	
	// Generate consistent UID
	uid := tb.registry.GenerateConsistentUID(filepath.Base(audioPath))
	
	// Create audio asset manually (transaction doesn't have specific audio asset method)
	audioAsset := &fcp.Asset{
		ID:            id,
		Name:          baseName,
		UID:           uid,
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
			Sig:  uid,
			Src:  "file://" + absAudioPath,
		},
	}
	
	tb.registry.RegisterAsset(audioAsset)
	return audioAsset.ID, nil
}

func (tb *TimelineBuilder) createCompoundClipMedia(tx *core.ResourceTransaction, baseName, duration, videoAssetID, audioAssetID string) (string, error) {
	// Create compound clip media
	id := tx.ReserveIDs(1)[0]
	media, err := tx.CreateCompoundClipMedia(id, baseName, duration, videoAssetID, audioAssetID)
	if err != nil {
		return "", err
	}
	
	return media.ID, nil
}

func (tb *TimelineBuilder) updateTotalDuration(offset, newClipDuration string) {
	// Calculate the end time of the new clip
	offsetFrames := parseDurationToFrames(offset)
	newClipFrames := parseDurationToFrames(newClipDuration)
	endFrames := offsetFrames + newClipFrames
	
	// Update total duration to be the end time of the latest clip
	tb.totalDuration = fmt.Sprintf("%d/24000s", endFrames)
}

// parseDurationToFrames converts a duration string to frame count
func parseDurationToFrames(duration string) int {
	if duration == "0s" {
		return 0
	}
	
	if strings.HasSuffix(duration, "/24000s") {
		framesStr := strings.TrimSuffix(duration, "/24000s")
		if frames, err := strconv.Atoi(framesStr); err == nil {
			return frames
		}
	}
	
	return 0
}

// Helper functions from original build package

func isPNGFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".png"
}

func isImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg"
}

// getVideoDuration and getAudioDuration are imported from utils package

// calculateTimelineOffset calculates where the next clip should start based on existing elements
func (tb *TimelineBuilder) calculateTimelineOffset() string {
	// First check existing spine content for any pre-existing clips
	existingOffset := "0s"
	if len(tb.fcpxml.Library.Events) > 0 && len(tb.fcpxml.Library.Events[0].Projects) > 0 {
		project := &tb.fcpxml.Library.Events[0].Projects[0]
		if len(project.Sequences) > 0 {
			spineContent := strings.TrimSpace(project.Sequences[0].Spine.Content)
			if spineContent != "" {
				existingOffset = tb.calculateTotalDurationFromSpine(spineContent)
			}
		}
	}
	
	// Calculate total duration of elements already added to this builder
	existingFrames := parseDurationToFrames(existingOffset)
	
	// Add duration of all elements in memory
	for _, element := range tb.elements {
		elementFrames := parseDurationToFrames(element.GetDuration())
		existingFrames += elementFrames
	}
	
	// Return the total offset for the next clip
	return fmt.Sprintf("%d/24000s", existingFrames)
}

// calculateTotalDurationFromSpine parses spine content and calculates the total timeline duration
func (tb *TimelineBuilder) calculateTotalDurationFromSpine(spineContent string) string {
	if strings.TrimSpace(spineContent) == "" {
		return "0s"
	}
	
	// Find all duration values in asset-clips, video elements, and ref-clips
	totalFrames := 0
	
	// Use regex to find all duration attributes more precisely
	lines := strings.Split(spineContent, "\n")
	for _, line := range lines {
		// Look for asset-clip, video, or ref-clip elements with duration
		if (strings.Contains(line, "asset-clip") || strings.Contains(line, "<video") || strings.Contains(line, "ref-clip")) && strings.Contains(line, "duration=") {
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