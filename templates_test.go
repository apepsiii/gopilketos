package main

import (
	"bytes"
	"database/sql"
	"html/template"
	"testing"

	"gopilketos/database"
	"gopilketos/handlers"
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

	// Test rendering landing.html
	var buf bytes.Buffer
	landingData := handlers.LandingPageData{
		Announcement: "Pengumuman",
		Credits: []handlers.CreditView{
			{ID: 1, Name: "Ahmad", ClassName: "XII RPL 1", Division: "Ketua OSIS", Period: "2024/2025", OrderNum: 1},
		},
		TotalCredits: 5,
	}
	if err := tmpl.ExecuteTemplate(&buf, "landing.html", landingData); err != nil {
		t.Fatalf("Failed to execute landing.html: %v", err)
	}

	// Test rendering demisioner.html
	buf.Reset()
	demisionerData := handlers.DemisionerPageData{
		Credits: []handlers.CreditView{
			{ID: 1, Name: "Ahmad", ClassName: "XII RPL 1", Division: "Ketua OSIS", Period: "2024/2025", OrderNum: 1},
		},
		Periods:    []string{"2024/2025"},
		Divisions:  []string{"Ketua OSIS"},
		TotalCount: 1,
	}
	if err := tmpl.ExecuteTemplate(&buf, "demisioner.html", demisionerData); err != nil {
		t.Fatalf("Failed to execute demisioner.html: %v", err)
	}

	// Verify testimonials table exists
	var testimCount int
	err = db.QueryRow("SELECT COUNT(*) FROM testimonials").Scan(&testimCount)
	if err != nil {
		t.Fatalf("testimonials table query failed: %v", err)
	}

	// Test rendering kiosk.html
	buf.Reset()
	kioskData := handlers.KioskPageData{
		Title:        "Video Booth Kesan & Pesan",
		AutoApprove:  true,
		GDriveActive: false,
	}
	if err := tmpl.ExecuteTemplate(&buf, "kiosk.html", kioskData); err != nil {
		t.Fatalf("Failed to execute kiosk.html: %v", err)
	}

	// Test rendering admin_testimonials.html
	buf.Reset()
	adminTestimData := handlers.AdminTestimonialsData{
		AdminLayoutData: handlers.AdminLayoutData{
			Title:           "Video Kiosk",
			PageTitle:       "Video Kiosk",
			PageSubtitle:    "Kelola video",
			ContentTemplate: "admin_testimonials_content",
			ActiveTab:       "testimonials",
		},
		Testimonials: []handlers.TestimonialView{
			{ID: 1, Name: "Budi", ClassName: "X RPL 1", Message: "Semoga sukses!", IsApproved: true, SyncStatus: "local", CreatedAt: "30 Sep 2026 19:00 WIB"},
		},
		TotalCount:   1,
		LocalCount:   1,
		GDriveActive: true,
	}
	if err := tmpl.ExecuteTemplate(&buf, "admin_testimonials.html", adminTestimData); err != nil {
		t.Fatalf("Failed to execute admin_testimonials.html: %v", err)
	}
}
