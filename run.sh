go build
rm b.fcpxml
./cutlass build b.fcpxml
./cutlass build b.fcpxml add-video ./assets/speech1.mov --with-text "hello"
./cutlass build b.fcpxml add-video ./assets/cs.pitt.edu.png
