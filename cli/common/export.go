package common

import (
	"fmt"
	"os"
	"path/filepath"
)

// Export writes content to a file in the given directory, creating the
// directory if it does not already exist.
//
// It returns the absolute path to the exported file.
func Export(path, filename string, content []byte) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for export path %s: %w", path, err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0o750)
		if err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	filePath := filepath.Join(path, filename)
	if err := CreateFileWithContent(filePath, string(content), false); err != nil {
		return "", fmt.Errorf("failed to write content to file %s: %w", filePath, err)
	}

	return filepath.Join(path, filename), nil
}
