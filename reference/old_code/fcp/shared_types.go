package fcp

import "time"

// Clip represents a video clip with timing information
type Clip struct {
	StartTime        time.Duration
	EndTime          time.Duration
	Duration         time.Duration
	Text             string
	FirstSegmentText string // Just the first VTT segment for previews
	ClipNum          int
}

// TableData represents the data structure for table generation
type TableData struct {
	Headers []string    `json:"headers"`
	Rows    []TableRow  `json:"rows"`
}

// TableRow represents a row in the table
type TableRow struct {
	Cells []TableCell `json:"cells"`
}

// TableCell represents a cell in the table
type TableCell struct {
	Content    string            `json:"content"`
	Style      map[string]string `json:"style"`
	Class      string            `json:"class"`
	ColSpan    int               `json:"colspan"`
	RowSpan    int               `json:"rowspan"`
	Attributes map[string]string `json:"attributes"`
}

// ConvertToTableData converts wikipedia data to the expected table format
func ConvertToTableData(tables []interface{}) *TableData {
	if len(tables) == 0 {
		return &TableData{
			Headers: []string{"Column 1", "Column 2"},
			Rows:    []TableRow{},
		}
	}
	
	tableMap, ok := tables[0].(map[string]interface{})
	if !ok {
		return &TableData{
			Headers: []string{"Column 1", "Column 2"},
			Rows:    []TableRow{},
		}
	}
	
	// Extract headers
	var headers []string
	if hdrIface, ok := tableMap["Headers"].([]string); ok {
		headers = hdrIface
	} else if hdrSlice, ok := tableMap["Headers"].([]interface{}); ok {
		for _, h := range hdrSlice {
			if s, ok := h.(string); ok {
				headers = append(headers, s)
			}
		}
	}
	
	if len(headers) == 0 {
		headers = []string{"Column 1", "Column 2"}
	}
	
	// Extract rows
	var rows []TableRow
	if rowsIface, ok := tableMap["Rows"].([]interface{}); ok {
		for _, rowIface := range rowsIface {
			if rowMap, ok := rowIface.(map[string]interface{}); ok {
				if cellsIface, ok := rowMap["Cells"].([]interface{}); ok {
					var cells []TableCell
					for _, cellIface := range cellsIface {
						if cellMap, ok := cellIface.(map[string]interface{}); ok {
							cell := TableCell{
								ColSpan: 1,
								RowSpan: 1,
								Style:   make(map[string]string),
								Attributes: make(map[string]string),
							}
							
							if content, ok := cellMap["Content"].(string); ok {
								cell.Content = content
							}
							if class, ok := cellMap["Class"].(string); ok {
								cell.Class = class
							}
							if colspan, ok := cellMap["ColSpan"].(int); ok {
								cell.ColSpan = colspan
							}
							if rowspan, ok := cellMap["RowSpan"].(int); ok {
								cell.RowSpan = rowspan
							}
							
							// Handle style map
							if styleIface, ok := cellMap["Style"]; ok {
								switch s := styleIface.(type) {
								case map[string]string:
									cell.Style = s
								case map[string]interface{}:
									for k, v := range s {
										if vs, ok := v.(string); ok {
											cell.Style[k] = vs
										}
									}
								}
							}
							
							// Handle attributes map
							if attrIface, ok := cellMap["Attributes"]; ok {
								switch a := attrIface.(type) {
								case map[string]string:
									cell.Attributes = a
								case map[string]interface{}:
									for k, v := range a {
										if vs, ok := v.(string); ok {
											cell.Attributes[k] = vs
										}
									}
								}
							}
							
							cells = append(cells, cell)
						}
					}
					rows = append(rows, TableRow{Cells: cells})
				}
			}
		}
	}
	
	return &TableData{
		Headers: headers,
		Rows:    rows,
	}
}