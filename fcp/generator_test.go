package fcp

import (
	"os"
	"testing"
)

func TestGenerateEmpty(t *testing.T) {
	// Create a temporary test file
	testFile := "test_generate_empty.fcpxml"
	
	// Ensure cleanup even if test fails
	defer func() {
		if err := os.Remove(testFile); err != nil && !os.IsNotExist(err) {
			t.Errorf("Failed to clean up test file: %v", err)
		}
	}()
	
	// Call GenerateEmpty with the test file
	err := GenerateEmpty(testFile)
	if err != nil {
		t.Fatalf("GenerateEmpty failed: %v", err)
	}
	
	// Read the generated file
	generatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}
	
	// Compare with expected XML string
	if string(generatedContent) != xmlstring {
		t.Errorf("Generated XML does not match expected output.\nExpected:\n%s\n\nGenerated:\n%s", xmlstring, string(generatedContent))
	}
}

var xmlstring = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

<fcpxml version="1.13">
    <resources>
        <format id="r1" name="FFVideoFormat720p2398" frameDuration="1001/24000s" width="1280" height="720" colorSpace="1-1-1 (Rec. 709)"></format>
    </resources>
    <library location="file:///Users/aa/Movies/Untitled.fcpbundle/">
        <event name="6-13-25" uid="78463397-97FD-443D-B4E2-07C581674AFC">
            <project name="wiki" uid="DEA19981-DED5-4851-8435-14515931C68A" modDate="2025-06-13 11:46:22 -0700">
                <sequence format="r1" duration="0s" tcStart="0s" tcFormat="NDF" audioLayout="stereo" audioRate="48k">
                    <spine></spine>
                </sequence>
            </project>
        </event>
        <smart-collection name="Projects" match="all">
            <match-clip rule="is" type="project"></match-clip>
        </smart-collection>
        <smart-collection name="All Video" match="any">
            <match-media rule="is" type="videoOnly"></match-media>
            <match-media rule="is" type="videoWithAudio"></match-media>
        </smart-collection>
        <smart-collection name="Audio Only" match="all">
            <match-media rule="is" type="audioOnly"></match-media>
        </smart-collection>
        <smart-collection name="Stills" match="all">
            <match-media rule="is" type="stills"></match-media>
        </smart-collection>
        <smart-collection name="Favorites" match="all">
            <match-ratings value="favorites"></match-ratings>
        </smart-collection>
    </library>
</fcpxml>`
