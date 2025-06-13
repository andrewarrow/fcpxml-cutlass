go build
rm b.fcpxml
./cutlass build b.fcpxml
./cutlass build b.fcpxml add-video ./assets/cs.pitt.edu.png --with-text "hello"
./cutlass build b.fcpxml add-video ./assets/bizrate.com.png --with-text "hello from cli" --with-sound data/wiki_tongue_tied_red_dwarf_song.wav
