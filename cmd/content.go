package cmd

import (
	"cutlass/time"

	"github.com/spf13/cobra"
)

var contentCmd = &cobra.Command{
	Use:   "content",
	Short: "Text processing and content generation",
	Long:  "Commands for processing text files, speech, and time-based content.",
}

func init() {
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
	contentCmd.AddCommand(timeCmd)
}
