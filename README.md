# m3u-playlist-creator

## a command-line tool to create and maintain m3u playlists.

M3U playlists group songs without copying or moving their files.
Internally, they are a listing of the files to play.
The application provides a command-line interface to search for song files and add them to playlists.
Songs in playlists can be reordered and given unique display names.
The application can load existing playlists, but it only saves playlists to new files.

M3U playlists format are supported on many devices, including vehicles. 
Use a USB flash drive to create music catalogs with playlists.
First, copy all of the music and the m3u-playlist-creator application onto the flash drive.
Then, run the application to create playlists and write (save) the playlist.
Finally, remove the USB drive and insert into the vehicle or desired device.
Do not move the music files once playlists are created.
The playlists reference them and will not work correctly if any of the referenced files are altered.

The application supports mp3 and m4a file types.
To list the distribution of file types in a folder, run `find -type f | sed 's/.*\.//' | sort | uniq -c | sort -k1 -h`

### Dependencies

* The application is written in [Go](https://go.dev).
* [github.com/dhowden/tag](https://github.com/dhowden/tag) parses the artist, album, and title from music files.

### Building

1. Clone this repository to your computer using [git](https://git-scm.com/)) or download it via HTTPS.
1. [Download](https://go.dev/dl/) the Go programming language.
Follow the [installation instructions](https://go.dev/doc/install) to install it.
1. Build the application with `go build -o build/m3u-playlist-creator`.
This creates the executable in a `/build` subfolder.
Move it to the playlist root directory for use.

Build a Windows version of the application with `GOOS=windows GOARCH=amd64 go build -o build/m3u-playlist-creator.exe`

Run tests with `go test --cover`

### Running

Songs are loaded from the directory the application is run from.

If the app is launched with the -md5 parameter, md5sums are computed for each song.
They are displayed and can be filtered on, but cause the app to load much more slowly.