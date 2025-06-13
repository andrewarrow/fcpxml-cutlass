package core

import (
	"fmt"
	"path/filepath"

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
	if isPNGFile(absPath) {
		asset.Duration = "0s" // PNG assets use 0s duration in FCP
		asset.VideoSources = "1" // Required for image assets
	} else {
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

// createCompoundClipSpineContent creates the spine content for a compound clip
func (tx *ResourceTransaction) createCompoundClipSpineContent(videoAssetID, audioAssetID, baseName, duration string) string {
	// This is a simplified approach - creates XML directly without marshaling structs
	return fmt.Sprintf(`
                        <video ref="%s" offset="0s" name="%s" start="86399313/24000s" duration="%s">
                            <asset-clip ref="%s" lane="-1" offset="28799771/8000s" name="%s" duration="%s" format="r1" tcFormat="NDF" audioRole="dialogue"/>
                        </video>`,
		videoAssetID, baseName, duration,
		audioAssetID, baseName, duration,
	)
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