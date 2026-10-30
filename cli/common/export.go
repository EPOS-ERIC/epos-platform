package common

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/EPOS-ERIC/epos-platform/cli/display"
)

// RenderOutputPath resolves and prepares the output directory for rendered files.
func RenderOutputPath(path string) (string, error) {
	if path == "" {
		display.Debug("output path not set, using current working directory")

		path, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current working directory: %w", err)
		}

		return path, nil
	}

	display.Debug("resolving output path")

	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for output path: %w", err)
	}

	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("environment directory %s already exists", path)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to check directory %s: %w", path, err)
	}

	if err := os.MkdirAll(path, 0o750); err != nil {
		return "", fmt.Errorf("failed to create environment directory %s: %w", path, err)
	}

	display.Debug("created output directory: %s", path)

	return path, nil
}

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

	return filePath, nil
}
