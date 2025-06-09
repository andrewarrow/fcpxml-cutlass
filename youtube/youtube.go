package youtube

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func IsYouTubeID(input string) bool {
	return len(input) == 11 && !strings.Contains(input, ".")
}

func DownloadVideo(youtubeID string) (string, error) {
	videoFile := youtubeID + ".mov"

	// Check if .mov file already exists
	if _, err := os.Stat(videoFile); err == nil {
		fmt.Printf("Video file %s already exists, skipping download\n", videoFile)
		return videoFile, nil
	}

	fmt.Printf("Detected YouTube ID: %s, downloading...\n", youtubeID)

	// Download as .mp4 first
	mp4File := youtubeID + ".mp4"
	cmd := exec.Command("yt-dlp", "-o", mp4File, youtubeID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error downloading YouTube video: %v", err)
	}

	// Convert mp4 to mov using ffmpeg
	fmt.Printf("Converting %s to %s...\n", mp4File, videoFile)
	convertCmd := exec.Command("ffmpeg", "-i", mp4File, videoFile)
	convertCmd.Stdout = os.Stdout
	convertCmd.Stderr = os.Stderr
	if err := convertCmd.Run(); err != nil {
		return "", fmt.Errorf("error converting video to .mov: %v", err)
	}

	// Remove the temporary mp4 file
	if err := os.Remove(mp4File); err != nil {
		fmt.Printf("Warning: Could not remove temporary file %s: %v\n", mp4File, err)
	}

	return videoFile, nil
}

func DownloadSubtitles(youtubeID string) error {
	vttFile := youtubeID + ".en.vtt"

	// Check if .en.vtt file already exists
	if _, err := os.Stat(vttFile); err == nil {
		fmt.Printf("Subtitles file %s already exists, skipping download\n", vttFile)
		return nil
	}

	fmt.Printf("Downloading subtitles...\n")
	youtubeURL := "https://www.youtube.com/watch?v=" + youtubeID

	// Retry with exponential backoff up to 50 times
	var lastErr error
	for attempt := 1; attempt <= 50; attempt++ {
		subCmd := exec.Command("yt-dlp", "-o", youtubeID, "--skip-download", "--write-auto-sub", "--sub-lang", "en", youtubeURL)
		subCmd.Stdout = os.Stdout
		subCmd.Stderr = os.Stderr
		
		if err := subCmd.Run(); err != nil {
			lastErr = err
			if attempt < 50 {
				// Exponential backoff: wait 2^(attempt-1) seconds, capped at 60 seconds
				delay := time.Duration(1<<uint(attempt-1)) * time.Second
				if delay > 60*time.Second {
					delay = 60 * time.Second
				}
				fmt.Printf("Subtitle download failed (attempt %d/50), retrying in %v...\n", attempt, delay)
				time.Sleep(delay)
				continue
			}
		} else {
			// Success
			return nil
		}
	}

	return fmt.Errorf("could not download subtitles after 50 attempts: %v", lastErr)
}

func DownloadMultipleVideos(idsFile string) error {
	// Read video IDs from file
	videoIDs, err := readVideoIDsFromFile(idsFile)
	if err != nil {
		return fmt.Errorf("failed to read video IDs: %v", err)
	}

	if len(videoIDs) == 0 {
		return fmt.Errorf("no video IDs found in file")
	}

	fmt.Printf("Found %d video IDs to download\n", len(videoIDs))

	// Download each video
	for i, videoID := range videoIDs {
		fmt.Printf("\n=== Downloading video %d/%d: %s ===\n", i+1, len(videoIDs), videoID)
		
		if err := downloadVideoWithYtDlp(videoID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to download video %s: %v\n", videoID, err)
			continue
		}
		
		fmt.Printf("Successfully downloaded video %s\n", videoID)
	}

	fmt.Printf("\nBulk download completed\n")
	return nil
}

func readVideoIDsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var videoIDs []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Validate YouTube ID
		if !IsYouTubeID(line) {
			fmt.Fprintf(os.Stderr, "Warning: Invalid YouTube ID on line %d: %s\n", lineNum, line)
			continue
		}
		
		videoIDs = append(videoIDs, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return videoIDs, nil
}

func downloadVideoWithYtDlp(videoID string) error {
	outputPattern := "./data/output.%(ext)s"
	videoURL := "https://www.youtube.com/watch?v=" + videoID
	
	// Create data directory if it doesn't exist
	if err := os.MkdirAll("./data", 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}
	
	cmd := exec.Command("yt-dlp", 
		"-o", outputPattern,
		"--download-sections", "*120-240",
		videoURL)
	
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yt-dlp command failed: %v", err)
	}
	
	return nil
}
