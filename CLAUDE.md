# Project Context for AI Assistance

never generate xml from hard coded string templates with %s placeholders, use structs

## REQUIRED: DTD Validation
ALWAYS test for DTD validation after changing any code that generates FCPXML. This is MANDATORY.

xmllint --dtdvalid FCPXMLv1_13.dtd output.fcpxml

This validation MUST pass without errors. If it fails, the XML structure is broken and must be fixed before the changes are complete.

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
