# cutlass

## Wikipedia Table Example

![table](https://i.imgur.com/mcAUx49.png)

^ this is from [Andre_Agassi Wikipedia](https://en.wikipedia.org/wiki/Andre_Agassi#Career_statistics)

```
./cutlass -i "Andre_Agassi" -w tennis.fcpxml
```

And after running cutlass you get `tennis.fcpxml` which opened in fcp looks like:

![fcp1](https://i.imgur.com/8CQmlQ4.png)

## Sources

[https://fcp.cafe/developers/fcpxml/](https://fcp.cafe/developers/fcpxml/)

[https://fcp.cafe/developer-case-studies/fcpxml/](https://fcp.cafe/developer-case-studies/fcpxml/)

[fcpxml/dtd](https://github.com/CommandPost/CommandPost/tree/develop/src/extensions/cp/apple/fcpxml/dtd)

[https://github.com/orchetect/DAWFileKit](https://github.com/orchetect/DAWFileKit)

[apple doc](https://developer.apple.com/documentation/professional-video-applications/fcpxml-reference)

## 🎯 Overview

cutlass is a swift army knife for generating fcpxml files.

Making tables is just one swift army knife blade. It can also take a youtube video id and download it (using yt-dlp) and download the vtt subtiles, and then make an fcpxml with video already cut into nice logical segments based on the timecode of the vtt info.



