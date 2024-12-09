package internal

import (
	"fmt"
	"jit_vcs/config"
	"os"
	"path/filepath"
	"strings"
)

type TreeEntry struct {
	Path   string
	BlobID string
}

type Tree struct {
	ID      string
	Entries []TreeEntry
}

func BuildTreeFromFiles(files *Index) (*Tree, error) {
	var entries []TreeEntry

	for _, f := range *files {
		entries = append(entries, TreeEntry{Path: f.Filepath, BlobID: f.Hash})
	}

	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("%s %s\n", e.BlobID, e.Path))
	}

	treeID := ComputeHash([]byte(sb.String()))

	t := &Tree{
		ID:      treeID,
		Entries: entries,
	}

	err := t.Save()
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Tree) Save() error {
	var sb strings.Builder
	for _, e := range t.Entries {
		sb.WriteString(fmt.Sprintf("%s %s\n", e.BlobID, e.Path))
	}
	return os.WriteFile(
		filepath.Join(config.REPO_DIR, config.OBJECTS_DIR, t.ID),
		[]byte(sb.String()), 0644,
	)
}
