package genvoices

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func HandleGenVoicesCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Error: Please provide a words file")
		return
	}

	wordsFile := args[0]
	if err := processWordsFile(wordsFile); err != nil {
		fmt.Printf("Error processing words file: %v\n", err)
	}
}

func processWordsFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Extract base name without extension for audio directory
	baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	audioDir := fmt.Sprintf("./data/%s_audio", baseName)

	scanner := bufio.NewScanner(file)
	currentSection := 0
	sentenceCount := 0
	inSentences := false

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Check for section headers
		if strings.HasPrefix(strings.TrimSpace(line), "## Section") {
			currentSection++
			sentenceCount = 0
			inSentences = false
			fmt.Printf("Found section %d\n", currentSection)
			continue
		}

		// Check for sentences marker
		if strings.Contains(line, "- Sentences:") {
			inSentences = true
			continue
		}

		// Process sentences
		if inSentences && strings.HasPrefix(line, "  - ") {
			sentenceCount++
			sentence := strings.TrimPrefix(line, "  - ")

			// Generate audio filename
			audioFilename := filepath.Join(audioDir, fmt.Sprintf("s%d_s%d.wav", currentSection, sentenceCount))

			// Check if audio file already exists
			if _, err := os.Stat(audioFilename); err == nil {
				fmt.Printf("Skipping section %d, sentence %d (already exists)\n", currentSection, sentenceCount)
				continue
			}

			// Call chatterbox
			if err := callChatterbox(sentence, audioFilename); err != nil {
				fmt.Printf("Error generating voice for sentence %d in section %d: %v\n", sentenceCount, currentSection, err)
				continue
			}

			fmt.Printf("Generated voice for section %d, sentence %d\n", currentSection, sentenceCount)
		}

		// Reset inSentences if we encounter a non-sentence line that doesn't start with spaces
		if inSentences && !strings.HasPrefix(line, "  ") && !strings.Contains(line, "- Sentences:") {
			inSentences = false
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	return nil
}

func callChatterbox(sentence, audioFilename string) error {
	cmd := exec.Command("/opt/miniconda3/envs/chatterbox/bin/python3",
		"/Users/aa/os/chatterbox/chatterbox/main.py",
		sentence,
		audioFilename)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(audioFilename), 0755); err != nil {
		return fmt.Errorf("failed to create audio directory: %v", err)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chatterbox command failed: %v", err)
	}

	return nil
}
