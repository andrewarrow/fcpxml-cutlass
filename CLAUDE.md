# Project Context for AI Assistance

## Project Overview

Cutalyst is a Go-based command-line tool that converts video files into Final Cut Pro XML (FCPXML) format. It supports both local video files and YouTube videos (by downloading them first using yt-dlp). The tool generates FCPXML files that can be imported into Final Cut Pro for video editing.

## Architecture

This is a simple command-line application with two main components:
- **Main CLI handler** (`main.go`): Handles command-line arguments, YouTube video detection/downloading, and orchestrates the conversion process
- **FCPXML generator** (`generateFCPXML` function): Creates the XML structure compatible with Final Cut Pro

The application follows a straightforward flow:
1. Parse command-line arguments
2. Detect if input is a YouTube ID or local file
3. Download YouTube video and subtitles if needed (using yt-dlp)
4. Generate FCPXML file with proper metadata
5. Output the converted file

## Key Technologies

- **Go 1.23.5**: Main programming language for cross-platform CLI tool
- **yt-dlp**: External dependency for YouTube video and subtitle downloading
- **FCPXML format**: XML-based project format used by Final Cut Pro

## Project Structure

```
cutalyst/
├── main.go           # Main CLI application and YouTube handling
├── fcpxml.go         # FCPXML generation logic (currently minimal)
├── go.mod            # Go module definition
├── test.fcpxml       # Generated output file (example)
└── *.mov, *.vtt      # Downloaded media files (temporary)
```

## Important Files and Their Purposes

- `main.go`: Contains the main CLI logic, YouTube video detection, yt-dlp integration, and FCPXML generation
- `fcpxml.go`: Currently contains placeholder function `breakIntoLogicalParts()` - appears to be work in progress
- `go.mod`: Defines the Go module as "cutalyst" with Go 1.23.5
- `*.vtt files`: WebVTT subtitle files downloaded from YouTube
- `*.mov files`: Downloaded video files from YouTube

## Current Development Status

- Latest features: YouTube video downloading with automatic subtitle extraction
- Work in progress: The `fcpxml.go` file has a stub function suggesting future refactoring
- Known issues: 
  - Hardcoded 3600s duration in FCPXML (1 hour)
  - Basic error handling for missing yt-dlp dependency
  - No support for custom video formats or resolutions

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
- Test with local video files: `go run . -i video.mp4`
- Test with YouTube IDs: `go run . -i dQw4w9WgXcQ`
- Verify generated FCPXML imports correctly into Final Cut Pro

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

Usage: `cutalyst -i <input_file> [output_file]`

- `-i`: Input file (required) - can be local file path or 11-character YouTube ID
- `output_file`: Optional output filename (defaults to "test.fcpxml")
- Automatically adds `.fcpxml` extension if not present

## YouTube Integration

- Detects YouTube IDs by checking for 11-character strings without dots
- Downloads video using yt-dlp with MP4 format
- Downloads English auto-generated subtitles when available
- Constructs YouTube URLs as: `https://www.youtube.com/watch?v={ID}`