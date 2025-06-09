package fcp

import (
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"
	"bytes"
)

type TemplateVideo struct {
	ID string
}

type NumberSection struct {
	Number  int
	VideoID string
	Offset  string
}

type TemplateData struct {
	FirstName       string
	LastName        string
	LastNameSuffix  string
	Videos          []TemplateVideo
	Numbers         []NumberSection
}

func GenerateTop5FCPXML(templatePath string, videoIDs []string, name, outputPath string) error {
	// Parse the name
	nameWords := strings.Fields(name)
	firstName := ""
	lastName := ""
	lastNameSuffix := ""
	
	if len(nameWords) >= 1 {
		firstName = nameWords[0]
	}
	if len(nameWords) >= 2 {
		lastName = nameWords[1]
		// Check if the original template had a suffix (like the "g" in "Dimoldenber")
		if lastName == "Dimoldenber" {
			lastNameSuffix = "g"
		}
	}

	// Create video data
	videos := make([]TemplateVideo, len(videoIDs))
	for i, id := range videoIDs {
		videos[i] = TemplateVideo{ID: id}
	}

	// Create number sections (5, 4, 3, 2, 1)
	numbers := make([]NumberSection, 5)
	offsets := []string{
		"8300/2500s",        // NUMBER 5
		"3827200/320000s",   // NUMBER 4  
		"8300/2500s",        // NUMBER 3 (same as 5)
		"3827200/320000s",   // NUMBER 2 (same as 4)
		"8300/2500s",        // NUMBER 1 (same as 5)
	}
	
	for i := 0; i < 5; i++ {
		numbers[i] = NumberSection{
			Number:  5 - i, // 5, 4, 3, 2, 1
			VideoID: videoIDs[i%len(videoIDs)], // Cycle through available videos
			Offset:  offsets[i],
		}
	}

	// Create template data
	data := TemplateData{
		FirstName:      firstName,
		LastName:       lastName,
		LastNameSuffix: lastNameSuffix,
		Videos:         videos,
		Numbers:        numbers,
	}

	// Create template functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"mul": func(a, b int) int { return a * b },
	}

	// Parse the template
	tmpl, err := template.New("top5.fcpxml").Funcs(funcMap).ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Execute the template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	// Write the result to the output file
	err = ioutil.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}

	return nil
}

