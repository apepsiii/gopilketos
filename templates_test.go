package main

import (
	"database/sql"
	"html/template"
	"testing"

	"gopilketos/database"
	_ "github.com/mattn/go-sqlite3"
)

func TestTemplatesAndMigration(t *testing.T) {
	// Test template parsing
	funcMap := template.FuncMap{
		"eq": func(a, b string) bool { return a == b },
	}
	tmpl := template.New("").Funcs(funcMap)
	viewsFS := getViewsFS()
	tmpl, err := tmpl.ParseFS(viewsFS, "voter/*.html")
	if err != nil {
		t.Fatalf("Failed to parse voter templates: %v", err)
	}
	tmpl, err = tmpl.ParseFS(viewsFS, "admin/*.html")
	if err != nil {
		t.Fatalf("Failed to parse admin templates: %v", err)
	}
	tmpl, err = tmpl.ParseFS(viewsFS, "layouts/*.html")
	if err != nil {
		t.Fatalf("Failed to parse layout templates: %v", err)
	}

	// Test SQLite in-memory migration
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open sqlite: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		t.Fatalf("Database migration failed: %v", err)
	}

	// Verify credits table exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM credits").Scan(&count)
	if err != nil {
		t.Fatalf("credits table query failed: %v", err)
	}
}
