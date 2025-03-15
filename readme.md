# qbot

qbot is a post-processing tool for [qbittorrent](https://www.qbittorrent.org/) that automates the renaming and moving of downloaded media files using [Filebot](https://www.filebot.net/). It processes files based on defined extensions and organizes them into the appropriate directories for Plex media libraries.

## Features

- **Automated File Processing:** Renames and moves files using Filebot based on user-specified rules.
- **Media Organization:** Supports multiple media categories (e.g., TV Shows, Anime, Movies) with custom directory layouts.
- **Customizable Options:** Configure file extensions, exclusion directories, and Filebot filters via command-line flags.
- **Logging:** Optional logging to a file for troubleshooting and tracking operations.
- **Seamless qbittorrent Integration:** Designed to work directly as a post-processing script.

## Pre-requisites

- **qbittorrent:** Install from the [official site](https://www.qbittorrent.org/).
- **Filebot:** Requires a valid license. Visit [Filebot](https://www.filebot.net/) for details.

## Installation

### Download Binary

You can download the latest release from the [releases page](https://github.com/sean1832/qbot/releases/latest).

#### Windows
```bash
curl -L https://github.com/sean1832/qbot/releases/latest/download/qbot-win-amd64.exe -o qbot.exe
```

#### Linux
```bash
curl -L https://github.com/sean1832/qbot/releases/latest/download/qbot-linux-amd64 -o qbot
```

Then, add the binary to your PATH.

### Compile from Source

1. Clone the repository:
```bash
git clone https://github.com/sean1832/qbot.git
cd qbot
```
2. Build the project:
```bash
go build .
```

## Usage

qbot is executed from the command line. The primary subcommand is `filebot`, which is used for media file processing.

```bash
qbot filebot [flags] <input_path> <media_category>
```

- `<input_path>`: The path of the downloaded file or folder.
- `<media_category>`: The type of media (supported values: `tv_show`, `anime`, `movie`).

### Global Options

- `-d, --destination`  
  **Description:** Destination path of the Plex media root.  
  **Example:** The Plex root should have subdirectories like `TV-Shows/Real`, `TV-Shows/Anime`, and `Movies`.

- `-n, --name`  
  **Description:** Torrent name to help Filebot identify the media.

- `-l, --language`  
  **Description:** Language of the media (default: `en`).

- `-a, --action`  
  **Description:** Action to perform on the file (e.g., `move`).

- `-c, --conflict`  
  **Description:** Conflict resolution strategy (e.g., `skip`).

- `-e, --ext`  
  **Description:** Comma-separated file extensions to process.  
  **Default:** `mkv,mp4,avi,mov,rmvb`

- `-x, --exclude`  
  **Description:** Comma-separated directories to exclude from processing.

- `--temp`  
  **Description:** Temporary directory used for processing (to avoid issues with folder names).  
  **Default:** `.temp`

- `--log`  
  **Description:** Path to a log file where output will be recorded.

- `-t, --tags`  
  **Description:** Comma-separated tags for additional Filebot options (format: `filter:xxx`).

### Media Configurations

qbot supports different media types with predefined directory structures and naming formats:

- **tv_show:**  
  - **Format:** `./{n}/Season {s}/{n} - {s00e00} - {t}`  
  - **Destination Subdirectory:** `/TV-Show/Real`

- **anime:**  
  - **Format:** `./{n}/Season {s}/{n} - {s00e00} - {t}`  
  - **Destination Subdirectory:** `/TV-Show/Anime`

- **movie:**  
  - **Format:** `./{ny}/{ny}`  
  - **Destination Subdirectory:** `/Movie`

## qbittorrent Integration

To integrate qbot as a post-processing script for qbittorrent, you can create a simple shell script that sets up the environment and calls qbot. For example:

**Create a script:** `usr/local/bin/qbot-script.sh`
```bash
#!/bin/bash
export HOME="/path/to/home"
exec qbot "$@"
```
Make the script executable:
```bash
chmod +x /usr/local/bin/qbot-script.sh
```

**Example qbittorrent command:**
```bash
qbot-script.sh filebot %F %L -d /path/to/media/root -n %N -a move -c skip -l en -e "mkv,mp4,avi,mov,rmvb" -x "sample,extras" -t "filter:myfilter" --temp /path/to/temp_root --log "/path/to/log"
```
Where:
- `%F` is the downloaded file/folder path.
- `%L` is the media label.
- `%N` is the torrent name.

Adjust the flags and paths as needed for your environment.

## Todo

- [ ] Add support for a configuration file to simplify complex setups.

---

## License

This project is licensed under the [Apache 2.0 License](LICENSE).