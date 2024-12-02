package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	repoPath   = ".jit"
	refsDir    = "refs"
	objectsDir = "objects"
	headPath   = "HEAD"
)

func Init(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: jit init")
	}

	if _, err := os.Stat(repoPath); errors.Is(err, os.ErrExist) {
		return fmt.Errorf("A repository already exists at %s", repoPath)
	}

	// create .jit directory
	if err := os.Mkdir(repoPath, 0755); err != nil {
		return fmt.Errorf("Failed to create directory: %s\n%s",
			repoPath, err,
		)
	}

	// create .jit/refs, .jit/objects dirs
	for _, dir := range []string{refsDir, objectsDir} {
		dirPath := filepath.Join(repoPath, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("Failed to create directory: %s\n%s",
				dirPath, err,
			)
		}
	}

	// create HEAD
	headFileContent := []byte("ref: refs/heads/main\n")
	headDir := filepath.Join(repoPath, headPath)
	if err := os.WriteFile(headDir, headFileContent, 0644); err != nil {
		return fmt.Errorf("Failed to write to file: %s\n%s",
			headDir, err,
		)
	}

	fmt.Printf("Initialized empty jit repository in %s\n", repoPath)
	return nil
}
