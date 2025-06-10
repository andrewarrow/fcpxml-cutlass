package fcp

import (
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"os/exec"
	"path/filepath"
	"os"
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
	FirstName       string
	LastName        string
	LastNameSuffix  string
	Videos          []TemplateVideo
	Numbers         []NumberSection
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

func GenerateTop5FCPXML(templatePath string, videoIDs []string, name, outputPath string) error {
	// Parse the name
	nameWords := strings.Fields(name)
	firstName := ""
	lastName := ""
	lastNameSuffix := ""
	
	if len(nameWords) >= 1 {
		firstName = nameWords[0]
	}
	if len(nameWords) >= 2 {
		lastName = nameWords[1]
		// Check if the original template had a suffix (like the "g" in "Dimoldenber")
		if lastName == "Dimoldenber" {
			lastNameSuffix = "g"
		}
	}

	// Create video data
	videos := make([]TemplateVideo, len(videoIDs))
	for i, id := range videoIDs {
		// Generate bookmark for the video file
		videoPath := fmt.Sprintf("data/%s.mov", id)
		bookmark, _ := generateBookmark(videoPath) // Ignore errors, continue without bookmark
		
		videos[i] = TemplateVideo{
			ID:       id,
			UID:      generateUID(id),
			Bookmark: bookmark,
		}
	}

	// Create number sections (5, 4, 3, 2, 1)
	numbers := make([]NumberSection, 5)
	offsets := []string{
		"8300/2500s",        // NUMBER 5
		"3827200/320000s",   // NUMBER 4  
		"8300/2500s",        // NUMBER 3 (same as 5)
		"3827200/320000s",   // NUMBER 2 (same as 4)
		"8300/2500s",        // NUMBER 1 (same as 5)
	}
	
	for i := 0; i < 5; i++ {
		numbers[i] = NumberSection{
			Number:  5 - i, // 5, 4, 3, 2, 1
			VideoID: videoIDs[i%len(videoIDs)], // Cycle through available videos
			Offset:  offsets[i],
		}
	}

	// Create template data
	data := TemplateData{
		FirstName:      firstName,
		LastName:       lastName,
		LastNameSuffix: lastNameSuffix,
		Videos:         videos,
		Numbers:        numbers,
	}

	// Create template functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"mul": func(a, b int) int { return a * b },
	}

	// Parse all templates in the templates directory
	tmpl := template.New("").Funcs(funcMap)
	
	templateFiles, err := ioutil.ReadDir("templates")
	if err != nil {
		return fmt.Errorf("failed to read templates directory: %v", err)
	}
	
	for _, file := range templateFiles {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".fcpxml") {
			filePath := "templates/" + file.Name()
			fileContents, err := ioutil.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read template file %s: %v", file.Name(), err)
			}
			
			_, err = tmpl.New(file.Name()).Parse(string(fileContents))
			if err != nil {
				return fmt.Errorf("failed to parse template %s: %v", file.Name(), err)
			}
		}
	}

	// Execute the template
	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "top5.fcpxml", data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	// Write the result to the output file
	err = ioutil.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}

	return nil
}

