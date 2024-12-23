package internal

import (
	"fmt"
	"io/fs"
	"jit/config"
	"os"
	"path/filepath"
	"strings"
)

type IndexEntry struct {
	Hash     string
	Filepath string
}

type Index []IndexEntry

// AddToIndex adds a file with <path> to the staging area
func AddToIndex(path string) error {

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return err
		}
		hasFiles := false
		for _, entry := range entries {
			if !entry.IsDir() {
				hasFiles = true
				break
			}
		}

		if !hasFiles {
			return nil
		}
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
			if len(parts) == 2 {
				indexEntries[parts[1]] = line
			}
		}
	}

	indexEntries[path] = fmt.Sprintf("%s %s", hash, path)

	var updatedIndexContent strings.Builder
	for _, entry := range indexEntries {
		updatedIndexContent.WriteString(entry + "\n")
	}

	return os.WriteFile(indexPath, []byte(updatedIndexContent.String()), 0644)
}

var index Index = nil

// loadIndex reads the index file and returns an Index
func loadIndex() (*Index, error) {

	data, err := os.ReadFile(filepath.Join(config.REPO_DIR, "index"))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			idxEntries := strings.Fields(l)
			if len(idxEntries) != 2 {
				continue
			}
			idxEntry := IndexEntry{
				Hash:     idxEntries[0],
				Filepath: idxEntries[1],
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
		indexEntry := fmt.Sprintf("%s %s\n", entry.Hash, entry.Filepath)
		sb.WriteString(indexEntry)
	}

	return os.WriteFile(indexPath, []byte(sb.String()), 0644)
}

// CreateFakeIndex generates a fake index from the current working directory
// Used for building working directory tree for change detection
func CreateFakeIndex(basePath string) (*Index, error) {
	var fakeIndex Index
	patterns, err := LoadIgnorePatterns()
	if err != nil {
		return nil, fmt.Errorf("failed to load .jitignore: %w", err)
	}

	// Walk the directory structure starting from basePath
	err = filepath.Walk(basePath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and the .jit repository
		if info.IsDir() {
			if strings.HasPrefix(path, config.REPO_DIR) {
				return filepath.SkipDir
			}
			return nil
		}

		if IsIgnonored(path, patterns) {
			fmt.Printf("skipping ingored file: %s\n", path)
			return nil
		}

		// Compute the hash of the file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file '%s': %w", path, err)
		}
		hash := ComputeHash(content)

		// Convert the file path to a relative path
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for '%s': %w", path, err)
		}
		relPath = filepath.ToSlash(relPath) // Normalize for consistency

		// Add to fake index
		fakeIndex = append(fakeIndex, IndexEntry{
			Filepath: relPath,
			Hash:     hash,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create fake index: %w", err)
	}

	return &fakeIndex, nil
}

// updateIndexFromTree updates the index given a tree hash
func updateIndexFromTree(treeHash string) error {
	tree, err := loadTree(treeHash)
	if err != nil {
		return fmt.Errorf("failed to load tree: %w", err)
	}

	var index Index
	for _, entry := range tree.Entries {
		fmt.Println(entry)
		index = append(index, IndexEntry{
			Filepath: entry.Name,
			Hash:     entry.Hash,
		})
	}

	err = saveIndex(&index)
	if err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

// printIndex prints an Index for debugging
func printIndex(idx *Index) {
	for _, i := range *idx {
		fmt.Println(i)
	}
}
