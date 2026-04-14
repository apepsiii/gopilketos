package database

import (
	"database/sql"
)

func Migrate(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			announcement_text TEXT,
			onesender_enabled INTEGER DEFAULT 0,
			onesender_api_url TEXT,
			onesender_api_key TEXT,
			onesender_template TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS candidates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			class_name TEXT NOT NULL,
			photo_url TEXT,
			vision TEXT,
			mission TEXT,
			program TEXT,
			position TEXT NOT NULL CHECK(position IN ('CHAIRMAN', 'VICE_CHAIRMAN')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS voters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			uuid TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			class_name TEXT NOT NULL,
			phone_number TEXT NOT NULL,
			has_voted INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS votes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			masked_uuid TEXT NOT NULL,
			chairman_id INTEGER NOT NULL,
			vice_chairman_id INTEGER NOT NULL,
			voted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (chairman_id) REFERENCES candidates(id),
			FOREIGN KEY (vice_chairman_id) REFERENCES candidates(id)
		);`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}

	columnChecks := map[string]string{
		"onesender_enabled":  "INTEGER DEFAULT 0",
		"onesender_api_url":  "TEXT",
		"onesender_api_key":  "TEXT",
		"onesender_template": "TEXT",
	}
	for col, colType := range columnChecks {
		var dummy int
		err := db.QueryRow("SELECT " + col + " FROM settings LIMIT 1").Scan(&dummy)
		if err != nil {
			db.Exec("ALTER TABLE settings ADD COLUMN " + col + " " + colType)
		}
	}

	voterColumnChecks := map[string]string{
		"presence_status": "INTEGER DEFAULT 0",
		"attended_at":     "DATETIME",
	}
	for col, colType := range voterColumnChecks {
		var dummy int
		err := db.QueryRow("SELECT " + col + " FROM voters LIMIT 1").Scan(&dummy)
		if err != nil {
			db.Exec("ALTER TABLE voters ADD COLUMN " + col + " " + colType)
		}
	}

	return nil
}
