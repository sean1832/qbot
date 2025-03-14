package core

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Action enum
type Action int

const (
	move Action = iota
	copy
	symlink
	hardlink
	test
)

func (a Action) ToString() string {
	return [...]string{"move", "copy", "symlink", "hardlink", "test"}[a]
}

func ActionFromString(action string) (Action, error) {
	switch action {
	case "move":
		return move, nil
	case "copy":
		return copy, nil
	case "symlink":
		return symlink, nil
	case "hardlink":
		return hardlink, nil
	case "test":
		return test, nil
	default:
		return -1, errors.New("invalid action")
	}
}

// Conflict enum
type Conflict int

const (
	skip Conflict = iota
	replace
	auto
	index
	fail
)

func (c Conflict) ToString() string {
	return [...]string{"skip", "replace", "auto", "index", "fail"}[c]
}

func ConflictFromString(conflict string) (Conflict, error) {
	switch conflict {
	case "skip":
		return skip, nil
	case "replace":
		return replace, nil
	case "auto":
		return auto, nil
	case "index":
		return index, nil
	case "fail":
		return fail, nil
	default:
		return -1, errors.New("invalid conflict")
	}
}

// DB enum
type DB int

const (
	TheMovieDB_TV DB = iota
	TheMovieDB
	TheTVDB
	AniDB
	OMDb
)

func (d DB) ToString() string {
	return [...]string{"TheMovieDB::TV", "TheMovieDB", "TheTVDB", "AniDB", "OMDb"}[d]
}

func DBFromString(db string) (DB, error) {
	switch db {
	case "TheMovieDB::TV":
		return TheMovieDB_TV, nil
	case "TheMovieDB":
		return TheMovieDB, nil
	case "TheTVDB":
		return TheTVDB, nil
	case "AniDB":
		return AniDB, nil
	case "OMDb":
		return OMDb, nil
	default:
		return -1, errors.New("invalid db")
	}
}

func Rename(inputPath string, outputPath string, query string, format string, db DB, action Action, conflict Conflict, language string) (string, error) {
	// Common arguments for filebot.
	commonArgs := []string{
		"-r",
		"--db", db.ToString(),
		"--format", fmt.Sprintf(`"%s"`, format),
		"--q", fmt.Sprintf(`"%s"`, query),
		"--action", action.ToString(),
		"--conflict", conflict.ToString(),
		"--lang", language,
		"--output", fmt.Sprintf(`"%s"`, outputPath),
		"-non-strict",
	}

	var cmd *exec.Cmd

	// Windows does not support shell globbing so we expand manually.
	if runtime.GOOS == "windows" {
		matches, err := filepath.Glob(inputPath)
		if err != nil {
			return "", fmt.Errorf("error expanding glob: %w", err)
		}
		if len(matches) == 0 {
			return "", fmt.Errorf("no files found for pattern %s", inputPath)
		}
		// Build the argument slice: first the '-rename', then the expanded file paths, then the common args.
		args := append([]string{"-rename"}, matches...)
		args = append(args, commonArgs...)
		cmd = exec.Command("filebot", args...)
		log.Printf("Executing command: filebot %v", args)
	} else {
		// On Unix-like systems, let the shell expand the glob.
		// Do NOT quote inputPath, so that the shell can expand wildcards.
		cmdStr := fmt.Sprintf("filebot -rename %s %s", inputPath, strings.Join(commonArgs, " "))
		cmd = exec.Command("sh", "-c", cmdStr)
		log.Printf("Executing command: %s", cmdStr)
	}

	output, err := cmd.CombinedOutput()
	return string(output), err
}
