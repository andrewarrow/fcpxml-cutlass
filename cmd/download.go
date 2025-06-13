package cmd

import (
	"cutlass/youtube"
	"cutlass/wikipedia"

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

var youtubeBulkCmd = &cobra.Command{
	Use:   "youtube-bulk <ids-file>",
	Short: "Download multiple YouTube videos from file",
	Long:  "Download multiple YouTube videos from a file containing video IDs.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		youtube.HandleYouTubeBulkCommand(args)
		return nil
	},
}

var youtubeBulkAssembleCmd = &cobra.Command{
	Use:   "youtube-bulk-assemble <ids-file> <name>",
	Short: "Create top5.fcpxml from downloaded videos",
	Long:  "Create a top5.fcpxml file from previously downloaded YouTube videos.",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		youtube.HandleYouTubeBulkAssembleCommand(args)
		return nil
	},
}

var wikipediaCmd = &cobra.Command{
	Use:   "wikipedia <article-title>",
	Short: "Generate FCPXML from Wikipedia tables",
	Long:  "Download Wikipedia article and generate FCPXML from its tables.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wikipedia.HandleWikipediaCommand(args)
		return nil
	},
}

var tableCmd = &cobra.Command{
	Use:   "table <article-title>",
	Short: "Display Wikipedia table data",
	Long:  "Display table data from a Wikipedia article.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wikipedia.HandleTableCommand(args)
		return nil
	},
}

func init() {
	downloadCmd.AddCommand(youtubeCmd)
	downloadCmd.AddCommand(youtubeBulkCmd)
	downloadCmd.AddCommand(youtubeBulkAssembleCmd)
	downloadCmd.AddCommand(wikipediaCmd)
	downloadCmd.AddCommand(tableCmd)
}