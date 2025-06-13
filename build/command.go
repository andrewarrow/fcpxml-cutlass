package build

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cutlass/fcp"
	"github.com/spf13/cobra"
)

var BuildCmd = &cobra.Command{
	Use:   "build [filename] [add-video] [video-file]",
	Short: "Build a blank FCP project or add video to existing project",
	Long:  "Create a blank Final Cut Pro project from empty.fcpxml template, or add a video to it",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		if !strings.HasSuffix(filename, ".fcpxml") {
			filename += ".fcpxml"
		}
		
		// Check if this is an add-video command
		if len(args) >= 3 && args[1] == "add-video" {
			videoFile := args[2]
			
			// First ensure the project exists, create if it doesn't
			if _, err := os.Stat(filename); os.IsNotExist(err) {
				err := createBlankProject(filename)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating blank project: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Created blank project: %s\n", filename)
			}
			
			// Add video to the project
			err := addVideoToProject(filename, videoFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error adding video to project: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Added video %s to project %s\n", videoFile, filename)
		} else {
			// Just create a blank project
			err := createBlankProject(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating blank project: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Created blank project: %s\n", filename)
		}
	},
}


func createBlankProject(filename string) error {
	// Read the empty.fcpxml template
	emptyContent, err := ioutil.ReadFile("empty.fcpxml")
	if err != nil {
		return fmt.Errorf("failed to read empty.fcpxml: %v", err)
	}
	
	// Parse the XML to modify timestamps and UIDs
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
			baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
			fcpxml.Library.Events[0].Projects[0].Name = baseName
		}
	}
	
	// Generate the XML output with proper formatting
	output, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %v", err)
	}
	
	// Add XML declaration and DOCTYPE
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

` + string(output)
	
	// Write to output file
	err = ioutil.WriteFile(filename, []byte(xmlContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}
	
	return nil
}

func addVideoToProject(projectFile, videoFile string) error {
	// Read the existing project file
	content, err := ioutil.ReadFile(projectFile)
	if err != nil {
		return fmt.Errorf("failed to read project file: %v", err)
	}
	
	// Parse the XML
	var fcpxml fcp.FCPXML
	err = xml.Unmarshal(content, &fcpxml)
	if err != nil {
		return fmt.Errorf("failed to parse project file: %v", err)
	}
	
	// Get absolute path for the video file
	absVideoPath, err := filepath.Abs(videoFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}
	
	// Check if video file exists
	if _, err := os.Stat(absVideoPath); os.IsNotExist(err) {
		return fmt.Errorf("video file does not exist: %s", absVideoPath)
	}
	
	// Create asset ID and UID
	baseName := strings.TrimSuffix(filepath.Base(videoFile), filepath.Ext(videoFile))
	
	// Check if asset already exists in the project
	existingAssetID := findExistingAsset(&fcpxml, absVideoPath)
	if existingAssetID != "" {
		// Asset already exists, just add asset-clip to spine
		duration, err := getVideoDuration(absVideoPath)
		if err != nil {
			duration = "240240/24000s" // 10 seconds at 23.976fps
		}
		
		if len(fcpxml.Library.Events) > 0 && len(fcpxml.Library.Events[0].Projects) > 0 {
			project := &fcpxml.Library.Events[0].Projects[0]
			if len(project.Sequences) > 0 {
				assetClip := fcp.AssetClip{
					Ref:      existingAssetID,
					Offset:   "0s",
					Name:     baseName,
					Duration: duration,
					Format:   "r1",
					TCFormat: "NDF",
				}
				
				// Convert asset clip to XML string and add to spine
				assetClipXML, err := xml.Marshal(assetClip)
				if err != nil {
					return fmt.Errorf("failed to marshal asset clip: %v", err)
				}
				
				// Add proper indentation
				indentedXML := strings.ReplaceAll(string(assetClipXML), "\n", "\n                        ")
				project.Sequences[0].Spine.Content = "\n                        " + indentedXML + "\n                    "
				
				// Update sequence duration to match the content
				project.Sequences[0].Duration = duration
			}
		}
		
		// Generate the XML output
		output, err := xml.MarshalIndent(fcpxml, "", "    ")
		if err != nil {
			return fmt.Errorf("failed to marshal XML: %v", err)
		}
		
		// Add XML declaration and DOCTYPE
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

` + string(output)
		
		// Write back to project file
		err = ioutil.WriteFile(projectFile, []byte(xmlContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write project file: %v", err)
		}
		
		return nil
	}
	
	assetID := fmt.Sprintf("r%d", len(fcpxml.Resources.Assets)+2) // Start from r2 since r1 is format
	
	// Get video duration using ffprobe
	duration, err := getVideoDuration(absVideoPath)
	if err != nil {
		// Fallback to default duration on frame boundary
		duration = "240240/24000s" // 10 seconds at 23.976fps
	}
	
	// Generate consistent UID from file path
	assetUID := generateUID(absVideoPath)
	
	// Generate bookmark for the video file
	_, _ = generateBookmark(absVideoPath) // Ignore errors, continue without bookmark
	
	// Add asset to resources
	asset := fcp.Asset{
		ID:            assetID,
		Name:          baseName,
		UID:           assetUID,
		Start:         "0s",
		HasVideo:      "1",
		Format:        "r1",
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
	
	fcpxml.Resources.Assets = append(fcpxml.Resources.Assets, asset)
	
	// Add asset-clip to the spine
	if len(fcpxml.Library.Events) > 0 && len(fcpxml.Library.Events[0].Projects) > 0 {
		project := &fcpxml.Library.Events[0].Projects[0]
		if len(project.Sequences) > 0 {
			assetClip := fcp.AssetClip{
				Ref:      assetID,
				Offset:   "0s",
				Name:     baseName,
				Duration: duration,
				Format:   "r1",
				TCFormat: "NDF",
			}
			
			// Convert asset clip to XML string and add to spine
			assetClipXML, err := xml.Marshal(assetClip)
			if err != nil {
				return fmt.Errorf("failed to marshal asset clip: %v", err)
			}
			
			// Add proper indentation
			indentedXML := strings.ReplaceAll(string(assetClipXML), "\n", "\n                        ")
			project.Sequences[0].Spine.Content = "\n                        " + indentedXML + "\n                    "
			
			// Update sequence duration to match the content
			project.Sequences[0].Duration = duration
		}
	}
	
	// Generate the XML output
	output, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %v", err)
	}
	
	// Add XML declaration and DOCTYPE
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

` + string(output)
	
	// Write back to project file
	err = ioutil.WriteFile(projectFile, []byte(xmlContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write project file: %v", err)
	}
	
	return nil
}

// generateUID creates a consistent UID from a file path using MD5 hash
func generateUID(filePath string) string {
	// Create a hash from the file path to ensure consistent UIDs
	hasher := md5.New()
	hasher.Write([]byte("cutlass_video_" + filePath))
	hash := hasher.Sum(nil)
	// Convert to uppercase hex string and format as UID
	hexStr := strings.ToUpper(hex.EncodeToString(hash))
	return fmt.Sprintf("%s-%s-%s-%s-%s", 
		hexStr[0:8], hexStr[8:12], hexStr[12:16], hexStr[16:20], hexStr[20:32])
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