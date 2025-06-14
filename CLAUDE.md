# Project Context for AI Assistance

never generate xml from hard coded string templates with %s placeholders, use structs

## REQUIRED: DTD Validation
ALWAYS test for DTD validation after changing any code that generates FCPXML. This is MANDATORY.

xmllint --dtdvalid FCPXMLv1_13.dtd output.fcpxml

This validation MUST pass without errors. If it fails, the XML structure is broken and must be fixed before the changes are complete.

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

make sure your code compiles, but do not run any of the menu options yourself. You can run xmllint but do not run ./cutlass

review reference/FCPCAFE.md
reference/FCPXML.md
reference/ANIMATION.md
and FCPXMLv1_13.dtd

## CRITICAL: Build Package Mission
The build package (build/command.go) and build2 package are the foundation for ALL FCPXML generation in this repo. They provide:
- **MANDATORY**: Rock solid FCPXML generation using proper structs (NEVER string templates with %s placeholders)
- Correct time calculations, R2 vs R1 handling, UID generation, and spine management
- Template-based approach for reusable FCPXML components
- DTD validation compliance
- Timeline continuity without gaps between clips

**CRITICAL**: All new FCPXML features MUST be built on the build/build2 package foundation. Other packages may have legacy logic that should be learned from but NEVER duplicated. Any FCPXML generation that bypasses these packages and uses string templates is a critical violation.
