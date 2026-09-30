package handlers

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v5"
)

type LandingPageData struct {
	Announcement           string
	TotalVoters            int
	TotalVotes             int
	Participation          int
	ChairmanCandidates     []CandidateView
	ViceChairmanCandidates []CandidateView
	Credits                []CreditView
}

type CreditView struct {
	ID        int
	Name      string
	ClassName string
	Division  string
	PhotoURL  string
	Period    string
	OrderNum  int
}

type CandidateView struct {
	ID              int
	CandidateNumber int
	Name            string
	ClassName       string
	PhotoURL        string
	Vision          string
	Mission         string
	Program         string
	Position        string
}

func LandingPageHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var announcement string
		_ = db.QueryRow("SELECT announcement_text FROM settings ORDER BY updated_at DESC LIMIT 1").Scan(&announcement)

		if announcement == "" {
			announcement = "Selamat datang di sistem voting OSIS SMK NIBA Business School. Silakan pilih Ketua dan Wakil Ketua OSIS periode berikutnya."
		}

		var totalVoters, totalVotes int
		_ = db.QueryRow("SELECT COUNT(*) FROM voters").Scan(&totalVoters)
		_ = db.QueryRow("SELECT COUNT(*) FROM votes").Scan(&totalVotes)

		participation := 0
		if totalVoters > 0 {
			participation = (totalVotes * 100) / totalVoters
		}

		chairmen := []CandidateView{}
		rows, err := db.Query("SELECT id, COALESCE(candidate_number, 1), name, class_name, photo_url, vision, mission, program FROM candidates WHERE position = 'CHAIRMAN' ORDER BY candidate_number ASC, id ASC")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var c CandidateView
				if err := rows.Scan(&c.ID, &c.CandidateNumber, &c.Name, &c.ClassName, &c.PhotoURL, &c.Vision, &c.Mission, &c.Program); err == nil {
					c.Position = "CHAIRMAN"
					chairmen = append(chairmen, c)
				}
			}
		}

		vice := []CandidateView{}
		rows2, err2 := db.Query("SELECT id, COALESCE(candidate_number, 1), name, class_name, photo_url, vision, mission, program FROM candidates WHERE position = 'VICE_CHAIRMAN' ORDER BY candidate_number ASC, id ASC")
		if err2 == nil {
			defer rows2.Close()
			for rows2.Next() {
				var c CandidateView
				if err := rows2.Scan(&c.ID, &c.CandidateNumber, &c.Name, &c.ClassName, &c.PhotoURL, &c.Vision, &c.Mission, &c.Program); err == nil {
					c.Position = "VICE_CHAIRMAN"
					vice = append(vice, c)
				}
			}
		}

		credits := []CreditView{}
		rows3, err3 := db.Query("SELECT id, name, class_name, division, COALESCE(photo_url, ''), COALESCE(period, ''), COALESCE(order_num, 1) FROM credits ORDER BY order_num ASC, id ASC")
		if err3 == nil {
			defer rows3.Close()
			for rows3.Next() {
				var cr CreditView
				if err := rows3.Scan(&cr.ID, &cr.Name, &cr.ClassName, &cr.Division, &cr.PhotoURL, &cr.Period, &cr.OrderNum); err == nil {
					credits = append(credits, cr)
				}
			}
		}

		data := LandingPageData{
			Announcement:           announcement,
			TotalVoters:            totalVoters,
			TotalVotes:             totalVotes,
			Participation:          participation,
			ChairmanCandidates:     chairmen,
			ViceChairmanCandidates: vice,
			Credits:                credits,
		}
		return c.Render(http.StatusOK, "landing.html", data)
	}
}
