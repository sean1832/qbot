# qbot
qbot is designed to be used as a post-processing script for qbittorrent. It automates renaming and moving of downloaded media files using filebot. The tool processes media files according to defined file extensions and organizes them into appropriate directories.

## Todo
- [ ] Add support for configuration file
- [ ] Add support for logging output to a file

## Pre-requisites
- [qbittorrent](https://www.qbittorrent.org/)
- [filebot](https://www.filebot.net/) with a valid license

## Installation
1. Download the latest release from the [releases page](https://github.com/sean1832/qbot/releases/latest) or with `curl`:
### Windows
```bash
curl -L https://github.com/sean1832/qbot/releases/latest/download/qbot-win-amd64.exe -o qbot.exe
```
### Linux
```bash
curl -L https://github.com/sean1832/qbot/releases/latest/download/qbot-linux-amd64 -o qbot
```

2. Add the binary to your PATH

### Compile from source
1. Clone the repository
2. Run the `build` command:
```bash
go build .
```

## Usage
```bash
qbot.exe filebot [flags]
```


### `filebot` Options
- `-d, --destination`  
  Destination path of the Plex media root. This root should contain directories like `TV-Shows/Real`, `TV-Shows/Anime`, and `Movies`.

- `-n, --name`  
  Torrent name to help filebot identify the media.

- `-l, --language`  
  Language of the media (default: `en`).

- `-a, --action`  
  Action to take (e.g., `move`).

- `-c, --conflict`  
  Conflict resolution strategy (e.g., `skip`).

- `-e, --ext`  
  Comma-separated file extensions to process (default: `"mkv,mp4,avi,mov,rmvb"`).

- `-x, --exclude`  
  Comma-separated directories to exclude (if any).

### Use with qbittorrent
For integration with qbittorrent as a post-processing script, you can use the following example command:
```bash
./qbot.exe filebot %F %L -d /path/to/media/root -n %N -a move -c skip -l en -e "mkv,mp4,avi,mov,rmvb" -x "sample,extras" -t /path/to/temp_root
```
In this command:
- `%F` is the downloaded file/folder path.
- `%L` is the label (which identifies the media type).
- `%N` is the torrent name.
- Adjust the destination path and flags as needed.

## License
This project is licensed under the [Apache 2.0](LICENSE).