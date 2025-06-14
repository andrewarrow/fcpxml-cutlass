package api

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cutlass/build2/core"
	"cutlass/build2/timeline"
	"cutlass/fcp"
)

// ProjectBuilder provides high-level operations for FCPXML projects
type ProjectBuilder struct {
	projectFile string
	registry    *core.ResourceRegistry
	fcpxml      *fcp.FCPXML
	builder     *timeline.TimelineBuilder
	lastError   error
}

// NewProjectBuilder creates a new project builder
func NewProjectBuilder(projectFile string) (*ProjectBuilder, error) {
	pb := &ProjectBuilder{
		projectFile: projectFile,
	}
	
	// Load or create project
	err := pb.loadOrCreateProject()
	if err != nil {
		return nil, err
	}
	
	// Initialize registry and timeline builder
	pb.registry = core.NewResourceRegistry(pb.fcpxml)
	pb.builder = timeline.NewTimelineBuilder(pb.registry)
	
	return pb, nil
}

// loadOrCreateProject loads existing project or creates blank one
func (pb *ProjectBuilder) loadOrCreateProject() error {
	if _, err := os.Stat(pb.projectFile); os.IsNotExist(err) {
		// Create blank project
		return pb.createBlankProject()
	}
	
	// Load existing project
	content, err := os.ReadFile(pb.projectFile)
	if err != nil {
		return fmt.Errorf("failed to read project file: %v", err)
	}
	
	// Parse the XML
	var fcpxml fcp.FCPXML
	err = xml.Unmarshal(content, &fcpxml)
	if err != nil {
		return fmt.Errorf("failed to parse project file: %v", err)
	}
	
	pb.fcpxml = &fcpxml
	return nil
}

// createBlankProject creates a new blank FCPXML project
func (pb *ProjectBuilder) createBlankProject() error {
	// Read the empty.fcpxml template
	emptyContent, err := os.ReadFile("empty.fcpxml")
	if err != nil {
		return fmt.Errorf("failed to read empty.fcpxml: %v", err)
	}
	
	// Parse the XML
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
			baseName := strings.TrimSuffix(filepath.Base(pb.projectFile), filepath.Ext(pb.projectFile))
			fcpxml.Library.Events[0].Projects[0].Name = baseName
		}
	}
	
	pb.fcpxml = &fcpxml
	return nil
}

// AddClip adds a clip to the timeline using the fluent API
func (pb *ProjectBuilder) AddClip(config ClipConfig) *ProjectBuilder {
	err := pb.builder.AddClipWithConfig(timeline.ClipConfig{
		VideoFile:      config.VideoFile,
		AudioFile:      config.AudioFile,
		Text:           config.Text,
		CustomDuration: config.CustomDuration,
	})
	if err != nil {
		// Store error for later retrieval
		pb.lastError = err
	}
	
	return pb
}

// ClipConfig contains configuration for adding a clip
type ClipConfig struct {
	VideoFile      string
	AudioFile      string
	Text           string
	CustomDuration string // Optional: if provided, overrides automatic duration calculation
}

// getLastError returns the last error encountered
func (pb *ProjectBuilder) getLastError() error {
	return pb.lastError
}

// Save saves the project to file
func (pb *ProjectBuilder) Save() error {
	// Check for any stored errors
	if err := pb.getLastError(); err != nil {
		return err
	}
	
	// Build the timeline
	err := pb.builder.Build()
	if err != nil {
		return err
	}
	
	// Write to file
	return pb.writeProjectFile()
}

// writeProjectFile writes the FCPXML to file
func (pb *ProjectBuilder) writeProjectFile() error {
	// Generate the XML output with proper formatting
	output, err := xml.MarshalIndent(pb.fcpxml, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %v", err)
	}
	
	// Add XML declaration and DOCTYPE
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

` + string(output)
	
	// Write to output file
	err = os.WriteFile(pb.projectFile, []byte(xmlContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}
	
	return nil
}

// Legacy API compatibility

// CreateBlankProject creates a blank FCPXML project
func CreateBlankProject(filename string) error {
	pb, err := NewProjectBuilder(filename)
	if err != nil {
		return err
	}
	
	return pb.Save()
}

// AddVideoToProject adds a video to an existing project (legacy compatibility)
func AddVideoToProject(projectFile, videoFile, withText, withSound string) error {
	pb, err := NewProjectBuilder(projectFile)
	if err != nil {
		return err
	}
	
	return pb.AddClip(ClipConfig{
		VideoFile: videoFile,
		AudioFile: withSound,
		Text:      withText,
	}).Save()
}

// AddClipSafe adds a clip and returns error immediately (non-fluent API)
func (pb *ProjectBuilder) AddClipSafe(config ClipConfig) error {
	return pb.builder.AddClipWithConfig(timeline.ClipConfig{
		VideoFile:      config.VideoFile,
		AudioFile:      config.AudioFile,
		Text:           config.Text,
		CustomDuration: config.CustomDuration,
	})
}

// AddVideoOnlySafe adds a video-only clip (for separate track approach)
func (pb *ProjectBuilder) AddVideoOnlySafe(videoFile, text, customDuration string) error {
	return pb.builder.AddVideoOnly(videoFile, text, customDuration)
}

// AddAudioOnlySafe adds an audio-only clip on lane -1 (for separate track approach)
func (pb *ProjectBuilder) AddAudioOnlySafe(audioFile, offset string) error {
	return pb.builder.AddAudioOnly(audioFile, offset)
}

// AddVideoWithNestedAudioSafe adds a video clip with nested audio clip inside
func (pb *ProjectBuilder) AddVideoWithNestedAudioSafe(videoFile, audioFile, text, customDuration string) error {
	return pb.builder.AddVideoWithNestedAudio(videoFile, audioFile, text, customDuration)
}

// AddVideoWithNestedAudioWithDurationSafe adds a video clip with nested audio clip using specified audio duration
func (pb *ProjectBuilder) AddVideoWithNestedAudioWithDurationSafe(videoFile, audioFile, text, customDuration, audioDuration string) error {
	return pb.builder.AddVideoWithNestedAudioWithDuration(videoFile, audioFile, text, customDuration, audioDuration)
}