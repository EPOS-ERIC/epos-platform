package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ApplyDir(db *sql.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".sql") {
			files = append(files, name)
		}
	}

	sort.Strings(files)
	for _, name := range files {
		path := filepath.Join(dir, name)
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read migration %s: %w", name, readErr)
		}
		if _, execErr := db.Exec(string(content)); execErr != nil {
			return fmt.Errorf("execute migration %s: %w", name, execErr)
		}
	}

	return nil
}
