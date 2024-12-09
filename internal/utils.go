package internal

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

func ComputeHash(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func FilterEmptyLines(s string) []string {
	lines := strings.Split(s, "\n")
	var res []string

	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			res = append(res, l)
		}
	}

	return res
}
