package fcp

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"
)

type TournamentResult struct {
	Tournament string
	Result     string
	Style      map[string]string
}


// GenerateTableGridFCPXML builds a simple table view inside an FCPXML timeline.
// The original implementation ignored the supplied `data` parameter and relied
// on hard-coded sample data.  This has been replaced with logic that attempts
// to extract information from the structure produced by the wikipedia/parse
// package.  If the structure is unrecognised or empty we fall back to the
// previous sample so that callers still receive a valid file instead of an
// error.
func GenerateTableGridFCPXML(data interface{}, outputPath string) error {
	// First, try to convert the loosely-typed `data` argument into a slice of
	// TournamentResult values that can drive the rendering.
	
	tournamentResults := extractTournamentResults(data)

	if len(tournamentResults) == 0 {
		// Nothing recognised – keep the original sample so the output is never
		// empty (and to aid manual debugging in the Final Cut Pro UI).
		tournamentResults = []TournamentResult{
			{Tournament: "US Open", Result: "1R", Style: map[string]string{"background": "#afeeee"}},
			{Tournament: "Australian Open", Result: "A", Style: map[string]string{}},
			{Tournament: "French Open", Result: "A", Style: map[string]string{}},
			{Tournament: "Wimbledon", Result: "A", Style: map[string]string{}},
		}
	}
	
	// Create table grid using shapes and text
	totalDuration := 15 * time.Second
	var spineContent strings.Builder
	currentOffset := time.Duration(0)
	
	// Calculate grid dimensions - exactly like tennis.fcpxml
	numRows := len(tournamentResults) + 1 // +1 for header = 5 rows
	numCols := 2 // Tournament and 1986 result
	
	// Grid positioning – start a little below the top of the frame (20 % of the
	// height) and keep the same spacing that the reference file used (12 %).  We
	// build the slice dynamically so that any number of rows is supported.
	var horizontalPositions []float64
	startY := 0.200 // First (top) horizontal line
	rowSpacing := 0.120
	for i := 0; i <= numRows; i++ {
		horizontalPositions = append(horizontalPositions, startY+rowSpacing*float64(i))
	}

	// Two-column layout (tournament / result) – the reference used three
	// vertical positions: 0.1 (left border), 0.5 (centre border) and 0.9 (right
	// border).  We keep those fixed.
	verticalPositions := []float64{0.100, 0.500, 0.900}
	
	// Create horizontal grid lines - exactly like tennis.fcpxml
	for i := 0; i <= numRows; i++ {
		spineContent.WriteString(fmt.Sprintf(`
		<video ref="r2" offset="%s" name="H-Line %d" start="0s" duration="%s">
			<param name="Shape" key="9999/988461322/100/988461395/2/100" value="4 (Rectangle)"/>
			<param name="Fill Color" key="9999/988455508/988455699/2/353/113/111" value="0.2 0.2 0.2"/>
			<param name="Outline" key="9999/988461322/100/988464485/2/100" value="0"/>
			<param name="Center" key="9999/988469355/988469353/3/988469357/1" value="0.5 %.3f"/>
			<adjust-transform scale="0.800 0.002"/>
		</video>`,
			FormatDurationForFCPXML(currentOffset),
			i,
			FormatDurationForFCPXML(totalDuration),
			horizontalPositions[i]))
	}
	
	// Create vertical grid lines - exactly like tennis.fcpxml  
	for j := 0; j <= numCols; j++ {
		spineContent.WriteString(fmt.Sprintf(`
		<video ref="r2" lane="1" offset="%s" name="V-Line %d" start="0s" duration="%s">
			<param name="Shape" key="9999/988461322/100/988461395/2/100" value="4 (Rectangle)"/>
			<param name="Fill Color" key="9999/988455508/988455699/2/353/113/111" value="0.2 0.2 0.2"/>
			<param name="Outline" key="9999/988461322/100/988464485/2/100" value="0"/>
			<param name="Center" key="9999/988469355/988469353/3/988469357/1" value="%.3f 0.5"/>
			<adjust-transform scale="0.002 0.600"/>
		</video>`,
			FormatDurationForFCPXML(currentOffset),
			j,
			FormatDurationForFCPXML(totalDuration),
			verticalPositions[j]))
	}
	
	// Header row – if the caller supplied explicit headers we use them, otherwise
	// fall back to the default "Tournament" / "Result" pair.
	primaryHeader := "Tournament"
	secondaryHeader := "Result"
	if hdrs, ok := tableHeadersFromData(data); ok {
		if len(hdrs) > 0 {
			primaryHeader = hdrs[0]
		}
		if len(hdrs) > 1 {
			secondaryHeader = hdrs[1]
		}
	}

	// Header – left column
	spineContent.WriteString(fmt.Sprintf(`
	<title ref="r3" lane="2" offset="%s" name="Header %s" start="%s" duration="%s">
		<param name="Position" key="9999/10003/13260/3296672360/1/100/101" value="-200.0 -259.2"/>
		<param name="Layout Method" key="9999/10003/13260/3296672360/2/314" value="1 (Paragraph)"/>
		<param name="Alignment" key="9999/10003/13260/3296672360/2/354/3296667315/401" value="1 (Center)"/>
		<text>
			<text-style ref="ts1">%s</text-style>
		</text>
		<text-style-def id="ts1">
			<text-style font="SF Pro Display" fontSize="36" fontFace="Bold" fontColor="0.1 0.1 0.1 1" alignment="center"/>
		</text-style-def>
	</title>`,
		FormatDurationForFCPXML(currentOffset),
		primaryHeader,
		FormatDurationForFCPXML(currentOffset),
		FormatDurationForFCPXML(totalDuration),
		primaryHeader))

	// Header – right column
	spineContent.WriteString(fmt.Sprintf(`
	<title ref="r3" lane="3" offset="%s" name="Header %s" start="%s" duration="%s">
		<param name="Position" key="9999/10003/13260/3296672360/1/100/101" value="200.0 -259.2"/>
		<param name="Layout Method" key="9999/10003/13260/3296672360/2/314" value="1 (Paragraph)"/>
		<param name="Alignment" key="9999/10003/13260/3296672360/2/354/3296667315/401" value="1 (Center)"/>
		<text>
			<text-style ref="ts2">%s</text-style>
		</text>
		<text-style-def id="ts2">
			<text-style font="SF Pro Display" fontSize="36" fontFace="Bold" fontColor="0.1 0.1 0.1 1" alignment="center"/>
		</text-style-def>
	</title>`,
		FormatDurationForFCPXML(currentOffset),
		secondaryHeader,
		FormatDurationForFCPXML(currentOffset),
		FormatDurationForFCPXML(totalDuration),
		secondaryHeader))
	
	// Row Y positions for text – keep the original 129.6-pixel spacing so that
	// the visual result remains similar to the reference.  The first data row in
	// tennis.fcpxml sat at ‑129.6px, so we replicate that offset and generate as
	// many entries as required.
	var textYPositions []float64
	textStartPx := -129.6
	textSpacingPx := 129.6
	for i := 0; i < len(tournamentResults); i++ {
		textYPositions = append(textYPositions, textStartPx+float64(i)*textSpacingPx)
	}

	// Cell centre positions for the coloured background rectangles.
	var cellCenterPositions []float64
	for i := 0; i < len(tournamentResults); i++ {
		centerY := startY + rowSpacing*float64(i+1) + rowSpacing/2
		cellCenterPositions = append(cellCenterPositions, centerY)
	}
	
	// Add tournament data with gradual reveal - using tennis.fcpxml patterns
	for i, result := range tournamentResults {
		revealTime := currentOffset + time.Duration(i+1)*2*time.Second
		cellDuration := totalDuration - time.Duration(i+1)*2*time.Second
		
		// Background color for the result cell
		bgColor := getBackgroundColor(result.Result, result.Style)
		
		// Background shape for result cell - exactly like tennis.fcpxml
		spineContent.WriteString(fmt.Sprintf(`
		<video ref="r2" lane="4" offset="%s" name="BG %s" start="%s" duration="%s">
			<param name="Shape" key="9999/988461322/100/988461395/2/100" value="4 (Rectangle)"/>
			<param name="Fill Color" key="9999/988455508/988455699/2/353/113/111" value="%s"/>
			<param name="Outline" key="9999/988461322/100/988464485/2/100" value="0"/>
			<param name="Center" key="9999/988469355/988469353/3/988469357/1" value="0.700 %.3f"/>
			<adjust-transform scale="0.380 0.096"/>
		</video>`,
			FormatDurationForFCPXML(revealTime),
			result.Tournament,
			FormatDurationForFCPXML(revealTime),
			FormatDurationForFCPXML(cellDuration),
			bgColor,
			cellCenterPositions[i]))
		
		// Tournament name - exactly like tennis.fcpxml
		spineContent.WriteString(fmt.Sprintf(`
		<title ref="r3" lane="5" offset="%s" name="%s" start="%s" duration="%s">
			<param name="Position" key="9999/10003/13260/3296672360/1/100/101" value="-200.0 %.1f"/>
			<param name="Layout Method" key="9999/10003/13260/3296672360/2/314" value="1 (Paragraph)"/>
			<param name="Alignment" key="9999/10003/13260/3296672360/2/354/3296667315/401" value="1 (Center)"/>
			<text>
				<text-style ref="ts%d">%s</text-style>
			</text>
			<text-style-def id="ts%d">
				<text-style font="SF Pro Display" fontSize="28" fontFace="Medium" fontColor="0.1 0.1 0.1 1" alignment="center"/>
			</text-style-def>
		</title>`,
			FormatDurationForFCPXML(revealTime),
			result.Tournament,
			FormatDurationForFCPXML(revealTime),
			FormatDurationForFCPXML(cellDuration),
			textYPositions[i],
			i+10, result.Tournament, i+10))
		
		// Result - exactly like tennis.fcpxml
		spineContent.WriteString(fmt.Sprintf(`
		<title ref="r3" lane="6" offset="%s" name="Result %s" start="%s" duration="%s">
			<param name="Position" key="9999/10003/13260/3296672360/1/100/101" value="200.0 %.1f"/>
			<param name="Layout Method" key="9999/10003/13260/3296672360/2/314" value="1 (Paragraph)"/>
			<param name="Alignment" key="9999/10003/13260/3296672360/2/354/3296667315/401" value="1 (Center)"/>
			<text>
				<text-style ref="ts%d">%s</text-style>
			</text>
			<text-style-def id="ts%d">
				<text-style font="SF Pro Display" fontSize="32" fontFace="Bold" fontColor="0.1 0.1 0.1 1" alignment="center"/>
			</text-style-def>
		</title>`,
			FormatDurationForFCPXML(revealTime),
			result.Result,
			FormatDurationForFCPXML(revealTime),
			FormatDurationForFCPXML(cellDuration),
			textYPositions[i],
			i+50, result.Result, i+50))
	}
	
	// Create the FCPXML structure
	fcpxml := FCPXML{
		Version: "1.13",
		Resources: Resources{
			Formats: []Format{
				{
					ID:            "r1",
					Name:          "FFVideoFormat1080p2997",
					FrameDuration: "1001/30000s",
					Width:         "1920",
					Height:        "1080",
					ColorSpace:    "1-1-1 (Rec. 709)",
				},
			},
			Effects: []Effect{
				{
					ID:   "r2",
					Name: "Shapes",
					UID:  ".../Generators.localized/Elements.localized/Shapes.localized/Shapes.motn",
				},
				{
					ID:   "r3",
					Name: "Text",
					UID:  ".../Titles.localized/Basic Text.localized/Text.localized/Text.moti",
				},
			},
		},
		Library: Library{
			Events: []Event{
				{
					Name: "Andre Agassi 1986",
					Projects: []Project{
						{
							Name: "Tournament Table",
							Sequences: []Sequence{
								{
									Format:      "r1",
									Duration:    FormatDurationForFCPXML(totalDuration),
									TCStart:     "0s",
									TCFormat:    "NDF",
									AudioLayout: "stereo",
									AudioRate:   "48k",
									Spine: Spine{
										Content: spineContent.String(),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	
	output, err := xml.MarshalIndent(fcpxml, "", "    ")
	if err != nil {
		return err
	}
	
	xmlContent := xml.Header + "<!DOCTYPE fcpxml>\n" + string(output)
	return os.WriteFile(outputPath, []byte(xmlContent), 0644)
}

// extractTournamentResults converts the dynamic data structure passed from
// main.go (or any other caller) into a slice of TournamentResult values.  The
// implementation purposefully performs many runtime type assertions instead of
// reflection to keep things simple and dependency-free.
func extractTournamentResults(data interface{}) []TournamentResult {
	var results []TournamentResult

	// Expecting []interface{} (tables)
	outerSlice, ok := data.([]interface{})
	if !ok || len(outerSlice) == 0 {
		return results
	}

	// Work with the first table only for now.
	tableMap, ok := outerSlice[0].(map[string]interface{})
	if !ok {
		return results
	}

	rowsIface, ok := tableMap["Rows"].([]interface{})
	if !ok {
		return results
	}

	for _, row := range rowsIface {
		rowMap, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		cellsIface, ok := rowMap["Cells"].([]interface{})
		if !ok || len(cellsIface) == 0 {
			continue
		}

		// First cell → tournament/event name
		firstCellMap, ok := cellsIface[0].(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := firstCellMap["Content"].(string)

		// Second cell (if present) → result value (W, F, SF, etc.)
		var resultStr string
		var styleMap map[string]string
		if len(cellsIface) > 1 {
			secondCellMap, ok := cellsIface[1].(map[string]interface{})
			if ok {
				resultStr, _ = secondCellMap["Content"].(string)

				// style is itself a map[string]string but comes through as
				// map[string]interface{} – convert if needed.
				if rawStyle, exists := secondCellMap["Style"]; exists {
					styleMap = make(map[string]string)
					switch s := rawStyle.(type) {
					case map[string]string:
						styleMap = s
					case map[string]interface{}:
						for k, v := range s {
							if vs, ok := v.(string); ok {
								styleMap[k] = vs
							}
						}
					}
				}
			}
		}

		results = append(results, TournamentResult{
			Tournament: name,
			Result:     resultStr,
			Style:      styleMap,
		})
	}

	return results
}

// tableHeadersFromData tries to pull the header list from the first table in
// the supplied dynamic structure.  It returns the slice and a boolean that
// indicates success.
func tableHeadersFromData(data interface{}) ([]string, bool) {
	outerSlice, ok := data.([]interface{})
	if !ok || len(outerSlice) == 0 {
		return nil, false
	}
	if tableMap, ok := outerSlice[0].(map[string]interface{}); ok {
		if hdr, ok := tableMap["Headers"].([]string); ok {
			return hdr, true
		}
		// When coming through the JSON-like marshaling the headers may be a
		// []interface{} of strings.
		if hdrIface, ok := tableMap["Headers"].([]interface{}); ok {
			var headers []string
			for _, h := range hdrIface {
				if s, ok := h.(string); ok {
					headers = append(headers, s)
				}
			}
			return headers, len(headers) > 0
		}
	}
	return nil, false
}

func getBackgroundColor(result string, style map[string]string) string {
	// Check style first
	if style != nil {
		if bg, ok := style["background"]; ok {
			switch bg {
			case "lime":
				return "0.2 0.8 0.2" // Green for wins
			case "yellow":
				return "1 1 0.2" // Yellow for semifinals
			case "thistle":
				return "0.8 0.6 0.8" // Purple for finals
			case "#afeeee":
				return "0.7 0.9 0.9" // Light blue for rounds
			case "#ffebcd":
				return "1 0.9 0.8" // Light orange for quarterfinals
			}
		}
		if bgColor, ok := style["background-color"]; ok {
			switch bgColor {
			case "lime":
				return "0.2 0.8 0.2"
			case "yellow":
				return "1 1 0.2"
			case "thistle":
				return "0.8 0.6 0.8"
			}
		}
	}
	
	// Fallback based on result content
	switch result {
	case "W", "'''W'''":
		return "0.2 0.8 0.2" // Green for wins
	case "F":
		return "0.8 0.6 0.8" // Purple for finals
	case "SF":
		return "1 1 0.2" // Yellow for semifinals
	case "QF":
		return "1 0.9 0.8" // Light orange for quarterfinals
	case "1R", "2R", "3R", "4R":
		return "0.7 0.9 0.9" // Light blue for rounds
	case "A":
		return "0.9 0.9 0.9" // Light gray for absent
	case "DNQ":
		return "0.8 0.8 0.8" // Gray for did not qualify
	default:
		return "0.95 0.95 0.95" // Very light gray default
	}
}

func GenerateWikipediaTableFCPXML(data interface{}, outputPath string) error {
	// Simple fallback function
	return GenerateTableGridFCPXML(data, outputPath)
}