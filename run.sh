go build
rm filename.fcpxml
./cutlass fcp filename.fcpxml
./cutlass fcp add-video filename.fcpxml ./assets/speech1.mov

