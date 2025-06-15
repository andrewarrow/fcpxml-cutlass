package fcp

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAddTextFromFile tests the AddTextFromFile function with various scenarios
func TestAddTextFromFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create test text file
	testTextFile := filepath.Join(tempDir, "test_text.txt")
	testTextContent := `First Text Line
Second Text Line
Third Text Line`
	
	err := os.WriteFile(testTextFile, []byte(testTextContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test text file: %v", err)
	}

	// Create base FCPXML with a video element (similar to samples/png.fcpxml)
	baseFCPXML := &FCPXML{
		Version: "1.13",
		Resources: Resources{
			Assets: []Asset{
				{
					ID:           "r2",
					Name:         "test_image",
					UID:          "TEST123456789",
					Start:        "0s",
					Duration:     "0s",
					HasVideo:     "1",
					Format:       "r3",
					VideoSources: "1",
					MediaRep: MediaRep{
						Kind: "original-media",
						Sig:  "TEST123456789",
						Src:  "file:///test/image.png",
					},
				},
			},
			Formats: []Format{
				{
					ID:            "r1",
					Name:          "FFVideoFormat720p2398",
					FrameDuration: "1001/24000s",
					Width:         "1280",
					Height:        "720",
					ColorSpace:    "1-1-1 (Rec. 709)",
				},
				{
					ID:         "r3",
					Name:       "FFVideoFormatRateUndefined",
					Width:      "1280",
					Height:     "800",
					ColorSpace: "1-13-1",
				},
			},
		},
		Library: Library{
			Location: "file:///Users/test/Movies/Test.fcpbundle/",
			Events: []Event{
				{
					Name: "Test Event",
					UID:  "TEST-EVENT-UID",
					Projects: []Project{
						{
							Name:    "Test Project",
							UID:     "TEST-PROJECT-UID",
							ModDate: "2025-06-15 12:00:00 -0700",
							Sequences: []Sequence{
								{
									Format:      "r1",
									Duration:    "241241/24000s",
									TCStart:     "0s",
									TCFormat:    "NDF",
									AudioLayout: "stereo",
									AudioRate:   "48k",
									Spine: Spine{
										Videos: []Video{
											{
												Ref:      "r2",
												Offset:   "0s",
												Name:     "test_image",
												Start:    "86399313/24000s",
												Duration: "241241/24000s",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Test AddTextFromFile
	err = AddTextFromFile(baseFCPXML, testTextFile, 1.0)
	if err != nil {
		t.Fatalf("AddTextFromFile failed: %v", err)
	}

	// Verify the structure was modified correctly
	sequence := &baseFCPXML.Library.Events[0].Projects[0].Sequences[0]
	video := &sequence.Spine.Videos[0]

	// Test 1: Verify slide animation was added
	if video.AdjustTransform == nil {
		t.Error("Expected slide animation (AdjustTransform) to be added to video element")
	} else {
		// Check that position animation exists
		hasPositionParam := false
		for _, param := range video.AdjustTransform.Params {
			if param.Name == "position" && param.KeyframeAnimation != nil {
				hasPositionParam = true
				// Verify keyframes
				if len(param.KeyframeAnimation.Keyframes) != 2 {
					t.Errorf("Expected 2 position keyframes, got %d", len(param.KeyframeAnimation.Keyframes))
				}
				// Check slide values
				if param.KeyframeAnimation.Keyframes[1].Value != "51.3109 0" {
					t.Errorf("Expected slide end position '51.3109 0', got '%s'", param.KeyframeAnimation.Keyframes[1].Value)
				}
			}
		}
		if !hasPositionParam {
			t.Error("Expected position parameter with keyframe animation in slide animation")
		}
	}

	// Test 2: Verify text elements were added as nested titles
	if len(video.NestedTitles) != 3 {
		t.Errorf("Expected 3 nested title elements, got %d", len(video.NestedTitles))
	}

	// Test 3: Verify text content matches input
	expectedTexts := []string{"First Text Line", "Second Text Line", "Third Text Line"}
	for i, title := range video.NestedTitles {
		if title.Text == nil || title.Text.TextStyle.Text != expectedTexts[i] {
			t.Errorf("Expected text '%s' at index %d, got '%s'", expectedTexts[i], i, getTextContent(title))
		}
	}

	// Test 4: Verify lane assignments (descending order)
	expectedLanes := []string{"3", "2", "1"}
	for i, title := range video.NestedTitles {
		if title.Lane != expectedLanes[i] {
			t.Errorf("Expected lane '%s' at index %d, got '%s'", expectedLanes[i], i, title.Lane)
		}
	}

	// Test 5: Verify staggered timing (1 second intervals)
	videoStart := 86399313 // frames from the video start time
	for i, title := range video.NestedTitles {
		expectedOffset := videoStart + 24024*(i+1) // 24024 frames = 1 second
		actualOffsetStr := title.Offset
		actualOffset := parseFCPDuration(actualOffsetStr)
		
		if actualOffset != expectedOffset {
			t.Errorf("Expected offset %d frames at index %d, got %d frames (%s)", expectedOffset, i, actualOffset, actualOffsetStr)
		}
	}

	// Test 6: Verify Y position offsets (300px increments)
	for i, title := range video.NestedTitles {
		if i == 0 {
			// First element should have no Position parameter (defaults to 0,0)
			hasPosition := false
			for _, param := range title.Params {
				if param.Name == "Position" {
					hasPosition = true
					break
				}
			}
			if hasPosition {
				t.Error("First text element should not have Position parameter (defaults to 0,0)")
			}
		} else {
			// Subsequent elements should have Position parameter with -300px increments
			expectedY := -300 * i
			actualValue := getPositionValue(title)
			if actualValue == "" {
				t.Errorf("Expected Position parameter for text element %d", i)
			} else {
				parts := strings.Fields(actualValue)
				if len(parts) >= 2 {
					actualY := parts[1]
					expectedYStr := fmt.Sprintf("%d", expectedY)
					if actualY != expectedYStr {
						t.Errorf("Expected Y position '%d' for element %d, got '%s'", expectedY, i, actualY)
					}
				}
			}
		}
	}

	// Test 7: Verify video duration was extended
	extendedDuration := parseFCPDuration(video.Duration)
	originalDuration := 241241
	if extendedDuration <= originalDuration {
		t.Errorf("Expected video duration to be extended beyond %d frames, got %d frames", originalDuration, extendedDuration)
	}

	// Test 8: Verify text effect was added to resources
	hasTextEffect := false
	for _, effect := range baseFCPXML.Resources.Effects {
		if effect.Name == "Text" && strings.Contains(effect.UID, "Text.moti") {
			hasTextEffect = true
			break
		}
	}
	if !hasTextEffect {
		t.Error("Expected Text effect to be added to resources")
	}

	// Test 9: Verify unique text-style-def IDs
	styleIDs := make(map[string]bool)
	for _, title := range video.NestedTitles {
		if title.TextStyleDef != nil {
			if styleIDs[title.TextStyleDef.ID] {
				t.Errorf("Duplicate text-style-def ID found: %s", title.TextStyleDef.ID)
			}
			styleIDs[title.TextStyleDef.ID] = true
		}
	}

	// Test 10: Verify XML marshaling works without errors
	_, err = xml.MarshalIndent(baseFCPXML, "", "    ")
	if err != nil {
		t.Errorf("Failed to marshal FCPXML to XML: %v", err)
	}
}

// TestAddTextFromFileErrorCases tests error handling
func TestAddTextFromFileErrorCases(t *testing.T) {
	baseFCPXML := &FCPXML{
		Version: "1.13",
		Library: Library{
			Events: []Event{
				{
					Projects: []Project{
						{
							Sequences: []Sequence{
								{
									Spine: Spine{
										Videos: []Video{}, // No videos
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Test 1: Non-existent file
	err := AddTextFromFile(baseFCPXML, "/non/existent/file.txt", 1.0)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test 2: No video element in spine
	tempDir := t.TempDir()
	testTextFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testTextFile, []byte("Test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = AddTextFromFile(baseFCPXML, testTextFile, 1.0)
	if err == nil || !strings.Contains(err.Error(), "no video element found") {
		t.Error("Expected error about no video element found")
	}
}

// TestAddTextFromFileIntegration tests the function with a real-world scenario
func TestAddTextFromFileIntegration(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a test text file similar to slide_text.txt
	testTextFile := filepath.Join(tempDir, "integration_test.txt")
	testContent := `Line One
Line Two
Line Three
Line Four`
	
	err := os.WriteFile(testTextFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test text file: %v", err)
	}

	// Create an empty FCPXML and add an image first
	fcpxml, err := GenerateEmpty("")
	if err != nil {
		t.Fatalf("Failed to create empty FCPXML: %v", err)
	}

	// Simulate adding an image (like png.fcpxml)
	registry := NewResourceRegistry(fcpxml)
	tx := NewTransaction(registry)
	
	ids := tx.ReserveIDs(2)
	assetID := ids[0]
	formatID := ids[1]

	// Create format for image
	_, err = tx.CreateFormat(formatID, "FFVideoFormatRateUndefined", "1280", "800", "1-13-1")
	if err != nil {
		t.Fatalf("Failed to create format: %v", err)
	}

	// Create asset
	_, err = tx.CreateAsset(assetID, "/test/image.png", "test_image", "0s", formatID)
	if err != nil {
		t.Fatalf("Failed to create asset: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Add video element to spine
	sequence := &fcpxml.Library.Events[0].Projects[0].Sequences[0]
	video := Video{
		Ref:      assetID,
		Offset:   "0s",
		Name:     "test_image",
		Start:    "86399313/24000s",
		Duration: "241241/24000s",
	}
	sequence.Spine.Videos = append(sequence.Spine.Videos, video)

	// Now test AddTextFromFile
	err = AddTextFromFile(fcpxml, testTextFile, 2.0) // 2 second offset
	if err != nil {
		t.Fatalf("AddTextFromFile failed: %v", err)
	}

	// Verify the integration worked
	updatedVideo := &sequence.Spine.Videos[0]
	
	// Should have 4 text elements
	if len(updatedVideo.NestedTitles) != 4 {
		t.Errorf("Expected 4 nested titles, got %d", len(updatedVideo.NestedTitles))
	}

	// Should have slide animation
	if updatedVideo.AdjustTransform == nil {
		t.Error("Expected slide animation to be added")
	}

	// Verify timing starts at 2 seconds offset
	if len(updatedVideo.NestedTitles) > 0 {
		firstOffset := parseFCPDuration(updatedVideo.NestedTitles[0].Offset)
		expectedFirstOffset := 86399313 + 48048 // video start + 2 seconds
		if firstOffset != expectedFirstOffset {
			t.Errorf("Expected first text offset %d, got %d", expectedFirstOffset, firstOffset)
		}
	}

	// Test that the XML can be marshaled successfully
	outputXML, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		t.Fatalf("Failed to marshal final FCPXML: %v", err)
	}

	// Basic sanity check on the XML output
	xmlStr := string(outputXML)
	if !strings.Contains(xmlStr, "Line One") || !strings.Contains(xmlStr, "Line Four") {
		t.Error("Expected text content not found in XML output")
	}
	
	if !strings.Contains(xmlStr, "adjust-transform") {
		t.Error("Expected slide animation not found in XML output")
	}
}

// Helper functions
func getTextContent(title Title) string {
	if title.Text != nil {
		return title.Text.TextStyle.Text
	}
	return ""
}

func getPositionValue(title Title) string {
	for _, param := range title.Params {
		if param.Name == "Position" {
			return param.Value
		}
	}
	return ""
}