package fcp

import "testing"

func TestGenerateEmpty(t *testing.T) {
	GenerateEmpty()
}

var xmlstring = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

<fcpxml version="1.13">
    <resources>
        <format id="r1" name="FFVideoFormat720p2398" frameDuration="1001/24000s" width="1280" height="720" colorSpace="1-1-1 (Rec. 709)"/>
    </resources>
    <library location="file:///Users/aa/Movies/Untitled.fcpbundle/">
        <event name="6-13-25" uid="78463397-97FD-443D-B4E2-07C581674AFC">
            <project name="wiki" uid="DEA19981-DED5-4851-8435-14515931C68A" modDate="2025-06-13 11:46:22 -0700">
                <sequence format="r1" duration="0s" tcStart="0s" tcFormat="NDF" audioLayout="stereo" audioRate="48k">
                    <spine/>
                </sequence>
            </project>
        </event>
        <smart-collection name="Projects" match="all">
            <match-clip rule="is" type="project"/>
        </smart-collection>
        <smart-collection name="All Video" match="any">
            <match-media rule="is" type="videoOnly"/>
            <match-media rule="is" type="videoWithAudio"/>
        </smart-collection>
        <smart-collection name="Audio Only" match="all">
            <match-media rule="is" type="audioOnly"/>
        </smart-collection>
        <smart-collection name="Stills" match="all">
            <match-media rule="is" type="stills"/>
        </smart-collection>
        <smart-collection name="Favorites" match="all">
            <match-ratings value="favorites"/>
        </smart-collection>
    </library>
</fcpxml>`
