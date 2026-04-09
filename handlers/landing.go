package handlers

import (
	"database/sql"
	"net/http"
	"github.com/labstack/echo/v5"
)

type LandingPageData struct {
	Announcement string
	ChairmanCandidates []CandidateView
	ViceChairmanCandidates []CandidateView
}

type CandidateView struct {
	ID int
	Name string
	ClassName string
	PhotoURL string
	Vision string
	Mission string
	Program string
	Position string
}

func LandingPageHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		// Ambil pengumuman
		var announcement string
		_ = db.QueryRow("SELECT announcement_text FROM settings ORDER BY updated_at DESC LIMIT 1").Scan(&announcement)

		// Ambil kandidat ketua
		chairmen := []CandidateView{}
		rows, _ := db.Query("SELECT name, class_name, photo_url, vision, mission, program FROM candidates WHERE position = 'CHAIRMAN'")
		defer func() { if rows != nil { rows.Close() } }()
		for rows != nil && rows.Next() {
			var c CandidateView
			_ = rows.Scan(&c.Name, &c.ClassName, &c.PhotoURL, &c.Vision, &c.Mission, &c.Program)
			chairmen = append(chairmen, c)
		}

		// Ambil kandidat wakil
		vice := []CandidateView{}
		rows2, _ := db.Query("SELECT name, class_name, photo_url, vision, mission, program FROM candidates WHERE position = 'VICE_CHAIRMAN'")
		defer func() { if rows2 != nil { rows2.Close() } }()
		for rows2 != nil && rows2.Next() {
			var c CandidateView
			_ = rows2.Scan(&c.Name, &c.ClassName, &c.PhotoURL, &c.Vision, &c.Mission, &c.Program)
			vice = append(vice, c)
		}

		data := LandingPageData{
			Announcement: announcement,
			ChairmanCandidates: chairmen,
			ViceChairmanCandidates: vice,
		}
		return c.Render(http.StatusOK, "landing.html", data)
	}
}
