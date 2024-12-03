package command

import (
	"fmt"
	"jit_vcs/config"
	"os"
	"path/filepath"
)

func Init(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("usage: jit init")
	}

	if _, err := os.Stat(config.REPO_DIR); err == nil {
		return fmt.Errorf("A repository already exists at %s", config.REPO_DIR)
	}

	// create .jit directory
	if err := os.Mkdir(config.REPO_DIR, 0755); err != nil {
		return fmt.Errorf("Failed to create directory: %s\n%s",
			config.REPO_DIR, err,
		)
	}

	// create .jit/refs, .jit/objects dirs
	dirs := []string{
		filepath.Join(config.REPO_DIR, config.OBJECTS_DIR),
		filepath.Join(config.REPO_DIR, config.REFS_DIR, "heads"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("Failed to create directory: %s\n%s",
				dir, err,
			)
		}
	}

	// create .jit/HEAD
	headFilePath := filepath.Join(config.REPO_DIR, config.HEAD_PATH)
	headFileContent := []byte("ref: refs/heads/master\n")
	if err := os.WriteFile(headFilePath, headFileContent, 0644); err != nil {
		return fmt.Errorf("Failed to write to file: %s\n%s",
			headFilePath, err,
		)
	}

	fmt.Printf("Initialized empty jit repository in %s\n", config.REPO_DIR)
	return nil
}
