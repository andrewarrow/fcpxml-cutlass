package cmd

import (
	"fmt"
	"os"

	"cutlass/fcp"

	"github.com/spf13/cobra"
)

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Analysis and inspection tools",
	Long:  "Commands for parsing and analyzing FCPXML files.",
}

var fcpxmlCmd = &cobra.Command{
	Use:   "fcpxml <file>",
	Short: "Parse and display FCPXML contents",
	Long:  "Parse an FCPXML file and display its contents with various detail levels.",
	Args:  cobra.ExactArgs(1),
	RunE:  runParseCommand,
}

var tier int
var showElements, showParams, showAnimation, showResources, showStructure bool

func init() {
	parseCmd.AddCommand(fcpxmlCmd)
	
	fcpxmlCmd.Flags().IntVarP(&tier, "tier", "t", 1, "Display tier (1=core, 2=advanced, 3=detailed)")
	fcpxmlCmd.Flags().BoolVarP(&showElements, "elements", "e", false, "Show story elements (clips, titles, effects)")
	fcpxmlCmd.Flags().BoolVarP(&showParams, "params", "p", false, "Show parameters and keyframes")
	fcpxmlCmd.Flags().BoolVarP(&showAnimation, "animation", "a", false, "Show animation and keyframe details")
	fcpxmlCmd.Flags().BoolVarP(&showResources, "resources", "r", false, "Show detailed resource information")
	fcpxmlCmd.Flags().BoolVarP(&showStructure, "structure", "s", false, "Show XML structure hierarchy")
}

func runParseCommand(cmd *cobra.Command, args []string) error {
	inputFile := args[0]
	
	// Check if file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("file '%s' does not exist", inputFile)
	}

	parseOptions := fcp.ParseOptions{
		Tier:          tier,
		ShowElements:  showElements,
		ShowParams:    showParams,
		ShowAnimation: showAnimation,
		ShowResources: showResources,
		ShowStructure: showStructure,
	}

	if err := parseFCPXMLWithOptions(inputFile, parseOptions); err != nil {
		return fmt.Errorf("error parsing FCPXML: %v", err)
	}
	
	return nil
}

func parseFCPXMLWithOptions(filePath string, options fcp.ParseOptions) error {
	fcpxml, err := fcp.ParseFCPXML(filePath)
	if err != nil {
		return err
	}

	fcp.DisplayFCPXMLWithOptions(fcpxml, options)
	return nil
}