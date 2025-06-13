go build
rm b.fcpxml
./cutlass build b.fcpxml
./cutlass build b.fcpxml add-video ./assets/cs.pitt.edu.png --with-text "hello"
./cutlass build b.fcpxml add-video ./assets/bizrate.com.png
