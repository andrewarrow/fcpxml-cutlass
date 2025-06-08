package fcp

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cutlass/vtt"
)

func GenerateClipFCPXML(clips []vtt.Clip, videoPath, outputPath string) error {
	fcpxml, err := BuildClipFCPXML(clips, videoPath)
	if err != nil {
		return err
	}

	output, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		return err
	}

	xmlContent := xml.Header + "<!DOCTYPE fcpxml>\n" + string(output)
	return os.WriteFile(outputPath, []byte(xmlContent), 0644)
}

func BuildClipFCPXML(clips []vtt.Clip, videoPath string) (FCPXML, error) {
	absVideoPath, err := filepath.Abs(videoPath)
	if err != nil {
		return FCPXML{}, err
	}

	videoName := filepath.Base(absVideoPath)
	nameWithoutExt := strings.TrimSuffix(videoName, filepath.Ext(videoName))

	// Calculate total duration - textblock, clip, textblock, clip pattern
	var totalDuration time.Duration
	for _, clip := range clips {
		totalDuration += clip.Duration + 10*time.Second // Add 10s for textblock
	}
	totalDuration += 10 * time.Second // Add final textblock

	var spineContent strings.Builder
	currentOffset := time.Duration(0)

	for i, clip := range clips {
		// Textblock gap before each clip - show just the first segment of what's coming next
		clipText := clip.FirstSegmentText

		escapedText := escapeXMLText(clipText)

		// Create gap with title element
		gap := Gap{
			Name:     "Gap",
			Offset:   FormatDurationForFCPXML(currentOffset),
			Duration: FormatDurationForFCPXML(10 * time.Second),
			Titles: []Title{
				{
					Ref:      "r2",
					Lane:     "1",
					Offset:   FormatDurationForFCPXML(360 * time.Millisecond),
					Name:     "Graphic Text Block",
					Start:    FormatDurationForFCPXML(360 * time.Millisecond),
					Duration: FormatDurationForFCPXML(10*time.Second - 133*time.Millisecond),
					Text: &TitleText{
						TextStyle: TextStyleRef{
							Ref:  fmt.Sprintf("ts%d", i+1),
							Text: escapedText,
						},
					},
					TextStyleDef: &TextStyleDef{
						ID: fmt.Sprintf("ts%d", i+1),
						TextStyle: TextStyle{
							Font:      "Helvetica Neue",
							FontSize:  "176.8",
							FontColor: "1 1 1 1",
						},
					},
				},
			},
		}

		gapXML, err := xml.Marshal(gap)
		if err != nil {
			return FCPXML{}, fmt.Errorf("error marshaling gap XML: %v", err)
		}
		spineContent.Write(gapXML)

		currentOffset += 10 * time.Second

		// Video clip
		assetClip := AssetClip{
			Ref:       "r3",
			Offset:    FormatDurationForFCPXML(currentOffset),
			Name:      fmt.Sprintf("%s Clip %d", nameWithoutExt, clip.ClipNum),
			Start:     FormatDurationForFCPXML(clip.StartTime),
			Duration:  FormatDurationForFCPXML(clip.Duration),
			TCFormat:  "NDF",
			AudioRole: "dialogue",
		}

		clipXML, err := xml.Marshal(assetClip)
		if err != nil {
			return FCPXML{}, fmt.Errorf("error marshaling asset clip XML: %v", err)
		}
		spineContent.Write(clipXML)

		currentOffset += clip.Duration
	}

	return FCPXML{
		Version: "1.11",
		Resources: Resources{
			Formats: []Format{
				{
					ID:            "r1",
					Name:          "FFVideoFormat1080p30",
					FrameDuration: "1001/30000s",
					Width:         "1920",
					Height:        "1080",
					ColorSpace:    "1-1-1 (Rec. 709)",
				},
			},
			Assets: []Asset{
				{
					ID:            "r3",
					Name:          nameWithoutExt,
					UID:           absVideoPath,
					Start:         "0s",
					HasVideo:      "1",
					Format:        "r1",
					HasAudio:      "1",
					AudioSources:  "1",
					AudioChannels: "2",
					Duration:      FormatDurationForFCPXML(totalDuration),
					MediaRep: MediaRep{
						Kind: "original-media",
						Sig:  absVideoPath,
						Src:  "file://" + absVideoPath,
					},
				},
			},
			Effects: []Effect{
				{
					ID:   "r2",
					Name: "Graphic Text Block",
					UID:  ".../Titles.localized/Basic Text.localized/Graphic Text Block.localized/Graphic Text Block.moti",
				},
			},
		},
		Library: Library{
			Events: []Event{
				{
					Name: "Auto Generated Clips",
					Projects: []Project{
						{
							Name: nameWithoutExt + " Clips",
							Sequences: []Sequence{
								{
									Format:      "r1",
									Duration:    FormatDurationForFCPXML(totalDuration),
									TCStart:     "0s",
									TCFormat:    "NDF",
									AudioLayout: "stereo",
									AudioRate:   "48k",
									Spine: Spine{
										Content: spineContent.String(),
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}
