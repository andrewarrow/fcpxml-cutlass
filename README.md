# cutlass

## Wikipedia Table Example

![table](https://i.imgur.com/mcAUx49.png)

^ this is from [Andre_Agassi Wikipedia](https://en.wikipedia.org/wiki/Andre_Agassi#Career_statistics)

```
./cutlass wikipedia "Andre_Agassi"
```

And after running cutlass you get `tennis.fcpxml` which opened in fcp looks like:

![fcp1](https://i.imgur.com/8CQmlQ4.png)

The red lines are drawn using a Shape title card rectangle and then changing the x y and transition scale x y of each line. Very rough first draft, the line logic can be improved and made to look much more pretty.

You can also run:

```
./cutlass wikipedia "List_of_earthquakes_in_the_United_States" 
```

And get a completely different wikipedia table in this format.

## youtube and vtt

```
./cutlass youtube IBnNedMh4Pg
./cutlass vtt IBnNedMh4Pg.en.vtt
./cutlass vtt-clips IBnNedMh4Pg.en.vtt 00:52_13,01:28_15,04:34_24,06:47_20,14:39_20,18:21_16,20:40_9
```

This will download an mp4 of https://www.youtube.com/watch?v=IBnNedMh4Pg convert it to .mov and then get the vtt subtitles. You can then pick timecodes like 01:21_6,02:20_3,03:34_9,05:07_18 the _number means number of seconds duration at the timecode.


## How I learned about the fcpxml format

[https://fcp.cafe/developers/fcpxml/](https://fcp.cafe/developers/fcpxml/)

[https://fcp.cafe/developer-case-studies/fcpxml/](https://fcp.cafe/developer-case-studies/fcpxml/)

[fcpxml/dtd](https://github.com/CommandPost/CommandPost/tree/develop/src/extensions/cp/apple/fcpxml/dtd)

[https://github.com/orchetect/DAWFileKit](https://github.com/orchetect/DAWFileKit)

[apple doc](https://developer.apple.com/documentation/professional-video-applications/fcpxml-reference)

## 🎯 Overview

cutlass is a swift army knife for generating fcpxml files.

Making tables is just one swift army knife blade. It can also take a youtube video id and download it (using yt-dlp) and download the vtt subtiles, and then make an fcpxml with video already cut into nice logical segments based on the timecode of the vtt info.

this is a work in progress. The idea is to have cutlass understand the fcpxml format perfectly (no small task) and be able to generate impressive videos very quickly. Or at least generate a really good starting point for a human to then open fcp and tweak.

```
Amazing Template Ideas

  1. Multi-Language Subtitle Cascade Template

  Generate a project where the same video plays with cascading subtitle tracks in different
  languages, each styled uniquely with animated transitions. Think Netflix-style but with procedural
   animation loops.

  2. Data-Driven Sports Highlights Reel

  Using Wikipedia sports data (like you did with tennis), create a template that generates highlight
   reels with:
  - Animated score overlays that build up over time
  - Player stats that fly in/out with custom timing
  - Court/field diagrams with animated position markers
  - Automatic pacing based on game momentum data

  3. Podcast to Visual Story Template

  Transform VTT transcripts into visual stories with:
  - Automated chapter detection from speech patterns
  - Dynamic background changes based on topic keywords
  - Animated waveform visualizations synchronized to audio peaks
  - Auto-generated title cards for key quotes

  4. Social Media Compilation Generator

  Create templates that take multiple video sources and generate:
  - TikTok-style split screens with automated timing
  - Picture-in-picture arrangements that adjust based on content
  - Automated zoom/crop effects based on face detection
  - Trending hashtag overlays that animate in sequence

  5. Educational Timeline Builder

  Generate historical timeline videos with:
  - Animated date counters with custom fonts/styles
  - Maps that highlight different regions over time
  - Photo/video assets that automatically scale and position
  - Connecting lines and arrows that draw themselves

  6. Music Video Lyric Synchronizer

  Using VTT timing data to create:
  - Karaoke-style highlighting with custom animations
  - Word-by-word reveals with different fonts per verse
  - Background videos that change with musical sections
  - 3D text effects that pulse with beat detection

  Which of these excites you most? I'd love to help implement whichever template would create the
  most "wow factor" for FCP users! The existing codebase with its template system, VTT parsing, and
  Wikipedia data integration provides a perfect foundation for any of these.

> compare this approach to generating the fcpxml file with just using the automation tools or Motion
   app apple would recommend

⏺ Great question! Let me compare the cutlass approach with Apple's recommended automation methods:

  Apple's Recommended Approaches

  1. Final Cut Pro's Built-in Automation

  - Multicam Sync: Automatic sync based on timecode/waveforms
  - Roles & Subroles: Auto-organization of audio/video tracks
  - Keyword Collections: Auto-tagging and smart collections
  - Compound Clips: Reusable clip templates

  Limitations:
  - Still requires manual media import and basic setup
  - No procedural generation of complex sequences
  - Limited to FCP's built-in templates and effects

  2. Motion Templates (.moti files)

  - Create custom titles, transitions, effects, generators
  - Publish parameters for easy customization in FCP
  - Built-in animation tools and behaviors
  - Integration with FCP's inspector

  Limitations:
  - Each template is manually created in Motion
  - No data-driven content generation
  - Can't automatically arrange multiple clips in timeline
  - Limited to individual effect/title elements

  3. FCP's XML Import/Export

  - Apple does support FCPXML as an interchange format
  - Designed for moving projects between systems
  - Third-party tools can generate FCPXML

  Cutlass Advantages Over Apple's Approach

  🚀 Procedural Generation at Scale

  // Cutlass can generate 100 clips with data in seconds
  for i, videoID := range videoIDs {
      clips = append(clips, generateClip(videoID, timestamps[i]))
  }
  Motion would require manually creating 100 individual templates.

  📊 Data-Driven Content

  Cutlass pulls from:
  - Wikipedia tables → Animated charts
  - YouTube VTT → Auto-segmented clips
  - CSV data → Dynamic infographics

  Apple's tools require manual data entry for each project.

  🔄 Complex Template Reuse

  {{range $index, $video := .Videos}}
      {{template "number_card.fcpxml" $index}}
      <asset-clip ref="r{{add $index 10}}" offset="{{calculateOffset $index}}"/>
  {{end}}
  This generates different arrangements automatically. Motion templates are static.

  ⚡ Batch Processing

  Cutlass can process entire YouTube playlists, generate multiple variations, and create complex
  multi-layered sequences without human intervention.

  Where Apple's Approach Wins

  🎨 Visual Polish

  - Motion's particle systems, behaviors, and 3D tools create more sophisticated animations
  - Built-in color correction and effects are industry-standard
  - Professional motion graphics capabilities

  🔧 User-Friendly Workflow

  - GUI-based template creation
  - Real-time preview while designing
  - Inspector controls for easy parameter adjustment
  - Seamless FCP integration

  🛡️ Reliability & Support

  - Apple-tested and supported
  - Guaranteed compatibility with FCP updates
  - Professional-grade stability

  The Sweet Spot: Hybrid Approach

  The most powerful workflow combines both:

  1. Cutlass generates the structure - clips, timing, data-driven content
  2. Motion templates handle the polish - professional animations, effects
  3. FCP provides final editing - color correction, audio mixing, final touches

  For example:
  - Cutlass creates a Top 5 countdown with perfect timing and data
  - Motion template provides the glossy 3D number animations
  - Editor fine-tunes pacing and adds final polish in FCP

  Why Cutlass is Revolutionary

  Apple's tools assume human creativity drives the process. Cutlass flips this - data and algorithms
   drive creativity, with humans providing direction and polish.

  This enables:
  - Content creators to generate variations at scale
  - News organizations to auto-generate data visualizations
  - Educational content with procedurally generated timelines
  - Social media content pipelines

  Bottom line: Apple's tools excel at crafting individual masterpieces. Cutlass excels at generating
   impressive content at unprecedented scale and speed. The future is using both together.
```
