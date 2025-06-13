package build

import (
	"encoding/xml"
	"fmt"
	"os"

	"cutlass/fcp"
)

// parseXML parses XML content into the provided struct
func parseXML(content []byte, fcpxml *fcp.FCPXML) error {
	return xml.Unmarshal(content, fcpxml)
}

// marshalXML marshals a struct to XML
func marshalXML(v interface{}) ([]byte, error) {
	return xml.Marshal(v)
}

// writeProjectFile writes an FCPXML struct to a file with proper formatting
func writeProjectFile(filename string, fcpxml *fcp.FCPXML) error {
	// Generate the XML output with proper formatting
	output, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %v", err)
	}
	
	// Add XML declaration and DOCTYPE
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

` + string(output)
	
	// Write to output file
	err = os.WriteFile(filename, []byte(xmlContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}
	
	return nil
}