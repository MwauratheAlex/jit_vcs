package vcs

import (
	"errors"
	"fmt"
	"io/fs"
	"jit_vcs/config"
	"os"
	"path/filepath"
)

func AddToIndex(path string) error {
	//TODO: check if file is ignored

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	hash := ComputeHash(content)

	objectPath := filepath.Join(config.REPO_DIR, config.OBJECTS_DIR, hash)
	if _, err := os.Stat(objectPath); errors.Is(err, fs.ErrNotExist) {
		if err := os.WriteFile(objectPath, content, 0644); err != nil {
			return err
		}
	}

	indexPath := filepath.Join(config.REPO_DIR, "index")
	indexEntry := fmt.Sprintf("%s %s\n", hash, path)
	f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(indexEntry)

	return err
}
