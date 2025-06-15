go build
rm b.fcpxml
./cutlass build2 b.fcpxml
./cutlass build2 b.fcpxml add-video ./assets/yammer.com.png --with-duration 900 --with-slide

