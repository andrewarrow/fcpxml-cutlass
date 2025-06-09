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

cutlass is a swiss army knife for generating fcpxml files.

Making tables is just one swiss army knife blade. It can also take a youtube video id and download it (using yt-dlp) and download the vtt subtiles, and then make an fcpxml with video already cut into nice logical segments based on the timecode of the vtt info.

this is a work in progress. The idea is to have cutlass understand the fcpxml format perfectly (no small task) and be able to generate impressive videos very quickly. Or at least generate a really good starting point for a human to then open fcp and tweak.

