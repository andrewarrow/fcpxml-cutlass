# fcpxml-cutlass

## Table Example

![table](https://i.imgur.com/mcAUx49.png)
![fcp1](https://i.imgur.com/8CQmlQ4.png)

## Sources

[https://fcp.cafe/developers/fcpxml/](https://fcp.cafe/developers/fcpxml/)

[https://fcp.cafe/developer-case-studies/fcpxml/](https://fcp.cafe/developer-case-studies/fcpxml/)

[fcpxml/dtd](https://github.com/CommandPost/CommandPost/tree/develop/src/extensions/cp/apple/fcpxml/dtd)

[https://github.com/orchetect/DAWFileKit](https://github.com/orchetect/DAWFileKit)

[apple doc](https://developer.apple.com/documentation/professional-video-applications/fcpxml-reference)

## 🎯 Overview

Cutalyst is a powerful command-line tool that converts video files into Final Cut Pro XML (FCPXML) format, making it effortless to import media into Final Cut Pro for editing. It supports both local video files and YouTube videos, automatically downloading them with subtitles when needed.

Built with a modular architecture, Cutalyst offers intelligent video segmentation that can automatically break YouTube videos into logical clips with title cards based on subtitle timing - perfect for creating short-form content from longer videos.

## ✨ Features

- **🎬 Local Video Conversion**: Convert any local video file to FCPXML format
- **📺 YouTube Integration**: Automatically download YouTube videos by providing just the video ID
- **📝 Subtitle Support**: Downloads English auto-generated subtitles from YouTube when available
- **🔍 Smart Auto-detection**: Automatically detects whether input is a local file or YouTube ID
- **📁 Flexible Output**: Specify custom output filename or use intelligent default naming
- **✂️ Segment Mode**: Intelligently break YouTube videos into logical clips (6-18 seconds) with title cards
- **🧠 Smart Clip Detection**: Uses subtitle timing and natural speech breaks to create meaningful segments
- **🏷️ Title Cards**: Automatically generates title cards between clips for easy navigation
- **⚡ Modular Architecture**: Clean package separation for maintainability and extensibility

## 🚀 Installation

### Prerequisites

1. **Go 1.23.5 or later**
   ```bash
   # Check your Go version
   go version
   ```

2. **yt-dlp** (required for YouTube functionality)
   ```bash
   # Install via pip
   pip install yt-dlp
   
   # Or via homebrew on macOS
   brew install yt-dlp
   
   # Or download binary from: https://github.com/yt-dlp/yt-dlp/releases
   ```

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd cutalyst

# Build the binary
go build -o cutalyst

# Or run directly
go run .
```

## 📖 Usage

### Basic Commands

#### Standard Mode (Single Video)
```bash
cutalyst -i <input_file> [output_file]
```

#### Segment Mode (Smart Clips)
```bash
cutalyst -i <youtube_id> -s [output_file]
```

### Parameters

- `-i <input_file>` (required): Input file path or YouTube video ID
- `-s` (optional): Segment mode - intelligently break video into clips with title cards
- `[output_file]` (optional): Output FCPXML filename
  - Standard mode: defaults to "test.fcpxml"
  - Segment mode: defaults to "{youtube_id}_clips.fcpxml"

### Examples

#### Standard Mode: Convert Local Video File

```bash
# Convert local video to FCPXML
cutalyst -i video.mp4

# Convert with custom output name
cutalyst -i video.mp4 my_project.fcpxml
```

#### Standard Mode: Download and Convert YouTube Video

```bash
# Using YouTube video ID (11 characters)
cutalyst -i dQw4w9WgXcQ

# With custom output filename
cutalyst -i dQw4w9WgXcQ rick_roll_project.fcpxml
```

#### ✂️ Segment Mode: Smart Clip Generation

```bash
# Break YouTube video into intelligent clips with title cards
cutalyst -i dQw4w9WgXcQ -s

# Custom output filename for segments
cutalyst -i dQw4w9WgXcQ -s my_clips.fcpxml
```

**Segment Mode Features:**
- 📺 Automatically downloads video and subtitles
- ✂️ Creates 6-18 second clips based on natural speech breaks
- 🏷️ Adds title cards between clips showing preview text
- ⏱️ Generates frame-accurate timing for smooth editing
- 🎯 Perfect for creating short-form content from longer videos
- 🧠 Uses AI-like intelligence to find natural breaking points

### YouTube Video ID

To get a YouTube video ID:
1. Go to any YouTube video
2. Look at the URL: `https://www.youtube.com/watch?v=dQw4w9WgXcQ`
3. The ID is the part after `v=`: `dQw4w9WgXcQ`

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PATH` | Must include yt-dlp location | System PATH | Yes (for YouTube) |

### Output Format

The tool generates FCPXML files with the following specifications:

#### Standard Mode
- **Format**: 1080p30 (1920x1080, 30fps)
- **Duration**: 3600s (1 hour) - currently hardcoded
- **Color Space**: Rec. 709
- **Audio**: Stereo, 48kHz
- **Timeline**: Single video asset with basic metadata

#### Segment Mode
- **Format**: 1080p30 (1920x1080, 30fps)
- **Duration**: Calculated based on actual clip content
- **Color Space**: Rec. 709
- **Audio**: Stereo, 48kHz
- **Timeline**: Multiple video clips with title cards
- **Clip Duration**: 6-18 seconds each (intelligently determined)
- **Title Cards**: 10 seconds between clips
- **Timing**: Frame-accurate for seamless editing

### Project Architecture

The project is organized into modular packages for clean separation of concerns:

```
cutalyst/
├── main.go              # Main CLI application and orchestration
├── go.mod              # Go module definition
├── CLAUDE.md           # AI assistant context
├── README.md           # This file
├── vtt/                # VTT subtitle processing package
│   └── vtt.go          # Parsing, segmentation, types
├── youtube/            # YouTube integration package
│   └── youtube.go      # Download, ID detection
├── fcp/                # Final Cut Pro XML generation
│   ├── types.go        # FCPXML data structures
│   └── generator.go    # Generation functions
├── wikipedia/          # Future feature (placeholder)
│   └── parse.go        # Wikipedia integration (TBD)
├── foo/                # Sample/test files
└── generated files:
    ├── *.fcpxml        # Generated Final Cut Pro projects
    ├── *.mov           # Downloaded YouTube videos
    └── *.vtt           # Downloaded subtitle files
```

## 🛠️ Development

### Building

```bash
# Build for current platform
go build -o cutalyst

# Build for multiple platforms
GOOS=windows GOARCH=amd64 go build -o cutalyst.exe
GOOS=darwin GOARCH=amd64 go build -o cutalyst-mac
GOOS=linux GOARCH=amd64 go build -o cutalyst-linux
```

### Package Functions

#### VTT Package (`vtt/`)
- `ParseFile(vttPath)`: Parse WebVTT subtitle files
- `ParseTime(timeStr)`: Convert VTT timestamps to Go time.Duration
- `SegmentIntoClips(segments, min, max)`: Intelligently segment into clips

#### YouTube Package (`youtube/`)
- `IsYouTubeID(input)`: Detect if string is YouTube video ID
- `DownloadVideo(youtubeID)`: Download video using yt-dlp
- `DownloadSubtitles(youtubeID)`: Download subtitles using yt-dlp

#### FCP Package (`fcp/`)
- `GenerateStandard(input, output)`: Create basic FCPXML
- `GenerateClipFCPXML(clips, video, output)`: Create segmented FCPXML
- `FormatDurationForFCPXML(duration)`: Frame-accurate duration formatting

### Testing

Currently, testing is done manually:

```bash
# Test standard mode with local file
go run . -i test_video.mp4

# Test standard mode with YouTube video
go run . -i dQw4w9WgXcQ

# Test segment mode (requires YouTube video with subtitles)
go run . -i dQw4w9WgXcQ -s

# Verify the generated FCPXML imports correctly into Final Cut Pro
```

## 🔧 Troubleshooting

### Common Issues

1. **"yt-dlp not found" error**
   - Install yt-dlp using the installation instructions above
   - Verify it's in your PATH: `which yt-dlp`

2. **YouTube download fails**
   - Check your internet connection
   - Verify the YouTube video ID is correct (11 characters)
   - Some videos may be region-restricted or private

3. **Input file not found**
   - Verify the file path is correct
   - Check file permissions

4. **FCPXML doesn't import properly**
   - Ensure you're using Final Cut Pro 10.4 or later
   - Check that video file paths in the FCPXML are accessible

5. **Segment mode fails**
   - Verify the YouTube video has English subtitles available
   - Check that both .mov and .vtt files were downloaded
   - Some videos may not have auto-generated subtitles

6. **Clips are too short/long in segment mode**
   - Segment duration is automatically determined (6-18 seconds)
   - Based on natural speech breaks and subtitle timing
   - Currently not customizable

7. **Compilation errors**
   - Ensure Go 1.23.5 or later is installed
   - Check that all package dependencies are correct
   - Run `go mod tidy` to clean up dependencies

### Debug Mode

Add print statements in the code for debugging:

```go
fmt.Printf("Debug: Processing input: %s\n", inputFile)
```

### Package Development

When adding new features:

1. **Determine the appropriate package**: vtt/, youtube/, fcp/, or main.go
2. **Follow Go conventions**: Exported functions start with capital letters
3. **Update imports**: Add new package imports where needed
4. **Test compilation**: Always run `go build` to verify changes
5. **Test functionality**: Use manual testing approach above

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and test them
4. Ensure code compiles: `go build`
5. Commit your changes: `git commit -am 'Add feature'`
6. Push to the branch: `git push origin feature-name`
7. Submit a pull request

### Development Guidelines

- Follow the modular package structure
- Add functions to the appropriate package
- Export functions (capitalize) if they need to be used by other packages
- Keep packages focused on single responsibilities
- Avoid circular dependencies between packages

## 📋 Roadmap

### Near Term
- [ ] Support for custom video formats and resolutions
- [ ] Dynamic duration detection instead of hardcoded 3600s in standard mode
- [ ] Customizable clip duration ranges for segment mode
- [ ] Better error handling and user feedback

### Medium Term
- [ ] Batch processing for multiple files
- [ ] Support for other subtitle languages in segment mode
- [ ] Segment mode for local video files with subtitle tracks
- [ ] Integration with other video platforms
- [ ] Automated testing framework

### Long Term
- [ ] GUI version for non-technical users
- [ ] Clip preview generation for segment mode
- [ ] Wikipedia integration (parse.go functionality)
- [ ] Advanced clip analysis and optimization
- [ ] Plugin system for custom processors
- [ ] Cloud-based processing options

## 🔗 Related Tools

- [yt-dlp](https://github.com/yt-dlp/yt-dlp) - YouTube video downloader
- [Final Cut Pro](https://www.apple.com/final-cut-pro/) - Video editing software
- [FCPXML](https://developer.apple.com/documentation/professional_video_applications/fcpxml_reference) - Final Cut Pro XML format documentation
