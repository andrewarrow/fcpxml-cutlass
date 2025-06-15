// Package fcp defines the struct types for FCPXML generation.
//
// 🚨 CRITICAL: These structs are the ONLY way to generate XML (see CLAUDE.md)
// - NEVER use string templates → USE xml.MarshalIndent() function only
// - NEVER set .Content or .InnerXML → APPEND to struct slices (e.g., spine.AssetClips)  
// - VALIDATE output → RUN ValidateClaudeCompliance() + xmllint DTD validation
// - FOR frame alignment → USE ConvertSecondsToFCPDuration() function
package fcp

import (
	"encoding/xml"
	"strconv"
	"strings"
)

type FCPXML struct {
	XMLName   xml.Name  `xml:"fcpxml"`
	Version   string    `xml:"version,attr"`
	Resources Resources `xml:"resources"`
	Library   Library   `xml:"library"`
}

// Resources contains all assets, formats, effects, and media definitions.
//
// 🚨 CLAUDE.md Rule: Unique ID Requirements → USE this counting pattern:
// resourceCount := len(Assets)+len(Formats)+len(Effects)+len(Media)
// nextID := fmt.Sprintf("r%d", resourceCount+1)
// NEVER hardcode IDs like "r1", "r2" - ALWAYS count existing resources
type Resources struct {
	Assets     []Asset     `xml:"asset,omitempty"`
	Formats    []Format    `xml:"format"`
	Effects    []Effect    `xml:"effect,omitempty"`
	Media      []Media     `xml:"media,omitempty"`
}

// Effect represents a Motion or standard FCP title effect referenced by <title ref="…"> elements.
type Effect struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
	UID  string `xml:"uid,attr,omitempty"`
}


type Format struct {
	ID            string `xml:"id,attr"`
	Name          string `xml:"name,attr,omitempty"` // CRITICAL: omitempty allows compatible formats without names
	FrameDuration string `xml:"frameDuration,attr,omitempty"`
	Width         string `xml:"width,attr,omitempty"`
	Height        string `xml:"height,attr,omitempty"`
	ColorSpace    string `xml:"colorSpace,attr,omitempty"`
}

// Asset represents a media asset (video, audio, image) in FCPXML.
//
// 🚨 CLAUDE.md Rule: UID Consistency Requirements → USE generateUID() function
// - UID = generateUID(filename) for deterministic UIDs based on filename  
// - NEVER base UID on file path (causes "cannot be imported again" errors)
// - FOR durations → USE ConvertSecondsToFCPDuration() function
type Asset struct {
	ID            string   `xml:"id,attr"`
	Name          string   `xml:"name,attr"`
	UID           string   `xml:"uid,attr"`
	Start         string   `xml:"start,attr"`
	HasVideo      string   `xml:"hasVideo,attr"`
	Format        string   `xml:"format,attr"`
	VideoSources  string   `xml:"videoSources,attr,omitempty"`
	HasAudio      string   `xml:"hasAudio,attr,omitempty"`
	AudioSources  string   `xml:"audioSources,attr,omitempty"`
	AudioChannels string   `xml:"audioChannels,attr,omitempty"`
	AudioRate     string   `xml:"audioRate,attr,omitempty"`
	Duration      string   `xml:"duration,attr"`
	MediaRep      MediaRep `xml:"media-rep"`
}

type MediaRep struct {
	Kind string `xml:"kind,attr"`
	Sig  string `xml:"sig,attr"`
	Src  string `xml:"src,attr"`
}

type Media struct {
	ID       string   `xml:"id,attr"`
	Name     string   `xml:"name,attr"`
	UID      string   `xml:"uid,attr"`
	ModDate  string   `xml:"modDate,attr,omitempty"`
	Sequence Sequence `xml:"sequence"`
}

type RefClip struct {
	XMLName         xml.Name         `xml:"ref-clip"`
	Ref             string           `xml:"ref,attr"`
	Offset          string           `xml:"offset,attr"`
	Name            string           `xml:"name,attr"`
	Duration        string           `xml:"duration,attr"`
	AdjustTransform *AdjustTransform `xml:"adjust-transform,omitempty"`
	Titles          []Title          `xml:"title,omitempty"`
}

type Library struct {
	Location          string            `xml:"location,attr,omitempty"`
	Events            []Event           `xml:"event"`
	SmartCollections  []SmartCollection `xml:"smart-collection,omitempty"`
}

type Event struct {
	Name     string    `xml:"name,attr"`
	UID      string    `xml:"uid,attr,omitempty"`
	Projects []Project `xml:"project"`
}

type Project struct {
	Name      string     `xml:"name,attr"`
	UID       string     `xml:"uid,attr,omitempty"`
	ModDate   string     `xml:"modDate,attr,omitempty"`
	Sequences []Sequence `xml:"sequence"`
}

type Sequence struct {
	Format      string `xml:"format,attr"`
	Duration    string `xml:"duration,attr"`
	TCStart     string `xml:"tcStart,attr"`
	TCFormat    string `xml:"tcFormat,attr"`
	AudioLayout string `xml:"audioLayout,attr"`
	AudioRate   string `xml:"audioRate,attr"`
	Spine       Spine  `xml:"spine"`
}

// TimelineElement represents any element that can appear in a spine with an offset
type TimelineElement interface {
	GetOffset() string
	GetEndOffset() string
}

// Spine represents the main timeline container in FCPXML.
//
// 🚨 CLAUDE.md Rule: NO XML STRING TEMPLATES → USE struct slices:
// spine.AssetClips = append(spine.AssetClips, assetClip) ✅
// spine.Content = fmt.Sprintf("<asset-clip...") ❌ CRITICAL VIOLATION!
// FOR durations → USE ConvertSecondsToFCPDuration() function
type Spine struct {
	XMLName    xml.Name    `xml:"spine"`
	AssetClips []AssetClip `xml:"asset-clip,omitempty"`
	Gaps       []Gap       `xml:"gap,omitempty"`
	Titles     []Title     `xml:"title,omitempty"`
	Videos     []Video     `xml:"video,omitempty"`
}

// MarshalXML implements custom XML marshaling to maintain chronological order
func (s Spine) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Start the spine element
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Collect all elements with their offsets
	type elementWithOffset struct {
		offset  int
		element interface{}
	}
	var elements []elementWithOffset

	// Add all element types
	for _, clip := range s.AssetClips {
		elements = append(elements, elementWithOffset{
			offset:  parseFCPDurationForSort(clip.Offset),
			element: clip,
		})
	}
	for _, video := range s.Videos {
		elements = append(elements, elementWithOffset{
			offset:  parseFCPDurationForSort(video.Offset),
			element: video,
		})
	}
	for _, title := range s.Titles {
		elements = append(elements, elementWithOffset{
			offset:  parseFCPDurationForSort(title.Offset),
			element: title,
		})
	}
	for _, gap := range s.Gaps {
		elements = append(elements, elementWithOffset{
			offset:  parseFCPDurationForSort(gap.Offset),
			element: gap,
		})
	}

	// Sort by offset
	for i := 0; i < len(elements)-1; i++ {
		for j := 0; j < len(elements)-i-1; j++ {
			if elements[j].offset > elements[j+1].offset {
				elements[j], elements[j+1] = elements[j+1], elements[j]
			}
		}
	}

	// Encode elements in chronological order
	for _, elem := range elements {
		if err := e.Encode(elem.element); err != nil {
			return err
		}
	}

	// End the spine element
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// parseFCPDurationForSort parses FCP duration for sorting (similar to existing function)
func parseFCPDurationForSort(duration string) int {
	if duration == "0s" {
		return 0
	}
	
	// Parse format like "12345/24000s"
	if strings.HasSuffix(duration, "/24000s") {
		framesStr := strings.TrimSuffix(duration, "/24000s")
		if frames, err := strconv.Atoi(framesStr); err == nil {
			return frames
		}
	}
	
	return 0
}

type AssetClip struct {
	XMLName         xml.Name         `xml:"asset-clip"`
	Ref             string           `xml:"ref,attr"`
	Lane            string           `xml:"lane,attr,omitempty"`
	Offset          string           `xml:"offset,attr"`
	Name            string           `xml:"name,attr"`
	Start           string           `xml:"start,attr,omitempty"`
	Duration        string           `xml:"duration,attr"`
	Format          string           `xml:"format,attr"`
	TCFormat        string           `xml:"tcFormat,attr"`
	AudioRole       string           `xml:"audioRole,attr,omitempty"`
	AdjustTransform *AdjustTransform `xml:"adjust-transform,omitempty"`
	Titles          []Title          `xml:"title,omitempty"`
	Videos          []Video          `xml:"video,omitempty"`
}

// GetOffset implements TimelineElement interface
func (ac AssetClip) GetOffset() string {
	return ac.Offset
}

// GetEndOffset implements TimelineElement interface
func (ac AssetClip) GetEndOffset() string {
	// This would require parsing offset and duration to calculate end time
	// For now, return offset (implementation can be added later if needed)
	return ac.Offset
}

type Gap struct {
	XMLName        xml.Name        `xml:"gap"`
	Name           string          `xml:"name,attr"`
	Offset         string          `xml:"offset,attr"`
	Duration       string          `xml:"duration,attr"`
	Titles         []Title         `xml:"title,omitempty"`
	GeneratorClips []GeneratorClip `xml:"generator-clip,omitempty"`
}

type Title struct {
	XMLName xml.Name `xml:"title"`
	Ref          string        `xml:"ref,attr"`
	Lane         string        `xml:"lane,attr,omitempty"`
	Offset       string        `xml:"offset,attr"`
	Name         string        `xml:"name,attr"`
	Duration     string        `xml:"duration,attr"`
	Start        string        `xml:"start,attr,omitempty"`
	Params       []Param       `xml:"param,omitempty"`
	Text         *TitleText    `xml:"text,omitempty"`      // Pointer so it can be nil
	TextStyleDef *TextStyleDef `xml:"text-style-def,omitempty"` // Pointer so it can be nil
}

// Video represents a video element (shapes, colors, etc.)
type Video struct {
	XMLName xml.Name `xml:"video"`
	Ref           string         `xml:"ref,attr"`
	Lane          string         `xml:"lane,attr,omitempty"`
	Offset        string         `xml:"offset,attr"`
	Name          string         `xml:"name,attr"`
	Duration      string         `xml:"duration,attr"`
	Start         string         `xml:"start,attr,omitempty"`
	Params        []Param        `xml:"param,omitempty"`
	AdjustTransform *AdjustTransform `xml:"adjust-transform,omitempty"`
	NestedVideos     []Video     `xml:"video,omitempty"`      // Support nested video elements with lanes
	NestedAssetClips []AssetClip `xml:"asset-clip,omitempty"` // Support nested asset-clip elements with lanes
	NestedTitles     []Title     `xml:"title,omitempty"`      // Support nested title elements with lanes
}

// GetOffset implements TimelineElement interface
func (v Video) GetOffset() string {
	return v.Offset
}

// GetEndOffset implements TimelineElement interface
func (v Video) GetEndOffset() string {
	// This would require parsing offset and duration to calculate end time
	// For now, return offset (implementation can be added later if needed)
	return v.Offset
}

type AdjustTransform struct {
	Position string  `xml:"position,attr,omitempty"`
	Scale    string  `xml:"scale,attr,omitempty"`
	Params   []Param `xml:"param,omitempty"`
}


type GeneratorClip struct {
	Ref      string  `xml:"ref,attr"`
	Lane     string  `xml:"lane,attr,omitempty"`
	Offset   string  `xml:"offset,attr"`
	Name     string  `xml:"name,attr"`
	Duration string  `xml:"duration,attr"`
	Start    string  `xml:"start,attr,omitempty"`
	Params   []Param `xml:"param,omitempty"`
}

type Param struct {
	Name               string              `xml:"name,attr"`
	Key                string              `xml:"key,attr,omitempty"`
	Value              string              `xml:"value,attr,omitempty"`
	KeyframeAnimation  *KeyframeAnimation  `xml:"keyframeAnimation,omitempty"`
	NestedParams       []Param             `xml:"param,omitempty"`
}

type KeyframeAnimation struct {
	Keyframes []Keyframe `xml:"keyframe"`
}

type Keyframe struct {
	Time  string `xml:"time,attr"`
	Value string `xml:"value,attr"`
	Curve string `xml:"curve,attr,omitempty"`
}

type TitleText struct {
	TextStyle TextStyleRef `xml:"text-style"`
}

type TextStyleRef struct {
	Ref  string `xml:"ref,attr"`
	Text string `xml:",chardata"`
}

type TextStyleDef struct {
	ID        string    `xml:"id,attr"`
	TextStyle TextStyle `xml:"text-style"`
}

type TextStyle struct {
	Font        string `xml:"font,attr"`
	FontSize    string `xml:"fontSize,attr"`
	FontFace    string `xml:"fontFace,attr"`
	FontColor   string `xml:"fontColor,attr"`
	Bold        string `xml:"bold,attr,omitempty"`
	Alignment   string `xml:"alignment,attr"`
	LineSpacing string `xml:"lineSpacing,attr,omitempty"`
}

type SmartCollection struct {
	Name     string      `xml:"name,attr"`
	Match    string      `xml:"match,attr"`
	Matches  []Match     `xml:"match-clip,omitempty"`
	MediaMatches []MediaMatch `xml:"match-media,omitempty"`
	RatingMatches []RatingMatch `xml:"match-ratings,omitempty"`
}

type Match struct {
	Rule string `xml:"rule,attr"`
	Type string `xml:"type,attr"`
}

type MediaMatch struct {
	Rule string `xml:"rule,attr"`
	Type string `xml:"type,attr"`
}

type RatingMatch struct {
	Value string `xml:"value,attr"`
}

type ParseOptions struct {
	Tier          int
	ShowElements  bool
	ShowParams    bool
	ShowAnimation bool
	ShowResources bool
	ShowStructure bool
}