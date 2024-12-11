package internal

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

func ComputeHash(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// CopyDir copies <src> directory to <dst> directory
func CopyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		fileInfo, err := os.Stat(srcPath)
		if err != nil {
			return err
		}
		switch {
		case fileInfo.IsDir():
			err = CopyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		default:
			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CopyFile copies <src> file to <dst> file
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)

	return err
}
