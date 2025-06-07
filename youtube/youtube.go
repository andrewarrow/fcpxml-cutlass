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
	fmt.Printf("Detected YouTube ID: %s, downloading...\n", youtubeID)
	
	cmd := exec.Command("yt-dlp", "-o", videoFile, youtubeID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error downloading YouTube video: %v", err)
	}
	
	return videoFile, nil
}

func DownloadSubtitles(youtubeID string) error {
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