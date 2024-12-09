package command

import (
	"os"
	"testing"
)

// Creates a temp directory.
// Changes directory into temp directory.
// Returns the previous directory
func SetupTempDirCd(t *testing.T) string {
	tempDir := t.TempDir()
	prevDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
		return ""
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change working directory: %v", err)
		return ""
	}

	return prevDir
}

// Changes directory to prevDir
func ChangeDirectory(prevDir string, t *testing.T) {
	if err := os.Chdir(prevDir); err != nil {
		t.Fatalf("Failed to restore working directory: %v", err)
	}
}
