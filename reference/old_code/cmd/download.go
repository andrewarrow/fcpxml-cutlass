package cmd

import (
	"cutlass/hackernews"
	"cutlass/wikipedia"
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

var wikipediaRandomCmd = &cobra.Command{
	Use:   "wikipedia-random",
	Short: "Download random Wikipedia page and Google image search",
	Long:  "Navigate to a random Wikipedia page, extract the title, perform Google image search, and save screenshot.",
	RunE: func(cmd *cobra.Command, args []string) error {
		max, _ := cmd.Flags().GetInt("max")
		wikipedia.HandleWikipediaRandomCommand(args, max)
		return nil
	},
}

var hnStep1Cmd = &cobra.Command{
	Use:   "hn-step-1 [newest]",
	Short: "Step 1: Get Hacker News articles and download thumbnails",
	Long:  "Fetch article titles from Hacker News and download video thumbnails. Use 'newest' argument to fetch from /newest page instead of homepage.",
	RunE: func(cmd *cobra.Command, args []string) error {
		hackernews.HandleHackerNewsStep1Command(args)
		return nil
	},
}

var hnStep2Cmd = &cobra.Command{
	Use:   "hn-step-2",
	Short: "Step 2: Generate audio files for Hacker News articles",
	Long:  "Generate audio files from article titles and create FCPXML output.",
	RunE: func(cmd *cobra.Command, args []string) error {
		hackernews.HandleHackerNewsStep2Command(args)
		return nil
	},
}

func init() {
	downloadCmd.AddCommand(youtubeCmd)
	downloadCmd.AddCommand(youtubeBulkCmd)
	downloadCmd.AddCommand(youtubeBulkAssembleCmd)
	downloadCmd.AddCommand(wikipediaCmd)
	downloadCmd.AddCommand(tableCmd)
	downloadCmd.AddCommand(wikipediaRandomCmd)
	downloadCmd.AddCommand(hnStep1Cmd)
	downloadCmd.AddCommand(hnStep2Cmd)
	
	// Add flags
	wikipediaRandomCmd.Flags().IntP("max", "m", 10, "Maximum number of articles to process")
}
