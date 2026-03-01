package config

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashWorkspaceContent(path string, data []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(path))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
