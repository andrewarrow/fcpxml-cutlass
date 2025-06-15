package cmd

import (
	"cutlass/utils"

	"github.com/spf13/cobra"
)

var utilsCmd = &cobra.Command{
	Use:   "utils",
	Short: "Utility commands",
	Long:  "Miscellaneous utility commands for various tasks.",
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
	utilsCmd.AddCommand(genaudioCmd)
}
