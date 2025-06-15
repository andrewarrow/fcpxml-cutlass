package cmd

import (
	"cutlass/genvoices"
	"cutlass/genvideo"
	"cutlass/resume"
	"cutlass/utils"

	"github.com/spf13/cobra"
)

var utilsCmd = &cobra.Command{
	Use:   "utils",
	Short: "Utility commands",
	Long:  "Miscellaneous utility commands for various tasks.",
}

var resumeCmd = &cobra.Command{
	Use:   "resume <file>",
	Short: "Take screenshots of domains found in resume file",
	Long: `Take screenshots of domains found in resume file.
Looks for lines with two spaces containing domains like bizrate.com.
Saves screenshots to ./assets/ as domain.png`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resume.HandleResumeCommand(args)
		return nil
	},
}

var genvoicesCmd = &cobra.Command{
	Use:   "genvoices <file>",
	Short: "Generate voice files from sentences in a words file",
	Long: `Generate voice files from sentences in a words file.
Processes .words files looking for '- Sentences:' sections and generates
audio files for each sentence using chatterbox Python script.
Audio files are saved as ./data/<basename>_audio/s<section>_s<sentence>.wav`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		genvoices.HandleGenVoicesCommand(args)
		return nil
	},
}

var genvideoCmd = &cobra.Command{
	Use:   "genvideo <file.genvideo>",
	Short: "Generate FCPXML from .genvideo file with audio, frames, and text overlays",
	Long: `Generate FCPXML from a .genvideo file specification.

The .genvideo file format:
Line 1: audio_file.wav (main audio track for entire video)
Following lines: comma-separated segments with frames and text groups
Example:
  audio_track.wav
  001_frame.jpg, 002_frame.jpg, "text1", "text2", "text3"
  003_frame.jpg, 004_frame.jpg, "text1a", "text2a", "text3a"

Creates an FCPXML with:
- Single audio track for the entire duration
- Video segments with specified frame sequences
- Staggered text overlays on 3 lanes (3, 2, 1) matching antiedit.fcpxml style
- Frame-aligned timing and unique text style IDs`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		genvideo.HandleGenVideoCommand(args)
		return nil
	},
}

var genaudioCmd = &cobra.Command{
	Use:   "genaudio <file.txt>",
	Short: "Generate audio files from simple text file (one sentence per line)",
	Long: `Generate audio files from a simple text file format.

The input file should have one sentence per line. Empty lines are skipped.
Uses the filename (without extension) as the video ID.

Example with waymo.txt:
- Creates ./data/waymo_audio/ directory
- Generates s1_duration.wav, s2_duration.wav, etc.
- Duration is automatically detected and added to filename

Audio files are generated using chatterbox TTS and skip existing files.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		utils.HandleGenAudioCommand(args)
		return nil
	},
}

func init() {
	utilsCmd.AddCommand(resumeCmd)
	utilsCmd.AddCommand(genvoicesCmd)
	utilsCmd.AddCommand(genvideoCmd)
	utilsCmd.AddCommand(genaudioCmd)
}