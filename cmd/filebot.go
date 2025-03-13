/*
Copyright © 2025 qbot <dev@zekezhang.com>
*/
package cmd

import (
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	logging "github.com/sean1832/qbot/internal"
	filebot "github.com/sean1832/qbot/pkg"
	"github.com/spf13/cobra"
)

var plexRootPath string
var query string
var actionStr string
var conflictStr string
var language string
var extensionsStr string
var excludedDirsStr string
var tempRoot string

// filebotCmd represents the filebot command
var filebotCmd = &cobra.Command{
	Use:   "filebot",
	Short: "Activate filebot automation",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Initial processing of arguments and enums.
		inputDir := filepath.Clean(args[0])
		mediaCategory := args[1]

		db, action, conflict, err := ProcessEnums(mediaCategory, actionStr, conflictStr)
		if err != nil {
			log.Println("Error processing enums:", err)
			return
		}

		format, err := GetFormat(mediaCategory)
		if err != nil {
			log.Println("Error getting format:", err)
			return
		}

		mediaRoot, err := GetMediaCategoryRoot(mediaCategory)
		if err != nil {
			log.Println("Error getting media category root:", err)
			return
		}

		// Build the output directory using filepath.Join for cross-platform compatibility.
		outputDir := filepath.Clean(filepath.Join(plexRootPath, mediaRoot))
		inputDir = filepath.ToSlash(inputDir)
		outputDir = filepath.ToSlash(outputDir)

		// ==========================================================================
		// Always move all files to a temporary directory with a clean folder name.
		// Avoids issues with folder names (like spaces) when using filebot.
		// ==========================================================================
		cleanTempDir := filepath.Join(tempRoot, "temp")
		if err := os.MkdirAll(cleanTempDir, os.ModePerm); err != nil {
			log.Println("Error creating temporary directory:", err)
			return
		}

		// Move files to the temporary directory, excluding certain paths.
		var excludedDirs []string
		if excludedDirsStr != "" {
			excludedDirs = strings.Split(excludedDirsStr, ",")
		}
		if err := MoveFilesWithExclusion(inputDir, cleanTempDir, excludedDirs, false); err != nil {
			log.Println("Error moving files to temporary directory:", err)
			return
		}
		// Update the input directory to point to the temporary folder.
		inputDir = cleanTempDir

		// Scan the updated inputDir for existing extensions.
		userExtensions := strings.Split(extensionsStr, ",")
		extensions := TryUseExtensions(userExtensions, GetExistingExtensions(inputDir))
		if len(extensions) == 0 {
			log.Println("No valid extensions found in the input path")
			// Continue
		}

		// Process files for each valid extension.
		for _, ext := range extensions {
			tempInputPath := filepath.Join(inputDir, "*."+ext)
			log.Println("Processing:", tempInputPath)
			var msg string
			if msg, err = filebot.Rename(tempInputPath, outputDir, query, format, db, action, conflict, language); err != nil {
				log.Println("Error renaming files:", err)
				log.Println("Error message:", msg)
				return
			}
		}

		// ==========================================================================
		// Cleanup: remove the temporary directory after processing.
		// ==========================================================================
		if err := os.RemoveAll(inputDir); err != nil {
			log.Println("Error cleaning up temporary directory:", err)
			return
		}
		log.Println("Cleaned up temporary directory.")
	},
}

func ProcessEnums(tag string, actionStr string, conflictStr string) (filebot.DB, filebot.Action, filebot.Conflict, error) {
	db, err := GetDB(tag)
	if err != nil {
		return -1, -1, -1, logging.LogErrorf("failed to get DB: %w", err)
	}

	action, err := filebot.ActionFromString(actionStr)
	if err != nil {
		return -1, -1, -1, logging.LogErrorf("failed to parse action: %w", err)
	}

	conflict, err := filebot.ConflictFromString(conflictStr)
	if err != nil {
		return -1, -1, -1, logging.LogErrorf("failed to parse conflict: %w", err)
	}

	return db, action, conflict, nil
}

func GetDB(category string) (filebot.DB, error) {
	switch category {
	case "tv_show", "anime":
		return filebot.TheMovieDB_TV, nil
	case "movie":
		return filebot.TheMovieDB, nil
	default:
		return -1, logging.LogErrorf("type of media %s is not supported", category)
	}
}

func GetFormat(category string) (string, error) {
	switch category {
	case "tv_show", "anime":
		return "/{n}/Season {s}/{n} - {s00e00} - {t}", nil
	case "movie":
		return "/{ny}/{ny}", nil
	default:
		return "", logging.LogErrorf("type of media %s is not supported", category)
	}
}

func GetMediaCategoryRoot(category string) (string, error) {
	switch category {
	case "tv_show":
		return "/TV-Shows/Real", nil
	case "anime":
		return "/TV-Shows/Anime", nil
	case "movie":
		return "/Movies", nil
	default:
		return "", logging.LogErrorf("type of media %s is not supported", category)
	}
}

func TryUseExtensions(extensionsInputs []string, existingExtensions []string) []string {
	// Build a set for existing extensions (normalized to lower-case).
	existingSet := make(map[string]struct{})
	for _, ext := range existingExtensions {
		normalized := strings.ToLower(strings.TrimPrefix(ext, "."))
		existingSet[normalized] = struct{}{}
	}

	// Include only those user inputs that exist.
	var result []string
	for _, inputExt := range extensionsInputs {
		normalized := strings.ToLower(strings.TrimPrefix(inputExt, "."))
		if _, ok := existingSet[normalized]; ok {
			result = append(result, normalized)
		}
	}
	return result
}

func GetExistingExtensions(filePath string) []string {
	extMap := make(map[string]struct{})
	err := filepath.Walk(filePath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			// Skip this file on error.
			return nil
		}
		if !info.IsDir() {
			ext := strings.TrimPrefix(filepath.Ext(info.Name()), ".")
			if ext != "" {
				extMap[ext] = struct{}{}
			}
		}
		return nil
	})

	if err != nil {
		logging.LogErrorf("Error scanning path %s: %v\n", filePath, err)
		return nil
	}

	uniqueExts := make([]string, 0, len(extMap))
	for ext := range extMap {
		uniqueExts = append(uniqueExts, ext)
	}
	return uniqueExts
}

// MoveFile moves a file from source to dest.
// It first tries os.Rename which is fast and atomic if source and dest are on the same filesystem.
// If os.Rename fails due to a cross-device error, it falls back to copying the file and then removing the source.
func MoveFile(source, dest string) error {
	// Ensure the destination directory exists.
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return logging.LogErrorf("failed to create destination directory %s: %w", destDir, err)
	}

	// Try to move the file via os.Rename.
	if err := os.Rename(source, dest); err == nil {
		return nil
	} else if !isCrossDeviceError(err) {
		// If the error is not because of a cross-device link, return it.
		return logging.LogErrorf("failed to rename file from %s to %s: %w", source, dest, err)
	}

	// Fallback: source and dest are on different filesystems.
	inputFile, err := os.Open(source)
	if err != nil {
		return logging.LogErrorf("couldn't open source file %s: %w", source, err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(dest)
	if err != nil {
		return logging.LogErrorf("couldn't create destination file %s: %w", dest, err)
	}
	defer outputFile.Close()

	// Copy file content.
	if _, err := io.Copy(outputFile, inputFile); err != nil {
		return logging.LogErrorf("couldn't copy file from %s to %s: %w", source, dest, err)
	}

	// Ensure the data is flushed to disk.
	if err := outputFile.Sync(); err != nil {
		return logging.LogErrorf("couldn't sync destination file %s: %w", dest, err)
	}

	// Optionally, preserve the file mode.
	if fi, err := os.Stat(source); err == nil {
		if err = os.Chmod(dest, fi.Mode()); err != nil {
			return logging.LogErrorf("couldn't set permissions on %s: %w", dest, err)
		}
	}

	// Remove the source file.
	if err := os.Remove(source); err != nil {
		return logging.LogErrorf("couldn't remove source file %s: %w", source, err)
	}

	return nil
}

// isCrossDeviceError checks if the error is due to a cross-device link.
func isCrossDeviceError(err error) bool {
	if linkErr, ok := err.(*os.LinkError); ok {
		if errno, ok := linkErr.Err.(syscall.Errno); ok {
			return errno == syscall.EXDEV
		}
	}
	return false
}

// MoveFilesWithExclusion walks the source directory and moves files that do not belong
// to the excluded paths to the destination directory while preserving the directory structure.
// In our updated implementation, passing an empty slice ensures that all files are moved.
func MoveFilesWithExclusion(sourceDir, destinationDir string, excludedPaths []string, preserveStruct bool) error {
	return filepath.Walk(sourceDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return logging.LogErrorf("error walking path %s: %w", path, err)
		}
		if !info.IsDir() {
			// Check if the current file should be excluded.
			exclude := false
			for _, excludedPath := range excludedPaths {
				if strings.Contains(path, excludedPath) {
					exclude = true
					break
				}
			}
			if !exclude {
				var destPath string
				if preserveStruct {
					// Preserve directory structure by computing the relative path.
					relPath, err := filepath.Rel(sourceDir, path)
					if err != nil {
						return err
					}
					destPath = filepath.Join(destinationDir, relPath)
				} else {
					// Move the file directly to the destination directory.
					destPath = filepath.Join(destinationDir, filepath.Base(path))
				}
				if err := MoveFile(path, destPath); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func init() {
	rootCmd.AddCommand(filebotCmd)
	filebotCmd.Flags().StringVarP(&plexRootPath, "destination", "d", ".", "destination path of the Plex media root. The root should contain directories like `TV-Shows/Real`, `TV-Shows/Anime`, `Movies`")
	filebotCmd.Flags().StringVarP(&query, "name", "n", "", "torrent name")
	filebotCmd.Flags().StringVarP(&language, "language", "l", "en", "language of the media")
	filebotCmd.Flags().StringVarP(&actionStr, "action", "a", "move", "action to take")
	filebotCmd.Flags().StringVarP(&conflictStr, "conflict", "c", "skip", "conflict resolution")
	filebotCmd.Flags().StringVarP(&extensionsStr, "ext", "e", "mkv,mp4,avi,mov,rmvb", "file extensions to process (comma separated)")
	filebotCmd.Flags().StringVarP(&excludedDirsStr, "exclude", "x", "", "directories to exclude (comma separated)")
	filebotCmd.Flags().StringVarP(&tempRoot, "temp", "t", ".temp", "temporary root directory for moving files with exclusion")

	// Example usage for qbittorrent post-process script:
	// ./qbot.exe filebot %F %L -d /path/to/media/root -n %N -a move -c skip -l en -e "mkv,mp4,avi,mov,rmvb" -x "sample,extras" -t /path/to/temp_root
}
