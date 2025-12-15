package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// Calculate computes SHA-256 hash of data and returns hex string
func Calculate(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// CalculateFile computes SHA-256 hash of a file
func CalculateFile(filepath string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// Verify checks if data matches expected hash
func Verify(data []byte, expectedHash string) bool {
	return Calculate(data) == expectedHash
}
