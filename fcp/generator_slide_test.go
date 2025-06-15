package fcp

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"
)

// TestAddImageWithSlide tests the slide animation functionality
func TestAddImageWithSlide(t *testing.T) {
	// Create test image file
	testImagePath := "test_slide_image.png"
	err := os.WriteFile(testImagePath, []byte("fake png data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test image file: %v", err)
	}
	defer os.Remove(testImagePath)

	// Generate empty FCPXML
	fcpxml, err := GenerateEmpty("")
	if err != nil {
		t.Fatalf("Failed to generate empty FCPXML: %v", err)
	}

	// Add image with slide animation
	err = AddImageWithSlide(fcpxml, testImagePath, 9.0, true)
	if err != nil {
		t.Fatalf("Failed to add image with slide: %v", err)
	}

	// Verify the structure
	if len(fcpxml.Resources.Assets) != 1 {
		t.Errorf("Expected 1 asset, got %d", len(fcpxml.Resources.Assets))
	}

	if len(fcpxml.Resources.Formats) != 2 {
		t.Errorf("Expected 2 formats, got %d", len(fcpxml.Resources.Formats))
	}

	// Check if video element has adjust-transform with slide animation
	sequence := &fcpxml.Library.Events[0].Projects[0].Sequences[0]
	if len(sequence.Spine.Videos) != 1 {
		t.Fatalf("Expected 1 video element, got %d", len(sequence.Spine.Videos))
	}

	video := sequence.Spine.Videos[0]
	if video.AdjustTransform == nil {
		t.Fatalf("Expected video to have adjust-transform")
	}

	// Verify keyframe animation parameters
	params := video.AdjustTransform.Params
	if len(params) != 4 {
		t.Errorf("Expected 4 animation params (anchor, position, rotation, scale), got %d", len(params))
	}

	// Check parameter names
	expectedParams := []string{"anchor", "position", "rotation", "scale"}
	for i, expectedParam := range expectedParams {
		if i >= len(params) {
			t.Errorf("Missing param: %s", expectedParam)
			continue
		}
		if params[i].Name != expectedParam {
			t.Errorf("Expected param %s, got %s", expectedParam, params[i].Name)
		}
	}

	// Verify position keyframes specifically
	positionParam := params[1] // position is second param
	if positionParam.KeyframeAnimation == nil {
		t.Fatalf("Position param should have keyframe animation")
	}

	keyframes := positionParam.KeyframeAnimation.Keyframes
	if len(keyframes) != 2 {
		t.Errorf("Expected 2 position keyframes, got %d", len(keyframes))
	}

	// Check keyframe values
	if keyframes[0].Value != "0 0" {
		t.Errorf("Expected first keyframe value '0 0', got '%s'", keyframes[0].Value)
	}
	if keyframes[1].Value != "51.3109 0" {
		t.Errorf("Expected second keyframe value '51.3109 0', got '%s'", keyframes[1].Value)
	}

	// Check keyframe timing matches samples/slide.fcpxml pattern
	if keyframes[0].Time != "86399313/24000s" {
		t.Errorf("Expected first keyframe time '86399313/24000s', got '%s'", keyframes[0].Time)
	}
	if keyframes[1].Time != "86423337/24000s" {
		t.Errorf("Expected second keyframe time '86423337/24000s', got '%s'", keyframes[1].Time)
	}

	// Verify curve attributes on static keyframes
	anchorParam := params[0]
	if len(anchorParam.KeyframeAnimation.Keyframes) > 0 {
		anchorKeyframe := anchorParam.KeyframeAnimation.Keyframes[0]
		if anchorKeyframe.Curve != "linear" {
			t.Errorf("Expected anchor keyframe curve 'linear', got '%s'", anchorKeyframe.Curve)
		}
	}
}

// TestAddImageWithoutSlide tests that images without slide don't have animations
func TestAddImageWithoutSlide(t *testing.T) {
	// Create test image file
	testImagePath := "test_no_slide_image.png"
	err := os.WriteFile(testImagePath, []byte("fake png data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test image file: %v", err)
	}
	defer os.Remove(testImagePath)

	// Generate empty FCPXML
	fcpxml, err := GenerateEmpty("")
	if err != nil {
		t.Fatalf("Failed to generate empty FCPXML: %v", err)
	}

	// Add image without slide animation
	err = AddImageWithSlide(fcpxml, testImagePath, 9.0, false)
	if err != nil {
		t.Fatalf("Failed to add image without slide: %v", err)
	}

	// Check if video element has no adjust-transform
	sequence := &fcpxml.Library.Events[0].Projects[0].Sequences[0]
	if len(sequence.Spine.Videos) != 1 {
		t.Fatalf("Expected 1 video element, got %d", len(sequence.Spine.Videos))
	}

	video := sequence.Spine.Videos[0]
	if video.AdjustTransform != nil {
		t.Errorf("Expected video to have no adjust-transform, but it has one")
	}
}

// TestSlideAnimationXMLOutput tests the actual XML output structure
func TestSlideAnimationXMLOutput(t *testing.T) {
	// Create test image file
	testImagePath := "test_xml_slide_image.png"
	err := os.WriteFile(testImagePath, []byte("fake png data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test image file: %v", err)
	}
	defer os.Remove(testImagePath)

	// Generate empty FCPXML
	fcpxml, err := GenerateEmpty("")
	if err != nil {
		t.Fatalf("Failed to generate empty FCPXML: %v", err)
	}

	// Add image with slide animation
	err = AddImageWithSlide(fcpxml, testImagePath, 9.0, true)
	if err != nil {
		t.Fatalf("Failed to add image with slide: %v", err)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		t.Fatalf("Failed to marshal XML: %v", err)
	}

	xmlString := string(output)

	// Check for key XML elements that should be present
	expectedElements := []string{
		"<adjust-transform>",
		`<param name="anchor">`,
		`<param name="position">`,
		`<param name="rotation">`,
		`<param name="scale">`,
		"<keyframeAnimation>",
		`time="86399313/24000s"`,
		`time="86423337/24000s"`,
		`value="0 0"`,
		`value="51.3109 0"`,
		`curve="linear"`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(xmlString, expected) {
			t.Errorf("Expected XML to contain '%s', but it doesn't", expected)
		}
	}

	// Check that empty key/value attributes are not present
	if strings.Contains(xmlString, `key=""`) {
		t.Errorf("XML should not contain empty key attributes")
	}
	if strings.Contains(xmlString, `value=""`) {
		t.Errorf("XML should not contain empty value attributes on params")
	}
}

// TestCreateSlideAnimation tests the slide animation creation function directly
func TestCreateSlideAnimation(t *testing.T) {
	// Test the createSlideAnimation function
	adjustTransform := createSlideAnimation("0s", 9.0)

	if adjustTransform == nil {
		t.Fatalf("createSlideAnimation returned nil")
	}

	// Check that we have 4 parameters
	if len(adjustTransform.Params) != 4 {
		t.Errorf("Expected 4 params, got %d", len(adjustTransform.Params))
	}

	// Verify position parameter keyframes
	var positionParam *Param
	for _, param := range adjustTransform.Params {
		if param.Name == "position" {
			positionParam = &param
			break
		}
	}

	if positionParam == nil {
		t.Fatalf("Could not find position parameter")
	}

	if positionParam.KeyframeAnimation == nil {
		t.Fatalf("Position parameter should have keyframe animation")
	}

	keyframes := positionParam.KeyframeAnimation.Keyframes
	if len(keyframes) != 2 {
		t.Errorf("Expected 2 position keyframes, got %d", len(keyframes))
	}

	// Test timing calculation (should be exactly 1 second apart)
	// 86423337 - 86399313 = 24024 frames = exactly 1 second in 1001/24000s timebase
	if keyframes[1].Time != "86423337/24000s" {
		t.Errorf("Expected end time 86423337/24000s, got %s", keyframes[1].Time)
	}
	if keyframes[0].Time != "86399313/24000s" {
		t.Errorf("Expected start time 86399313/24000s, got %s", keyframes[0].Time)
	}

	// Verify the slide values
	if keyframes[0].Value != "0 0" {
		t.Errorf("Expected start position '0 0', got '%s'", keyframes[0].Value)
	}
	if keyframes[1].Value != "51.3109 0" {
		t.Errorf("Expected end position '51.3109 0', got '%s'", keyframes[1].Value)
	}
}

// TestSlideAnimationBackwardsCompatibility tests that AddImage still works without slide
func TestSlideAnimationBackwardsCompatibility(t *testing.T) {
	// Create test image file
	testImagePath := "test_compat_image.png"
	err := os.WriteFile(testImagePath, []byte("fake png data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test image file: %v", err)
	}
	defer os.Remove(testImagePath)

	// Generate empty FCPXML
	fcpxml, err := GenerateEmpty("")
	if err != nil {
		t.Fatalf("Failed to generate empty FCPXML: %v", err)
	}

	// Test that the original AddImage function still works (should call AddImageWithSlide with false)
	err = AddImage(fcpxml, testImagePath, 9.0)
	if err != nil {
		t.Fatalf("Failed to add image using original AddImage function: %v", err)
	}

	// Verify no animation was added
	sequence := &fcpxml.Library.Events[0].Projects[0].Sequences[0]
	if len(sequence.Spine.Videos) != 1 {
		t.Fatalf("Expected 1 video element, got %d", len(sequence.Spine.Videos))
	}

	video := sequence.Spine.Videos[0]
	if video.AdjustTransform != nil {
		t.Errorf("AddImage should not add animation, but adjust-transform was found")
	}
}