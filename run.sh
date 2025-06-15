go build
rm filename.fcpxml
./cutlass fcp add-image ./assets/waymo.png --duration 28 -o filename.fcpxml
./cutlass fcp add-slide 0 --input filename.fcpxml -o filename.fcpxml
./cutlass fcp add-text slide_text.txt --input filename.fcpxml -o filename.fcpxml
./cutlass fcp add-image ./assets/waymo.png --duration 28 --input filename.fcpxml -o filename.fcpxml
./cutlass fcp add-slide 28 --input filename.fcpxml -o filename.fcpxml
./cutlass fcp add-text slide_text2.txt --offset 28 --input filename.fcpxml -o filename.fcpxml
./cutlass fcp add-audio ./data/waymo_audio/output.wav --input filename.fcpxml -o filename.fcpxml


