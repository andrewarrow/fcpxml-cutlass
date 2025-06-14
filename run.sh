go build
rm b.fcpxml
./cutlass build2 b.fcpxml
./cutlass build2 b.fcpxml add-video ./assets/yammer.com.png --with-text "hello from cli" --with-sound data/waymo_audio/s001_2.wav
./cutlass build2 b.fcpxml add-video ./assets/bizrate.com.png --with-text "hello from cli" --with-sound data/waymo_audio/s002_7.wav
