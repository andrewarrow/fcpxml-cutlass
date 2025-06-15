package fcp

import (
	"fmt"
	"sync"
)

// ResourceRegistry provides centralized resource management with global ID uniqueness
type ResourceRegistry struct {
	mu sync.RWMutex

	// Resource tracking maps
	resources map[string]Resource // All resources by ID
	assets    map[string]*Asset
	formats   map[string]*Format
	effects   map[string]*Effect
	media     map[string]*Media

	// ID generation state
	nextResourceID int
	usedIDs        map[string]bool

	// UID tracking for FCP compatibility
	fileUIDs map[string]string // filename -> UID mapping

	// Project reference
	ml *FCPXML
}

// Resource represents any FCPXML resource
type Resource interface {
	GetID() string
	GetType() ResourceType
}

// ResourceType defines the different types of FCPXML resources
type ResourceType int

const (
	AssetResource ResourceType = iota
	FormatResource
	EffectResource
	MediaResource
)

// NewResourceRegistry creates a new resource registry
func NewResourceRegistry(ml *FCPXML) *ResourceRegistry {
	registry := &ResourceRegistry{
		resources: make(map[string]Resource),
		assets:    make(map[string]*Asset),
		formats:   make(map[string]*Format),
		effects:   make(map[string]*Effect),
		media:     make(map[string]*Media),
		usedIDs:   make(map[string]bool),
		fileUIDs:  make(map[string]string),
		ml:        ml,
	}

	// Initialize from existing FCPXML
	registry.initializeFromFCPXML()

	return registry
}

// initializeFromFCPXML scans existing FCPXML and registers all resources
func (r *ResourceRegistry) initializeFromFCPXML() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Register existing assets
	for i := range r.ml.Resources.Assets {
		asset := &r.ml.Resources.Assets[i]
		r.assets[asset.ID] = asset
		r.usedIDs[asset.ID] = true
		r.resources[asset.ID] = &AssetWrapper{asset}
	}

	// Register existing formats
	for i := range r.ml.Resources.Formats {
		format := &r.ml.Resources.Formats[i]
		r.formats[format.ID] = format
		r.usedIDs[format.ID] = true
		r.resources[format.ID] = &FormatWrapper{format}
	}

	// Register existing effects
	for i := range r.ml.Resources.Effects {
		effect := &r.ml.Resources.Effects[i]
		r.effects[effect.ID] = effect
		r.usedIDs[effect.ID] = true
		r.resources[effect.ID] = &EffectWrapper{effect}
	}

	// Register existing media
	for i := range r.ml.Resources.Media {
		media := &r.ml.Resources.Media[i]
		r.media[media.ID] = media
		r.usedIDs[media.ID] = true
		r.resources[media.ID] = &MediaWrapper{media}
	}

	// Calculate next available resource ID
	r.nextResourceID = len(r.resources) + 1
}

// ReserveIDs reserves multiple IDs in sequence to avoid collisions
func (r *ResourceRegistry) ReserveIDs(count int) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	ids := make([]string, count)
	for i := 0; i < count; i++ {
		for {
			id := fmt.Sprintf("r%d", r.nextResourceID)
			r.nextResourceID++

			if !r.usedIDs[id] {
				r.usedIDs[id] = true
				ids[i] = id
				break
			}
		}
	}

	return ids
}

// ReserveNextID reserves a single ID
func (r *ResourceRegistry) ReserveNextID() string {
	return r.ReserveIDs(1)[0]
}

// RegisterAsset registers an asset in the registry
func (r *ResourceRegistry) RegisterAsset(asset *Asset) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.assets[asset.ID] = asset
	r.resources[asset.ID] = &AssetWrapper{asset}
	r.ml.Resources.Assets = append(r.ml.Resources.Assets, *asset)
}

// RegisterFormat registers a format in the registry
func (r *ResourceRegistry) RegisterFormat(format *Format) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.formats[format.ID] = format
	r.resources[format.ID] = &FormatWrapper{format}
	r.ml.Resources.Formats = append(r.ml.Resources.Formats, *format)
}

// RegisterEffect registers an effect in the registry
func (r *ResourceRegistry) RegisterEffect(effect *Effect) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.effects[effect.ID] = effect
	r.resources[effect.ID] = &EffectWrapper{effect}
	r.ml.Resources.Effects = append(r.ml.Resources.Effects, *effect)
}

// RegisterMedia registers media in the registry
func (r *ResourceRegistry) RegisterMedia(media *Media) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.media[media.ID] = media
	r.resources[media.ID] = &MediaWrapper{media}
	r.ml.Resources.Media = append(r.ml.Resources.Media, *media)
}

// GetAsset retrieves an asset by ID
func (r *ResourceRegistry) GetAsset(id string) (*Asset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	asset, exists := r.assets[id]
	return asset, exists
}

// GetOrCreateAsset finds existing asset by file path or creates new one
func (r *ResourceRegistry) GetOrCreateAsset(filepath string) (*Asset, bool) {
	r.mu.RLock()

	// Check if asset already exists for this file
	for _, asset := range r.assets {
		if asset.MediaRep.Src == "file://"+filepath {
			r.mu.RUnlock()
			return asset, true // existing
		}
	}
	r.mu.RUnlock()

	return nil, false // not found
}

// GetResource retrieves any resource by ID
func (r *ResourceRegistry) GetResource(id string) (Resource, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resource, exists := r.resources[id]
	return resource, exists
}

// GenerateConsistentUID generates a consistent UID for a filename
func (r *ResourceRegistry) GenerateConsistentUID(filename string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if uid, exists := r.fileUIDs[filename]; exists {
		return uid
	}

	// Generate new UID using deterministic method
	uid := GenerateUID(filename)
	r.fileUIDs[filename] = uid
	return uid
}

// GetResourceCount returns the total number of resources
func (r *ResourceRegistry) GetResourceCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.resources)
}

// GetFCPXML returns the FCPXML project
func (r *ResourceRegistry) GetFCPXML() *FCPXML {
	return r.ml
}

// Wrapper types to implement Resource interface

type AssetWrapper struct {
	*Asset
}

func (a *AssetWrapper) GetID() string         { return a.ID }
func (a *AssetWrapper) GetType() ResourceType { return AssetResource }

type FormatWrapper struct {
	*Format
}

func (f *FormatWrapper) GetID() string         { return f.ID }
func (f *FormatWrapper) GetType() ResourceType { return FormatResource }

type EffectWrapper struct {
	*Effect
}

func (e *EffectWrapper) GetID() string         { return e.ID }
func (e *EffectWrapper) GetType() ResourceType { return EffectResource }

type MediaWrapper struct {
	*Media
}

func (m *MediaWrapper) GetID() string         { return m.ID }
func (m *MediaWrapper) GetType() ResourceType { return MediaResource }
