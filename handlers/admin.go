package handlers

import (
	"database/sql"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type AdminDashboardData struct {
	AdminLayoutData
	TotalVoters         int
	VotesCast           int
	Participation       string
	ChairmanResults     []CandidateResult
	ViceChairmanResults []CandidateResult
	Announcement        string
}

type CandidateResult struct {
	Name  string
	Count int
}

type AdminCandidatesData struct {
	AdminLayoutData
	Candidates []CandidateView
}

type AdminCandidateFormData struct {
	AdminLayoutData
	ID        int
	Name      string
	ClassName string
	PhotoURL  string
	Vision    string
	Mission   string
	Program   string
	Position  string
	Action    string
}

type AdminVotersData struct {
	AdminLayoutData
	Voters []VoterView
}

type AdminVoterFormData struct {
	AdminLayoutData
}

type VoterView struct {
	ID          int
	UUID        string
	Name        string
	ClassName   string
	PhoneNumber string
	HasVoted    string
}

type AdminLogsData struct {
	AdminLayoutData
	Logs []AuditLogView
}

type AuditLogView struct {
	MaskedUUID   string
	ChairmanName string
	ViceName     string
	VotedAt      string
}

type AdminSettingsData struct {
	AdminLayoutData
	Announcement string
}

type AdminLoginData struct {
	Error string
}

type AdminLayoutData struct {
	Title           string
	PageTitle       string
	PageSubtitle    string
	ContentTemplate string
	ActiveTab       string
}

func adminLayout(title, pageTitle, pageSubtitle, contentTemplate, activeTab string) AdminLayoutData {
	return AdminLayoutData{
		Title:           title,
		PageTitle:       pageTitle,
		PageSubtitle:    pageSubtitle,
		ContentTemplate: contentTemplate,
		ActiveTab:       activeTab,
	}
}

func createAdminSessionValue(username, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(username))
	return username + "|" + fmt.Sprintf("%x", h.Sum(nil))
}

func validateAdminSessionValue(value, secret, expectedUser string) bool {
	parts := strings.SplitN(value, "|", 2)
	if len(parts) != 2 {
		return false
	}
	if parts[0] != expectedUser {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0]))
	expectedSig := fmt.Sprintf("%x", mac.Sum(nil))
	return hmac.Equal([]byte(parts[1]), []byte(expectedSig))
}

func AdminSessionMiddleware(secret, expectedUser string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			cookie, err := c.Cookie("admin_session")
			if err != nil || !validateAdminSessionValue(cookie.Value, secret, expectedUser) {
				return c.Redirect(http.StatusSeeOther, "/admin/login")
			}
			return next(c)
		}
	}
}

func AdminLoginPageHandler(secret, expectedUser string) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if cookie, err := c.Cookie("admin_session"); err == nil && validateAdminSessionValue(cookie.Value, secret, expectedUser) {
			return c.Redirect(http.StatusSeeOther, "/admin")
		}
		return c.Render(http.StatusOK, "admin_login.html", AdminLoginData{})
	}
}

func AdminLoginHandler(adminUser, adminPass, secret string) echo.HandlerFunc {
	return func(c *echo.Context) error {
		username := c.FormValue("username")
		password := c.FormValue("password")
		if username != adminUser || password != adminPass {
			return c.Render(http.StatusUnauthorized, "admin_login.html", AdminLoginData{Error: "Username atau kata sandi salah."})
		}
		cookie := new(http.Cookie)
		cookie.Name = "admin_session"
		cookie.Value = createAdminSessionValue(adminUser, secret)
		cookie.Path = "/"
		cookie.HttpOnly = true
		cookie.SameSite = http.SameSiteLaxMode
		c.SetCookie(cookie)
		return c.Redirect(http.StatusSeeOther, "/admin")
	}
}

func AdminLogoutHandler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		cookie := new(http.Cookie)
		cookie.Name = "admin_session"
		cookie.Value = ""
		cookie.Path = "/"
		cookie.HttpOnly = true
		cookie.MaxAge = -1
		cookie.Expires = time.Unix(0, 0)
		c.SetCookie(cookie)
		return c.Redirect(http.StatusSeeOther, "/admin/login")
	}
}

func AdminDashboardHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var totalVoters, votesCast int
		_ = db.QueryRow("SELECT COUNT(*) FROM voters").Scan(&totalVoters)
		_ = db.QueryRow("SELECT COUNT(*) FROM votes").Scan(&votesCast)

		participation := "0%"
		if totalVoters > 0 {
			participation = fmt.Sprintf("%.0f%%", float64(votesCast)*100.0/float64(totalVoters))
		}

		chairmanResults, _ := fetchCandidateResults(db, "CHAIRMAN", "chairman_id")
		viceResults, _ := fetchCandidateResults(db, "VICE_CHAIRMAN", "vice_chairman_id")

		var announcement string
		_ = db.QueryRow("SELECT announcement_text FROM settings ORDER BY updated_at DESC LIMIT 1").Scan(&announcement)

		data := AdminDashboardData{
			AdminLayoutData: adminLayout("Dashboard | OSIS Admin", "Election Dashboard", "Status Overview", "admin_dashboard_content", "dashboard"),
			TotalVoters:         totalVoters,
			VotesCast:           votesCast,
			Participation:       participation,
			ChairmanResults:     chairmanResults,
			ViceChairmanResults: viceResults,
			Announcement:        announcement,
		}
		return c.Render(http.StatusOK, "admin_dashboard.html", data)
	}
}

func fetchCandidateResults(db *sql.DB, position string, voteField string) ([]CandidateResult, error) {
	query := fmt.Sprintf(`SELECT c.name, COUNT(v.id) FROM candidates c
		LEFT JOIN votes v ON v.%s = c.id
		WHERE c.position = ?
		GROUP BY c.id
		ORDER BY COUNT(v.id) DESC, c.name`, voteField)
	rows, err := db.Query(query, position)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []CandidateResult{}
	for rows.Next() {
		var r CandidateResult
		if err := rows.Scan(&r.Name, &r.Count); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

func AdminCandidatesHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		rows, err := db.Query("SELECT id, name, class_name, photo_url, vision, mission, program, position FROM candidates ORDER BY position, id")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memuat daftar kandidat")
		}
		defer rows.Close()

		candidates := []CandidateView{}
		for rows.Next() {
			var candidate CandidateView
			if err := rows.Scan(&candidate.ID, &candidate.Name, &candidate.ClassName, &candidate.PhotoURL, &candidate.Vision, &candidate.Mission, &candidate.Program, &candidate.Position); err != nil {
				return c.String(http.StatusInternalServerError, "Gagal memproses kandidat")
			}
			candidates = append(candidates, candidate)
		}

		return c.Render(http.StatusOK, "admin_candidates.html", AdminCandidatesData{
			AdminLayoutData: adminLayout("Candidates Management | OSIS Admin", "Candidates Management", "Manage candidate records and election roster.", "admin_candidates_content", "candidates"),
			Candidates:      candidates,
		})
	}
}

func AdminCandidateFormHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		idStr := c.QueryParam("id")
		form := AdminCandidateFormData{AdminLayoutData: adminLayout("Tambah Kandidat | OSIS Admin", "Tambah Kandidat", "Create or edit candidate profiles.", "admin_candidate_form_content", "candidates"), Action: "Tambah Kandidat"}
		if idStr != "" {
			id, err := strconv.Atoi(idStr)
			if err == nil {
				var photoURL string
				_ = db.QueryRow("SELECT id, name, class_name, photo_url, vision, mission, program, position FROM candidates WHERE id = ?", id).Scan(
					&form.ID, &form.Name, &form.ClassName, &photoURL, &form.Vision, &form.Mission, &form.Program, &form.Position)
				form.PhotoURL = photoURL
				form.Action = "Simpan Perubahan"
				form.AdminLayoutData = adminLayout("Edit Kandidat | OSIS Admin", "Edit Kandidat", "Create or edit candidate profiles.", "admin_candidate_form_content", "candidates")
			}
		}
		return c.Render(http.StatusOK, "admin_candidate_form.html", form)
	}
}

func AdminCandidateSaveHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		idStr := c.FormValue("id")
		name := c.FormValue("name")
		className := c.FormValue("class_name")
		vision := c.FormValue("vision")
		mission := c.FormValue("mission")
		program := c.FormValue("program")
		position := c.FormValue("position")
		photoURL := c.FormValue("photo_url")

		if name == "" || className == "" || position == "" {
			return c.String(http.StatusBadRequest, "Nama, kelas, dan posisi wajib diisi")
		}

		if file, err := c.FormFile("photo"); err == nil && file != nil {
			src, err := file.Open()
			if err == nil {
				defer src.Close()
				filename := fmt.Sprintf("%s_%s", uuid.NewString(), filepath.Base(file.Filename))
				destPath := filepath.Join("public", "uploads", filename)
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err == nil {
				out, err := os.Create(destPath)
				if err == nil {
					defer out.Close()
					io.Copy(out, src)
					photoURL = "/static/uploads/" + filename
				}
				}
			}
		}

		if idStr == "" {
			_, err := db.Exec("INSERT INTO candidates (name, class_name, photo_url, vision, mission, program, position) VALUES (?, ?, ?, ?, ?, ?, ?)", name, className, photoURL, vision, mission, program, position)
			if err != nil {
				return c.String(http.StatusInternalServerError, "Gagal menyimpan kandidat")
			}
		} else {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return c.String(http.StatusBadRequest, "ID kandidat tidak valid")
			}
			_, err = db.Exec("UPDATE candidates SET name = ?, class_name = ?, photo_url = ?, vision = ?, mission = ?, program = ?, position = ? WHERE id = ?", name, className, photoURL, vision, mission, program, position, id)
			if err != nil {
				return c.String(http.StatusInternalServerError, "Gagal memperbarui kandidat")
			}
		}

		return c.Redirect(http.StatusSeeOther, "/admin/candidates")
	}
}

func AdminCandidateDeleteHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			return c.String(http.StatusBadRequest, "ID kandidat tidak valid")
		}
		_, err = db.Exec("DELETE FROM candidates WHERE id = ?", id)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal menghapus kandidat")
		}
		return c.Redirect(http.StatusSeeOther, "/admin/candidates")
	}
}

func AdminVotersHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		rows, err := db.Query("SELECT id, uuid, name, class_name, phone_number, has_voted FROM voters ORDER BY id DESC")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memuat daftar pemilih")
		}
		defer rows.Close()

		voters := []VoterView{}
		for rows.Next() {
			var v VoterView
			var hasVoted int
			if err := rows.Scan(&v.ID, &v.UUID, &v.Name, &v.ClassName, &v.PhoneNumber, &hasVoted); err != nil {
				return c.String(http.StatusInternalServerError, "Gagal memproses pemilih")
			}
			if hasVoted == 1 {
				v.HasVoted = "Sudah"
			} else {
				v.HasVoted = "Belum"
			}
			voters = append(voters, v)
		}
		return c.Render(http.StatusOK, "admin_voters.html", AdminVotersData{
			AdminLayoutData: adminLayout("Voters Management | OSIS Admin", "Voters Management", "Manage institutional electorate records.", "admin_voters_content", "voters"),
			Voters:          voters,
		})
	}
}

func AdminVoterFormHandler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		return c.Render(http.StatusOK, "admin_voter_form.html", AdminVoterFormData{
			AdminLayoutData: adminLayout("Tambah Pemilih | OSIS Admin", "Tambah Pemilih", "Add a new eligible voter to the DPT.", "admin_voter_form_content", "voters"),
		})
	}
}

func AdminVoterSaveHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		name := c.FormValue("name")
		className := c.FormValue("class_name")
		phoneNumber := c.FormValue("phone_number")
		if name == "" || className == "" || phoneNumber == "" {
			return c.String(http.StatusBadRequest, "Semua field wajib diisi")
		}
		uuidValue := uuid.NewString()
		_, err := db.Exec("INSERT INTO voters (uuid, name, class_name, phone_number) VALUES (?, ?, ?, ?)", uuidValue, name, className, phoneNumber)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal menyimpan pemilih")
		}
		return c.Redirect(http.StatusSeeOther, "/admin/voters")
	}
}

func AdminVoterDeleteHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			return c.String(http.StatusBadRequest, "ID pemilih tidak valid")
		}
		_, err = db.Exec("DELETE FROM voters WHERE id = ?", id)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal menghapus pemilih")
		}
		return c.Redirect(http.StatusSeeOther, "/admin/voters")
	}
}

func AdminLogsHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		rows, err := db.Query(`SELECT v.masked_uuid, c1.name, c2.name, v.voted_at
			FROM votes v
			JOIN candidates c1 ON v.chairman_id = c1.id
			JOIN candidates c2 ON v.vice_chairman_id = c2.id
			ORDER BY v.voted_at DESC`)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memuat log audit")
		}
		defer rows.Close()

		logs := []AuditLogView{}
		for rows.Next() {
			var entry AuditLogView
			if err := rows.Scan(&entry.MaskedUUID, &entry.ChairmanName, &entry.ViceName, &entry.VotedAt); err != nil {
				return c.String(http.StatusInternalServerError, "Gagal memproses log")
			}
			logs = append(logs, entry)
		}
		return c.Render(http.StatusOK, "admin_logs.html", AdminLogsData{
			AdminLayoutData: adminLayout("Audit Logs | OSIS Admin", "Audit Ledger", "Immutable record of all ballots cast.", "admin_logs_content", "logs"),
			Logs:            logs,
		})
	}
}

func AdminSettingsHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var announcement string
		_ = db.QueryRow("SELECT announcement_text FROM settings ORDER BY updated_at DESC LIMIT 1").Scan(&announcement)
		return c.Render(http.StatusOK, "admin_settings.html", AdminSettingsData{
			AdminLayoutData: adminLayout("Pengumuman | OSIS Admin", "Pengumuman Publik", "Atur teks pengumuman yang akan tampil di halaman landing page pemilih.", "admin_settings_content", "settings"),
			Announcement:    announcement,
		})
	}
}

func AdminSettingsSaveHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		announcement := c.FormValue("announcement")
		_, err := db.Exec("INSERT INTO settings (announcement_text) VALUES (?)", announcement)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal menyimpan pengumuman")
		}
		return c.Redirect(http.StatusSeeOther, "/admin/settings")
	}
}
