package clips

import (
	"html"
	"path/filepath"
	"strings"
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
	Offset          string
	Text            string
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
	
	// For image files with audio, we need compound clips
	if isImageFile(video) && audio != "" {
		return true
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
}

func (v *VideoElement) GetXML() string {
	xml := `<video ref="` + v.Ref + `" offset="` + v.Offset + `" name="` + v.Name + `" start="` + v.Start + `" duration="` + v.Duration + `"`
	
	if v.HasText {
		xml += `>`
		
		// Add adjust-transform before title (DTD requirement)
		if v.HasAnimation {
			xml += `
                            <adjust-transform>
                                <param name="position" key="" value="">
                                    <keyframeAnimation>
                                        <keyframe time="0s" value="0 0"/>
                                        <keyframe time="48048/24000s" value="0 -22.1038"/>
                                    </keyframeAnimation>
                                </param>
                            </adjust-transform>`
		}
		
		xml += `
                            <title ref="` + v.TextEffectID + `" lane="1" offset="0s" name="` + v.Name + ` - Text" duration="` + v.Duration + `" start="86486400/24000s">
                                ` + s.getTextParams() + `
                                <text>
                                    <text-style ref="` + s.generateTextStyleID(v.Text, v.Name) + `">` + html.EscapeString(v.Text) + `</text-style>
                                </text>
                                <text-style-def id="` + s.generateTextStyleID(v.Text, v.Name) + `">
                                    <text-style font="Helvetica Neue" fontSize="196" fontColor="1 1 1 1" bold="1" alignment="center" lineSpacing="-19"/>
                                </text-style-def>
                            </title>`
		
		xml += `
                        </video>`
	} else {
		xml += `/>`
	}
	
	return xml
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
}

func (a *AssetClipElement) GetXML() string {
	xml := `<asset-clip ref="` + a.Ref + `" offset="` + a.Offset + `" name="` + a.Name + `" duration="` + a.Duration + `" format="` + a.Format + `" tcFormat="` + a.TCFormat + `"`
	
	if a.HasText {
		xml += `>`
		
		// Add adjust-transform before title (DTD requirement)
		if a.HasAnimation {
			xml += `
                            <adjust-transform>
                                <param name="position" key="" value="">
                                    <keyframeAnimation>
                                        <keyframe time="0s" value="0 0"/>
                                        <keyframe time="48048/24000s" value="0 -22.1038"/>
                                    </keyframeAnimation>
                                </param>
                            </adjust-transform>`
		}
		
		xml += `
                            <title ref="` + a.TextEffectID + `" lane="1" offset="0s" name="` + a.Name + ` - Text" duration="` + a.Duration + `" start="86486400/24000s">
                                ` + s.getTextParams() + `
                                <text>
                                    <text-style ref="` + s.generateTextStyleID(a.Text, a.Name) + `">` + html.EscapeString(a.Text) + `</text-style>
                                </text>
                                <text-style-def id="` + s.generateTextStyleID(a.Text, a.Name) + `">
                                    <text-style font="Helvetica Neue" fontSize="196" fontColor="1 1 1 1" bold="1" alignment="center" lineSpacing="-19"/>
                                </text-style-def>
                            </title>`
		
		xml += `
                        </asset-clip>`
	} else {
		xml += `/>`
	}
	
	return xml
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
	xml := `<ref-clip ref="` + r.Ref + `" offset="` + r.Offset + `" name="` + r.Name + `" duration="` + r.Duration + `"`
	
	if r.HasText {
		xml += `>`
		
		// Add adjust-transform before title (DTD requirement)
		if r.HasAnimation {
			xml += `
                            <adjust-transform>
                                <param name="position" key="" value="">
                                    <keyframeAnimation>
                                        <keyframe time="0s" value="0 0"/>
                                        <keyframe time="48048/24000s" value="0 -22.1038"/>
                                    </keyframeAnimation>
                                </param>
                            </adjust-transform>`
		}
		
		xml += `
                            <title ref="` + r.TextEffectID + `" lane="1" offset="0s" name="` + r.Name + ` - Text" duration="` + r.Duration + `" start="86486400/24000s">
                                ` + s.getTextParams() + `
                                <text>
                                    <text-style ref="` + s.generateTextStyleID(r.Text, r.Name) + `">` + html.EscapeString(r.Text) + `</text-style>
                                </text>
                                <text-style-def id="` + s.generateTextStyleID(r.Text, r.Name) + `">
                                    <text-style font="Helvetica Neue" fontSize="196" fontColor="1 1 1 1" bold="1" alignment="center" lineSpacing="-19"/>
                                </text-style-def>
                            </title>`
		
		xml += `
                        </ref-clip>`
	} else {
		xml += `/>`
	}
	
	return xml
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
	HasText       bool
	TextEffectID  string
	Text          string
	HasAnimation  bool
}

func (v *VideoWithAudioElement) GetXML() string {
	xml := `<video ref="` + v.VideoRef + `" offset="` + v.Offset + `" name="` + v.Name + `" start="` + v.Start + `" duration="` + v.Duration + `">`
	
	// Add the audio asset-clip on lane -1 (audio lane)
	xml += `
                            <asset-clip ref="` + v.AudioRef + `" lane="-1" offset="0s" name="` + v.Name + ` - Audio" duration="` + v.Duration + `" format="r1" tcFormat="NDF"/>`
	
	// Add adjust-transform before title if text is present (DTD requirement)
	if v.HasText && v.HasAnimation {
		xml += `
                            <adjust-transform>
                                <param name="position" key="" value="">
                                    <keyframeAnimation>
                                        <keyframe time="0s" value="0 0"/>
                                        <keyframe time="48048/24000s" value="0 -22.1038"/>
                                    </keyframeAnimation>
                                </param>
                            </adjust-transform>`
	}
	
	// Add text overlay if requested
	if v.HasText {
		xml += `
                            <title ref="` + v.TextEffectID + `" lane="1" offset="0s" name="` + v.Name + ` - Text" duration="` + v.Duration + `" start="86486400/24000s">
                                ` + s.getTextParams() + `
                                <text>
                                    <text-style ref="` + s.generateTextStyleID(v.Text, v.Name) + `">` + html.EscapeString(v.Text) + `</text-style>
                                </text>
                                <text-style-def id="` + s.generateTextStyleID(v.Text, v.Name) + `">
                                    <text-style font="Helvetica Neue" fontSize="196" fontColor="1 1 1 1" bold="1" alignment="center" lineSpacing="-19"/>
                                </text-style-def>
                            </title>`
	}
	
	xml += `
                        </video>`
	
	return xml
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

func (s *SmartClipStrategy) generateTextStyleID(text, baseName string) string {
	// Simple hash-based ID generation
	hash := 0
	for _, c := range text + baseName {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return "ts" + strings.ToUpper(strings.Replace(strings.Replace(strings.Replace(
		strings.Replace(string(rune('A'+hash%26))+string(rune('0'+hash%10))+string(rune('A'+(hash/10)%26))+string(rune('0'+(hash/100)%10))+
		string(rune('A'+(hash/1000)%26))+string(rune('0'+(hash/10000)%10))+string(rune('A'+(hash/100000)%26))+string(rune('0'+(hash/1000000)%10)),
		" ", "", -1), "\n", "", -1), "\t", "", -1), "\r", "", -1))[:8]
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
	return `<asset-clip ref="` + a.AudioRef + `" lane="-1" offset="` + a.Offset + `" name="` + a.Name + `" duration="` + a.Duration + `" format="r1" tcFormat="NDF" audioRole="dialogue"/>`
}

func (a *AudioOnlyElement) GetDuration() string { return a.Duration }
func (a *AudioOnlyElement) GetOffset() string { return a.Offset }

// isImageFile checks if the given file is an image (PNG or JPG)
func isImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg"
}