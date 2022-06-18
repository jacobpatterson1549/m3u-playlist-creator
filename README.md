# m3u-playlist-creator

M3U-playlist is a command-line tool to create and maintain m3u playlists.
The m3u playlist format is well-supported across many devices including vehicles.

The application supports mp3 and m4a files.

To list the distribution of file types in a folder, run `find -type f | sed 's/.*\.//' | sort | uniq -c | sort -k1 -h`

## Dependencies

* The application is written in [Go](https://go.dev).
* [github.com/dhowden/tag](https://github.com/dhowden/tag) parses the artist, album, and title from music files.

## Building

The following examples build all executables to a `./build` folder.  Move the executable application to the base location of the music files for the playlists.

Build the application with `go build ./... -o build/m3u-playlist-creator`

Build a Windows version of the application with `GOOS=windows GOARCH=amd64 go build -o build/m3u-playlist-creator.exe`

Run tests with `go test ./... --cover`