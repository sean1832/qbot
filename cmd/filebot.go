/*
Copyright © 2025 qbot <dev@zekezhang.com>
*/
package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	filebot "github.com/sean1832/qbot/pkg/core"
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
var logFile string
var tagsStr string

type Options struct {
	filter string // e.g. s==1 (see https://www.filebot.net/forums/viewtopic.php?t=2127)
}

var filebotCmd = &cobra.Command{
	Use:   "filebot",
	Short: "Activate filebot automation",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize logging to a file.
		InitLogging()

		inputDir := filepath.Clean(args[0])
		mediaCategory := args[1]

		db, action, conflict, err := ProcessEnums(mediaCategory, actionStr, conflictStr)
		if err != nil {
			log.Fatalf("Error processing enums: %s", err)
			return
		}

		config, err := GetMediaConfig(mediaCategory)
		if err != nil {
			log.Fatalf("Error getting media configuration: %s", err)
		}

		// Build the output directory using the media root from the config.
		outputDir := filepath.Clean(filepath.Join(plexRootPath, config.Root))
		inputDir = filepath.ToSlash(inputDir)
		outputDir = filepath.ToSlash(outputDir)

		// =======================================================================
		// Always move all files to a temporary directory with a clean folder name.
		// Avoids issues with folder names (like spaces) when using filebot.
		// =======================================================================
		if err := ValidateInputPath(tempRoot); err != nil {
			log.Fatalf("Error validating temporary directory: %s", err)
		}

		if err := os.MkdirAll(tempRoot, os.ModePerm); err != nil {
			log.Fatalf("Error creating temporary directory: %s", err)
		}

		// Move files to the temporary directory, excluding certain paths.
		var excludedDirs []string
		if excludedDirsStr != "" {
			excludedDirs = strings.Split(excludedDirsStr, ",")
		}
		if err := MoveFilesWithExclusion(inputDir, tempRoot, excludedDirs, false); err != nil {
			log.Fatalf("Error moving files to temporary directory: %s", err)
		}
		log.Printf("Moved files to temporary directory: %s \n", tempRoot)

		// Scan the updated inputDir for existing extensions.
		userExtensions := strings.Split(extensionsStr, ",")
		existingExts, err := GetExistingExtensions(inputDir)
		if err != nil {
			log.Fatalf("Error scanning for existing extensions: %s", err)
		}
		extensions := TryUseExtensions(userExtensions, existingExts)
		if len(extensions) == 0 {
			log.Println("No valid extensions found in the input path")
		}
		opt, err := deconstructTags(tagsStr)
		if err != nil {
			log.Fatalf("Error deconstructing tags: %s", err)
		}

		// Process each extension.
		for _, ext := range extensions {
			tempInputPath := filepath.Join(tempRoot, "*."+ext)
			log.Println("Processing:", tempInputPath)
			msg, err := filebot.Rename(tempInputPath, outputDir, query, config.Format, db, action, conflict, language, opt.filter)
			if err != nil {
				log.Fatalf("Error renaming files: %s, [Message: %s]", err, msg)
			}
		}

		// =======================================================================
		// Cleanup: remove the temporary directory after processing.
		// =======================================================================
		if err := os.RemoveAll(tempRoot); err != nil {
			log.Fatalf("Error cleaning up temporary directory: %s", err)
		}
		log.Println("Cleaned up temporary directory.")
	},
}

type MediaConfig struct {
	Format string
	Root   string
}

var mediaConfigs = map[string]MediaConfig{
	"tv_show": {Format: "./{n}/Season {s}/{n} - {s00e00} - {t}", Root: "/TV-Show/Real"},
	"anime":   {Format: "./{n}/Season {s}/{n} - {s00e00} - {t}", Root: "/TV-Show/Anime"},
	"movie":   {Format: "./{ny}/{ny}", Root: "/Movie"},
}

func GetMediaConfig(category string) (MediaConfig, error) {
	if config, ok := mediaConfigs[category]; ok {
		return config, nil
	}
	return MediaConfig{}, fmt.Errorf("type of media %s is not supported", category)
}

// ValidateInputPath ensures the input path does not contain spaces.
func ValidateInputPath(inputPath string) error {
	if strings.Contains(inputPath, " ") {
		return fmt.Errorf("input path contains spaces: %s", inputPath)
	}
	return nil
}

// ProcessEnums converts string parameters into proper enum types.
func ProcessEnums(tag string, actionStr string, conflictStr string) (filebot.DB, filebot.Action, filebot.Conflict, error) {
	db, err := GetDB(tag)
	if err != nil {
		return -1, -1, -1, fmt.Errorf("failed to get DB: %w", err)
	}

	action, err := filebot.ActionFromString(actionStr)
	if err != nil {
		return -1, -1, -1, fmt.Errorf("failed to parse action: %w", err)
	}

	conflict, err := filebot.ConflictFromString(conflictStr)
	if err != nil {
		return -1, -1, -1, fmt.Errorf("failed to parse conflict: %w", err)
	}

	return db, action, conflict, nil
}

// GetDB returns the appropriate DB based on the media category.
func GetDB(category string) (filebot.DB, error) {
	switch category {
	case "tv_show", "anime":
		return filebot.TheMovieDB_TV, nil
	case "movie":
		return filebot.TheMovieDB, nil
	default:
		return -1, fmt.Errorf("type of media %s is not supported", category)
	}
}

// TryUseExtensions returns only those user-provided extensions that exist.
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

// GetExistingExtensions walks the filePath and returns a slice of unique extensions.
func GetExistingExtensions(filePath string) ([]string, error) {
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
		return nil, fmt.Errorf("error scanning path %s: %w", filePath, err)
	}

	uniqueExts := make([]string, 0, len(extMap))
	for ext := range extMap {
		uniqueExts = append(uniqueExts, ext)
	}
	return uniqueExts, nil
}

// MoveFile moves a file from source to dest.
// It tries os.Rename first, then falls back to copying if needed.
func MoveFile(source, dest string) error {
	// Ensure the destination directory exists.
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
	}

	// Try to move the file via os.Rename.
	if err := os.Rename(source, dest); err == nil {
		return nil
	} else if !isCrossDeviceError(err) {
		return fmt.Errorf("failed to rename file from %s to %s: %w", source, dest, err)
	}

	// Fallback: source and dest are on different filesystems.
	inputFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("couldn't open source file %s: %w", source, err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("couldn't create destination file %s: %w", dest, err)
	}
	defer outputFile.Close()

	// Copy file content.
	if _, err := io.Copy(outputFile, inputFile); err != nil {
		return fmt.Errorf("couldn't copy file from %s to %s: %w", source, dest, err)
	}

	// Ensure the data is flushed to disk.
	if err := outputFile.Sync(); err != nil {
		return fmt.Errorf("couldn't sync destination file %s: %w", dest, err)
	}

	// Optionally, preserve the file mode.
	fi, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("couldn't stat source file %s: %w", source, err)
	}
	if err = os.Chmod(dest, fi.Mode()); err != nil {
		return fmt.Errorf("couldn't set permissions on %s: %w", dest, err)
	}

	// Remove the source file.
	if err := os.Remove(source); err != nil {
		return fmt.Errorf("couldn't remove source file %s: %w", source, err)
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
func MoveFilesWithExclusion(sourceDir, destinationDir string, excludedPaths []string, preserveStruct bool) error {
	return filepath.Walk(sourceDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking path %s: %w", path, err)
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
						return fmt.Errorf("failed to get relative path for %s: %w", path, err)
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

func deconstructTags(tags string) (Options, error) {
	var options Options
	for _, tag := range strings.Split(tags, ",") {
		if strings.HasPrefix(tag, "filter:") {
			log.Println("tag:", tag)
			filter := strings.Split(tag, ":")[1]
			options.filter = filter
		}
	}
	return options, nil
}

// InitLogging initializes logging. If a log file is specified, it ensures the directory exists
// and opens the file in append mode; otherwise, logging defaults to stderr.
func InitLogging() {
	if logFile == "" {
		log.SetOutput(os.Stderr)
		return
	}
	// Ensure the log file directory exists.
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		log.Fatalf("failed to create log directory %s: %v", logDir, err)
	}

	// Open (or create) the log file with append mode.
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("failed to open log file %s: %v", logFile, err)
	}
	log.SetOutput(file)
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
	filebotCmd.Flags().StringVarP(&tempRoot, "temp", "", ".temp", "temporary root directory for moving files with exclusion")
	filebotCmd.Flags().StringVarP(&logFile, "log", "", "", "log file path")
	filebotCmd.Flags().StringVarP(&tagsStr, "tags", "t", "", "tags for processing (comma separated) (format: filter:xxx)")

	// Example usage for qbittorrent post-process script:
	// ./qbot.exe filebot %F %L -d /path/to/media/root -n %N -t %G -a move -c skip -l en -e "mkv,mp4,avi,mov,rmvb" -x "sample,extras" --temp /path/to/temp_root --log /var/log/qbot.log
}
