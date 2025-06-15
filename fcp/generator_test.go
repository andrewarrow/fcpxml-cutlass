// Package fcp provides tests for FCPXML generation.
//
// 🚨 CRITICAL: Tests MUST validate CLAUDE.md compliance:
// - AFTER changes → RUN: xmllint --dtdvalid FCPXMLv1_13.dtd test_file.fcpxml  
// - BEFORE commits → RUN: ValidateClaudeCompliance() function
// - FOR durations → USE: ConvertSecondsToFCPDuration() function  
// - VERIFY: No fmt.Sprintf() with XML content in any test
package fcp

import (
	"fmt"
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
	_, err := GenerateEmpty(testFile)
	if err != nil {
		t.Fatalf("GenerateEmpty failed: %v", err)
	}

	// Read the generated file
	generatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	// Compare with expected XML string
	if string(generatedContent) != emptyxml {
		t.Errorf("Generated XML does not match expected output.\nExpected:\n%s\n\nGenerated:\n%s", emptyxml, string(generatedContent))
	}
}

var emptyxml = `<?xml version="1.0" encoding="UTF-8"?>
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

var pngxmlTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

<fcpxml version="1.13">
    <resources>
        <asset id="r2" name="cs.pitt.edu" uid="%s" start="0s" hasVideo="1" format="r3" videoSources="1" duration="0s">
            <media-rep kind="original-media" sig="%s" src="file:///Users/aa/cs/cutlass/assets/cs.pitt.edu.png"></media-rep>
        </asset>
        <format id="r1" name="FFVideoFormat720p2398" frameDuration="1001/24000s" width="1280" height="720" colorSpace="1-1-1 (Rec. 709)"></format>
        <format id="r3" name="FFVideoFormatRateUndefined" width="1280" height="720" colorSpace="1-13-1"></format>
    </resources>
    <library location="file:///Users/aa/Movies/Untitled.fcpbundle/">
        <event name="6-13-25" uid="78463397-97FD-443D-B4E2-07C581674AFC">
            <project name="wiki" uid="DEA19981-DED5-4851-8435-14515931C68A" modDate="2025-06-13 11:46:22 -0700">
                <sequence format="r1" duration="216216/24000s" tcStart="0s" tcFormat="NDF" audioLayout="stereo" audioRate="48k">
                    <spine>
                        <video ref="r2" offset="0s" name="cs.pitt.edu" duration="216216/24000s" start="86399313/24000s"></video>
                    </spine>
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

var movxmlTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

<fcpxml version="1.13">
    <resources>
        <asset id="r2" name="speech1" uid="%s" start="0s" hasVideo="1" format="r1" hasAudio="1" audioSources="1" audioChannels="2" duration="240240/24000s">
            <media-rep kind="original-media" sig="%s" src="file:///Users/aa/cs/cutlass/assets/speech1.mov"></media-rep>
        </asset>
        <format id="r1" name="FFVideoFormat720p2398" frameDuration="1001/24000s" width="1280" height="720" colorSpace="1-1-1 (Rec. 709)"></format>
    </resources>
    <library location="file:///Users/aa/Movies/Untitled.fcpbundle/">
        <event name="6-13-25" uid="78463397-97FD-443D-B4E2-07C581674AFC">
            <project name="wiki" uid="DEA19981-DED5-4851-8435-14515931C68A" modDate="2025-06-13 11:46:22 -0700">
                <sequence format="r1" duration="240240/24000s" tcStart="0s" tcFormat="NDF" audioLayout="stereo" audioRate="48k">
                    <spine>
                        <asset-clip ref="r2" offset="0s" name="speech1" duration="240240/24000s" format="r1" tcFormat="NDF" audioRole="dialogue"></asset-clip>
                    </spine>
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

var appendpngxmlTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

<fcpxml version="1.13">
    <resources>
        <asset id="r2" name="cs.pitt.edu" uid="%s" start="0s" hasVideo="1" format="r3" videoSources="1" duration="0s">
            <media-rep kind="original-media" sig="%s" src="file:///Users/aa/cs/cutlass/assets/cs.pitt.edu.png"></media-rep>
        </asset>
        <asset id="r4" name="cutlass_logo_t" uid="%s" start="0s" hasVideo="1" format="r5" videoSources="1" duration="0s">
            <media-rep kind="original-media" sig="%s" src="file:///Users/aa/cs/cutlass/assets/cutlass_logo_t.png"></media-rep>
        </asset>
        <format id="r1" name="FFVideoFormat720p2398" frameDuration="1001/24000s" width="1280" height="720" colorSpace="1-1-1 (Rec. 709)"></format>
        <format id="r3" name="FFVideoFormatRateUndefined" width="1280" height="720" colorSpace="1-13-1"></format>
        <format id="r5" name="FFVideoFormatRateUndefined" width="1280" height="720" colorSpace="1-13-1"></format>
    </resources>
    <library location="file:///Users/aa/Movies/Untitled.fcpbundle/">
        <event name="6-13-25" uid="78463397-97FD-443D-B4E2-07C581674AFC">
            <project name="wiki" uid="DEA19981-DED5-4851-8435-14515931C68A" modDate="2025-06-13 11:46:22 -0700">
                <sequence format="r1" duration="457457/24000s" tcStart="0s" tcFormat="NDF" audioLayout="stereo" audioRate="48k">
                    <spine>
                        <video ref="r2" offset="0s" name="cs.pitt.edu" duration="241241/24000s" start="86399313/24000s"></video>
                        <video ref="r4" offset="241241/24000s" name="cutlass_logo_t" duration="216216/24000s" start="86399313/24000s"></video>
                    </spine>
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

var appendmovtopngxmlTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

<fcpxml version="1.13">
    <resources>
        <asset id="r2" name="cs.pitt.edu" uid="%s" start="0s" hasVideo="1" format="r3" videoSources="1" duration="0s">
            <media-rep kind="original-media" sig="%s" src="file:///Users/aa/cs/cutlass/assets/cs.pitt.edu.png"></media-rep>
        </asset>
        <asset id="r4" name="speech1" uid="%s" start="0s" hasVideo="1" format="r1" hasAudio="1" audioSources="1" audioChannels="2" duration="240240/24000s">
            <media-rep kind="original-media" sig="%s" src="file:///Users/aa/cs/cutlass/assets/speech1.mov"></media-rep>
        </asset>
        <format id="r1" name="FFVideoFormat720p2398" frameDuration="1001/24000s" width="1280" height="720" colorSpace="1-1-1 (Rec. 709)"></format>
        <format id="r3" name="FFVideoFormatRateUndefined" width="1280" height="720" colorSpace="1-13-1"></format>
    </resources>
    <library location="file:///Users/aa/Movies/Untitled.fcpbundle/">
        <event name="6-13-25" uid="78463397-97FD-443D-B4E2-07C581674AFC">
            <project name="wiki" uid="DEA19981-DED5-4851-8435-14515931C68A" modDate="2025-06-13 11:46:22 -0700">
                <sequence format="r1" duration="481481/24000s" tcStart="0s" tcFormat="NDF" audioLayout="stereo" audioRate="48k">
                    <spine>
                        <video ref="r2" offset="0s" name="cs.pitt.edu" duration="241241/24000s" start="86399313/24000s"></video>
                        <asset-clip ref="r4" offset="241241/24000s" name="speech1" duration="240240/24000s" format="r1" tcFormat="NDF" audioRole="dialogue"></asset-clip>
                    </spine>
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

func TestGeneratePng(t *testing.T) {
	testFile := "test_generate_png.fcpxml"

	defer func() {
		if err := os.Remove(testFile); err != nil && !os.IsNotExist(err) {
			t.Errorf("Failed to clean up test file: %v", err)
		}
	}()

	fcpxml, err := GenerateEmpty("")
	if err != nil {
		t.Fatalf("GenerateEmpty failed: %v", err)
	}

	err = AddImage(fcpxml, "/Users/aa/cs/cutlass/assets/cs.pitt.edu.png", 9.0)
	if err != nil {
		t.Fatalf("AddImage failed: %v", err)
	}

	err = WriteToFile(fcpxml, testFile)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	generatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	// Generate expected XML with correct UIDs
	pngUID := GenerateUID("/Users/aa/cs/cutlass/assets/cs.pitt.edu.png")
	expectedXML := fmt.Sprintf(pngxmlTemplate, pngUID, pngUID)

	if string(generatedContent) != expectedXML {
		t.Errorf("Generated XML does not match expected output.\nExpected:\n%s\n\nGenerated:\n%s", expectedXML, string(generatedContent))
	}
}

func TestGenerateMov(t *testing.T) {
	testFile := "test_generate_mov.fcpxml"

	defer func() {
		if err := os.Remove(testFile); err != nil && !os.IsNotExist(err) {
			t.Errorf("Failed to clean up test file: %v", err)
		}
	}()

	fcpxml, err := GenerateEmpty("")
	if err != nil {
		t.Fatalf("GenerateEmpty failed: %v", err)
	}

	err = AddVideo(fcpxml, "/Users/aa/cs/cutlass/assets/speech1.mov")
	if err != nil {
		t.Fatalf("AddVideo failed: %v", err)
	}

	err = WriteToFile(fcpxml, testFile)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	generatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	// Generate expected XML with correct UIDs
	movUID := GenerateUID("/Users/aa/cs/cutlass/assets/speech1.mov")
	expectedXML := fmt.Sprintf(movxmlTemplate, movUID, movUID)

	if string(generatedContent) != expectedXML {
		t.Errorf("Generated XML does not match expected output.\nExpected:\n%s\n\nGenerated:\n%s", expectedXML, string(generatedContent))
	}
}

func TestAppendPng(t *testing.T) {
	testFile := "test_append_png.fcpxml"

	defer func() {
		if err := os.Remove(testFile); err != nil && !os.IsNotExist(err) {
			t.Errorf("Failed to clean up test file: %v", err)
		}
	}()

	fcpxml, err := GenerateEmpty("")
	if err != nil {
		t.Fatalf("GenerateEmpty failed: %v", err)
	}

	err = AddImage(fcpxml, "/Users/aa/cs/cutlass/assets/cs.pitt.edu.png", 10.05)
	if err != nil {
		t.Fatalf("First AddImage failed: %v", err)
	}

	err = AddImage(fcpxml, "/Users/aa/cs/cutlass/assets/cutlass_logo_t.png", 9.0)
	if err != nil {
		t.Fatalf("Second AddImage failed: %v", err)
	}

	err = WriteToFile(fcpxml, testFile)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	generatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	// Generate expected XML with correct UIDs
	pngUID := GenerateUID("/Users/aa/cs/cutlass/assets/cs.pitt.edu.png")
	logoUID := GenerateUID("/Users/aa/cs/cutlass/assets/cutlass_logo_t.png")
	expectedXML := fmt.Sprintf(appendpngxmlTemplate, pngUID, pngUID, logoUID, logoUID)

	if string(generatedContent) != expectedXML {
		t.Errorf("Generated XML does not match expected output.\nExpected:\n%s\n\nGenerated:\n%s", expectedXML, string(generatedContent))
	}
}

func TestAppendMovToPng(t *testing.T) {
	testFile := "test_append_mov_to_png.fcpxml"

	defer func() {
		if err := os.Remove(testFile); err != nil && !os.IsNotExist(err) {
			t.Errorf("Failed to clean up test file: %v", err)
		}
	}()

	fcpxml, err := GenerateEmpty("")
	if err != nil {
		t.Fatalf("GenerateEmpty failed: %v", err)
	}

	err = AddImage(fcpxml, "/Users/aa/cs/cutlass/assets/cs.pitt.edu.png", 10.05)
	if err != nil {
		t.Fatalf("AddImage failed: %v", err)
	}

	err = AddVideo(fcpxml, "/Users/aa/cs/cutlass/assets/speech1.mov")
	if err != nil {
		t.Fatalf("AddVideo failed: %v", err)
	}

	err = WriteToFile(fcpxml, testFile)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	generatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	// Generate expected XML with correct UIDs
	pngUID := GenerateUID("/Users/aa/cs/cutlass/assets/cs.pitt.edu.png")
	movUID := GenerateUID("/Users/aa/cs/cutlass/assets/speech1.mov")
	expectedXML := fmt.Sprintf(appendmovtopngxmlTemplate, pngUID, pngUID, movUID, movUID)

	if string(generatedContent) != expectedXML {
		t.Errorf("Generated XML does not match expected output.\nExpected:\n%s\n\nGenerated:\n%s", expectedXML, string(generatedContent))
	}
}
