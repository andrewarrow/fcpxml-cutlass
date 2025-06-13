package cmd

import (
	"cutlass/resume"

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

func init() {
	utilsCmd.AddCommand(resumeCmd)
}