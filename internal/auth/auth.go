package auth

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

const authDBPath = "data/auth.db"

// InitAuthDatabase ensures that the auth database and users table exist,
// and inserts a default user if the table is empty.
func InitAuthDatabase() error {
	// Ensure the data directory exists.
	if err := os.MkdirAll("data", os.ModePerm); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite3", authDBPath)
	if err != nil {
		return fmt.Errorf("failed to open auth database: %w", err)
	}
	defer db.Close()

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password TEXT
	);`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Check if there are any users
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query users table: %w", err)
	}

	// Insert a default user if table is empty.
	if count == 0 {
		// For demonstration purposes: default username "john", password "secret123"
		_, err = db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", "john", "secret123")
		if err != nil {
			return fmt.Errorf("failed to insert default user: %w", err)
		}
	}

	return nil
}

// CreateUser inserts a new user with the provided username and password into the auth database.
func CreateUser(username, password string) error {
	// Open auth database.
	db, err := sql.Open("sqlite3", authDBPath)
	if err != nil {
		return fmt.Errorf("failed to open auth database: %w", err)
	}
	defer db.Close()

	// Insert the user.
	_, err = db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, password)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

// ValidateUserCredentials checks the users table for the given username and password.
// Returns true if credentials match, false otherwise.
func ValidateUserCredentials(username, password string) (bool, error) {
	db, err := sql.Open("sqlite3", authDBPath)
	if err != nil {
		return false, fmt.Errorf("failed to open auth database: %w", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ? AND password = ?", username, password).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to query auth database: %w", err)
	}

	return count > 0, nil
}
