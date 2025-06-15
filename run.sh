go build
rm filename.fcpxml
./cutlass fcp add-image ./assets/yammer.com.png --duration 20 -o filename.fcpxml
./cutlass fcp add-slide 0 --input filename.fcpxml -o filename.fcpxml
./cutlass fcp add-text slide_text.txt --input filename.fcpxml -o filename.fcpxml


