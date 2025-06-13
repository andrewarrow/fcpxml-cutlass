# Build Package Refactoring Plan

## Executive Summary

After extensive debugging of ID collisions, DTD validation failures, and FCP import issues, this document outlines a comprehensive refactoring plan for the build package to create a robust, maintainable, and reliable FCPXML generation system.

## Current Pain Points

### 1. **ID Management Chaos**
- **Inconsistent resource counting**: Different functions count different resource types when generating IDs
- **Race conditions**: Sequential resource creation causes ID collisions  
- **No centralized ID registry**: Functions generate IDs independently without coordination
- **Hardcoded IDs**: Magic strings like "r1", "r2" scattered throughout code

### 2. **UID Generation Problems**
- **Path-dependent UIDs**: Same file gets different UIDs from different directories
- **FCP library conflicts**: Once FCP imports a file with a UID, that UID is permanent
- **No collision detection**: Multiple files with same name but different paths cause conflicts

### 3. **Timeline Management Issues**
- **Duration calculation bugs**: Timeline duration stuck at "0s" causing FCP errors
- **Mixed element type handling**: Inconsistent parsing of `<video>`, `<asset-clip>`, `<ref-clip>`
- **Offset calculation errors**: Clips placed at wrong timeline positions

### 4. **Compound Clip Complexity**
- **Unnecessary compound clips**: All clips with audio become compound clips, even when simple structure would work
- **Multiple compound clip problems**: FCP rejects timelines with multiple compound clips in sequence
- **Resource structure bloat**: Compound clips create complex nested resource hierarchies

## Proposed Architecture

### Core Components

```
build/
├── core/
│   ├── registry.go           # Centralized resource registry
│   ├── transaction.go        # Atomic multi-resource operations
│   └── validation.go         # DTD and consistency validation
├── ids/
│   ├── generator.go          # Unified ID/UID generation
│   ├── resolver.go           # ID conflict detection/resolution
│   └── registry.go           # ID tracking and reuse
├── resources/
│   ├── assets.go             # Asset creation and management
│   ├── formats.go            # Format resource handling
│   ├── effects.go            # Effect resource management
│   └── media.go              # Compound clip media creation
├── timeline/
│   ├── builder.go            # Timeline construction
│   ├── calculator.go         # Duration and offset calculations
│   └── clips.go              # Clip placement and sequencing
├── templates/
│   ├── base.go               # Base FCPXML templates
│   ├── clips.go              # Clip templates
│   └── text.go               # Text overlay templates
└── api/
    ├── project.go            # High-level project operations
    ├── builder.go            # Fluent API for timeline building
    └── validator.go          # Input validation and sanitization
```

## 1. Resource Registry System

### Core Concept
A centralized registry that tracks all resources and their relationships, ensuring global uniqueness and consistency.

```go
type ResourceRegistry struct {
    resources map[string]Resource
    assets    map[string]*Asset
    formats   map[string]*Format  
    effects   map[string]*Effect
    media     map[string]*Media
    
    // ID generation state
    nextResourceID int
    usedIDs        map[string]bool
    
    // UID tracking for FCP compatibility
    fileUIDs       map[string]string  // filename -> UID mapping
}

func (r *ResourceRegistry) ReserveIDs(count int) []string
func (r *ResourceRegistry) GetOrCreateAsset(filepath string) *Asset
func (r *ResourceRegistry) GenerateConsistentUID(filename string) string
```

**Benefits:**
- **Global ID uniqueness**: Impossible to have ID collisions
- **Atomic operations**: Reserve multiple IDs in a single transaction
- **UID consistency**: Same file always gets same UID regardless of path
- **Resource reuse**: Detect and reuse existing assets

### Implementation Strategy
```go
// Before: Scattered ID generation
assetID := fmt.Sprintf("r%d", totalResources+1)  // Inconsistent counting

// After: Centralized registry
assetID := registry.ReserveNextID()
registry.RegisterAsset(assetID, asset)
```

## 2. Transaction-Based Operations

### Problem
Current code creates resources sequentially, leading to race conditions and partial failures.

### Solution
Atomic transactions that reserve all required resources upfront.

```go
type ResourceTransaction struct {
    registry *ResourceRegistry
    reserved []string
    created  []Resource
}

func (tx *ResourceTransaction) CreateCompoundClip(videoFile, audioFile string) error {
    // Reserve all IDs upfront
    ids := tx.registry.ReserveIDs(3) // video asset, audio asset, media
    
    // Create all resources atomically
    videoAsset := tx.CreateAsset(ids[0], videoFile)
    audioAsset := tx.CreateAsset(ids[1], audioFile)  
    media := tx.CreateMedia(ids[2], videoAsset, audioAsset)
    
    return tx.Commit()
}
```

**Benefits:**
- **No ID collisions**: All IDs reserved before any resource creation
- **Rollback capability**: Failed operations don't leave partial state
- **Consistency**: Either all resources created or none

## 3. Smart Clip Strategy

### Current Problem
All clips with audio become compound clips, causing FCP compatibility issues.

### Proposed Solution
Intelligent clip type selection based on content and context.

```go
type ClipStrategy interface {
    ShouldCreateCompoundClip(video, audio string, context TimelineContext) bool
    CreateOptimalClip(video, audio, text string) TimelineElement
}

type SmartClipStrategy struct{}

func (s *SmartClipStrategy) ShouldCreateCompoundClip(video, audio string, context TimelineContext) bool {
    // Only create compound clips when necessary
    if audio == "" {
        return false  // Simple video element
    }
    
    if context.HasExistingCompoundClips() && !context.SupportsMultipleCompoundClips() {
        return false  // Use simple video + audio lane approach
    }
    
    return true  // Safe to create compound clip
}
```

**Clip Type Matrix:**
| Video | Audio | Text | Result | Structure |
|-------|-------|------|--------|-----------|
| PNG   | None  | Yes  | `<video>` with `<title>` | Simple |
| PNG   | WAV   | Yes  | `<ref-clip>` to compound | Complex |
| PNG   | WAV   | Yes  | `<video>` with audio lane | Alternative |

## 4. Timeline Builder API

### Fluent Interface
```go
timeline := NewTimelineBuilder(registry).
    AddClip("image1.png").
        WithText("Hello World").
        WithDuration("10s").
    AddClip("image2.png").
        WithText("Second Slide").
        WithAudio("narration.wav").
        WithDuration("5s").
    Build()
```

### Benefits
- **Type safety**: Compile-time validation of clip configurations
- **Automatic optimization**: Builder chooses optimal clip structure
- **Duration management**: Automatic timeline duration calculation
- **Offset calculation**: Automatic clip positioning

## 5. Validation Framework

### Multi-Layer Validation
```go
type ValidationPipeline struct {
    validators []Validator
}

// Input validation
type InputValidator struct{}
func (v *InputValidator) Validate(project *Project) []ValidationError

// Structure validation  
type StructureValidator struct{}
func (v *StructureValidator) Validate(project *Project) []ValidationError

// DTD validation
type DTDValidator struct{}
func (v *DTDValidator) Validate(xmlContent []byte) []ValidationError

// FCP compatibility validation
type FCPValidator struct{}
func (v *FCPValidator) Validate(project *Project) []ValidationError
```

### Validation Rules
- **ID uniqueness**: All resource IDs unique across document
- **UID consistency**: File UIDs match expected patterns
- **Timeline coherence**: Duration calculations correct
- **Reference integrity**: All refs point to existing resources
- **DTD compliance**: Valid against FCPXML schema
- **FCP compatibility**: Known FCP limitations and workarounds

## 6. Template System

### Parameterized Templates
```go
type ClipTemplate struct {
    Name     string
    Template string
    Params   []TemplateParam
}

// Video clip with text overlay
var VideoWithTextTemplate = ClipTemplate{
    Name: "video-with-text",
    Template: `<video ref="{{.VideoRef}}" offset="{{.Offset}}" duration="{{.Duration}}">
        {{if .Text}}<title ref="{{.TextEffectRef}}" lane="1" offset="0s">
            {{template "text-style" .Text}}
        </title>{{end}}
    </video>`,
}
```

**Benefits:**
- **Consistency**: All clips use same template structure
- **Maintainability**: Changes in one place affect all instances
- **Flexibility**: Easy to add new clip types
- **Validation**: Templates can be pre-validated

## 7. Error Handling Strategy

### Structured Error Types
```go
type FCPXMLError struct {
    Type     ErrorType
    Location string
    Message  string
    Context  map[string]interface{}
}

type ErrorType int
const (
    IDCollisionError ErrorType = iota
    UIDConflictError
    DTDValidationError
    TimelineError
    ResourceNotFoundError
)
```

### Error Recovery
```go
func (b *Builder) AddClipWithRecovery(clip ClipConfig) error {
    err := b.AddClip(clip)
    if err != nil {
        switch e := err.(type) {
        case *IDCollisionError:
            // Regenerate ID and retry
            return b.retryWithNewID(clip)
        case *UIDConflictError:
            // Use existing UID
            return b.retryWithExistingUID(clip, e.ExistingUID)
        default:
            return err
        }
    }
    return nil
}
```

## 8. Testing Strategy

### Test Categories
1. **Unit Tests**: Individual component functionality
2. **Integration Tests**: Multi-component workflows  
3. **DTD Validation Tests**: Generated XML validates against schema
4. **FCP Import Tests**: Generated files import successfully into FCP
5. **Regression Tests**: Specific bugs don't reoccur

### Test Data Management
```go
// Standard test assets
var TestAssets = struct {
    PNG1080p    string
    PNG720p     string
    WAVMono     string
    WAVStereo   string
    FCPXMLValid string
}{
    PNG1080p:    "testdata/sample_1080p.png",
    PNG720p:     "testdata/sample_720p.png", 
    WAVMono:     "testdata/mono_audio.wav",
    WAVStereo:   "testdata/stereo_audio.wav",
    FCPXMLValid: "testdata/valid_project.fcpxml",
}
```

## 9. Migration Plan

### Phase 1: Foundation (Week 1-2)
- [ ] Implement ResourceRegistry
- [ ] Create unified ID generation system
- [ ] Build transaction framework
- [ ] Add comprehensive unit tests

### Phase 2: Core Features (Week 3-4)  
- [ ] Refactor asset creation using registry
- [ ] Implement smart clip strategy
- [ ] Create timeline builder API
- [ ] Add DTD validation pipeline

### Phase 3: Advanced Features (Week 5-6)
- [ ] Template system implementation
- [ ] Error recovery mechanisms
- [ ] FCP compatibility validation
- [ ] Performance optimization

### Phase 4: Polish (Week 7-8)
- [ ] Integration testing
- [ ] Documentation updates
- [ ] Migration guide creation
- [ ] Legacy code cleanup

## 10. Backward Compatibility

### API Preservation
```go
// Legacy API (preserved)
func addVideoToProject(projectFile, videoFile, withText, withSound string) error {
    // Delegate to new system
    builder := NewProjectBuilder(projectFile)
    return builder.AddClip(ClipConfig{
        VideoFile: videoFile,
        Text:      withText,
        AudioFile: withSound,
    }).Save()
}
```

### Migration Utilities
```go
func MigrateLegacyProject(oldProject string) (*Project, error)
func ValidateProjectCompatibility(project *Project) []CompatibilityIssue  
func FixCommonIssues(project *Project) *Project
```

## Success Metrics

### Quality Metrics
- **Zero ID collisions**: No duplicate resource IDs in any generated FCPXML
- **100% DTD compliance**: All generated files validate against FCPXML DTD
- **FCP import success**: All generated files import successfully into FCP
- **UID consistency**: Same files always generate same UIDs

### Performance Metrics
- **Sub-second generation**: Complex projects generate in <1 second
- **Memory efficiency**: <50MB memory usage for large projects
- **Incremental updates**: Add clips to existing projects in <100ms

### Developer Experience
- **Type safety**: Compile-time validation of all operations
- **Clear error messages**: Actionable error descriptions with context
- **Comprehensive docs**: Every API documented with examples
- **Easy testing**: Simple test data setup and validation

## Conclusion

This refactoring plan addresses all major issues discovered during debugging:
- **ID management** through centralized registry
- **UID consistency** through deterministic generation
- **Timeline reliability** through proper duration calculation
- **FCP compatibility** through smart clip strategies
- **Code maintainability** through clear architecture

The proposed architecture provides a solid foundation for reliable FCPXML generation while maintaining backward compatibility and enabling future extensibility.