package cmd

import (
	"cutlass/speech"
	"cutlass/time"

	"github.com/spf13/cobra"
)

var contentCmd = &cobra.Command{
	Use:   "content",
	Short: "Text processing and content generation",
	Long:  "Commands for processing text files, speech, and time-based content.",
}

var speechCmd = &cobra.Command{
	Use:   "speech <text-file> <video-or-image-file>",
	Short: "Generate FCPXML with multiple text elements appearing over time",
	Long: `Generate FCPXML with slide animation where each line from text file appears over time.
The video or image file will be used as background media for the text animations.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		speech.HandleSpeechCommand(args)
		return nil
	},
}

var speechResumeCmd = &cobra.Command{
	Use:   "resume <resume-file>",
	Short: "Generate FCPXML with multiple image/text pairs from resume file",
	Long: `Generate FCPXML with multiple image/text pairs from resume file.
The resume file should contain PNG filenames followed by their associated text lines.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFile, _ := cmd.Flags().GetString("output")
		speech.HandleResumeCommandWithOutput(args, outputFile)
		return nil
	},
}

func init() {
	speechResumeCmd.Flags().StringP("output", "o", "data/resume.fcpxml", "Output file")
}

var timeCmd = &cobra.Command{
	Use:   "time <time-file>",
	Short: "Generate FCPXML from .time format file",
	Long:  "Generate FCPXML from .time format file containing video paths and timed slide animations with text elements.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		time.HandleTimeCommand(args)
		return nil
	},
}

func init() {
	contentCmd.AddCommand(speechCmd)
	contentCmd.AddCommand(speechResumeCmd)
	contentCmd.AddCommand(timeCmd)
}