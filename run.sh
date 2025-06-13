go build
rm b.fcpxml
./cutlass build2 b.fcpxml
./cutlass build2 b.fcpxml add-video ./assets/yammer.com.png --with-text "hello from cli" --with-sound data/wiki_tongue_tied_red_dwarf_song.wav
./cutlass build2 b.fcpxml add-video ./assets/bizrate.com.png --with-text "hello from cli" --with-sound data/wiki_tacoma_film_festival.wav
