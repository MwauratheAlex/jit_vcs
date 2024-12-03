package command

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	REPO_DIR    = ".jit"
	REFS_DIR    = "refs"
	OBJECTS_DIR = "objects"
	HEAD_PATH   = "HEAD"
)

func Init(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("usage: jit init")
	}

	if _, err := os.Stat(REPO_DIR); err == nil {
		return fmt.Errorf("A repository already exists at %s", REPO_DIR)
	}

	// create .jit directory
	if err := os.Mkdir(REPO_DIR, 0755); err != nil {
		return fmt.Errorf("Failed to create directory: %s\n%s",
			REPO_DIR, err,
		)
	}

	// create .jit/refs, .jit/objects dirs
	dirs := []string{
		filepath.Join(REPO_DIR, OBJECTS_DIR),
		filepath.Join(REPO_DIR, REFS_DIR, "heads"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("Failed to create directory: %s\n%s",
				dir, err,
			)
		}
	}

	// create .jit/HEAD
	headFilePath := filepath.Join(REPO_DIR, HEAD_PATH)
	headFileContent := []byte("ref: refs/heads/master\n")
	if err := os.WriteFile(headFilePath, headFileContent, 0644); err != nil {
		return fmt.Errorf("Failed to write to file: %s\n%s",
			headFilePath, err,
		)
	}

	fmt.Printf("Initialized empty jit repository in %s\n", REPO_DIR)
	return nil
}
