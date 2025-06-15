package fcp

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
)

// ResourceTransaction provides atomic multi-resource operations
type ResourceTransaction struct {
	registry *ResourceRegistry
	reserved []string
	created  []Resource
	rolled   bool
}

// NewTransaction creates a new resource transaction
func NewTransaction(registry *ResourceRegistry) *ResourceTransaction {
	return &ResourceTransaction{
		registry: registry,
		reserved: make([]string, 0),
		created:  make([]Resource, 0),
	}
}

// ReserveIDs reserves multiple IDs for this transaction
func (tx *ResourceTransaction) ReserveIDs(count int) []string {
	if tx.rolled {
		return nil
	}

	ids := tx.registry.ReserveIDs(count)
	tx.reserved = append(tx.reserved, ids...)
	return ids
}

// CreateAsset creates an asset with transaction management
func (tx *ResourceTransaction) CreateAsset(id, filePath, baseName, duration string, formatID string) (*Asset, error) {
	if tx.rolled {
		return nil, fmt.Errorf("transaction has been rolled back")
	}

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Generate consistent UID
	uid := tx.registry.GenerateConsistentUID(filepath.Base(filePath))

	// Create asset
	asset := &Asset{
		ID:       id,
		Name:     baseName,
		UID:      uid,
		Start:    "0s",
		Duration: duration,
		HasVideo: "1",
		Format:   formatID,
		MediaRep: MediaRep{
			Kind: "original-media",
			Sig:  uid,
			Src:  "file://" + absPath,
		},
	}

	// Set file-type specific properties
	if isImageFile(absPath) {
		// 🚨 CRITICAL: Images are timeless - asset duration MUST be "0s"
		// Display duration is applied only to Video element in spine, not asset
		// This matches working samples/png.fcpxml pattern: asset duration="0s"
		asset.Duration = "0s" // CRITICAL: Override caller duration for images
		asset.VideoSources = "1" // Required for image assets
		// Image files (PNG, JPG, JPEG) should NOT have audio properties
	} else if isAudioFile(absPath) {
		// Audio files have only audio properties, NO video properties
		asset.HasVideo = "" // Remove video properties for audio
		asset.HasAudio = "1"
		asset.AudioSources = "1"
		asset.AudioChannels = "2"
		asset.AudioRate = "48000"
		// Note: Duration remains as provided by caller (audio duration)
	} else {
		// Video files have both audio and video properties
		asset.HasAudio = "1"
		asset.AudioSources = "1"
		asset.AudioChannels = "2"
		asset.AudioRate = "48000"
	}

	tx.created = append(tx.created, &AssetWrapper{asset})
	return asset, nil
}

// CreateFormat creates a format with transaction management
// 🚨 CRITICAL: frameDuration should ONLY be set for sequence formats, NOT image formats
// Image formats must NOT have frameDuration or FCP's performAudioPreflightCheckForObject crashes
// Analysis of working top5orig.fcpxml shows image formats have NO frameDuration attribute
func (tx *ResourceTransaction) CreateFormat(id, name, width, height, colorSpace string) (*Format, error) {
	if tx.rolled {
		return nil, fmt.Errorf("transaction has been rolled back")
	}

	format := &Format{
		ID:         id,
		Name:       name,
		Width:      width,
		Height:     height,
		ColorSpace: colorSpace,
		// Note: FrameDuration intentionally omitted - only sequence formats need frameDuration
	}

	tx.created = append(tx.created, &FormatWrapper{format})
	return format, nil
}

// CreateEffect creates an effect with transaction management
func (tx *ResourceTransaction) CreateEffect(id, name, uid string) (*Effect, error) {
	if tx.rolled {
		return nil, fmt.Errorf("transaction has been rolled back")
	}

	effect := &Effect{
		ID:   id,
		Name: name,
		UID:  uid,
	}

	tx.created = append(tx.created, &EffectWrapper{effect})
	return effect, nil
}

// createCompoundClipSpineContent creates the spine content for a compound clip using structs
func (tx *ResourceTransaction) createCompoundClipSpineContent(videoAssetID, audioAssetID, baseName, duration string) string {
	// Create audio asset-clip struct
	audioClip := AssetClip{
		Ref:       audioAssetID,
		Lane:      "-1",
		Offset:    "28799771/8000s",
		Name:      baseName,
		Duration:  duration,
		Format:    "r1",
		TCFormat:  "NDF",
		AudioRole: "dialogue",
	}

	// Create video element with nested audio
	video := Video{
		Ref:      videoAssetID,
		Offset:   "0s",
		Name:     baseName,
		Start:    "86399313/24000s",
		Duration: duration,
	}

	// Note: The Video struct doesn't support nested asset-clips directly
	// So we need a hybrid approach here - marshal the video and manually insert the audio clip
	videoXML, err := xml.MarshalIndent(&video, "                        ", "    ")
	if err != nil {
		return "<!-- Error marshaling compound clip video: " + err.Error() + " -->"
	}

	audioXML, err := xml.MarshalIndent(&audioClip, "                            ", "    ")
	if err != nil {
		return "<!-- Error marshaling compound clip audio: " + err.Error() + " -->"
	}

	// Insert the audio clip before the closing video tag
	videoStr := string(videoXML)
	videoStr = strings.Replace(videoStr, "</video>", "    "+string(audioXML)+"\n                        </video>", 1)

	return videoStr
}

// Commit commits all created resources to the registry
func (tx *ResourceTransaction) Commit() error {
	if tx.rolled {
		return fmt.Errorf("transaction has been rolled back")
	}

	// Register all created resources
	for _, resource := range tx.created {
		switch r := resource.(type) {
		case *AssetWrapper:
			tx.registry.RegisterAsset(r.Asset)
		case *FormatWrapper:
			tx.registry.RegisterFormat(r.Format)
		case *EffectWrapper:
			tx.registry.RegisterEffect(r.Effect)
		case *MediaWrapper:
			tx.registry.RegisterMedia(r.Media)
		}
	}

	return nil
}

// Rollback rolls back the transaction (IDs remain reserved)
func (tx *ResourceTransaction) Rollback() {
	tx.rolled = true
	tx.created = nil
}
