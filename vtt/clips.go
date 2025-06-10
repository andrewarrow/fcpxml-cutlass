package vtt

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// createSmartClipsWithAudio creates clips using both text analysis and audio boundaries
func createSmartClipsWithAudio(segments []Segment, silenceGaps []SilenceGap) []Segment {
	if len(silenceGaps) == 0 {
		// Fallback to text-only analysis
		return createSmartClips(segments)
	}

	var smartClips []Segment
	i := 0

	for i < len(segments) {
		currentClip := segments[i]
		currentText := strings.TrimSpace(currentClip.Text)

		// Look ahead to merge incomplete thoughts
		j := i + 1
		for j < len(segments) {
			nextSegment := segments[j]
			nextText := strings.TrimSpace(nextSegment.Text)

			// Check if we should merge with next segment
			shouldMerge := false

			// Standard text-based merging logic
			if !endsWithCompletePunctuation(currentText) {
				shouldMerge = true
			}
			if len(strings.Fields(currentText)) < 4 {
				shouldMerge = true
			}
			if len(nextText) > 0 && nextText[0] >= 'a' && nextText[0] <= 'z' {
				shouldMerge = true
			}

			// Audio-based boundary detection
			if shouldMerge {
				// Check if there's a natural pause between segments
				gapBetween := nextSegment.StartTime - currentClip.EndTime
				if gapBetween > 800*time.Millisecond {
					// Long gap suggests natural boundary
					shouldMerge = false
				} else {
					// Look for silence gaps near the boundary
					boundaryTime := nextSegment.StartTime
					for _, gap := range silenceGaps {
						// If there's a silence gap within 1 second of the boundary
						if gap.Start <= boundaryTime+time.Second && gap.End >= boundaryTime-time.Second {
							if gap.Duration >= 400*time.Millisecond {
								shouldMerge = false
								break
							}
						}
					}
				}
			}

			// Don't merge if combined clip would be too long
			combinedDuration := nextSegment.EndTime - currentClip.StartTime
			if combinedDuration > 20*time.Second {
				break
			}

			// Don't merge if we've hit a topic change
			if isTopicChange(currentText, nextText) {
				break
			}

			if shouldMerge {
				currentClip.EndTime = nextSegment.EndTime
				currentClip.Text = currentText + " " + nextText
				currentText = currentClip.Text
				j++
			} else {
				break
			}
		}

		// Refine clip boundaries using audio analysis
		currentClip = refineClipBoundariesWithAudio(currentClip, silenceGaps)

		smartClips = append(smartClips, currentClip)
		i = j
	}

	return smartClips
}

// createSmartClips merges segments into complete thoughts with natural boundaries
func createSmartClips(segments []Segment) []Segment {
	if len(segments) == 0 {
		return segments
	}

	var smartClips []Segment
	i := 0

	for i < len(segments) {
		currentClip := segments[i]
		currentText := strings.TrimSpace(currentClip.Text)

		// Look ahead to merge incomplete thoughts
		j := i + 1
		for j < len(segments) {
			nextSegment := segments[j]
			nextText := strings.TrimSpace(nextSegment.Text)

			// Check if we should merge with next segment
			shouldMerge := false

			// Merge if current ends without punctuation and next is continuation
			if !endsWithCompletePunctuation(currentText) {
				shouldMerge = true
			}

			// Merge if current text is very short (likely incomplete)
			if len(strings.Fields(currentText)) < 4 {
				shouldMerge = true
			}

			// Merge if next starts with lowercase (continuation)
			if len(nextText) > 0 && nextText[0] >= 'a' && nextText[0] <= 'z' {
				shouldMerge = true
			}

			// Don't merge if the combined clip would be too long
			combinedDuration := nextSegment.EndTime - currentClip.StartTime
			if combinedDuration > 25*time.Second {
				break
			}

			// Don't merge if we've hit a clear topic change
			if isTopicChange(currentText, nextText) {
				break
			}

			if shouldMerge {
				// Merge the segments
				currentClip.EndTime = nextSegment.EndTime
				currentClip.Text = currentText + " " + nextText
				currentText = currentClip.Text
				j++
			} else {
				break
			}
		}

		// Add timing padding for natural speech boundaries
		currentClip = addNaturalPadding(currentClip)

		smartClips = append(smartClips, currentClip)
		i = j
	}

	return smartClips
}

func HandleVTTClipsCommand(args []string) {
	fs := flag.NewFlagSet("vtt-clips", flag.ExitOnError)
	var outputFile string

	fs.StringVar(&outputFile, "o", "", "Output file (default: <vtt-basename>_clips.fcpxml)")
	fs.StringVar(&outputFile, "output", "", "Output file (default: <vtt-basename>_clips.fcpxml)")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "Error: VTT file and timecodes required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s vtt-clips <vtt-file> <timecodes>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s vtt-clips IBnNedMh4Pg.en.vtt 01:21_6,02:20_3,03:34_9,05:07_18\n", os.Args[0])
		os.Exit(1)
	}

	vttFile := fs.Arg(0)
	timecodesStr := fs.Arg(1)

	if _, err := os.Stat(vttFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: VTT file '%s' does not exist\n", vttFile)
		os.Exit(1)
	}

	if err := GenerateVTTClips(vttFile, timecodesStr, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating VTT clips: %v\n", err)
		os.Exit(1)
	}
}
