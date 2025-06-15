package clips

import (
	"encoding/xml"
	"html"
	"path/filepath"
	"strings"

	"cutlass/fcp"
)

// ClipStrategy defines how clips should be created based on content and context
type ClipStrategy interface {
	ShouldCreateCompoundClip(video, audio string, context TimelineContext) bool
	CreateOptimalClip(video, audio, text string, config ClipConfig) TimelineElement
}

// TimelineContext provides context about the current timeline state
type TimelineContext struct {
	HasExistingCompoundClips bool
	ClipCount               int
	TotalDuration           string
}

// ClipConfig contains all parameters needed to create a clip
type ClipConfig struct {
	VideoAssetID    string
	AudioAssetID    string
	MediaID         string
	TextEffectID    string
	BaseName        string
	Duration        string
	AudioDuration   string
	Offset          string
	Text            string
	WithSlide       bool
}

// TimelineElement represents any element that can be added to a timeline
type TimelineElement interface {
	GetXML() string
	GetDuration() string
	GetOffset() string
}

// SmartClipStrategy implements intelligent clip type selection
type SmartClipStrategy struct{}

// NewSmartClipStrategy creates a new smart clip strategy
func NewSmartClipStrategy() *SmartClipStrategy {
	return &SmartClipStrategy{}
}

// ShouldCreateCompoundClip determines if a compound clip should be created
func (s *SmartClipStrategy) ShouldCreateCompoundClip(video, audio string, context TimelineContext) bool {
	// Only create compound clips when necessary
	if audio == "" {
		return false // Simple video element for video-only clips
	}
	
	// For image files with audio, use video elements with nested audio clips
	// This matches the structure used in Info.fcpxml
	if isImageFile(video) && audio != "" {
		return false
	}
	
	// For video files with audio, we can use simple asset-clips in most cases
	// Only use compound clips if we need complex audio mixing
	return false
}

// CreateOptimalClip creates the best clip type for the given parameters
func (s *SmartClipStrategy) CreateOptimalClip(video, audio, text string, config ClipConfig) TimelineElement {
	if audio == "" {
		// Video-only clip
		if isImageFile(video) {
			return s.createVideoElement(config)
		} else {
			return s.createAssetClip(config)
		}
	}
	
	// Clip with audio - prefer direct audio lanes over compound clips
	return s.createVideoWithAudioLane(config)
}

// createVideoElement creates a video element (for PNG files)
func (s *SmartClipStrategy) createVideoElement(config ClipConfig) TimelineElement {
	video := &VideoElement{
		Ref:      config.VideoAssetID,
		Offset:   config.Offset,
		Name:     config.BaseName,
		Start:    "0s",
		Duration: config.Duration,
		WithSlide: config.WithSlide,
	}
	
	// Add text overlay if requested
	if config.Text != "" {
		video.HasText = true
		video.TextEffectID = config.TextEffectID
		video.Text = config.Text
		video.HasAnimation = true
	}
	
	return video
}

// createAssetClip creates an asset-clip element (for video files)
func (s *SmartClipStrategy) createAssetClip(config ClipConfig) TimelineElement {
	clip := &AssetClipElement{
		Ref:      config.VideoAssetID,
		Offset:   config.Offset,
		Name:     config.BaseName,
		Duration: config.Duration,
		Format:   "r1",
		TCFormat: "NDF",
	}
	
	// Add text overlay if requested
	if config.Text != "" {
		clip.HasText = true
		clip.TextEffectID = config.TextEffectID
		clip.Text = config.Text
		clip.HasAnimation = true
	}
	
	return clip
}

// createRefClip creates a ref-clip element (for compound clips)
func (s *SmartClipStrategy) createRefClip(config ClipConfig) TimelineElement {
	clip := &RefClipElement{
		Ref:      config.MediaID,
		Offset:   config.Offset,
		Name:     config.BaseName + " Clip",
		Duration: config.Duration,
	}
	
	// Add text overlay if requested
	if config.Text != "" {
		clip.HasText = true
		clip.TextEffectID = config.TextEffectID
		clip.Text = config.Text
		clip.HasAnimation = true
	}
	
	return clip
}

// createVideoWithAudioLane creates a video element with audio on a separate lane
func (s *SmartClipStrategy) createVideoWithAudioLane(config ClipConfig) TimelineElement {
	// Create a video element with nested audio asset-clip
	video := &VideoWithAudioElement{
		VideoRef:      config.VideoAssetID,
		AudioRef:      config.AudioAssetID,
		Offset:        config.Offset,
		Name:          config.BaseName,
		Start:         "0s",
		Duration:      config.Duration,
		AudioDuration: config.AudioDuration,
		HasText:       config.Text != "",
		TextEffectID:  config.TextEffectID,
		Text:          config.Text,
		HasAnimation:  config.Text != "",
	}
	
	return video
}

// Concrete implementations of TimelineElement

// VideoElement represents a video element
type VideoElement struct {
	Ref           string
	Offset        string
	Name          string
	Start         string
	Duration      string
	HasText       bool
	TextEffectID  string
	Text          string
	HasAnimation  bool
	WithSlide     bool
}

func (v *VideoElement) GetXML() string {
	video := &fcp.Video{
		Ref:      v.Ref,
		Offset:   v.Offset,
		Name:     v.Name,
		Start:    v.Start,
		Duration: v.Duration,
	}
	
	// Add adjust-transform for slide animation or text animation
	if v.WithSlide {
		video.AdjustTransform = &fcp.AdjustTransform{
			Params: []fcp.Param{
				{
					Name: "anchor",
					KeyframeAnimation: &fcp.KeyframeAnimation{
						Keyframes: []fcp.Keyframe{
							{Time: "0s", Value: "0 0"},
						},
					},
				},
				{
					Name: "position",
					KeyframeAnimation: &fcp.KeyframeAnimation{
						Keyframes: []fcp.Keyframe{
							{Time: "0s", Value: "0 0"},
							{Time: "48048/24000s", Value: "67.9349 0"},
						},
					},
				},
				{
					Name: "rotation",
					KeyframeAnimation: &fcp.KeyframeAnimation{
						Keyframes: []fcp.Keyframe{
							{Time: "0s", Value: "0"},
						},
					},
				},
				{
					Name: "scale",
					KeyframeAnimation: &fcp.KeyframeAnimation{
						Keyframes: []fcp.Keyframe{
							{Time: "0s", Value: "1 1"},
						},
					},
				},
			},
		}
	} else if v.HasAnimation {
		video.AdjustTransform = &fcp.AdjustTransform{
			Params: []fcp.Param{
				{
					Name:  "position",
					Key:   "",
					Value: "",
					KeyframeAnimation: &fcp.KeyframeAnimation{
						Keyframes: []fcp.Keyframe{
							{Time: "0s", Value: "0 0"},
							{Time: "48048/24000s", Value: "0 -22.1038"},
						},
					},
				},
			},
		}
	}
	
	// Add text overlay if requested
	if v.HasText {
		textStyleID := s.generateTextStyleID(v.Text, v.Name)
		title := fcp.Title{
			Ref:      v.TextEffectID,
			Lane:     "1",
			Offset:   "0s",
			Name:     v.Name + " - Text",
			Duration: v.Duration,
			Start:    "86486400/24000s",
			Params:   s.getTextParamsStruct(),
			Text: &fcp.TitleText{
				TextStyle: fcp.TextStyleRef{
					Ref:  textStyleID,
					Text: html.EscapeString(v.Text),
				},
			},
			TextStyleDef: &fcp.TextStyleDef{
				ID: textStyleID,
				TextStyle: fcp.TextStyle{
					Font:        "Helvetica Neue",
					FontSize:    "196",
					FontColor:   "1 1 1 1",
					Bold:        "1",
					Alignment:   "center",
					LineSpacing: "-19",
				},
			},
		}
		video.NestedTitles = []fcp.Title{title}
	}
	
	// Marshal to XML
	xmlBytes, err := xml.MarshalIndent(video, "", "    ")
	if err != nil {
		return "<!-- Error marshaling video element: " + err.Error() + " -->"
	}
	return string(xmlBytes)
}

func (v *VideoElement) GetDuration() string { return v.Duration }
func (v *VideoElement) GetOffset() string { return v.Offset }

// AssetClipElement represents an asset-clip element
type AssetClipElement struct {
	Ref           string
	Offset        string
	Name          string
	Duration      string
	Format        string
	TCFormat      string
	HasText       bool
	TextEffectID  string
	Text          string
	HasAnimation  bool
	WithSlide     bool
}

func (a *AssetClipElement) GetXML() string {
	assetClip := &fcp.AssetClip{
		Ref:      a.Ref,
		Offset:   a.Offset,
		Name:     a.Name,
		Duration: a.Duration,
		Format:   a.Format,
		TCFormat: a.TCFormat,
	}
	
	// Add adjust-transform before title (DTD requirement)
	if a.HasAnimation {
		assetClip.AdjustTransform = &fcp.AdjustTransform{
			Params: []fcp.Param{
				{
					Name:  "position",
					Key:   "",
					Value: "",
					KeyframeAnimation: &fcp.KeyframeAnimation{
						Keyframes: []fcp.Keyframe{
							{Time: "0s", Value: "0 0"},
							{Time: "48048/24000s", Value: "0 -22.1038"},
						},
					},
				},
			},
		}
	}
	
	// Add text overlay if requested
	if a.HasText {
		textStyleID := s.generateTextStyleID(a.Text, a.Name)
		title := fcp.Title{
			Ref:      a.TextEffectID,
			Lane:     "1",
			Offset:   "0s",
			Name:     a.Name + " - Text",
			Duration: a.Duration,
			Start:    "86486400/24000s",
			Params:   s.getTextParamsStruct(),
			Text: &fcp.TitleText{
				TextStyle: fcp.TextStyleRef{
					Ref:  textStyleID,
					Text: html.EscapeString(a.Text),
				},
			},
			TextStyleDef: &fcp.TextStyleDef{
				ID: textStyleID,
				TextStyle: fcp.TextStyle{
					Font:        "Helvetica Neue",
					FontSize:    "196",
					FontColor:   "1 1 1 1",
					Bold:        "1",
					Alignment:   "center",
					LineSpacing: "-19",
				},
			},
		}
		assetClip.Titles = []fcp.Title{title}
	}
	
	// Marshal to XML
	xmlBytes, err := xml.MarshalIndent(assetClip, "", "    ")
	if err != nil {
		return "<!-- Error marshaling asset-clip element: " + err.Error() + " -->"
	}
	return string(xmlBytes)
}

func (a *AssetClipElement) GetDuration() string { return a.Duration }
func (a *AssetClipElement) GetOffset() string { return a.Offset }

// RefClipElement represents a ref-clip element
type RefClipElement struct {
	Ref           string
	Offset        string
	Name          string
	Duration      string
	HasText       bool
	TextEffectID  string
	Text          string
	HasAnimation  bool
}

func (r *RefClipElement) GetXML() string {
	refClip := &fcp.RefClip{
		Ref:      r.Ref,
		Offset:   r.Offset,
		Name:     r.Name,
		Duration: r.Duration,
	}
	
	// Add adjust-transform before title (DTD requirement)
	if r.HasAnimation {
		refClip.AdjustTransform = &fcp.AdjustTransform{
			Params: []fcp.Param{
				{
					Name:  "position",
					Key:   "",
					Value: "",
					KeyframeAnimation: &fcp.KeyframeAnimation{
						Keyframes: []fcp.Keyframe{
							{Time: "0s", Value: "0 0"},
							{Time: "48048/24000s", Value: "0 -22.1038"},
						},
					},
				},
			},
		}
	}
	
	// Add text overlay if requested
	if r.HasText {
		textStyleID := s.generateTextStyleID(r.Text, r.Name)
		title := fcp.Title{
			Ref:      r.TextEffectID,
			Lane:     "1",
			Offset:   "0s",
			Name:     r.Name + " - Text",
			Duration: r.Duration,
			Start:    "86486400/24000s",
			Params:   s.getTextParamsStruct(),
			Text: &fcp.TitleText{
				TextStyle: fcp.TextStyleRef{
					Ref:  textStyleID,
					Text: html.EscapeString(r.Text),
				},
			},
			TextStyleDef: &fcp.TextStyleDef{
				ID: textStyleID,
				TextStyle: fcp.TextStyle{
					Font:        "Helvetica Neue",
					FontSize:    "196",
					FontColor:   "1 1 1 1",
					Bold:        "1",
					Alignment:   "center",
					LineSpacing: "-19",
				},
			},
		}
		refClip.Titles = []fcp.Title{title}
	}
	
	// Marshal to XML
	xmlBytes, err := xml.MarshalIndent(refClip, "", "    ")
	if err != nil {
		return "<!-- Error marshaling ref-clip element: " + err.Error() + " -->"
	}
	return string(xmlBytes)
}

func (r *RefClipElement) GetDuration() string { return r.Duration }
func (r *RefClipElement) GetOffset() string { return r.Offset }

// VideoWithAudioElement represents a video element with audio on a separate lane
type VideoWithAudioElement struct {
	VideoRef      string
	AudioRef      string
	Offset        string
	Name          string
	Start         string
	Duration      string
	AudioDuration string
	HasText       bool
	TextEffectID  string
	Text          string
	HasAnimation  bool
}

func (v *VideoWithAudioElement) GetXML() string {
	video := &fcp.Video{
		Ref:      v.VideoRef,
		Offset:   v.Offset,
		Name:     v.Name,
		Start:    v.Start,
		Duration: v.Duration,
	}
	
	// Add adjust-transform FIRST (intrinsic params come before anchor items per DTD)
	if v.HasText && v.HasAnimation {
		video.AdjustTransform = &fcp.AdjustTransform{
			Params: []fcp.Param{
				{
					Name:  "position",
					Key:   "",
					Value: "",
					KeyframeAnimation: &fcp.KeyframeAnimation{
						Keyframes: []fcp.Keyframe{
							{Time: "0s", Value: "0 0"},
							{Time: "48048/24000s", Value: "0 -22.1038"},
						},
					},
				},
			},
		}
	}
	
	// Add the audio asset-clip on lane -1 (audio lane) with correct audio duration
	audioDuration := v.AudioDuration
	if audioDuration == "" {
		audioDuration = v.Duration // fallback to video duration if not specified
	}
	
	// Create nested videos list with the audio asset-clip
	video.NestedVideos = []fcp.Video{}
	
	// We need to add this as a nested asset-clip, but fcp.Video doesn't support nested asset-clips
	// So we'll temporarily use a manual approach for this complex case
	xmlBytes, err := xml.MarshalIndent(video, "", "    ")
	if err != nil {
		return "<!-- Error marshaling video with audio element: " + err.Error() + " -->"
	}
	
	xmlStr := string(xmlBytes)
	
	// Insert the audio asset-clip before closing the video tag
	audioClipXML := `<asset-clip ref="` + v.AudioRef + `" lane="-1" offset="0s" name="` + v.Name + ` - Audio" duration="` + audioDuration + `" format="r1" tcFormat="NDF" audioRole="dialogue"/>`
	
	if v.HasText {
		textStyleID := s.generateTextStyleID(v.Text, v.Name)
		titleXML := `<title ref="` + v.TextEffectID + `" lane="1" offset="0s" name="` + v.Name + ` - Text" duration="` + v.Duration + `" start="86486400/24000s">`
		
		// Add text params as XML string temporarily (TODO: convert to struct approach)
		titleXML += s.getTextParams()
		titleXML += `<text><text-style ref="` + textStyleID + `">` + html.EscapeString(v.Text) + `</text-style></text>`
		titleXML += `<text-style-def id="` + textStyleID + `"><text-style font="Helvetica Neue" fontSize="196" fontColor="1 1 1 1" bold="1" alignment="center" lineSpacing="-19"/></text-style-def>`
		titleXML += `</title>`
		
		// Insert both audio and title before closing tag
		xmlStr = strings.Replace(xmlStr, "</video>", "    "+audioClipXML+"\n    "+titleXML+"\n</video>", 1)
	} else {
		// Insert just audio before closing tag
		xmlStr = strings.Replace(xmlStr, "</video>", "    "+audioClipXML+"\n</video>", 1)
	}
	
	return xmlStr
}

func (v *VideoWithAudioElement) GetDuration() string { return v.Duration }
func (v *VideoWithAudioElement) GetOffset() string { return v.Offset }

// Helper functions

var s = &SmartClipStrategy{} // Global instance for helper methods

func (s *SmartClipStrategy) getTextParams() string {
	return `<param name="Layout Method" key="9999/10003/13260/3296672360/2/314" value="1 (Paragraph)"/>
                                <param name="Left Margin" key="9999/10003/13260/3296672360/2/323" value="-1730"/>
                                <param name="Right Margin" key="9999/10003/13260/3296672360/2/324" value="1730"/>
                                <param name="Top Margin" key="9999/10003/13260/3296672360/2/325" value="960"/>
                                <param name="Bottom Margin" key="9999/10003/13260/3296672360/2/326" value="-960"/>
                                <param name="Alignment" key="9999/10003/13260/3296672360/2/354/3296667315/401" value="1 (Center)"/>
                                <param name="Line Spacing" key="9999/10003/13260/3296672360/2/354/3296667315/404" value="-19"/>
                                <param name="Auto-Shrink" key="9999/10003/13260/3296672360/2/370" value="3 (To All Margins)"/>
                                <param name="Alignment" key="9999/10003/13260/3296672360/2/373" value="0 (Left) 0 (Top)"/>
                                <param name="Opacity" key="9999/10003/13260/3296672360/4/3296673134/1000/1044" value="0"/>
                                <param name="Speed" key="9999/10003/13260/3296672360/4/3296673134/201/208" value="6 (Custom)"/>
                                <param name="Custom Speed" key="9999/10003/13260/3296672360/4/3296673134/201/209">
                                    <keyframeAnimation>
                                        <keyframe time="-469658744/1000000000s" value="0"/>
                                        <keyframe time="12328542033/1000000000s" value="1"/>
                                    </keyframeAnimation>
                                </param>
                                <param name="Apply Speed" key="9999/10003/13260/3296672360/4/3296673134/201/211" value="2 (Per Object)"/>`
}

func (s *SmartClipStrategy) getTextParamsStruct() []fcp.Param {
	return []fcp.Param{
		{Name: "Layout Method", Key: "9999/10003/13260/3296672360/2/314", Value: "1 (Paragraph)"},
		{Name: "Left Margin", Key: "9999/10003/13260/3296672360/2/323", Value: "-1730"},
		{Name: "Right Margin", Key: "9999/10003/13260/3296672360/2/324", Value: "1730"},
		{Name: "Top Margin", Key: "9999/10003/13260/3296672360/2/325", Value: "960"},
		{Name: "Bottom Margin", Key: "9999/10003/13260/3296672360/2/326", Value: "-960"},
		{Name: "Alignment", Key: "9999/10003/13260/3296672360/2/354/3296667315/401", Value: "1 (Center)"},
		{Name: "Line Spacing", Key: "9999/10003/13260/3296672360/2/354/3296667315/404", Value: "-19"},
		{Name: "Auto-Shrink", Key: "9999/10003/13260/3296672360/2/370", Value: "3 (To All Margins)"},
		{Name: "Alignment", Key: "9999/10003/13260/3296672360/2/373", Value: "0 (Left) 0 (Top)"},
		{Name: "Opacity", Key: "9999/10003/13260/3296672360/4/3296673134/1000/1044", Value: "0"},
		{Name: "Speed", Key: "9999/10003/13260/3296672360/4/3296673134/201/208", Value: "6 (Custom)"},
		{
			Name: "Custom Speed",
			Key:  "9999/10003/13260/3296672360/4/3296673134/201/209",
			KeyframeAnimation: &fcp.KeyframeAnimation{
				Keyframes: []fcp.Keyframe{
					{Time: "-469658744/1000000000s", Value: "0"},
					{Time: "12328542033/1000000000s", Value: "1"},
				},
			},
		},
		{Name: "Apply Speed", Key: "9999/10003/13260/3296672360/4/3296673134/201/211", Value: "2 (Per Object)"},
	}
}

func (s *SmartClipStrategy) generateTextStyleID(text, baseName string) string {
	// Create unique ID using text + baseName + timestamp component to ensure uniqueness
	// Include more variation in the hash to avoid collisions
	input := text + "|" + baseName + "|" + strings.Join(strings.Fields(text), "_")
	hash := 0
	for i, c := range input {
		hash = hash*31 + int(c) + i*7 // Add position weight to increase variation
	}
	if hash < 0 {
		hash = -hash
	}
	
	// Generate more varied character combinations
	chars := []rune{
		rune('A' + hash%26),
		rune('0' + (hash/26)%10),
		rune('A' + (hash/260)%26),
		rune('0' + (hash/6760)%10),
		rune('A' + (hash/67600)%26),
		rune('0' + (hash/1757600)%10),
		rune('A' + (hash/17576000)%26),
		rune('0' + (hash/456976000)%10),
	}
	
	return "ts" + string(chars)
}

// isPNGFile checks if the given file is a PNG image
func isPNGFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".png"
}

// AudioOnlyElement represents an audio-only clip on lane -1
type AudioOnlyElement struct {
	AudioRef string
	Offset   string
	Name     string
	Duration string
}

func (a *AudioOnlyElement) GetXML() string {
	assetClip := &fcp.AssetClip{
		Ref:       a.AudioRef,
		Lane:      "-1",
		Offset:    a.Offset,
		Name:      a.Name,
		Duration:  a.Duration,
		Format:    "r1",
		TCFormat:  "NDF",
		AudioRole: "dialogue",
	}
	
	// Marshal to XML
	xmlBytes, err := xml.MarshalIndent(assetClip, "", "    ")
	if err != nil {
		return "<!-- Error marshaling audio-only element: " + err.Error() + " -->"
	}
	return string(xmlBytes)
}

func (a *AudioOnlyElement) GetDuration() string { return a.Duration }
func (a *AudioOnlyElement) GetOffset() string { return a.Offset }

// isImageFile checks if the given file is an image (PNG or JPG)
func isImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg"
}