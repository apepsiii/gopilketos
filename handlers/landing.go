package handlers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

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
	TotalCredits           int
	Testimonials           []TestimonialView
}

type DemisionerPageData struct {
	Credits    []CreditView
	Periods    []string
	Divisions  []string
	TotalCount int
}

type DPTVoterItem struct {
	ID          int
	MaskedUUID  string
	Name        string
	ClassName   string
	MaskedPhone string
	HasVoted    bool
	IsPresent   bool
	AttendedAt  string
}

type DPTPageData struct {
	Voters        []DPTVoterItem
	Classes       []string
	TotalVoters   int
	TotalVoted    int
	TotalNotVoted int
	TotalPresent  int
	Participation int
	TotalClasses  int
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
	VideoURL        string
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
		rows, err := db.Query("SELECT id, COALESCE(candidate_number, 1), name, class_name, photo_url, vision, mission, program, COALESCE(video_url, '') FROM candidates WHERE position = 'CHAIRMAN' ORDER BY candidate_number ASC, id ASC")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var c CandidateView
				if err := rows.Scan(&c.ID, &c.CandidateNumber, &c.Name, &c.ClassName, &c.PhotoURL, &c.Vision, &c.Mission, &c.Program, &c.VideoURL); err == nil {
					c.Position = "CHAIRMAN"
					chairmen = append(chairmen, c)
				}
			}
		}

		vice := []CandidateView{}
		rows2, err2 := db.Query("SELECT id, COALESCE(candidate_number, 1), name, class_name, photo_url, vision, mission, program, COALESCE(video_url, '') FROM candidates WHERE position = 'VICE_CHAIRMAN' ORDER BY candidate_number ASC, id ASC")
		if err2 == nil {
			defer rows2.Close()
			for rows2.Next() {
				var c CandidateView
				if err := rows2.Scan(&c.ID, &c.CandidateNumber, &c.Name, &c.ClassName, &c.PhotoURL, &c.Vision, &c.Mission, &c.Program, &c.VideoURL); err == nil {
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

		totalCredits := len(credits)
		featuredCredits := credits
		if len(featuredCredits) > 4 {
			featuredCredits = featuredCredits[:4]
		}

		testimonials := []TestimonialView{}
		rowsTestim, errTestim := db.Query(`SELECT id, name, class_name, COALESCE(message, ''), 
			COALESCE(video_path, ''), COALESCE(video_type, 'video/webm'), 
			COALESCE(gdrive_file_id, ''), COALESCE(gdrive_url, ''), 
			COALESCE(sync_status, 'local'), COALESCE(created_at, '') 
			FROM testimonials WHERE is_approved = 1 ORDER BY id DESC LIMIT 8`)
		if errTestim == nil {
			defer rowsTestim.Close()
			for rowsTestim.Next() {
				var t TestimonialView
				var createdAt time.Time
				if err := rowsTestim.Scan(&t.ID, &t.Name, &t.ClassName, &t.Message, &t.VideoPath, &t.VideoType, &t.GDriveFileID, &t.GDriveURL, &t.SyncStatus, &createdAt); err == nil {
					t.CreatedAt = createdAt.Format("02 Jan 2006")
					testimonials = append(testimonials, t)
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
			Credits:                featuredCredits,
			TotalCredits:           totalCredits,
			Testimonials:           testimonials,
		}
		return c.Render(http.StatusOK, "landing.html", data)
	}
}

func DemisionerPageHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		credits := []CreditView{}
		rows, err := db.Query("SELECT id, name, class_name, division, COALESCE(photo_url, ''), COALESCE(period, ''), COALESCE(order_num, 1) FROM credits ORDER BY order_num ASC, id ASC")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var cr CreditView
				if err := rows.Scan(&cr.ID, &cr.Name, &cr.ClassName, &cr.Division, &cr.PhotoURL, &cr.Period, &cr.OrderNum); err == nil {
					credits = append(credits, cr)
				}
			}
		}

		// Extract unique periods and divisions preserving chronological order
		periodMap := make(map[string]bool)
		divisionMap := make(map[string]bool)
		periods := []string{}
		divisions := []string{}

		for _, cr := range credits {
			p := strings.TrimSpace(cr.Period)
			if p != "" && !periodMap[p] {
				periodMap[p] = true
				periods = append(periods, p)
			}
			d := strings.TrimSpace(cr.Division)
			if d != "" && !divisionMap[d] {
				divisionMap[d] = true
				divisions = append(divisions, d)
			}
		}

		data := DemisionerPageData{
			Credits:    credits,
			Periods:    periods,
			Divisions:  divisions,
			TotalCount: len(credits),
		}
		return c.Render(http.StatusOK, "demisioner.html", data)
	}
}

func maskPhoneNumber(phone string) string {
	phone = strings.TrimSpace(phone)
	if len(phone) < 7 {
		if phone == "" {
			return "-"
		}
		return phone
	}
	prefix := phone[:4]
	suffix := phone[len(phone)-4:]
	return prefix + "-****-" + suffix
}

func maskUUID(uid string) string {
	uid = strings.TrimSpace(uid)
	if len(uid) <= 4 {
		return "••••"
	}
	return "••••-" + uid[len(uid)-4:]
}

// DPTPageHandler serves the public Daftar Pemilih Tetap (DPT) directory
func DPTPageHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		rows, err := db.Query(`SELECT 
			id, uuid, name, class_name, 
			COALESCE(phone_number, ''), 
			COALESCE(has_voted, 0), 
			COALESCE(presence_status, 0), 
			COALESCE(attended_at, '') 
		FROM voters 
		ORDER BY class_name ASC, name ASC`)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memuat DPT: "+err.Error())
		}
		defer rows.Close()

		var voters []DPTVoterItem
		classMap := make(map[string]bool)
		var classes []string
		totalVoted := 0
		totalPresent := 0

		for rows.Next() {
			var v DPTVoterItem
			var rawUUID, rawPhone string
			var hasVotedInt, presenceInt int
			var rawAttendedAt string

			if err := rows.Scan(
				&v.ID, &rawUUID, &v.Name, &v.ClassName,
				&rawPhone, &hasVotedInt, &presenceInt, &rawAttendedAt,
			); err == nil {
				v.HasVoted = hasVotedInt == 1
				v.IsPresent = presenceInt == 1
				if v.HasVoted {
					totalVoted++
				}
				if v.IsPresent {
					totalPresent++
				}

				if rawAttendedAt != "" {
					if t, err := time.Parse("2006-01-02 15:04:05", rawAttendedAt); err == nil {
						v.AttendedAt = t.Format("15:04 WIB")
					} else if t, err := time.Parse(time.RFC3339, rawAttendedAt); err == nil {
						v.AttendedAt = t.Format("15:04 WIB")
					} else {
						v.AttendedAt = rawAttendedAt
					}
				}

				// Mask UUID & Phone for privacy and ballot security
				v.MaskedUUID = maskUUID(rawUUID)
				v.MaskedPhone = maskPhoneNumber(rawPhone)

				className := strings.TrimSpace(v.ClassName)
				if className != "" && !classMap[className] {
					classMap[className] = true
					classes = append(classes, className)
				}

				voters = append(voters, v)
			}
		}

		totalVoters := len(voters)
		participation := 0
		if totalVoters > 0 {
			participation = (totalVoted * 100) / totalVoters
		}

		data := DPTPageData{
			Voters:        voters,
			Classes:       classes,
			TotalVoters:   totalVoters,
			TotalVoted:    totalVoted,
			TotalNotVoted: totalVoters - totalVoted,
			TotalPresent:  totalPresent,
			Participation: participation,
			TotalClasses:  len(classes),
		}

		return c.Render(http.StatusOK, "dpt.html", data)
	}
}
