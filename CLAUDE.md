# Project Context for AI Assistance

## 🚨 ABSOLUTELY CRITICAL: NO XML STRING TEMPLATES 🚨

**NEVER EVER generate XML from hardcoded string templates with %s placeholders, use structs**

❌ BAD: xml := `<video ref="` + videoRef + `" name="` + name + `">` + content + `</video>`
❌ BAD: fmt.Sprintf(`<asset-clip ref="%s" name="%s"/>`, ref, name)
✅ GOOD: xml.MarshalIndent(&fcp.Video{Ref: videoRef, Name: name}, "", "    ")

**This is a CRITICAL VIOLATION if not followed. All FCPXML generation MUST use the fcp.* structs in the fcp package.**

  ## 🚨 ADDITIONAL VIOLATION PATTERNS TO AVOID 🚨

  **NEVER use any of these XML generation patterns:**
  ❌ BAD: `sequence.Spine.Content = fmt.Sprintf("<asset-clip...")`
  ❌ BAD: `xml := "<element>" + variable + "</element>"`
  ❌ BAD: Any manual XML string construction
  ❌ BAD: Setting .Content or .InnerXML fields with formatted strings

  **ALWAYS use struct fields and let xml.Marshal handle XML generation:**
  ✅ GOOD: `sequence.Spine.AssetClips = append(sequence.Spine.AssetClips, assetClip)`
  ✅ GOOD: Populate struct fields and use xml.MarshalIndent
  ✅ GOOD: Use proper XML struct tags for marshaling

  **If you see string concatenation or fmt.Sprintf with XML, STOP IMMEDIATELY.**

## REQUIRED: fcp/generate_test.go
ALWAYS run fcp pacakge tests after changing any code that generates FCPXML. This is MANDATORY.

These tests MUST pass without errors. If it fails, the logic is broken and must be fixed before the changes are complete.

## REQUIRED: DTD Validation
ALWAYS test for DTD validation after changing any code that generates FCPXML. This is MANDATORY.

xmllint --dtdvalid FCPXMLv1_13.dtd output.fcpxml

This validation MUST pass without errors. If it fails, the XML structure is broken and must be fixed before the changes are complete.

## 🚨 CRITICAL: Resource Management and ID Generation Requirements 🚨

**FCPXML crashes are caused by improper resource management and ID generation, NOT just format mismatches.**

### ❌ NAIVE APPROACHES THAT CAUSE CRASHES:

**Simple ID counting (BROKEN):**
```go
resourceCount := len(assets) + len(formats) + len(effects) + len(media)
assetID := fmt.Sprintf("r%d", resourceCount+1)  // RACE CONDITIONS!
```

**Problems with naive approach:**
- ID collisions when multiple resources created simultaneously
- No atomic transaction management
- No thread safety for concurrent access
- No validation of existing resource state
- Ignores FCP's complex resource relationship requirements

### ✅ REQUIRED PATTERN: Registry/Transaction System

**The old code in `reference/old_code/build2/core/` exists for critical reasons:**

1. **ResourceRegistry** - Centralized resource tracking:
   - Thread-safe ID management with mutex locks
   - Global uniqueness enforcement across all resource types
   - Existing resource detection to prevent duplicates
   - Consistent UID generation for file-based assets

2. **ResourceTransaction** - Atomic resource operations:
   - Reserve multiple IDs atomically to prevent collisions
   - Rollback capability if any operation fails
   - Ensure all-or-nothing resource creation
   - Proper cleanup on failure scenarios

3. **Critical FCP Crash Points:**
   - `addAssetClip:toObject:parentFormatID:` - The `parentFormatID` parameter suggests complex format relationships
   - ID collisions cause immediate crashes during import
   - Resource reference integrity must be maintained
   - Format compatibility between sequence and assets is complex
   - **🚨 CRITICAL: Missing frameDuration in format definitions causes immediate crashes**

### 🚨 LESSONS LEARNED FROM CRASH ANALYSIS:

1. **Don't guess at FCP requirements** - The crash happens deep in FCP's import logic
2. **Simple counting breaks** - Resource ID generation needs sophisticated state management  
3. **Format relationships are complex** - Not just asset→clip format matching
4. **The old code complexity exists for good reasons** - Registry/transaction pattern prevents crashes
5. **Thread safety matters** - Even in single-threaded contexts, atomic operations prevent corruption

### 📋 REQUIRED IMPLEMENTATION APPROACH:

**Before implementing any new FCPXML generation:**
1. Study the `ResourceRegistry` and `ResourceTransaction` patterns in detail
2. Understand WHY the old code was complex (crash prevention)
3. Implement proper resource management, don't bypass it for "simplicity"
4. Use atomic ID reservation, not naive counting
5. Test with actual FCP import, not just XML validation

**The simple approach of counting resources and incrementing IDs is fundamentally broken and causes FCP crashes.**

### 🔄 NEXT STEPS FOR CRASH RESOLUTION:

1. **Implement ResourceRegistry pattern** from `reference/old_code/build2/core/registry.go`
2. **Use ResourceTransaction pattern** from `reference/old_code/build2/core/transaction.go`  
3. **Study the `parentFormatID` relationship** in FCP's import logic
4. **Don't assume format consistency rules** without understanding FCP's actual requirements
5. **Test each change with actual FCP import** to verify crash resolution

**The complexity in the old code exists because FCPXML generation is inherently complex and FCP's requirements are strict.**

## 🚨 CRITICAL: Images Are Timeless - Asset Duration and Spine Element Requirements 🚨

**ROOT CAUSE IDENTIFIED: Analysis of working samples/png.fcpxml vs crash patterns revealed the critical "images are timeless" rule:**

### 🚨 CRITICAL DISCOVERY: Images Use Video Elements, NOT AssetClip Elements

**The addAssetClip:toObject:parentFormatID: crash occurs because images should use `<video>` elements in spine, not `<asset-clip>` elements.**

**Working Pattern Analysis (samples/png.fcpxml):**
1. **Asset duration**: `duration="0s"` (images are timeless)
2. **Spine element**: `<video ref="r2" ... duration="241241/24000s"/>` (display duration on Video element)
3. **Format**: No frameDuration attribute (image formats are timeless)

**Broken Pattern (our previous code):**
1. **Asset duration**: User-specified duration (e.g., 9 seconds converted to frames) ❌
2. **Spine element**: `<asset-clip>` ❌ (causes addAssetClip:toObject:parentFormatID crash)
3. **Format**: No frameDuration (this part was correct)

### 📋 MANDATORY IMAGE REQUIREMENTS:

**IMAGE ASSETS must follow the "timeless" pattern:**
1. **Asset duration**: MUST be `"0s"` (images have no inherent timeline duration)
2. **Display duration**: Applied ONLY to Video element in spine, NOT to asset
3. **Spine element**: MUST use `<video>` elements, NEVER `<asset-clip>` for images
4. **Format**: MUST NOT have frameDuration (image formats are timeless)

### ❌ BROKEN IMAGE PATTERN (causes addAssetClip:toObject:parentFormatID crash):
```xml
<!-- WRONG: Asset has duration, uses asset-clip in spine -->
<asset duration="215978/24000s" .../>
<spine>
    <asset-clip ref="r2" duration="215978/24000s"/>
</spine>
```

### ✅ CORRECT IMAGE PATTERN (works in FCP - from samples/png.fcpxml):
```xml
<!-- CORRECT: Asset duration="0s", Video element has display duration -->
<asset duration="0s" .../>
<spine>
    <video ref="r2" duration="241241/24000s"/>
</spine>
```

## 🚨 CRITICAL: Format FrameDuration Requirements 🚨

**Analysis of actual FCP crash (samples/crash.txt) revealed missing frameDuration causes addAssetClip:toObject:parentFormatID: crashes.**

### ❌ BROKEN IMAGE FORMAT DEFINITION (causes performAudioPreflightCheckForObject crash):
```xml
<format id="r3" name="FFVideoFormatRateUndefined" frameDuration="1001/24000s" width="1280" height="720" colorSpace="1-13-1"/>
```

### ✅ CORRECT IMAGE FORMAT DEFINITION (works in FCP - from top5orig.fcpxml):
```xml
<format id="r3" name="FFVideoFormatRateUndefined" width="262" height="282" colorSpace="1-13-1"/>
```

### ✅ CORRECT VIDEO FORMAT DEFINITION (works in FCP - sequence formats):
```xml
<format id="r1" name="FFVideoFormat720p2398" frameDuration="1001/24000s" width="1280" height="720" colorSpace="1-1-1 (Rec. 709)"/>
```

### 🔍 CRASH ANALYSIS FINDINGS:

**Comparing our crashing files vs working assets/top5orig.fcpxml:**

1. **Critical discovery**: samples/simple_video1.fcpxml is a VIDEO file, not an image example
2. **Real image format pattern** from working top5orig.fcpxml:
   - **Image format**: `name="FFVideoFormatRateUndefined"`, `colorSpace="1-13-1"`, **NO frameDuration**
   - **Image asset**: `duration="0s"`, `hasVideo="1"`, `videoSources="1"`, **NO audio properties**
3. **Crash cause identified**: Adding frameDuration to image formats triggers `performAudioPreflightCheckForObject` crash
4. **FCP audio preflight**: Expects image formats to be "timeless" (no frameDuration) for audio validation to pass
5. **Format type separation**: Sequence formats have frameDuration, image formats do not

### 📋 MANDATORY FORMAT REQUIREMENTS:

**Format definitions MUST follow type-specific rules:**

**SEQUENCE FORMATS (r1):**
1. **frameDuration**: REQUIRED - defines timeline timing (e.g., "1001/24000s")
2. **name**: REQUIRED - format identifier (e.g., "FFVideoFormat720p2398")
3. **colorSpace**: Use "1-1-1 (Rec. 709)" for HD video

**IMAGE ASSET FORMATS (r3+):**
1. **frameDuration**: FORBIDDEN - causes performAudioPreflightCheckForObject crash
2. **name**: REQUIRED - must be "FFVideoFormatRateUndefined" for image compatibility  
3. **colorSpace**: Use "1-13-1" for image formats (different from video)

**Critical Rules:**
- **Sequence formats**: Define timeline timing with frameDuration + named format
- **Image formats**: Must be "timeless" (no frameDuration) for FCP audio validation
- **Format separation**: Different format types serve different purposes in FCP's import logic
- **Audio preflight**: FCP validates audio relationships and expects image formats to have no timing

## CRITICAL: Unique ID Requirements
FCPXML requires ALL IDs to be unique within the document. Common violations include:

### Text Style IDs
- NEVER hardcode text-style-def IDs like "ts1"
- Multiple text overlays MUST have unique text-style-def IDs
- Use generateUID() or hash-based approach for uniqueness
- Example: "tsB139D196", "tsAC597A49" (not "ts1", "ts1")

### Asset and Resource IDs  
- All asset, format, effect, and media IDs must be unique
- Use proper ID generation functions that consider existing resources
- Check existing IDs before assigning new ones

### Common ID Collision Patterns to Avoid:
1. Hardcoded IDs in functions that get called multiple times
2. Not checking for existing IDs when adding new resources
3. Copy-pasting code without updating ID generation
4. Using simple counters that don't account for existing resources
5. **CRITICAL**: Inconsistent resource counting in ID generation functions - different functions counting different numbers of resource types (e.g., some counting 3 types: assets+formats+effects, others counting 4 types: assets+formats+effects+media)
6. Race conditions when creating multiple resources in the same transaction without using sequence generation

### ID Generation Best Practices:
- Use unified ID generation functions that count ALL resource types consistently
- For multiple resources created in one transaction, use sequence generation to avoid collisions
- Never assume resource counts are static during function execution

### UID Consistency Requirements:
- **CRITICAL**: Once FCP imports a media file with a specific UID, that UID is permanently associated with that file in the library
- Attempting to import the same file with a different UID causes "cannot be imported again with a different unique identifier" errors
- UID generation must be deterministic based on file content/name, not file path
- Use filename-based UID generation to ensure consistency across different working directories

When adding any new FCPXML elements with IDs, always ensure uniqueness across the entire document.

## CRITICAL: Frame Boundary Alignment
FCPXML durations MUST be aligned to frame boundaries to avoid "not on an edit frame boundary" errors in Final Cut Pro.

### Frame Rate and Time Base
- FCP uses a time base of 24000/1001 ≈ 23.976 fps for frame alignment
- Duration format: `(frames*1001)/24000s` where frames is an integer
- NEVER use simple `seconds * 24000` calculations - this creates non-frame-aligned durations

### Frame Boundary Violations:
- `21600000/24000s` = 900.0s (NON-FRAME-ALIGNED) ❌ - causes "not on an edit frame boundary" error
- `21599578/24000s` = 899.982s (FRAME-ALIGNED: 21578 frames) ✅
- The difference is small but FCP strictly enforces frame boundaries

### Correct Duration Conversion from Seconds:
```go
func ConvertSecondsToFCPDuration(seconds float64) string {
    // Convert to frame count using the sequence time base (1001/24000s frame duration)
    // This means 24000/1001 frames per second ≈ 23.976 fps
    framesPerSecond := 24000.0 / 1001.0
    exactFrames := seconds * framesPerSecond
    
    // Choose the frame count that gives the closest duration to the target
    floorFrames := int(math.Floor(exactFrames))
    ceilFrames := int(math.Ceil(exactFrames))
    
    floorDuration := float64(floorFrames) / framesPerSecond
    ceilDuration := float64(ceilFrames) / framesPerSecond
    
    var frames int
    if math.Abs(seconds-floorDuration) <= math.Abs(seconds-ceilDuration) {
        frames = floorFrames
    } else {
        frames = ceilFrames
    }
    
    // Format as rational using the sequence time base
    return fmt.Sprintf("%d/24000s", frames*1001)
}
```

### Exact Time Limitations:
- Due to FCP's 23.976fps timebase, exact round-second durations are often impossible
- For 900 seconds: closest frame-aligned durations are 899.982s (21578 frames) or 900.024s (21579 frames)
- The algorithm chooses the frame count that produces the duration closest to the target

### Always Use Frame-Aligned Durations:
- Asset durations must align to frame boundaries
- Clip durations must align to frame boundaries  
- Offset positions should align to frame boundaries when possible
- Use the build2/utils duration functions which implement proper frame alignment

this program is a swiff army knife for generating fcpxml files. There is a complex cli menu system for asking what specific army knife you want.

do not add complex logic to main.go that belongs in other packages.
have main.go call funcs in a package instead.

make sure your code compiles. You can run xmllint but do not run ./cutlass

review reference/FCPCAFE.md
reference/FCPXML.md
reference/ANIMATION.md
and FCPXMLv1_13.dtd

