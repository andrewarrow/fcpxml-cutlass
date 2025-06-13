package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cutlass/fcp"
)

// createBlankProject creates a new blank FCPXML project from the empty.fcpxml template
func createBlankProject(filename string) error {
	// Read the empty.fcpxml template
	emptyContent, err := os.ReadFile("empty.fcpxml")
	if err != nil {
		return fmt.Errorf("failed to read empty.fcpxml: %v", err)
	}
	
	// Parse the XML to modify timestamps and UIDs
	var fcpxml fcp.FCPXML
	err = parseXML(emptyContent, &fcpxml)
	if err != nil {
		return fmt.Errorf("failed to parse empty.fcpxml: %v", err)
	}
	
	// Update timestamps and generate new UIDs
	currentTime := time.Now().Format("2006-01-02 15:04:05 -0700")
	
	if len(fcpxml.Library.Events) > 0 {
		// Update event name to current date
		fcpxml.Library.Events[0].Name = time.Now().Format("1-2-06")
		
		if len(fcpxml.Library.Events[0].Projects) > 0 {
			// Update project modification date
			fcpxml.Library.Events[0].Projects[0].ModDate = currentTime
			
			// Extract base filename without extension
			baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
			fcpxml.Library.Events[0].Projects[0].Name = baseName
		}
	}
	
	// Write to output file
	return writeProjectFile(filename, &fcpxml)
}