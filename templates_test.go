package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"strings"
	"testing"
	"time"

	"gopilketos/database"
	"gopilketos/handlers"
	"gopilketos/services"
	_ "github.com/mattn/go-sqlite3"
)

func TestTemplatesAndMigration(t *testing.T) {
	// Test template parsing
	funcMap := template.FuncMap{
		"eq": func(a, b interface{}) bool {
			return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
		},
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

	// Test rendering dpt.html
	buf.Reset()
	dptData := handlers.DPTPageData{
		Voters: []handlers.DPTVoterItem{
			{
				ID:          1,
				MaskedUUID:  "••••-abcd",
				Name:        "Siti Aminah",
				ClassName:   "XII AKL 1",
				MaskedPhone: "0812-****-4321",
				HasVoted:    false,
				IsPresent:   true,
				AttendedAt:  "08:30 WIB",
			},
		},
		Classes:       []string{"XII AKL 1"},
		TotalVoters:   1,
		TotalVoted:    0,
		TotalNotVoted: 1,
		TotalPresent:  1,
		Participation: 0,
		TotalClasses:  1,
	}
	if err := tmpl.ExecuteTemplate(&buf, "dpt.html", dptData); err != nil {
		t.Fatalf("Failed to execute dpt.html: %v", err)
	}

	// Test rendering admin_candidate_form.html
	buf.Reset()
	candidateFormData := handlers.AdminCandidateFormData{
		AdminLayoutData: handlers.AdminLayoutData{
			Title:           "Tambah Kandidat",
			PageTitle:       "Tambah Kandidat",
			PageSubtitle:    "Form profil kandidat",
			ContentTemplate: "admin_candidate_form_content",
			ActiveTab:       "candidates",
		},
		CandidateNumber: 1,
		Action:          "Tambah Kandidat",
	}
	if err := tmpl.ExecuteTemplate(&buf, "admin_candidate_form.html", candidateFormData); err != nil {
		t.Fatalf("Failed to execute admin_candidate_form.html: %v", err)
	}
}

func TestTeacherSalutationsAndMessages(t *testing.T) {
	// Test teacher detection
	if !services.IsTeacherOrStaff("GURU") {
		t.Errorf("Expected GURU to be recognized as teacher/staff")
	}
	if !services.IsTeacherOrStaff("STAF") {
		t.Errorf("Expected STAF to be recognized as teacher/staff")
	}
	if !services.IsTeacherOrStaff("Satpam") {
		t.Errorf("Expected Satpam to be recognized as teacher/staff")
	}
	if services.IsTeacherOrStaff("X-MPLB") {
		t.Errorf("Expected X-MPLB to NOT be recognized as teacher/staff")
	}
	if services.IsTeacherOrStaff("XI-PM") {
		t.Errorf("Expected XI-PM to NOT be recognized as teacher/staff")
	}

	// Test salutations
	sapaanTeacher, ythTeacher, panggilanTeacher := services.GetSalutation("GURU")
	if sapaanTeacher != "Bapak/Ibu" || ythTeacher != "Yth. Bapak/Ibu" || panggilanTeacher != "Bapak/Ibu" {
		t.Errorf("Unexpected teacher salutations: %s, %s, %s", sapaanTeacher, ythTeacher, panggilanTeacher)
	}

	sapaanStudent, ythStudent, panggilanStudent := services.GetSalutation("X-MPLB")
	if sapaanStudent != "Halo" || ythStudent != "Halo" || panggilanStudent != "kamu" {
		t.Errorf("Unexpected student salutations: %s, %s, %s", sapaanStudent, ythStudent, panggilanStudent)
	}

	// Test vote messages
	client := &services.WhatsAppClient{}
	teacherMsg := client.BuildVoteMessage("Muhammad Saepurahman, S.E.", "GURU", "Kandidat A", "Kandidat B", time.Now())
	if !strings.Contains(teacherMsg, "Yth. Bapak/Ibu Muhammad Saepurahman, S.E.") {
		t.Errorf("Expected teacher vote message to contain 'Yth. Bapak/Ibu', got: %s", teacherMsg)
	}

	studentMsg := client.BuildVoteMessage("Ahmad Fauzi", "X-MPLB", "Kandidat A", "Kandidat B", time.Now())
	if !strings.Contains(studentMsg, "Halo Ahmad Fauzi (X-MPLB)!") {
		t.Errorf("Expected student vote message to contain 'Halo Ahmad Fauzi (X-MPLB)!', got: %s", studentMsg)
	}
}
