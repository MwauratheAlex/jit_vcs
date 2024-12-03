package vcs

import (
	"crypto/sha1"
	"encoding/hex"
)

func ComputeHash(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
