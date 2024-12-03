package command

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func cleanup() {
	os.RemoveAll(REPO_DIR)
}

func TestInit(t *testing.T) {
	defer cleanup()
	cleanup()

	t.Run("Initialize repository", func(t *testing.T) {
		err := Init([]string{})
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Check directories
		dirs := []string{
			REPO_DIR,
			filepath.Join(REPO_DIR, OBJECTS_DIR),
			filepath.Join(REPO_DIR, REFS_DIR, "heads"),
		}

		for _, dir := range dirs {
			dir := dir
			t.Run(fmt.Sprintf("Checking directory: %s", dir), func(t *testing.T) {
				_, err := os.Stat(dir)
				if errors.Is(err, fs.ErrNotExist) {
					t.Errorf("Directory %s not created", dir)
				}
			})
		}

		// Check .jit/HEAD
		t.Run("Check file .jit/HEAD", func(t *testing.T) {
			_, err := os.Stat(filepath.Join(REPO_DIR, HEAD_PATH))
			if errors.Is(err, fs.ErrNotExist) {
				t.Errorf(".jit/HEAD not created")
			}
		})

		// Check .jit/HEAD content
		t.Run("Check content of .jit/HEAD", func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(REPO_DIR, HEAD_PATH))
			if err != nil {
				t.Errorf("Failed to read .jit/HEAD: %v", err)
			}
			expected := "ref: refs/heads/master\n"
			if string(content) != expected {
				t.Errorf("Unexpected content in .jit/HEAD: got %s, want %s", string(content), expected)
			}
		})
	})

	t.Run("Fail if repository already exists", func(t *testing.T) {
		os.Mkdir(REPO_DIR, 0755) // Simulate existing repo
		defer cleanup()
		err := Init([]string{})
		if err == nil {
			t.Errorf("Expected error when repository already exists, got nil")
		}
	})

	t.Run("Fail if arguments are provided", func(t *testing.T) {
		err := Init([]string{"unexpected"})
		if err == nil {
			t.Errorf("Expected error for invalid usage, got nil")
		}
	})
}
