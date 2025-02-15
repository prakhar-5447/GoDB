package db

import (
	"database/sql"
	"log"
)

// RunMigrations ensures all required tables exist
func RunMigrations(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS indexes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL,
		table_name TEXT NOT NULL,
		index_name TEXT NOT NULL UNIQUE,
		columns TEXT NOT NULL,
		UNIQUE(user_id, index_name)
	)`)
	if err != nil {
		log.Println("❌ Failed to run migrations:", err)
		return err
	}

	log.Println("✅ Migrations applied successfully!")
	return nil
}
