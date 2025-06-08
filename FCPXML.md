# FCPXML Technical Reference

## Overview

FCPXML (Final Cut Pro XML) is Apple's XML-based interchange format for representing video editing projects, media assets, and timeline structures in Final Cut Pro. The format has evolved through multiple versions, with the current specification supporting both single-file (.fcpxml) and bundle (.fcpxmld) formats.

## Document Structure

### Root Element
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml PUBLIC "-//Apple//DTD FCPXML 1.7//EN" "http://www.apple.com/DTDs/FCPXMLv1_7.dtd">
<fcpxml version="1.7">
    <!-- Document content -->
</fcpxml>
```

### Main Structural Components

1. **Resources Section** - Contains all media assets, formats, effects, and generators
2. **Library/Event/Project Structure** - Organizational hierarchy
3. **Sequence** - Timeline container with spine and clips
4. **Story Elements** - Clips, gaps, transitions, titles arranged in time

## Core Elements and Attributes

### fcpxml (Root Element)
- **version**: Format version (1.0 through 1.10+)
- Contains: resources, library, event, project elements

### resources
Container for all referenced media and effects:
```xml
<resources>
    <format id="r1" name="FFVideoFormat1080p30" frameDuration="1001/30000s" width="1920" height="1080"/>
    <asset id="r2" name="video.mov" src="file:///path/to/video.mov" start="0s" duration="3600s" hasVideo="1" hasAudio="1"/>
</resources>
```

### project
- **name**: Project name
- **uid**: Unique identifier
- Contains: sequence elements

### sequence
- **duration**: Total sequence duration
- **format**: Reference to format resource
- **tcStart**: Timecode start
- **tcFormat**: Timecode format (DF, NDF)
- Contains: spine element

### spine
Primary timeline container:
```xml
<spine>
    <asset-clip ref="r2" offset="0s" duration="10s" start="0s"/>
    <gap name="Gap" offset="10s" duration="5s"/>
    <asset-clip ref="r3" offset="15s" duration="8s" start="100s"/>
</spine>
```

### Asset Elements

#### asset-clip
- **ref**: Reference to asset in resources
- **offset**: Position in timeline
- **duration**: Clip duration  
- **start**: Start position in source media
- **name**: Clip name
- **lane**: Vertical position (for stacking)
- **enabled**: Enable/disable (0 or 1)

#### ref-clip
References compound clips or other sequences

#### sync-clip
Container for synchronized multicam content

#### gap
Empty space in timeline:
- **name**: Gap identifier
- **offset**: Timeline position
- **duration**: Gap length

### Media Elements

#### video
Video portion of a clip:
```xml
<video ref="r2" offset="0s" duration="10s" start="0s">
    <param name="crop" value="0 0 1920 1080"/>
    <param name="scale" value="1 1"/>
</video>
```

#### audio
Audio portion of a clip:
```xml
<audio ref="r2" offset="0s" duration="10s" start="0s" role="dialogue">
    <param name="volume" value="0dB"/>
</audio>
```

### Effects and Filters

#### filter-video / filter-audio
Video and audio filters:
```xml
<filter-video ref="r5" name="Gaussian Blur">
    <param name="amount" value="10"/>
</filter-video>
```

#### transition
Transition between clips:
```xml
<transition name="Cross Dissolve" offset="9s" duration="2s"/>
```

### Annotations

#### marker
Timeline markers:
```xml
<marker start="30s" duration="1s" value="Scene 2" completed="0"/>
```

#### keyword
Content keywords:
```xml
<keyword start="0s" duration="30s" value="interview"/>
```

## Time Representation

FCPXML uses rational number representation for precise timing:

### Format
- Expressed as fractions: `"numerator/denominator" + "s"`
- Examples: `"1001/30000s"`, `"3600s"`, `"0s"`
- Uses 64-bit numerator and 32-bit denominator

### Common Frame Rates
- 24 fps: `"1001/24000s"`
- 25 fps: `"1/25s"`
- 29.97 fps: `"1001/30000s"`
- 30 fps: `"1/30s"`
- 60 fps: `"1/60s"`

## Resource Types

### format
Defines video/audio format parameters:
```xml
<format id="r1" name="FFVideoFormat1080p30" 
        frameDuration="1001/30000s" 
        width="1920" height="1080" 
        colorSpace="1-1-1 (Rec. 709)"/>
```

### asset
Media file reference:
```xml
<asset id="r2" name="video.mov" 
       src="file:///path/to/video.mov" 
       start="0s" duration="3600s" 
       hasVideo="1" hasAudio="1" 
       audioSources="1" audioChannels="2" 
       audioRate="48000"/>
```

### effect
Effect or generator:
```xml
<effect id="r3" name="Custom Title" uid=".../Titles.localized/Custom.localized/Custom.moti"/>
```

## Timeline and Lane Structure

### Lane System
- **lane="0"** or no lane: Main spine (horizontal arrangement)
- **lane="1", "2", etc.**: Vertical stacking above main spine
- **lane="-1", "-2", etc.**: Vertical stacking below main spine

### Correct Vertical Stacking
```xml
<spine>
    <video ref="r2" offset="0s" duration="10s" name="Main Video">
        <!-- This video stacks ABOVE the main video -->
        <video ref="r3" lane="1" offset="0s" duration="5s" name="Upper Video"/>
        <!-- This title stacks even higher -->
        <title ref="r4" lane="2" offset="2s" duration="3s" name="Title"/>
    </video>
</spine>
```

## Bundle Format (.fcpxmld)

Starting with Final Cut Pro 10.6, Apple introduced the bundle format:

### Structure
```
project.fcpxmld/
├── Info.plist
├── Settings.plist  
├── Contents/
│   ├── Resources/
│   │   ├── project.fcpxml
│   │   ├── tracking_data.bin
│   │   └── stabilization_data.bin
│   └── CurrentVersion.fcpxml -> Resources/project.fcpxml
```

### Benefits
- Separates XML from binary data (tracking, stabilization)
- Maintains backward compatibility
- Reduces memory overhead for XML processing
- Supports multiple projects in one bundle

## Validation and DTD

### DTD Versions
Apple maintains DTDs for all versions:
- FCPXMLv1.0.dtd through FCPXMLv1.7.dtd
- Latest: FCPXMLv1.10+ (with bundle support)

### Validation Command
```bash
xmllint --dtdvalid "/path/to/FCPXMLv1_7.dtd" "/path/to/file.fcpxml"
```

### Important Notes
- DTD validation doesn't guarantee successful import
- Final Cut Pro doesn't validate against DTD during import
- Invalid data can still cause import failures despite DTD compliance

## Common Attributes

### Universal Attributes
- **id**: Unique identifier for resources
- **name**: Human-readable name
- **offset**: Timeline position
- **duration**: Length in time
- **start**: Source media start position
- **ref**: Reference to resource by ID

### Time Attributes
- **tcStart**: Timecode start
- **tcFormat**: Timecode format (DF, NDF)
- **frameDuration**: Frame duration for format

### Media Attributes
- **hasVideo**: Video track present (0/1)
- **hasAudio**: Audio track present (0/1)
- **audioSources**: Number of audio sources
- **audioChannels**: Audio channel count
- **audioRate**: Audio sample rate

## Best Practices

### Structure
1. Always define resources before referencing them
2. Use meaningful IDs for resources (r1, r2, etc.)
3. Maintain proper parent-child relationships
4. Use lanes for vertical stacking, not separate spine elements

### Timing
1. Use rational numbers for frame-accurate timing
2. Ensure offsets and durations align with format frame rates
3. Calculate precise start times for source media
4. Maintain timeline continuity

### Performance
1. Minimize resource duplication
2. Use ref-clips for repeated elements
3. Consider bundle format for complex projects
4. Validate structure before import

### Compatibility
1. Test with target Final Cut Pro version
2. Use appropriate DTD version
3. Consider backward compatibility requirements
4. Handle version-specific features gracefully

## Version Differences

### Major Changes by Version
- **1.0-1.2**: Basic structure, simple effects
- **1.3**: Separated filter-audio/filter-video elements
- **1.4-1.6**: Enhanced multicam, rate conforming
- **1.7**: Improved metadata, keywords
- **1.8-1.9**: Advanced effects, color correction
- **1.10+**: Bundle format, binary data support

### Deprecated Elements
- `<filter>` → `<filter-video>` / `<filter-audio>` (v1.3+)
- `<timeMap>` → `<conformRate>` (v1.3+)

## Element Reference

### Complete Element Hierarchy
```
fcpxml
├── resources
│   ├── format
│   ├── asset
│   │   └── media-rep
│   ├── effect
│   ├── transition
│   └── generator
├── library
│   └── event
│       └── project
│           └── sequence
│               └── spine
│                   ├── asset-clip
│                   ├── ref-clip
│                   ├── sync-clip
│                   ├── gap
│                   ├── video
│                   ├── audio
│                   ├── title
│                   ├── generator-clip
│                   ├── transition
│                   ├── filter-video
│                   ├── filter-audio
│                   ├── marker
│                   └── keyword
```

### Parameter Types
Parameters can contain various data types:
- **Numeric**: `value="1.5"`
- **Boolean**: `value="0"` or `value="1"`
- **String**: `value="Custom Text"`
- **Color**: `value="1.0 0.5 0.2"` (RGB)
- **Position**: `value="100 200"` (X Y coordinates)
- **Keyframes**: Complex animation data

### Roles and Metadata
Audio and video can have roles for organization:
```xml
<audio role="dialogue">
<audio role="music">
<audio role="effects">
```

## Advanced Features

### Compound Clips
Referenced using ref-clip element:
```xml
<ref-clip ref="compoundClip1" offset="0s" duration="30s"/>
```

### Multicam Clips
Synchronized media using sync-clip:
```xml
<sync-clip offset="0s" duration="60s" name="Interview Multicam">
    <asset-clip ref="camera1" offset="0s" duration="60s" role="video.multicam1"/>
    <asset-clip ref="camera2" offset="0s" duration="60s" role="video.multicam2"/>
    <asset-clip ref="audio1" offset="0s" duration="60s" role="audio.mix"/>
</sync-clip>
```

### Color Correction
Color parameters and LUTs:
```xml
<filter-video ref="colorCorrector">
    <param name="saturation" value="1.2"/>
    <param name="exposure" value="0.5"/>
    <param name="shadows" value="0.1"/>
</filter-video>
```

This comprehensive reference covers the essential technical aspects of FCPXML format for developing applications that generate or consume Final Cut Pro XML files.