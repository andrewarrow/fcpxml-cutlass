# Project Context for AI Assistance

never generate xml from hard coded string templates with %s placeholders, use structs
never run the program, just edit code, compile and fix errors


## Project Overview

Cutalyst is a Go-based command-line tool that converts video files into Final Cut Pro XML (FCPXML) format. It supports both local video files and YouTube videos (by downloading them first using yt-dlp). The tool has two main modes:

1. **Standard mode**: Converts a single video file into a simple FCPXML project
2. **Segment mode** (-s flag): Intelligently breaks YouTube videos into logical clips with title cards based on subtitle timing

## Architecture

This is a modular command-line application organized into specialized packages:

- **Main CLI handler** (`main.go`): Handles command-line arguments, orchestrates the conversion process, and coordinates between packages
- **VTT Package** (`vtt/`): Advanced WebVTT subtitle parsing and intelligent clip segmentation
- **YouTube Package** (`youtube/`): YouTube video detection, video downloading, and subtitle fetching
- **FCP Package** (`fcp/`): FCPXML generation with both simple and complex timeline support
- **Wikipedia Package** (`wikipedia/`): Placeholder for future Wikipedia integration features

The application follows a clean separation of concerns with each package handling its specific domain.

### Standard Mode Flow:
1. Parse command-line arguments
2. Detect if input is a YouTube ID or local file (using `youtube.IsYouTubeID`)
3. Download YouTube video and subtitles if needed (using `youtube.DownloadVideo` and `youtube.DownloadSubtitles`)
4. Generate simple FCPXML file with single asset (using `fcp.GenerateStandard`)
5. Output the converted file

### Segment Mode Flow:
1. Parse command-line arguments (with -s flag)
2. Download YouTube video and subtitles using yt-dlp
3. Parse VTT subtitle file to extract timing and text (using `vtt.ParseFile`)
4. Intelligently segment video into 6-18 second clips based on natural speech breaks (using `vtt.SegmentIntoClips`)
5. Generate complex FCPXML with multiple asset clips and title cards (using `fcp.GenerateClipFCPXML`)
6. Output segmented project file

## Key Technologies

- **Go 1.23.5**: Main programming language for cross-platform CLI tool
- **yt-dlp**: External dependency for YouTube video and subtitle downloading
- **FCPXML format**: XML-based project format used by Final Cut Pro
- **WebVTT (VTT)**: Subtitle format parsed for segment timing and text extraction
- **XML marshaling**: Go's encoding/xml package for generating structured FCPXML output
- **Regular expressions**: For parsing VTT timestamps and cleaning subtitle text

## Project Structure

```
cutalyst/
├── main.go              # Main CLI application and orchestration
├── go.mod              # Go module definition
├── CLAUDE.md           # AI assistant context (this file)
├── README.md           # User documentation
├── vtt/                # VTT subtitle processing package
│   └── vtt.go          # VTT parsing, clip segmentation, types
├── youtube/            # YouTube integration package
│   └── youtube.go      # Video/subtitle download, ID detection
├── fcp/                # Final Cut Pro XML generation package
│   ├── types.go        # FCPXML data structures
│   └── generator.go    # FCPXML generation functions
├── wikipedia/          # Future Wikipedia integration
│   └── parse.go        # Placeholder for Wikipedia parsing
├── foo/                # Sample/test files directory
│   ├── *.fcpxml        # Sample FCPXML files
│   └── *.py            # Python scripts for testing/generation
└── generated files:
    ├── *.fcpxml        # Generated Final Cut Pro projects
    ├── *.mov           # Downloaded YouTube videos
    └── *.vtt           # Downloaded WebVTT subtitle files
```

## Important Files and Their Purposes

### Core Application
- `main.go`: Contains main CLI logic, coordinates between packages, handles segment mode orchestration (lines 16-138)
- `go.mod`: Defines the Go module as "cutalyst" with Go 1.23.5

### VTT Package (`vtt/`)
- `vtt.go`: VTT parsing, intelligent clip segmentation, and related data structures
  - `Segment` type: Represents individual VTT subtitle segments
  - `Clip` type: Represents generated video clips with timing and text
  - `ParseFile()`: Parses WebVTT files, extracts timestamps and text
  - `ParseTime()`: Converts VTT timestamp format to Go time.Duration
  - `SegmentIntoClips()`: Intelligently groups VTT segments into clips based on natural breaks

### YouTube Package (`youtube/`)
- `youtube.go`: YouTube integration functionality
  - `IsYouTubeID()`: Detects if input string is a YouTube video ID
  - `DownloadVideo()`: Downloads YouTube video using yt-dlp
  - `DownloadSubtitles()`: Downloads English auto-generated subtitles

### FCP Package (`fcp/`)
- `types.go`: Complete FCPXML data structures for XML marshaling
- `generator.go`: FCPXML generation functions
  - `GenerateStandard()`: Creates basic single-asset FCPXML
  - `BuildClipFCPXML()`: Creates complex multi-clip FCPXML with title cards
  - `GenerateClipFCPXML()`: Wrapper that writes clip FCPXML to file
  - `FormatDurationForFCPXML()`: Converts Go duration to frame-accurate FCPXML format

### Generated Files
- `*.vtt files`: WebVTT subtitle files downloaded from YouTube containing timing and text data
- `*.mov files`: Downloaded video files from YouTube
- `*_clips.fcpxml files`: Generated segmented project files with multiple clips and title cards

## Current Development Status

- **Latest refactoring**: 
  - Reorganized code into modular packages (vtt/, youtube/, fcp/)
  - Removed monolithic fcpxml.go and types.go files
  - Improved separation of concerns and maintainability
- **Latest features**: 
  - Segment mode (-s flag) for intelligent video clip generation
  - VTT subtitle parsing and text extraction
  - Automatic clip segmentation based on speech patterns (6-18 second clips)
  - Title card generation between clips
  - Frame-accurate timing for 30fps video
- **Fully implemented**: Standard mode conversion, YouTube integration, VTT processing, modular architecture
- **Known issues**: 
  - Hardcoded 3600s duration in standard mode FCPXML (segment mode calculates actual duration)
  - Basic error handling for missing yt-dlp dependency
  - No support for custom video formats or resolutions
  - Segment mode only works with YouTube videos that have subtitles
  - Wikipedia package is placeholder with no functionality

## Development Patterns

- **Package organization**: Domain-driven design with clear separation of concerns
- **Coding standards**: Standard Go conventions, error handling with early returns
- **Design patterns used**: Simple procedural approach, single responsibility functions
- **Import structure**: Uses relative imports for internal packages (cutalyst/vtt, cutalyst/youtube, cutalyst/fcp)
- **Error handling**: Consistent error propagation with descriptive messages
- **Testing approach**: No automated tests currently present

## Common Tasks

### Adding a new feature
1. Determine which package the feature belongs to (vtt/, youtube/, fcp/, or main.go)
2. Add new functions to the appropriate package
3. Export functions if they need to be used by other packages (capitalize function names)
4. Update imports in main.go if using new package functions
5. Test with both local files and YouTube videos
6. Ensure code compiles with `go build`

### Adding a new package
1. Create new directory under project root
2. Add Go files with appropriate package name
3. Define exported functions and types
4. Import package in main.go or other packages as needed
5. Update project documentation

### Debugging issues
Always add print statements to help debug:
- Check if yt-dlp is installed and accessible
- Verify input file exists before processing
- Print YouTube URL construction for debugging downloads
- Log VTT parsing steps and segment counts
- Print FCPXML generation steps and file paths
- Verify package imports and function calls

### Running tests
No formal test framework currently exists. Manual testing approach:
- Test standard mode with local files: `go run . -i video.mp4`
- Test standard mode with YouTube: `go run . -i dQw4w9WgXcQ`
- Test segment mode: `go run . -i dQw4w9WgXcQ -s`
- Verify generated FCPXML imports correctly into Final Cut Pro
- Check that segment mode creates clips with proper timing and title cards
- Ensure VTT parsing handles various subtitle formats correctly
- Test compilation: `go build`

## Integration Points

- **External tools**: yt-dlp (required for YouTube functionality)
- **File system**: Reads/writes local video files and FCPXML outputs
- **YouTube API**: Indirectly through yt-dlp for video/subtitle downloading
- **Package dependencies**: Internal packages depend on each other for data types and functions

## Performance Considerations

- YouTube downloads can be slow depending on video size and network speed
- FCPXML generation is fast as it's template-based
- VTT parsing is efficient with streaming file reading
- Consider file size limits for very large video files
- Memory usage is minimal as the tool doesn't process video content directly
- Package modularization improves compile times for incremental changes

## Dependencies

### External Dependencies
- **Required**: yt-dlp must be installed and available in system PATH
- **Go modules**: No external Go dependencies currently used
- **System requirements**: File system write permissions for output directory

### Internal Package Dependencies
- `main.go` → imports all packages (vtt, youtube, fcp)
- `fcp/generator.go` → imports vtt package for Clip type
- All packages are self-contained otherwise

## Command Line Interface

### Standard Mode
Usage: `cutalyst -i <input_file> [output_file]`

### Segment Mode  
Usage: `cutalyst -i <youtube_id> -s [output_file]`

### Parameters
- `-i <input_file>`: Input file (required) - can be local file path or 11-character YouTube ID
- `-s`: Segment mode flag - breaks video into logical clips with title cards (only works with YouTube videos)
- `[output_file]`: Optional output filename 
  - Standard mode: defaults to "test.fcpxml"
  - Segment mode: defaults to "{youtube_id}_clips.fcpxml"
- Automatically adds `.fcpxml` extension if not present

## Key Functions and Their Locations

### Main Orchestration (`main.go`)
- `main()`: CLI argument parsing and main application flow (lines 16-86)
- `breakIntoLogicalParts()`: Coordinates segment mode processing (lines 88-131)

### VTT Processing (`vtt/vtt.go`)
- `ParseFile(vttPath)`: Parses WebVTT files, extracts timestamps and text (lines 60-111)
- `ParseTime(timeStr)`: Converts VTT timestamp format to Go time.Duration (lines 29-58)
- `SegmentIntoClips(segments, minDuration, maxDuration)`: Intelligently groups VTT segments into clips (lines 113-178)

### YouTube Integration (`youtube/youtube.go`)
- `IsYouTubeID(input)`: Detects 11-character YouTube IDs (lines 10-12)
- `DownloadVideo(youtubeID)`: Downloads video using yt-dlp (lines 14-26)
- `DownloadSubtitles(youtubeID)`: Downloads subtitles using yt-dlp (lines 28-40)

### FCPXML Generation (`fcp/generator.go`)
- `GenerateStandard(inputFile, outputFile)`: Creates basic single-asset FCPXML (lines 22-94)
- `BuildClipFCPXML(clips, videoPath)`: Creates complex multi-clip FCPXML with title cards (lines 96-222)
- `GenerateClipFCPXML(clips, videoPath, outputPath)`: File writing wrapper (lines 224-237)
- `FormatDurationForFCPXML(d)`: Converts Go duration to frame-accurate FCPXML format (lines 14-20)

## YouTube Integration

- Detects YouTube IDs by checking for 11-character strings without dots
- Downloads video using yt-dlp with .mov format output
- Downloads English auto-generated subtitles when available
- Constructs YouTube URLs as: `https://www.youtube.com/watch?v={ID}`
- Uses exec.Command to run yt-dlp as external process

## Package Import Guidelines

- Use internal package imports: `"cutalyst/packagename"`
- Export functions and types that need to be used across packages (capitalize names)
- Keep package dependencies minimal and unidirectional
- Each package should have a clear, single responsibility
- Avoid circular dependencies between packages

## FCPXML Timeline Layout and Lane Stacking

**CRITICAL UNDERSTANDING:** FCPXML timeline layout works fundamentally differently than expected.

### Timeline Layout Rules
1. **Spine elements** = Horizontal arrangement in Final Cut Pro timeline
2. **Lane elements** = Vertical stacking ABOVE spine elements (NOT separate spine elements)

### Correct Structure for Vertical Stacking
To achieve vertical stacking in Final Cut Pro timeline, elements must be **nested** with lanes:

```xml
<spine>
    <!-- Main video on spine (appears horizontally) -->
    <video ref="r2" offset="0s" name="Main">
        <!-- Nested video with lane="1" stacks VERTICALLY above main video -->
        <video ref="r2" lane="1" offset="0s" name="Stacked Above">
            <!-- Content for stacked element -->
        </video>
        <!-- Nested video with lane="2" stacks even higher -->
        <video ref="r2" lane="2" offset="0s" name="Stacked Higher">
            <!-- Content for element stacked above lane="1" -->
        </video>
    </video>
    
    <!-- Second main video (appears horizontally, separate from first) -->
    <video ref="r2" offset="10s" name="Second Main">
        <!-- This appears horizontally next to first video -->
    </video>
</spine>
```

### INCORRECT Approach (Does NOT create vertical stacking)
```xml
<spine>
    <video ref="r2" offset="0s" name="Video 1" />
    <video ref="r2" lane="1" offset="0s" name="Video 2" />  <!-- Still horizontal! -->
    <video ref="r2" lane="2" offset="0s" name="Video 3" />  <!-- Still horizontal! -->
</spine>
```

### CORRECT Approach (Creates vertical stacking)
```xml
<spine>
    <video ref="r2" offset="0s" name="Main Video">
        <video ref="r2" lane="1" offset="0s" name="Stacked Video" />  <!-- Vertical! -->
        <title ref="r3" lane="2" offset="0s" name="Stacked Title" />  <!-- Higher! -->
    </video>
</spine>
```

### Implementation in Go Structs
The `Video` struct in `fcp/types.go` must support nested elements:

```go
type Video struct {
    XMLName xml.Name `xml:"video"`
    Ref     string   `xml:"ref,attr"`
    Lane    string   `xml:"lane,attr,omitempty"`  // Only for nested elements
    Offset  string   `xml:"offset,attr"`
    Name    string   `xml:"name,attr"`
    // ... other fields ...
    NestedVideos []Video `xml:"video,omitempty"`   // REQUIRED for vertical stacking
    NestedTitles []Title `xml:"title,omitempty"`   // REQUIRED for vertical stacking
}
```

### Key Learnings from Testing
1. **Reference file analysis**: `table.fcpxml` shows the correct pattern with nested `lane="1"` elements
2. **Iterative testing**: Started with minimal 2-element test, then added nested structure
3. **Structure validation**: Only nested elements with lanes create vertical stacking
4. **Timeline behavior**: Spine elements always appear horizontally; lanes stack vertically within their parent

### Testing Commands
- Generate test FCPXML: `go build && ./cutalyst -i "Andre_Agassi" -w tennis.fcpxml`
- Verify structure by opening generated `tennis.fcpxml` in Final Cut Pro
- Compare timeline layout with working reference `table.fcpxml`
