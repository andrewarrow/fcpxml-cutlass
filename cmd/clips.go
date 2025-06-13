package cmd

import (
	"cutlass/segments"
	"cutlass/vtt"

	"github.com/spf13/cobra"
)

var clipsCmd = &cobra.Command{
	Use:   "clips",
	Short: "Segment creation and VTT handling",
	Long:  "Commands for creating video segments and handling VTT subtitle files.",
}

var vttCmd = &cobra.Command{
	Use:   "vtt <file>",
	Short: "Parse VTT file and display cleaned text",
	Long:  "Parse a VTT subtitle file and display the cleaned text content.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vtt.HandleVTTCommand(args)
		return nil
	},
}

var vttClipsCmd = &cobra.Command{
	Use:   "vtt-clips <vtt-file> <timecodes>",
	Short: "Create FCPXML clips from VTT file at specified timecodes",
	Long: `Create FCPXML clips from VTT file at specified timecodes.
Timecodes can be MM:SS or MM:SS_duration format.
Example: 01:21_6,02:20_3,03:34_9,05:07_18`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		vtt.HandleVTTClipsCommand(args)
		return nil
	},
}

var segmentsCmd = &cobra.Command{
	Use:   "segments <video-id> <timecodes>",
	Short: "Create FCPXML clips from video ID at specified timecodes",
	Long: `Create FCPXML clips from video ID in ./data/ at specified timecodes.
Similar to vtt-clips but looks for video_id in ./data/id.mov`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		segments.HandleSegmentsCommand(args)
		return nil
	},
}

func init() {
	clipsCmd.AddCommand(vttCmd)
	clipsCmd.AddCommand(vttClipsCmd)
	clipsCmd.AddCommand(segmentsCmd)
}