package db

import (
	"fmt"
	"os"
	"path/filepath"
)

const DBDir = "data"

// GetDatabasePath returns the full path for a given database name
func GetDatabasePath(userID, dbName string) string {
	// The data directory is defined in the config package (or use a constant).
	return filepath.Join(DBDir, userID, fmt.Sprintf("%s.db", dbName))
}

// EnsureDBDirectory makes sure the `data/` directory exists
func EnsureDBDirectory() error {
	if _, err := os.Stat(DBDir); os.IsNotExist(err) {
		return os.Mkdir(DBDir, os.ModePerm)
	}
	return nil
}

func EnsureUserDBDirectory(userID string) error {
	userDir := filepath.Join(DBDir, userID)
	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		return os.Mkdir(userDir, os.ModePerm)
	}
	return nil
}
