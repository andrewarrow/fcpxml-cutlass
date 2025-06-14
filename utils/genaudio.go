package utils

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func HandleGenAudioCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Error: Please provide a text file")
		return
	}

	textFile := args[0]
	if err := processSimpleTextFile(textFile); err != nil {
		fmt.Printf("Error processing text file: %v\n", err)
	}
}

func processSimpleTextFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Extract video ID from filename (without extension)
	videoID := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	audioDir := fmt.Sprintf("./data/%s_audio", videoID)

	// Create audio directory
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return fmt.Errorf("failed to create audio directory: %v", err)
	}

	scanner := bufio.NewScanner(file)
	sentenceNum := 1

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines
		if line == "" {
			continue
		}

		// Generate initial audio filename without duration
		tempFilename := filepath.Join(audioDir, fmt.Sprintf("s%d.wav", sentenceNum))
		
		// Check if audio file already exists (with any duration)
		if existingFile := findExistingAudioFile(audioDir, sentenceNum); existingFile != "" {
			fmt.Printf("Skipping sentence %d (already exists: %s)\n", sentenceNum, existingFile)
			sentenceNum++
			continue
		}

		// Call chatterbox to generate audio
		if err := callChatterbox(line, tempFilename); err != nil {
			fmt.Printf("Error generating audio for sentence %d: %v\n", sentenceNum, err)
			sentenceNum++
			continue
		}

		// Get audio duration and rename file
		duration, err := getAudioDurationSeconds(tempFilename)
		if err != nil {
			fmt.Printf("Warning: Could not get duration for sentence %d: %v\n", sentenceNum, err)
			fmt.Printf("Generated audio for sentence %d\n", sentenceNum)
		} else {
			// Rename file to include duration
			finalFilename := filepath.Join(audioDir, fmt.Sprintf("s%d_%.0f.wav", sentenceNum, duration))
			if err := os.Rename(tempFilename, finalFilename); err != nil {
				fmt.Printf("Warning: Could not rename file: %v\n", err)
				fmt.Printf("Generated audio for sentence %d\n", sentenceNum)
			} else {
				fmt.Printf("Generated audio for sentence %d (%.1fs)\n", sentenceNum, duration)
			}
		}

		sentenceNum++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	return nil
}

func findExistingAudioFile(audioDir string, sentenceNum int) string {
	// Look for files with pattern s{num}*.wav
	pattern := fmt.Sprintf("s%d*.wav", sentenceNum)
	matches, err := filepath.Glob(filepath.Join(audioDir, pattern))
	if err != nil || len(matches) == 0 {
		return ""
	}
	return filepath.Base(matches[0])
}

func callChatterbox(sentence, audioFilename string) error {
	cmd := exec.Command("/opt/miniconda3/envs/chatterbox/bin/python3",
		"/Users/aa/os/chatterbox/chatterbox/main.py",
		sentence,
		audioFilename)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chatterbox command failed: %v", err)
	}

	return nil
}

func getAudioDurationSeconds(audioFile string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format=duration", "-of", "csv=p=0", audioFile)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %v", err)
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %v", err)
	}

	return duration, nil
}