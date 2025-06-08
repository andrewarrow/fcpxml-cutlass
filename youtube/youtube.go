package youtube

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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
	convertCmd := exec.Command("ffmpeg", "-i", mp4File, "-c", "copy", videoFile)
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
	
	subCmd := exec.Command("yt-dlp", "-o", youtubeID, "--skip-download", "--write-auto-sub", "--sub-lang", "en", youtubeURL)
	subCmd.Stdout = os.Stdout
	subCmd.Stderr = os.Stderr
	if err := subCmd.Run(); err != nil {
		return fmt.Errorf("could not download subtitles: %v", err)
	}
	
	return nil
}