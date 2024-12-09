package command

import (
	"crypto/sha1"
	"fmt"
	"jit_vcs/config"
	"os"
	"path/filepath"
	"testing"
)

func TestAdd(t *testing.T) {
	// setup
	currDir := SetupTempDirCd(t)
	defer ChangeDirectory(currDir, t)

	if err := Init([]string{}); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// testing
	testFileName := "testfile.txt"
	testFileContent := []byte("Yes. It's jit!")
	if err := os.WriteFile(testFileName, testFileContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if err := Add([]string{testFileName}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	expectedHash := computeHash(testFileContent)

	objectFilePath := filepath.Join(
		config.REPO_DIR, config.OBJECTS_DIR, expectedHash)
	if _, err := os.Stat(objectFilePath); os.IsNotExist(err) {
		t.Errorf("Expected object file does not exist: %s", objectFilePath)
	}

	indexFilePath := filepath.Join(config.REPO_DIR, "index")
	indexData, err := os.ReadFile(indexFilePath)
	if err != nil {
		t.Fatalf("Failed to read index file: %v", err)
	}

	expectedIndexEntry := fmt.Sprintf("%s %s\n", expectedHash, testFileName)
	if string(indexData) != expectedIndexEntry {
		t.Errorf("Index file content mismatch.\nExpected: %s\nGot: %s",
			expectedIndexEntry, string(indexData),
		)
	}
}

func computeHash(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}
