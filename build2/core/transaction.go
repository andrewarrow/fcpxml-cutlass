package core

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"

	"cutlass/fcp"
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
func (tx *ResourceTransaction) CreateAsset(id, filePath, baseName, duration string, formatID string) (*fcp.Asset, error) {
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
	asset := &fcp.Asset{
		ID:       id,
		Name:     baseName,
		UID:      uid,
		Start:    "0s",
		Duration: duration,
		HasVideo: "1",
		Format:   formatID,
		MediaRep: fcp.MediaRep{
			Kind: "original-media",
			Sig:  uid,
			Src:  "file://" + absPath,
		},
	}
	
	// Set file-type specific properties
	if isImageFile(absPath) {
		// Image files (PNG, JPG, JPEG) should NOT have audio properties
		asset.VideoSources = "1" // Required for image assets
		// Note: Duration is set by caller, not hardcoded to "0s"
	} else {
		// Video files have audio properties
		asset.HasAudio = "1"
		asset.AudioSources = "1"
		asset.AudioChannels = "2"
	}
	
	tx.created = append(tx.created, &AssetWrapper{asset})
	return asset, nil
}

// CreateFormat creates a format with transaction management
func (tx *ResourceTransaction) CreateFormat(id, name, width, height, colorSpace string) (*fcp.Format, error) {
	if tx.rolled {
		return nil, fmt.Errorf("transaction has been rolled back")
	}
	
	format := &fcp.Format{
		ID:         id,
		Name:       name,
		Width:      width,
		Height:     height,
		ColorSpace: colorSpace,
	}
	
	tx.created = append(tx.created, &FormatWrapper{format})
	return format, nil
}

// CreateEffect creates an effect with transaction management
func (tx *ResourceTransaction) CreateEffect(id, name, uid string) (*fcp.Effect, error) {
	if tx.rolled {
		return nil, fmt.Errorf("transaction has been rolled back")
	}
	
	effect := &fcp.Effect{
		ID:   id,
		Name: name,
		UID:  uid,
	}
	
	tx.created = append(tx.created, &EffectWrapper{effect})
	return effect, nil
}

// CreateCompoundClipMedia creates compound clip media with transaction management
func (tx *ResourceTransaction) CreateCompoundClipMedia(id, baseName, duration string, videoAssetID, audioAssetID string) (*fcp.Media, error) {
	if tx.rolled {
		return nil, fmt.Errorf("transaction has been rolled back")
	}
	
	// Generate UID for compound clip
	mediaUID := tx.registry.GenerateConsistentUID(baseName + "_compound")
	
	// Create spine content for the compound clip
	spineContent := tx.createCompoundClipSpineContent(videoAssetID, audioAssetID, baseName, duration)
	
	// Create compound clip media
	media := &fcp.Media{
		ID:      id,
		Name:    baseName + " Clip",
		UID:     mediaUID,
		ModDate: "2025-06-13 10:53:41 -0700", // Use current time format
		Sequence: fcp.Sequence{
			Format:      "r1",
			Duration:    duration,
			TCStart:     "0s",
			TCFormat:    "NDF",
			AudioLayout: "stereo",
			AudioRate:   "48k",
			Spine: fcp.Spine{
				Content: spineContent,
			},
		},
	}
	
	tx.created = append(tx.created, &MediaWrapper{media})
	return media, nil
}

// createCompoundClipSpineContent creates the spine content for a compound clip using structs
func (tx *ResourceTransaction) createCompoundClipSpineContent(videoAssetID, audioAssetID, baseName, duration string) string {
	// Create audio asset-clip struct
	audioClip := fcp.AssetClip{
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
	video := fcp.Video{
		Ref:      videoAssetID,
		Offset:   "0s",
		Name:     baseName,
		Start:    "86399313/24000s",
		Duration: duration,
	}
	
	// Note: The fcp.Video struct doesn't support nested asset-clips directly
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

// isPNGFile checks if the given file is a PNG image
func isPNGFile(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".png" || ext == ".PNG"
}

// isImageFile checks if the given file is an image (PNG, JPG, JPEG)
func isImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg"
}