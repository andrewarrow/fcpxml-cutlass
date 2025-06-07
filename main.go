package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var inputFile string
	flag.StringVar(&inputFile, "i", "", "Input file (required)")
	flag.Parse()

	args := flag.Args()
	if inputFile == "" || len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s -i <input_file> <output_file>\n", os.Args[0])
		os.Exit(1)
	}

	outputFile := args[0]
	if !strings.HasSuffix(strings.ToLower(outputFile), ".fcpxml") {
		outputFile += ".fcpxml"
	}

	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file '%s' does not exist\n", inputFile)
		os.Exit(1)
	}

	if err := generateFCPXML(inputFile, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating FCPXML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully converted '%s' to '%s'\n", inputFile, outputFile)
}

func generateFCPXML(inputFile, outputFile string) error {
	inputName := filepath.Base(inputFile)
	inputExt := strings.ToLower(filepath.Ext(inputFile))
	nameWithoutExt := strings.TrimSuffix(inputName, inputExt)

	fcpxml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>
<fcpxml version="1.11">
	<resources>
		<format id="r1" name="FFVideoFormat1080p30" frameDuration="1001/30000s" width="1920" height="1080" colorSpace="1-1-1 (Rec. 709)"/>
		<asset id="r2" name="%s" uid="%s" start="0s" hasVideo="1" format="r1" hasAudio="1" audioSources="1" audioChannels="2" duration="3600s">
			<media-rep kind="original-media" sig="%s" src="file://%s"/>
		</asset>
	</resources>
	<library>
		<event name="Converted Media">
			<project name="%s">
				<sequence format="r1" duration="3600s" tcStart="0s" tcFormat="NDF" audioLayout="stereo" audioRate="48k">
					<spine>
						<asset-clip ref="r2" offset="0s" name="%s" duration="3600s" format="r1" tcFormat="NDF">
						</asset-clip>
					</spine>
				</sequence>
			</project>
		</event>
	</library>
</fcpxml>`, nameWithoutExt, inputFile, inputFile, inputFile, nameWithoutExt, nameWithoutExt)

	return os.WriteFile(outputFile, []byte(fcpxml), 0644)
}
