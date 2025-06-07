# Project Context for AI Assistance

Never run the program, only edit code make sure it compiles.

## Project Overview

Cutalyst is a Go-based command-line tool that converts video files into Final Cut Pro XML (FCPXML) format. It supports both local video files and YouTube videos (by downloading them first using yt-dlp). The tool has two main modes:

1. **Standard mode**: Converts a single video file into a simple FCPXML project
2. **Segment mode** (-s flag): Intelligently breaks YouTube videos into logical clips with title cards based on subtitle timing

## Architecture

This is a command-line application with several key components:
- **Main CLI handler** (`main.go`): Handles command-line arguments, YouTube video detection/downloading, and orchestrates the conversion process
- **Standard FCPXML generator** (`generateFCPXML` function): Creates basic XML structure for single video files
- **Segment processor** (`fcpxml.go`): Advanced VTT subtitle parsing and intelligent clip segmentation
- **Advanced FCPXML builder** (`buildClipFCPXML` function): Creates complex timeline with multiple clips and title cards

The application follows two different flows:

### Standard Mode Flow:
1. Parse command-line arguments
2. Detect if input is a YouTube ID or local file
3. Download YouTube video and subtitles if needed (using yt-dlp)
4. Generate simple FCPXML file with single asset
5. Output the converted file

### Segment Mode Flow:
1. Parse command-line arguments (with -s flag)
2. Download YouTube video and subtitles using yt-dlp
3. Parse VTT subtitle file to extract timing and text
4. Intelligently segment video into 6-18 second clips based on natural speech breaks
5. Generate complex FCPXML with multiple asset clips and title cards
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
├── main.go              # Main CLI application and YouTube handling
├── fcpxml.go           # Advanced FCPXML generation and VTT processing
├── go.mod              # Go module definition
├── CLAUDE.md           # AI assistant context (this file)
├── README.md           # User documentation
├── foo/                # Sample/test files directory
│   ├── *.fcpxml        # Sample FCPXML files
│   └── *.py            # Python scripts for testing/generation
└── generated files:
    ├── *.fcpxml        # Generated Final Cut Pro projects
    ├── *.mov           # Downloaded YouTube videos
    └── *.vtt           # Downloaded WebVTT subtitle files
```

## Important Files and Their Purposes

- `main.go`: Contains main CLI logic, YouTube video detection, yt-dlp integration, and basic FCPXML generation (lines 13-175)
- `fcpxml.go`: Advanced features including VTT parsing, intelligent clip segmentation, and complex FCPXML generation with title cards (lines 1-528)
- `go.mod`: Defines the Go module as "cutalyst" with Go 1.23.5
- `*.vtt files`: WebVTT subtitle files downloaded from YouTube containing timing and text data
- `*.mov files`: Downloaded video files from YouTube
- `*_clips.fcpxml files`: Generated segmented project files with multiple clips and title cards

## Current Development Status

- **Latest features**: 
  - Segment mode (-s flag) for intelligent video clip generation
  - VTT subtitle parsing and text extraction
  - Automatic clip segmentation based on speech patterns (6-18 second clips)
  - Title card generation between clips
  - Frame-accurate timing for 30fps video
- **Fully implemented**: Standard mode conversion, YouTube integration, VTT processing
- **Known issues**: 
  - Hardcoded 3600s duration in standard mode FCPXML (segment mode calculates actual duration)
  - Basic error handling for missing yt-dlp dependency
  - No support for custom video formats or resolutions
  - Segment mode only works with YouTube videos that have subtitles

## Development Patterns

- Coding standards: Standard Go conventions, error handling with early returns
- Design patterns used: Simple procedural approach, single responsibility functions
- Testing approach: No automated tests currently present

## Common Tasks

### Adding a new feature
1. Modify the main CLI flow in `main.go` if it affects argument parsing
2. Add new functions for specific functionality
3. Update the FCPXML generation if it affects the output format
4. Test with both local files and YouTube videos

### Debugging issues
Always add print statements to help debug:
- Check if yt-dlp is installed and accessible
- Verify input file exists before processing
- Print YouTube URL construction for debugging downloads
- Log FCPXML generation steps

### Running tests
No formal test framework currently exists. Manual testing approach:
- Test standard mode with local files: `go run . -i video.mp4`
- Test standard mode with YouTube: `go run . -i dQw4w9WgXcQ`
- Test segment mode: `go run . -i dQw4w9WgXcQ -s`
- Verify generated FCPXML imports correctly into Final Cut Pro
- Check that segment mode creates clips with proper timing and title cards
- Ensure VTT parsing handles various subtitle formats correctly

## Integration Points

- **External tools**: yt-dlp (required for YouTube functionality)
- **File system**: Reads/writes local video files and FCPXML outputs
- **YouTube API**: Indirectly through yt-dlp for video/subtitle downloading

## Performance Considerations

- YouTube downloads can be slow depending on video size and network speed
- FCPXML generation is fast as it's template-based
- Consider file size limits for very large video files
- Memory usage is minimal as the tool doesn't process video content directly

## Dependencies

- **Required**: yt-dlp must be installed and available in system PATH
- **Go modules**: No external Go dependencies currently used
- **System requirements**: File system write permissions for output directory

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

### Key Functions and Their Purposes

#### VTT Processing (`fcpxml.go`)
- `parseVTTFile(vttPath)`: Parses WebVTT files, extracts timestamps and text (lines 197-248)
- `parseVTTTime(timeStr)`: Converts VTT timestamp format to Go time.Duration (lines 166-195)
- `segmentIntoClips(segments, minDuration, maxDuration)`: Intelligently groups VTT segments into clips based on natural breaks (lines 250-309)

#### FCPXML Generation
- `generateFCPXML(inputFile, outputFile)`: Creates basic single-asset FCPXML (main.go lines 94-175)
- `buildClipFCPXML(clips, videoPath)`: Creates complex multi-clip FCPXML with title cards (fcpxml.go lines 319-460)
- `formatDurationForFCPXML(d)`: Converts Go duration to frame-accurate FCPXML format (fcpxml.go lines 311-317)

## YouTube Integration

- Detects YouTube IDs by checking for 11-character strings without dots
- Downloads video using yt-dlp with MP4 format
- Downloads English auto-generated subtitles when available
- Constructs YouTube URLs as: `https://www.youtube.com/watch?v={ID}`
