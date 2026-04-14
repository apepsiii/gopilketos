package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) *sql.DB {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("failed to create database directory: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout=5000&_pragma=synchronous=NORMAL")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	return db
}
