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

this program is a swiff army knife for generating fcpxml files. There is a complex cli menu system for asking what specific army knife you want.

do not add complex logic to main.go that belongs in other packages.
have main.go call funcs in a package instead.

make sure your code compiles, but do not run any of the menu options yourself. You can run xmllint but do not run ./cutlass

review reference/FCPCAFE.md
reference/FCPXML.md
reference/ANIMATION.md
and FCPXMLv1_13.dtd

## Build Package Mission
The build package (build/command.go) is the foundation for ALL FCPXML generation in this repo. It provides:
- Rock solid FCPXML generation using proper structs (not string templates)
- Correct time calculations, R2 vs R1 handling, UID generation, and spine management
- Template-based approach for reusable FCPXML components
- DTD validation compliance

All new FCPXML features should be built on the build package foundation. Other packages may have legacy logic that should be learned from but not duplicated.
