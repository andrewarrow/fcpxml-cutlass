package cmd

import (
	"cutlass/genvoices"
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

func init() {
	utilsCmd.AddCommand(resumeCmd)
	utilsCmd.AddCommand(genvoicesCmd)
}