package fcp

import (
	"encoding/xml"
)

type FCPXML struct {
	XMLName   xml.Name  `xml:"fcpxml"`
	Version   string    `xml:"version,attr"`
	Resources Resources `xml:"resources"`
	Library   Library   `xml:"library"`
}

type Resources struct {
	Assets     []Asset     `xml:"asset,omitempty"`
	Effects    []Effect    `xml:"effect,omitempty"`
	Formats    []Format    `xml:"format"`
	// Remove generators as they should be effects
}

// Effect represents a Motion or standard FCP title effect referenced by <title ref="…"> elements.
type Effect struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
	UID  string `xml:"uid,attr,omitempty"`
}


type Format struct {
	ID            string `xml:"id,attr"`
	Name          string `xml:"name,attr"`
	FrameDuration string `xml:"frameDuration,attr"`
	Width         string `xml:"width,attr"`
	Height        string `xml:"height,attr"`
	ColorSpace    string `xml:"colorSpace,attr"`
}

type Asset struct {
	ID            string   `xml:"id,attr"`
	Name          string   `xml:"name,attr"`
	UID           string   `xml:"uid,attr"`
	Start         string   `xml:"start,attr"`
	HasVideo      string   `xml:"hasVideo,attr"`
	Format        string   `xml:"format,attr"`
	HasAudio      string   `xml:"hasAudio,attr"`
	AudioSources  string   `xml:"audioSources,attr"`
	AudioChannels string   `xml:"audioChannels,attr"`
	Duration      string   `xml:"duration,attr"`
	MediaRep      MediaRep `xml:"media-rep"`
}

type MediaRep struct {
	Kind string `xml:"kind,attr"`
	Sig  string `xml:"sig,attr"`
	Src  string `xml:"src,attr"`
}

type Library struct {
	Events []Event `xml:"event"`
}

type Event struct {
	Name     string    `xml:"name,attr"`
	Projects []Project `xml:"project"`
}

type Project struct {
	Name      string     `xml:"name,attr"`
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

type Spine struct {
	XMLName xml.Name `xml:"spine"`
	Content string   `xml:",innerxml"`
}

type AssetClip struct {
	Ref       string `xml:"ref,attr"`
	Offset    string `xml:"offset,attr"`
	Name      string `xml:"name,attr"`
	Start     string `xml:"start,attr,omitempty"`
	Duration  string `xml:"duration,attr"`
	Format    string `xml:"format,attr"`
	TCFormat  string `xml:"tcFormat,attr"`
	AudioRole string `xml:"audioRole,attr,omitempty"`
}

type Gap struct {
	Name           string          `xml:"name,attr"`
	Offset         string          `xml:"offset,attr"`
	Duration       string          `xml:"duration,attr"`
	Titles         []Title         `xml:"title,omitempty"`
	GeneratorClips []GeneratorClip `xml:"generator-clip,omitempty"`
}

type Title struct {
	XMLName xml.Name `xml:"title"`
	Ref          string       `xml:"ref,attr"`
	Lane         string       `xml:"lane,attr,omitempty"`
	Offset       string       `xml:"offset,attr"`
	Name         string       `xml:"name,attr"`
	Duration     string       `xml:"duration,attr"`
	Start        string       `xml:"start,attr,omitempty"`
	Params       []Param      `xml:"param,omitempty"`
	Text         TitleText    `xml:"text,omitempty"`
	TextStyleDef TextStyleDef `xml:"text-style-def,omitempty"`
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
	NestedVideos  []Video        `xml:"video,omitempty"`  // Support nested video elements
	NestedTitles  []Title        `xml:"title,omitempty"` // Support nested title elements
}

type AdjustTransform struct {
	Position string `xml:"position,attr,omitempty"`
	Scale    string `xml:"scale,attr,omitempty"`
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
	Name  string `xml:"name,attr"`
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
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
	Font      string `xml:"font,attr"`
	FontSize  string `xml:"fontSize,attr"`
	FontFace  string `xml:"fontFace,attr"`
	FontColor string `xml:"fontColor,attr"`
	Alignment string `xml:"alignment,attr"`
}