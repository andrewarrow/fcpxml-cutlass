go build
rm b.fcpxml
./cutlass build b.fcpxml
./cutlass build b.fcpxml add-video ./assets/speech1.mov
./cutlass build b.fcpxml add-video ./assets/speech2.mov
