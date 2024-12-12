package internal

import (
	"fmt"
	"io/fs"
	"jit/config"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type IndexEntry struct {
	Hash     string
	Filepath string
	Mode     os.FileMode
}

type Index []IndexEntry

// AddToIndex adds a file with <path> to the staging area
func AddToIndex(path string) error {
	//TODO: check if file is ignored

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		// walk
		return filepath.Walk(
			path,
			func(filePath string, fileInfo fs.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				// skip dirs, process only their files
				if fileInfo.IsDir() {
					return nil
				}

				return AddToIndex(filePath)
			})
	}

	// handle files
	content, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}

	hash := ComputeHash(content)

	// write obj to if does not exist
	objectPath := filepath.Join(config.REPO_DIR, config.OBJECTS_DIR, hash)
	if _, err := os.Stat(objectPath); err != nil {
		if err := os.WriteFile(objectPath, content, 0644); err != nil {
			return err
		}
	}

	// write to index
	mode := fmt.Sprintf("%06o", info.Mode().Perm())

	indexPath := filepath.Join(config.REPO_DIR, "index")
	indexEntries := map[string]string{}
	if _, err := os.Stat(indexPath); err == nil {
		content, err := os.ReadFile(indexPath)
		if err != nil {
			return err
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) == 3 {
				indexEntries[parts[2]] = line
			}
		}
	}

	indexEntries[path] = fmt.Sprintf("%s %s %s", hash, mode, path)

	var updatedIndexContent strings.Builder
	for _, entry := range indexEntries {
		updatedIndexContent.WriteString(entry + "\n")
	}

	return os.WriteFile(indexPath, []byte(updatedIndexContent.String()), 0644)
}

var index Index = nil

// loadIndex reads the index file and returns an Index
// cached if access earlier because index can not be updated and fetched
// in the same operation hence cannot become stale.
func loadIndex() (*Index, error) {
	if index != nil {
		return &index, nil
	}

	data, err := os.ReadFile(filepath.Join(config.REPO_DIR, "index"))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			idxEntries := strings.Split(l, " ")
			if len(idxEntries) < 3 {
				continue
			}
			modeUint, err := strconv.ParseUint(idxEntries[1], 8, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid mode in index: %s", idxEntries[1])
			}
			mode := os.FileMode(modeUint)
			idxEntry := IndexEntry{
				Hash:     idxEntries[0],
				Mode:     mode,
				Filepath: idxEntries[2],
			}

			index = append(index, idxEntry)
		}

	}

	return &index, nil
}

// saveIndex saves the Index to file
func saveIndex(index *Index) error {
	indexPath := filepath.Join(config.REPO_DIR, "index")

	var sb strings.Builder
	for _, entry := range *index {
		mode := fmt.Sprintf("%04o", entry.Mode.Perm())
		indexEntry := fmt.Sprintf("%s %s %s\n", entry.Hash, mode, entry.Filepath)
		sb.WriteString(indexEntry)
	}

	return os.WriteFile(indexPath, []byte(sb.String()), 0644)
}
