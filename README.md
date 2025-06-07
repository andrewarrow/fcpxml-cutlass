# Cutalyst

[![Go Version](https://img.shields.io/badge/go-1.23.5-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## 🎯 Overview

Cutalyst is a command-line tool that converts video files into Final Cut Pro XML (FCPXML) format, making it easy to import media into Final Cut Pro for editing. It supports both local video files and YouTube videos, automatically downloading them with subtitles when needed.

Perfect for content creators who want to quickly get YouTube videos or local media files into Final Cut Pro without manual import processes.

## ✨ Features

- **Local Video Conversion**: Convert any local video file to FCPXML format
- **YouTube Integration**: Automatically download YouTube videos by providing just the video ID
- **Subtitle Support**: Downloads English auto-generated subtitles from YouTube when available
- **Auto-detection**: Automatically detects whether input is a local file or YouTube ID
- **Flexible Output**: Specify custom output filename or use default naming

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

### Basic Usage

```bash
cutalyst -i <input_file> [output_file]
```

### Parameters

- `-i <input_file>` (required): Input file path or YouTube video ID
- `[output_file]` (optional): Output FCPXML filename (defaults to "test.fcpxml")

### Examples

#### Convert Local Video File

```bash
# Convert local video to FCPXML
cutalyst -i video.mp4

# Convert with custom output name
cutalyst -i video.mp4 my_project.fcpxml
```

#### Download and Convert YouTube Video

```bash
# Using YouTube video ID (11 characters)
cutalyst -i dQw4w9WgXcQ

# With custom output filename
cutalyst -i dQw4w9WgXcQ rick_roll_project.fcpxml
```

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
- **Format**: 1080p30 (1920x1080, 30fps)
- **Duration**: 3600s (1 hour) - currently hardcoded
- **Color Space**: Rec. 709
- **Audio**: Stereo, 48kHz

### Project Structure

```
cutalyst/
├── main.go              # Main CLI application
├── fcpxml.go           # FCPXML generation logic
├── go.mod              # Go module definition
├── CLAUDE.md           # AI assistant context
├── README.md           # This file
└── generated files:
    ├── *.fcpxml        # Generated Final Cut Pro projects
    ├── *.mp4           # Downloaded YouTube videos
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

### Testing

Currently, testing is done manually:

```bash
# Test with local file
go run . -i test_video.mp4

# Test with YouTube video
go run . -i dQw4w9WgXcQ

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

### Debug Mode

Add print statements in the code for debugging:

```go
fmt.Printf("Debug: Processing input: %s\\n", inputFile)
```

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and test them
4. Commit your changes: `git commit -am 'Add feature'`
5. Push to the branch: `git push origin feature-name`
6. Submit a pull request

## 📋 Roadmap

- [ ] Support for custom video formats and resolutions
- [ ] Dynamic duration detection instead of hardcoded 3600s
- [ ] Batch processing for multiple files
- [ ] Integration with other video platforms
- [ ] GUI version for non-technical users
- [ ] Automated testing framework