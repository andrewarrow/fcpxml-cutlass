package fcp

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GenerateClipFCPXML(clips []Clip, videoPath, outputPath string) error {
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

func BuildClipFCPXML(clips []Clip, videoPath string) (FCPXML, error) {
	absVideoPath, err := filepath.Abs(videoPath)
	if err != nil {
		return FCPXML{}, err
	}

	videoName := filepath.Base(absVideoPath)
	nameWithoutExt := strings.TrimSuffix(videoName, filepath.Ext(videoName))

	// Calculate total duration - just video clips
	var totalDuration time.Duration
	for _, clip := range clips {
		totalDuration += clip.Duration
	}

	var spineContent strings.Builder
	currentOffset := time.Duration(0)

	for _, clip := range clips {
		// Video clip
		assetClip := AssetClip{
			Ref:       "r3",
			Offset:    FormatDurationForFCPXML(currentOffset),
			Name:      fmt.Sprintf("%s Clip %d", nameWithoutExt, clip.ClipNum),
			Start:     FormatDurationForFCPXML(clip.StartTime),
			Duration:  FormatDurationForFCPXML(clip.Duration),
			Format:    "r1",
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
		Version: "1.13",
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
