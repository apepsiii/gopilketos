package database

import (
	"database/sql"
	"log"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
)

func InitDB(dbPath string) *sql.DB {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("failed to create database directory: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	return db
}
