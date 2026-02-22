package database

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations executes all .sql files in dir in lexicographic order.
func RunMigrations(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		path := filepath.Join(dir, name)
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = DB.Exec(string(body))
		if err != nil {
			log.Printf("Migration %s failed: %v", name, err)
			return err
		}
		log.Printf("Migration applied: %s", name)
	}
	return nil
}
