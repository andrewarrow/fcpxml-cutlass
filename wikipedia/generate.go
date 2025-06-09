package wikipedia

import (
	"cutlass/fcp"
	"fmt"
)

func GenerateFromWikipedia(articleTitle, outputFile string) error {
	// Fetch Wikipedia source
	fmt.Printf("Fetching Wikipedia source for: %s\n", articleTitle)
	source, err := FetchWikipediaSource(articleTitle)
	if err != nil {
		return fmt.Errorf("failed to fetch Wikipedia source: %v", err)
	}

	// Parse the source to extract tables
	fmt.Printf("Parsing Wikipedia source for tables...\n")
	tables, err := ParseWikitableFromSource(source)
	if err != nil {
		return fmt.Errorf("failed to parse Wikipedia source: %v", err)
	}

	if len(tables) == 0 {
		return fmt.Errorf("no tables found in Wikipedia article")
	}

	// Select best table
	bestTable := SelectBestTable(tables)
	if bestTable == nil {
		return fmt.Errorf("no suitable table found")
	}

	fmt.Printf("Table headers: %v\n", bestTable.Headers)
	fmt.Printf("Table has %d rows\n", len(bestTable.Rows))

	// Convert the selected table to the structured TableData format
	tableData := &fcp.TableData{
		Headers: bestTable.Headers,
		Rows:    make([]fcp.TableRow, len(bestTable.Rows)),
	}

	for i, row := range bestTable.Rows {
		tableData.Rows[i] = fcp.TableRow{
			Cells: make([]fcp.TableCell, len(row)),
		}
		for j, cell := range row {
			tableData.Rows[i].Cells[j] = fcp.TableCell{
				Content: cell,
			}
		}
	}

	// Convert to fcp-compatible format
	fcpTable := &fcp.WikiSimpleTable{
		Headers: bestTable.Headers,
		Rows:    bestTable.Rows,
	}

	// Generate FCPXML with multiple table views
	fmt.Printf("Generating FCPXML: %s\n", outputFile)
	err = fcp.GenerateMultiTableFCPXML(fcpTable, outputFile)
	if err != nil {
		return fmt.Errorf("failed to generate FCPXML: %v", err)
	}

	fmt.Printf("Successfully generated Wikipedia table FCPXML: %s\n", outputFile)
	return nil
}
