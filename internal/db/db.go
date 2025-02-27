package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/prakhar-5447/GoDB/internal/auth"
)

func OpenDatabase(ConnectionString string) (*sql.DB, error) {
	// Authenticate using the connection string from the request.
	// Parse the connection string to extract user info.
	username, password, dbName, err := ParseConnectionString(ConnectionString)
	if err != nil {
		return nil, err
	}
	// Validate credentials using the auth package.
	auth, err := auth.ValidateUserCredentials(username, password)
	if !auth || err != nil {
		return nil, fmt.Errorf("authentication failed")
	}

	// Log successful authentication.
	log.Printf("Authenticated user %s for database %s", username, dbName)

	// Ensure the user's database directory exists.
	if err := EnsureUserDBDirectory(username); err != nil {
		return nil, fmt.Errorf("failed to create user database directory: %w", err)
	}

	// Build the full database path using the user ID and database name.
	dbPath := GetDatabasePath(username, dbName)

	// Open a connection to the SQLite database.
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", dbName, err)
	}

	// Optionally enable foreign key constraints.
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return db, nil
}

// ParseConnectionString extracts the username, password, and database name.
// Expected format: grpc://username:password/databaseName
func ParseConnectionString(connStr string) (username, password, database string, err error) {
	// Ensure the connection string starts with "grpc://"
	if !strings.HasPrefix(connStr, "grpc://") {
		return "", "", "", fmt.Errorf("connection string must start with 'grpc://'")
	}
	trimmed := strings.TrimPrefix(connStr, "grpc://")

	// Split into credentials and database parts using "/"
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid connection string format: missing database name")
	}

	// Parse credentials (expected format "username:password")
	credParts := strings.SplitN(parts[0], ":", 2)
	if len(credParts) != 2 {
		return "", "", "", fmt.Errorf("invalid credentials format")
	}
	username = credParts[0]
	password = credParts[1]

	// The remainder is the database name.
	database = parts[1]

	return username, password, database, nil
}
