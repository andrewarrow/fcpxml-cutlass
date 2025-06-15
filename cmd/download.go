package cmd

import (
	"cutlass/youtube"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download content from external sources",
	Long:  "Commands for downloading content from YouTube, Wikipedia, and other sources.",
}

var youtubeCmd = &cobra.Command{
	Use:   "youtube <video-id>",
	Short: "Download YouTube video and generate FCPXML",
	Long:  "Download a YouTube video by ID and generate FCPXML from it.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		youtube.HandleYouTubeCommand(args)
		return nil
	},
}

func init() {
	downloadCmd.AddCommand(youtubeCmd)
}
