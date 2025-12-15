package hash

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculate(t *testing.T) {
	data := []byte("Hello, World!")
	hash := Calculate(data)

	// Known SHA-256 hash for "Hello, World!"
	expected := "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"

	if hash != expected {
		t.Errorf("Hash mismatch:\ngot:      %s\nexpected: %s", hash, expected)
	}
}

func TestCalculateFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := []byte("Hello, World!")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash, err := CalculateFile(testFile)
	if err != nil {
		t.Fatalf("CalculateFile failed: %v", err)
	}

	expected := "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"
	if hash != expected {
		t.Errorf("Hash mismatch:\ngot:      %s\nexpected: %s", hash, expected)
	}
}

func TestVerify(t *testing.T) {
	data := []byte("Hello, World!")
	correctHash := "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"
	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"

	if !Verify(data, correctHash) {
		t.Error("Verify should return true for correct hash")
	}

	if Verify(data, wrongHash) {
		t.Error("Verify should return false for wrong hash")
	}
}

func TestCalculateEmpty(t *testing.T) {
	hash := Calculate([]byte{})
	// SHA-256 of empty string
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	if hash != expected {
		t.Errorf("Empty hash mismatch:\ngot:      %s\nexpected: %s", hash, expected)
	}
}
