package handlers

import (
	"archive/zip"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"gopilketos/services"
)

type AdminDashboardData struct {
	AdminLayoutData
	TotalVoters         int
	VotesCast           int
	NotVoted            int
	Participation       string
	ChairmanResults     []CandidateResult
	ViceChairmanResults []CandidateResult
	Announcement        string
}

type CandidateResult struct {
	CandidateNumber int
	Name            string
	Count           int
}

type AdminCandidatesData struct {
	AdminLayoutData
	Candidates []CandidateView
}

type AdminCandidateFormData struct {
	AdminLayoutData
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
	Action          string
}

type AdminCreditsData struct {
	AdminLayoutData
	Credits        []CreditView
	Periods        []string
	SelectedPeriod string
}

type AdminCreditFormData struct {
	AdminLayoutData
	ID        int
	Name      string
	ClassName string
	Division  string
	PhotoURL  string
	Period    string
	OrderNum  int
	Action    string
}

type AdminVotersData struct {
	AdminLayoutData
	Voters        []VoterView
	ImportMessage string
}

type AdminVoterFormData struct {
	AdminLayoutData
	IsEdit      bool
	ID          int
	UUID        string
	Name        string
	ClassName   string
	PhoneNumber string
}

type VoterFormData struct {
	ID          int
	UUID        string
	Name        string
	ClassName   string
	PhoneNumber string
}

type VoterView struct {
	ID             int
	UUID           string
	Name           string
	ClassName      string
	PhoneNumber    string
	HasVoted       string
	PresenceStatus int
	AttendedAt     string
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
	Announcement      string
	OneSenderEnabled  bool
	OneSenderURL      string
	OneSenderKey      string
	OneSenderTemplate string
	WAEnabled         bool
	WAProvider        string
	WAURL                    string
	WADeviceID               string
	WAUsername               string
	WAPassword               string
	WATemplate               string
	GDriveEnabled            bool
	GDriveFolderID           string
	GDriveServiceAccountJSON string
	GDriveClientEmail        string
	GDriveDeleteLocal        bool
	TestimonialsAutoApprove  bool
	TestimonialsEnabled      bool
}

type AdminLoginData struct {
	Error string
}

type AdminAttendanceData struct {
	AdminLayoutData
	Voters     []VoterView
	TotalHadir int
	TotalBelum int
	Total      int
}

type VoteExport struct {
	ID               int    `json:"id"`
	MaskedUUID       string `json:"masked_uuid"`
	ChairmanID       int    `json:"chairman_id"`
	ChairmanName     string `json:"chairman_name"`
	ViceChairmanID   int    `json:"vice_chairman_id"`
	ViceChairmanName string `json:"vice_chairman_name"`
	VotedAt          string `json:"voted_at"`
}

type BackupData struct {
	Candidates []CandidateView `json:"candidates"`
	Voters     []VoterView     `json:"voters"`
	Votes      []VoteExport    `json:"votes"`
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

const AdminSessionDuration = 4 * time.Hour

func createAdminSessionValue(username, secret string) string {
	expiresAt := time.Now().Add(AdminSessionDuration).Unix()
	msg := fmt.Sprintf("%s:%d", username, expiresAt)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(msg))
	return fmt.Sprintf("%s:%x", msg, h.Sum(nil))
}

func validateAdminSessionValue(value, secret, expectedUser string) bool {
	parts := strings.SplitN(value, ":", 3)
	if len(parts) != 3 {
		return false
	}
	username := parts[0]
	expiryStr := parts[1]
	sig := parts[2]

	if username != expectedUser {
		return false
	}

	expiry, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return false
	}

	msg := fmt.Sprintf("%s:%s", username, expiryStr)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	expectedSig := fmt.Sprintf("%x", mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expectedSig))
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

var (
	loginAttemptsMu sync.Mutex
	loginAttempts   = make(map[string][]time.Time)
)

const (
	maxLoginAttempts     = 5
	loginLockoutDuration = 5 * time.Minute
)

func isLoginRateLimited(ip string) bool {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-loginLockoutDuration)

	var valid []time.Time
	for _, t := range loginAttempts[ip] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	loginAttempts[ip] = valid
	return len(valid) >= maxLoginAttempts
}

func recordFailedLogin(ip string) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()
	loginAttempts[ip] = append(loginAttempts[ip], time.Now())
}

func recordSuccessfulLogin(ip string) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()
	delete(loginAttempts, ip)
}

func AdminLoginHandler(adminUser, adminPass, secret string) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ip := c.RealIP()
		if isLoginRateLimited(ip) {
			return c.Render(http.StatusTooManyRequests, "admin_login.html", AdminLoginData{
				Error: "Terlalu banyak percobaan login gagal. Akun dikunci sementara selama 5 menit untuk keamanan.",
			})
		}

		username := c.FormValue("username")
		password := c.FormValue("password")
		if username != adminUser || password != adminPass {
			recordFailedLogin(ip)
			return c.Render(http.StatusUnauthorized, "admin_login.html", AdminLoginData{Error: "Username atau kata sandi salah."})
		}

		recordSuccessfulLogin(ip)
		cookie := new(http.Cookie)
		cookie.Name = "admin_session"
		cookie.Value = createAdminSessionValue(adminUser, secret)
		cookie.Path = "/"
		cookie.HttpOnly = true
		cookie.MaxAge = int(AdminSessionDuration.Seconds())
		cookie.Expires = time.Now().Add(AdminSessionDuration)
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
		var totalVoters, votesCast, notVoted int
		_ = db.QueryRow("SELECT COUNT(*) FROM voters").Scan(&totalVoters)
		_ = db.QueryRow("SELECT COUNT(*) FROM votes").Scan(&votesCast)
		notVoted = totalVoters - votesCast
		if notVoted < 0 {
			notVoted = 0
		}

		participation := "0%"
		if totalVoters > 0 {
			participation = fmt.Sprintf("%.0f%%", float64(votesCast)*100.0/float64(totalVoters))
		}

		chairmanResults, _ := fetchCandidateResults(db, "CHAIRMAN", "chairman_id")
		viceResults, _ := fetchCandidateResults(db, "VICE_CHAIRMAN", "vice_chairman_id")

		var announcement string
		_ = db.QueryRow("SELECT announcement_text FROM settings ORDER BY updated_at DESC LIMIT 1").Scan(&announcement)

		data := AdminDashboardData{
			AdminLayoutData:     adminLayout("Dashboard | OSIS Admin", "Election Dashboard", "Status Overview", "admin_dashboard_content", "dashboard"),
			TotalVoters:         totalVoters,
			VotesCast:           votesCast,
			NotVoted:            notVoted,
			Participation:       participation,
			ChairmanResults:     chairmanResults,
			ViceChairmanResults: viceResults,
			Announcement:        announcement,
		}
		return c.Render(http.StatusOK, "admin_dashboard.html", data)
	}
}

func fetchCandidateResults(db *sql.DB, position string, voteField string) ([]CandidateResult, error) {
	query := fmt.Sprintf(`SELECT COALESCE(c.candidate_number, 1), c.name, COUNT(v.id) FROM candidates c
		LEFT JOIN votes v ON v.%s = c.id
		WHERE c.position = ?
		GROUP BY c.id
		ORDER BY c.candidate_number ASC, COUNT(v.id) DESC, c.name`, voteField)
	rows, err := db.Query(query, position)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []CandidateResult{}
	for rows.Next() {
		var r CandidateResult
		if err := rows.Scan(&r.CandidateNumber, &r.Name, &r.Count); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

func AdminCandidatesHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		rows, err := db.Query("SELECT id, COALESCE(candidate_number, 1), name, class_name, photo_url, vision, mission, program, position, COALESCE(video_url, '') FROM candidates ORDER BY position, candidate_number ASC, id ASC")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memuat daftar kandidat")
		}
		defer rows.Close()

		candidates := []CandidateView{}
		for rows.Next() {
			var candidate CandidateView
			if err := rows.Scan(&candidate.ID, &candidate.CandidateNumber, &candidate.Name, &candidate.ClassName, &candidate.PhotoURL, &candidate.Vision, &candidate.Mission, &candidate.Program, &candidate.Position, &candidate.VideoURL); err != nil {
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
		if idStr == "" {
			idStr = c.Param("id")
		}
		form := AdminCandidateFormData{
			AdminLayoutData: adminLayout("Tambah Kandidat | OSIS Admin", "Tambah Kandidat", "Create or edit candidate profiles.", "admin_candidate_form_content", "candidates"),
			CandidateNumber: 1,
			Action:          "Tambah Kandidat",
		}
		if idStr != "" {
			id, err := strconv.Atoi(idStr)
			if err == nil {
				var photoURL, videoURL string
				_ = db.QueryRow("SELECT id, COALESCE(candidate_number, 1), name, class_name, photo_url, vision, mission, program, position, COALESCE(video_url, '') FROM candidates WHERE id = ?", id).Scan(
					&form.ID, &form.CandidateNumber, &form.Name, &form.ClassName, &photoURL, &form.Vision, &form.Mission, &form.Program, &form.Position, &videoURL)
				form.PhotoURL = photoURL
				form.VideoURL = videoURL
				form.Action = "Simpan Perubahan"
				form.AdminLayoutData = adminLayout("Edit Kandidat | OSIS Admin", "Edit Kandidat", "Create or edit candidate profiles.", "admin_candidate_form_content", "candidates")
			}
		}
		return c.Render(http.StatusOK, "admin_candidate_form.html", form)
	}
}

func AdminCandidateCreateHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		name := c.FormValue("name")
		className := c.FormValue("class_name")
		vision := c.FormValue("vision")
		mission := c.FormValue("mission")
		program := c.FormValue("program")
		position := c.FormValue("position")
		photoURL := c.FormValue("photo_url")
		videoURL := strings.TrimSpace(c.FormValue("video_url"))
		numberStr := c.FormValue("candidate_number")
		candidateNumber, _ := strconv.Atoi(numberStr)
		if candidateNumber < 1 || candidateNumber > 3 {
			candidateNumber = 1
		}

		if name == "" || className == "" || position == "" {
			return c.String(http.StatusBadRequest, "Nama, kelas, dan posisi wajib diisi")
		}

		if file, err := c.FormFile("photo"); err == nil && file != nil {
			uploaded, err := saveCandidatePhoto(file)
			if err == nil {
				photoURL = uploaded
			}
		}
		photoURL = normalizeCandidatePhotoURL(photoURL)

		_, err := db.Exec("INSERT INTO candidates (candidate_number, name, class_name, photo_url, video_url, vision, mission, program, position) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", candidateNumber, name, className, photoURL, videoURL, vision, mission, program, position)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal menyimpan kandidat")
		}
		return c.Redirect(http.StatusSeeOther, "/admin/candidates")
	}
}

func AdminCandidateUpdateHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			return c.String(http.StatusBadRequest, "ID kandidat tidak valid")
		}
		name := c.FormValue("name")
		className := c.FormValue("class_name")
		vision := c.FormValue("vision")
		mission := c.FormValue("mission")
		program := c.FormValue("program")
		position := c.FormValue("position")
		photoURL := c.FormValue("photo_url")
		videoURL := strings.TrimSpace(c.FormValue("video_url"))
		numberStr := c.FormValue("candidate_number")
		candidateNumber, _ := strconv.Atoi(numberStr)
		if candidateNumber < 1 || candidateNumber > 3 {
			candidateNumber = 1
		}

		if name == "" || className == "" || position == "" {
			return c.String(http.StatusBadRequest, "Nama, kelas, dan posisi wajib diisi")
		}

		if file, err := c.FormFile("photo"); err == nil && file != nil {
			uploaded, err := saveCandidatePhoto(file)
			if err == nil {
				photoURL = uploaded
			}
		}
		photoURL = normalizeCandidatePhotoURL(photoURL)

		_, err = db.Exec("UPDATE candidates SET candidate_number = ?, name = ?, class_name = ?, photo_url = ?, video_url = ?, vision = ?, mission = ?, program = ?, position = ? WHERE id = ?", candidateNumber, name, className, photoURL, videoURL, vision, mission, program, position, id)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memperbarui kandidat")
		}
		return c.Redirect(http.StatusSeeOther, "/admin/candidates")
	}
}

func saveCandidatePhoto(file *multipart.FileHeader) (string, error) {
	const maxPhotoSize = 2 * 1024 * 1024 // 2MB
	if file.Size > maxPhotoSize {
		return "", fmt.Errorf("ukuran file melebihi batas 2MB")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		return "", fmt.Errorf("format file tidak didukung: %s (hanya .jpg, .jpeg, .png, .webp)", ext)
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Sniff MIME type from the first 512 bytes
	buf := make([]byte, 512)
	n, err := src.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	mimeType := http.DetectContentType(buf[:n])
	if !strings.HasPrefix(mimeType, "image/") {
		return "", fmt.Errorf("file bukan gambar yang valid (tipe: %s)", mimeType)
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s%s", uuid.NewString(), ext)
	uploadsDir := findUploadsDir()
	destPath := filepath.Join(uploadsDir, filename)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return "", err
	}
	out, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		return "", err
	}
	return "/static/uploads/" + filename, nil
}

func findUploadsDir() string {
	exeDir, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exeDir), "public", "uploads")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	return filepath.Join("public", "uploads")
}

func normalizeCandidatePhotoURL(photoURL string) string {
	photoURL = strings.TrimSpace(photoURL)
	if photoURL == "" {
		return ""
	}
	if strings.HasPrefix(photoURL, "http://") || strings.HasPrefix(photoURL, "https://") || strings.HasPrefix(photoURL, "/") {
		return photoURL
	}
	photoURL = strings.TrimLeft(photoURL, "./")
	if strings.HasPrefix(photoURL, "uploads/") {
		return "/static/" + photoURL
	}
	return "/static/uploads/" + photoURL
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

func AdminCreditsHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		selectedPeriod := strings.TrimSpace(c.QueryParam("period"))

		// Get distinct periods
		periodRows, _ := db.Query("SELECT DISTINCT period FROM credits WHERE period != '' ORDER BY period DESC")
		periods := []string{}
		if periodRows != nil {
			defer periodRows.Close()
			for periodRows.Next() {
				var p string
				if err := periodRows.Scan(&p); err == nil && p != "" {
					periods = append(periods, p)
				}
			}
		}

		query := "SELECT id, name, class_name, division, COALESCE(photo_url, ''), COALESCE(period, ''), COALESCE(order_num, 1) FROM credits"
		var args []interface{}
		if selectedPeriod != "" {
			query += " WHERE period = ?"
			args = append(args, selectedPeriod)
		}
		query += " ORDER BY order_num ASC, id ASC"

		rows, err := db.Query(query, args...)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memuat daftar demisioner")
		}
		defer rows.Close()

		credits := []CreditView{}
		for rows.Next() {
			var cr CreditView
			if err := rows.Scan(&cr.ID, &cr.Name, &cr.ClassName, &cr.Division, &cr.PhotoURL, &cr.Period, &cr.OrderNum); err != nil {
				return c.String(http.StatusInternalServerError, "Gagal memproses data demisioner")
			}
			credits = append(credits, cr)
		}

		return c.Render(http.StatusOK, "admin_credits.html", AdminCreditsData{
			AdminLayoutData: adminLayout("Demisioner OSIS | OSIS Admin", "Kredit Demisioner OSIS", "Kelola daftar pengurus OSIS periode sebelumnya untuk apresiasi dan dokumentasi.", "admin_credits_content", "credits"),
			Credits:         credits,
			Periods:         periods,
			SelectedPeriod: selectedPeriod,
		})
	}
}

func AdminCreditFormHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		idStr := c.QueryParam("id")
		if idStr == "" {
			idStr = c.Param("id")
		}
		form := AdminCreditFormData{
			AdminLayoutData: adminLayout("Tambah Demisioner | OSIS Admin", "Tambah Pengurus Demisioner", "Tambahkan profil anggota OSIS dari kepengurusan periode sebelumnya.", "admin_credit_form_content", "credits"),
			OrderNum:        1,
			Action:          "Tambah Anggota",
		}
		if idStr != "" {
			id, err := strconv.Atoi(idStr)
			if err == nil {
				var photoURL, period string
				_ = db.QueryRow("SELECT id, name, class_name, division, COALESCE(photo_url, ''), COALESCE(period, ''), COALESCE(order_num, 1) FROM credits WHERE id = ?", id).Scan(
					&form.ID, &form.Name, &form.ClassName, &form.Division, &photoURL, &period, &form.OrderNum)
				form.PhotoURL = photoURL
				form.Period = period
				form.Action = "Simpan Perubahan"
				form.AdminLayoutData = adminLayout("Edit Demisioner | OSIS Admin", "Edit Pengurus Demisioner", "Perbarui profil anggota OSIS dari kepengurusan periode sebelumnya.", "admin_credit_form_content", "credits")
			}
		}
		return c.Render(http.StatusOK, "admin_credit_form.html", form)
	}
}

func AdminCreditCreateHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		name := strings.TrimSpace(c.FormValue("name"))
		className := strings.TrimSpace(c.FormValue("class_name"))
		division := strings.TrimSpace(c.FormValue("division"))
		period := strings.TrimSpace(c.FormValue("period"))
		orderNumStr := c.FormValue("order_num")
		orderNum, _ := strconv.Atoi(orderNumStr)
		if orderNum <= 0 {
			orderNum = 1
		}
		photoURL := strings.TrimSpace(c.FormValue("photo_url"))

		if name == "" || className == "" || division == "" {
			return c.String(http.StatusBadRequest, "Nama, kelas, dan divisi/jabatan wajib diisi")
		}

		if file, err := c.FormFile("photo"); err == nil && file != nil {
			uploaded, err := saveCandidatePhoto(file)
			if err == nil {
				photoURL = uploaded
			}
		}
		photoURL = normalizeCandidatePhotoURL(photoURL)

		_, err := db.Exec("INSERT INTO credits (name, class_name, division, photo_url, period, order_num) VALUES (?, ?, ?, ?, ?, ?)", name, className, division, photoURL, period, orderNum)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal menyimpan data demisioner: "+err.Error())
		}
		return c.Redirect(http.StatusSeeOther, "/admin/credits")
	}
}

func AdminCreditUpdateHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			return c.String(http.StatusBadRequest, "ID demisioner tidak valid")
		}
		name := strings.TrimSpace(c.FormValue("name"))
		className := strings.TrimSpace(c.FormValue("class_name"))
		division := strings.TrimSpace(c.FormValue("division"))
		period := strings.TrimSpace(c.FormValue("period"))
		orderNumStr := c.FormValue("order_num")
		orderNum, _ := strconv.Atoi(orderNumStr)
		if orderNum <= 0 {
			orderNum = 1
		}
		photoURL := strings.TrimSpace(c.FormValue("photo_url"))

		if name == "" || className == "" || division == "" {
			return c.String(http.StatusBadRequest, "Nama, kelas, dan divisi/jabatan wajib diisi")
		}

		if file, err := c.FormFile("photo"); err == nil && file != nil {
			uploaded, err := saveCandidatePhoto(file)
			if err == nil {
				photoURL = uploaded
			}
		}
		photoURL = normalizeCandidatePhotoURL(photoURL)

		_, err = db.Exec("UPDATE credits SET name = ?, class_name = ?, division = ?, photo_url = ?, period = ?, order_num = ? WHERE id = ?", name, className, division, photoURL, period, orderNum, id)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memperbarui data demisioner: "+err.Error())
		}
		return c.Redirect(http.StatusSeeOther, "/admin/credits")
	}
}

func AdminCreditDeleteHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			return c.String(http.StatusBadRequest, "ID demisioner tidak valid")
		}
		_, err = db.Exec("DELETE FROM credits WHERE id = ?", id)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal menghapus data demisioner")
		}
		return c.Redirect(http.StatusSeeOther, "/admin/credits")
	}
}

func AdminVotersHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		rows, err := db.Query("SELECT id, uuid, name, class_name, phone_number, has_voted, COALESCE(presence_status, 0) as presence_status FROM voters ORDER BY id DESC")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memuat daftar pemilih")
		}
		defer rows.Close()

		voters := []VoterView{}
		for rows.Next() {
			var v VoterView
			var hasVoted, presenceStatus int
			if err := rows.Scan(&v.ID, &v.UUID, &v.Name, &v.ClassName, &v.PhoneNumber, &hasVoted, &presenceStatus); err != nil {
				return c.String(http.StatusInternalServerError, "Gagal memproses voters")
			}
			if hasVoted == 1 {
				v.HasVoted = "Sudah"
			} else {
				v.HasVoted = "Belum"
			}
			v.PresenceStatus = presenceStatus
			voters = append(voters, v)
		}
		message := c.QueryParam("import_status")
		return c.Render(http.StatusOK, "admin_voters.html", AdminVotersData{
			AdminLayoutData: adminLayout("Voters Management | OSIS Admin", "Voters Management", "Manage institutional electorate records.", "admin_voters_content", "voters"),
			Voters:          voters,
			ImportMessage:   message,
		})
	}
}

func AdminVoterCardsHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		rows, err := db.Query("SELECT id, uuid, name, class_name, phone_number, has_voted FROM voters ORDER BY id ASC")
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memuat data pemilih untuk kartu")
		}
		defer rows.Close()

		voters := []VoterView{}
		for rows.Next() {
			var v VoterView
			var hasVoted int
			if err := rows.Scan(&v.ID, &v.UUID, &v.Name, &v.ClassName, &v.PhoneNumber, &hasVoted); err != nil {
				return c.String(http.StatusInternalServerError, "Gagal memproses data pemilih")
			}
			if hasVoted == 1 {
				v.HasVoted = "Sudah"
			} else {
				v.HasVoted = "Belum"
			}
			voters = append(voters, v)
		}
		return c.Render(http.StatusOK, "admin_voters_cards.html", AdminVotersData{
			AdminLayoutData: adminLayout("Print Voter Cards | OSIS Admin", "Print Voter Cards", "Print eight voter cards per A4 page with QR and barcode.", "admin_voters_cards_content", "voters"),
			Voters:          voters,
		})
	}
}

func AdminVoterFormHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		data := AdminVoterFormData{
			AdminLayoutData: adminLayout("Tambah Pemilih | OSIS Admin", "Tambah Pemilih", "Tambah data pemilih baru ke DPT.", "admin_voter_form_content", "voters"),
			IsEdit:          false,
		}
		return c.Render(http.StatusOK, "admin_voter_form.html", data)
	}
}

func AdminVoterEditHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			return c.String(http.StatusBadRequest, "ID tidak valid")
		}

		var v VoterFormData
		err = db.QueryRow("SELECT id, uuid, name, class_name, phone_number FROM voters WHERE id = ?", id).Scan(&v.ID, &v.UUID, &v.Name, &v.ClassName, &v.PhoneNumber)
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Pemilih tidak ditemukan")
		} else if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memuat data")
		}

		data := AdminVoterFormData{
			AdminLayoutData: adminLayout("Edit Pemilih | OSIS Admin", "Edit Pemilih", "Edit data pemilih.", "admin_voter_form_content", "voters"),
			IsEdit:          true,
			ID:              v.ID,
			UUID:            v.UUID,
			Name:            v.Name,
			ClassName:       v.ClassName,
			PhoneNumber:     v.PhoneNumber,
		}
		return c.Render(http.StatusOK, "admin_voter_form.html", data)
	}
}

func AdminVoterImportPageHandler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		return c.Render(http.StatusOK, "admin_voters_import.html", AdminVoterFormData{
			AdminLayoutData: adminLayout("Import Voters | OSIS Admin", "Import Pemilih", "Upload file Excel atau CSV untuk menambahkan data pemilih massal.", "admin_voters_import_content", "voters"),
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

func AdminVoterUpdateHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		id, err := strconv.Atoi(c.FormValue("id"))
		if err != nil || id <= 0 {
			return c.String(http.StatusBadRequest, "ID tidak valid")
		}
		name := c.FormValue("name")
		className := c.FormValue("class_name")
		phoneNumber := c.FormValue("phone_number")
		if name == "" || className == "" || phoneNumber == "" {
			return c.String(http.StatusBadRequest, "Semua field wajib diisi")
		}
		_, err = db.Exec("UPDATE voters SET name = ?, class_name = ?, phone_number = ? WHERE id = ?", name, className, phoneNumber, id)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memperbarui pemilih")
		}
		return c.Redirect(http.StatusSeeOther, "/admin/voters")
	}
}

func AdminVotersBulkDeleteHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ids := c.FormValue("ids")
		if ids == "" {
			return c.String(http.StatusBadRequest, "Pilih minimal satu pemilih")
		}

		idList := strings.Split(ids, ",")
		for _, idStr := range idList {
			id, err := strconv.Atoi(strings.TrimSpace(idStr))
			if err == nil && id > 0 {
				db.Exec("DELETE FROM voters WHERE id = ?", id)
			}
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

type VoterImportRow struct {
	UUID        string
	Name        string
	ClassName   string
	PhoneNumber string
}

func AdminVoterImportHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		formFile, err := c.FormFile("import_file")
		if err != nil || formFile == nil {
			return c.Redirect(http.StatusSeeOther, "/admin/voters?import_status="+url.QueryEscape("Pilih file Excel atau CSV untuk diimpor."))
		}

		const maxImportFileSize = 10 * 1024 * 1024 // 10MB
		if formFile.Size > maxImportFileSize {
			return c.Redirect(http.StatusSeeOther, "/admin/voters?import_status="+url.QueryEscape("Ukuran file melebihi batas maksimal 10MB."))
		}

		file, err := formFile.Open()
		if err != nil {
			return c.Redirect(http.StatusSeeOther, "/admin/voters?import_status="+url.QueryEscape("Gagal membuka file impor."))
		}
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(formFile.Filename))
		var voters []VoterImportRow
		switch ext {
		case ".csv":
			voters, err = parseVoterCSV(file)
		case ".xlsx":
			voters, err = parseVoterXLSX(file)
		default:
			err = fmt.Errorf("format file tidak didukung")
		}
		if err != nil {
			return c.Redirect(http.StatusSeeOther, "/admin/voters?import_status="+url.QueryEscape("Gagal mengimpor file: "+err.Error()))
		}
		if len(voters) == 0 {
			return c.Redirect(http.StatusSeeOther, "/admin/voters?import_status="+url.QueryEscape("Tidak ada data pemilih valid yang ditemukan dalam file."))
		}

		tx, err := db.Begin()
		if err != nil {
			return c.Redirect(http.StatusSeeOther, "/admin/voters?import_status="+url.QueryEscape("Gagal memulai transaksi database."))
		}
		inserted := 0
		skipped := 0
		for _, row := range voters {
			if row.UUID == "" {
				row.UUID = uuid.NewString()
			}
			if _, err := uuid.Parse(row.UUID); err != nil {
				row.UUID = uuid.NewString()
			}
			var count int
			err = tx.QueryRow("SELECT COUNT(*) FROM voters WHERE uuid = ?", row.UUID).Scan(&count)
			if err != nil {
				tx.Rollback()
				return c.Redirect(http.StatusSeeOther, "/admin/voters?import_status="+url.QueryEscape("Gagal memeriksa duplikat UUID."))
			}
			if count > 0 {
				skipped++
				continue
			}
			_, err = tx.Exec("INSERT INTO voters (uuid, name, class_name, phone_number) VALUES (?, ?, ?, ?)", row.UUID, row.Name, row.ClassName, row.PhoneNumber)
			if err != nil {
				tx.Rollback()
				return c.Redirect(http.StatusSeeOther, "/admin/voters?import_status="+url.QueryEscape("Gagal menyimpan data pemilih: "+err.Error()))
			}
			inserted++
		}
		if err := tx.Commit(); err != nil {
			return c.Redirect(http.StatusSeeOther, "/admin/voters?import_status="+url.QueryEscape("Gagal menyelesaikan impor pemilih."))
		}

		message := fmt.Sprintf("%d pemilih berhasil diimpor. %d baris diabaikan karena duplikat atau tidak lengkap.", inserted, skipped)
		return c.Redirect(http.StatusSeeOther, "/admin/voters?import_status="+url.QueryEscape(message))
	}
}

func parseVoterCSV(reader io.Reader) ([]VoterImportRow, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true
	csvReader.FieldsPerRecord = -1
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	return parseVoterRecords(records)
}

func parseVoterXLSX(reader io.Reader) ([]VoterImportRow, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	sharedStrings, err := readXLSXSharedStrings(zipReader)
	if err != nil {
		return nil, err
	}
	records, err := readXLSXWorksheet(zipReader, sharedStrings)
	if err != nil {
		return nil, err
	}
	return parseVoterRecords(records)
}

func readXLSXSharedStrings(zipReader *zip.Reader) ([]string, error) {
	file, err := zipReader.Open("xl/sharedStrings.xml")
	if err != nil {
		return nil, nil
	}
	defer file.Close()
	var result []string
	decoder := xml.NewDecoder(file)
	var insideSI bool
	var textBuilder strings.Builder
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch v := tok.(type) {
		case xml.StartElement:
			if v.Name.Local == "si" {
				insideSI = true
				textBuilder.Reset()
			}
			if insideSI && v.Name.Local == "t" {
				var text string
				if err := decoder.DecodeElement(&text, &v); err != nil {
					return nil, err
				}
				textBuilder.WriteString(text)
			}
		case xml.EndElement:
			if v.Name.Local == "si" && insideSI {
				result = append(result, textBuilder.String())
				insideSI = false
			}
		}
	}
	return result, nil
}

func readXLSXWorksheet(zipReader *zip.Reader, sharedStrings []string) ([][]string, error) {
	file, err := zipReader.Open("xl/worksheets/sheet1.xml")
	if err != nil {
		return nil, fmt.Errorf("file worksheet tidak ditemukan")
	}
	defer file.Close()
	decoder := xml.NewDecoder(file)
	var records [][]string
	var currentRow []string
	var currentCellRef, currentCellType string
	var inValue bool
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch v := tok.(type) {
		case xml.StartElement:
			switch v.Name.Local {
			case "c":
				currentCellRef = ""
				currentCellType = ""
				for _, attr := range v.Attr {
					switch attr.Name.Local {
					case "r":
						currentCellRef = attr.Value
					case "t":
						currentCellType = attr.Value
					}
				}
			case "v":
				inValue = true
			}
		case xml.CharData:
			if inValue {
				value := strings.TrimSpace(string(v))
				colIndex := cellColumnIndex(currentCellRef)
				for len(currentRow) <= colIndex {
					currentRow = append(currentRow, "")
				}
				if currentCellType == "s" {
					idx, err := strconv.Atoi(value)
					if err == nil && idx >= 0 && idx < len(sharedStrings) {
						currentRow[colIndex] = sharedStrings[idx]
					} else {
						currentRow[colIndex] = value
					}
				} else {
					currentRow[colIndex] = value
				}
			}
		case xml.EndElement:
			switch v.Name.Local {
			case "v":
				inValue = false
			case "c":
				currentCellRef = ""
				currentCellType = ""
			case "row":
				if len(currentRow) > 0 {
					records = append(records, currentRow)
				}
				currentRow = nil
			}
		}
	}
	return records, nil
}

func cellColumnIndex(cellRef string) int {
	for i, r := range cellRef {
		if r >= 'A' && r <= 'Z' {
			continue
		}
		return colLettersToIndex(cellRef[:i])
	}
	return colLettersToIndex(cellRef)
}

func colLettersToIndex(colLetters string) int {
	index := 0
	for _, r := range colLetters {
		if r < 'A' || r > 'Z' {
			break
		}
		index = index*26 + int(r-'A'+1)
	}
	return index - 1
}

func parseVoterRecords(records [][]string) ([]VoterImportRow, error) {
	if len(records) == 0 {
		return nil, nil
	}
	headers := records[0]
	lowerHeaders := make([]string, len(headers))
	for i, h := range headers {
		lowerHeaders[i] = strings.ToLower(strings.TrimSpace(h))
	}
	fieldNames := map[string]bool{
		"uuid":          true,
		"id":            true,
		"name":          true,
		"nama":          true,
		"full name":     true,
		"nama pemilih":  true,
		"class":         true,
		"class_name":    true,
		"class name":    true,
		"kelas":         true,
		"phone":         true,
		"phone_number":  true,
		"phone number":  true,
		"telepon":       true,
		"nomor hp":      true,
		"nomor telepon": true,
		"hp":            true,
	}
	hasHeader := false
	countHeaderFields := 0
	for _, h := range lowerHeaders {
		if fieldNames[h] {
			countHeaderFields++
		}
	}
	if countHeaderFields >= 2 {
		hasHeader = true
	}
	if hasHeader {
		records = records[1:]
	}

	parsed := make([]VoterImportRow, 0, len(records))
	for _, cols := range records {
		if len(cols) == 0 {
			continue
		}
		for i := range cols {
			cols[i] = strings.TrimSpace(cols[i])
		}
		row := VoterImportRow{}
		if hasHeader {
			for i, h := range lowerHeaders {
				if i >= len(cols) {
					continue
				}
				switch h {
				case "uuid", "id":
					row.UUID = cols[i]
				case "name", "nama":
					row.Name = cols[i]
				case "full name":
					row.Name = cols[i]
				case "class", "class_name", "kelas":
					row.ClassName = cols[i]
				case "class name":
					row.ClassName = cols[i]
				case "phone", "phone_number", "telepon", "hp":
					row.PhoneNumber = cols[i]
				case "phone number", "nomor hp", "nomor telepon":
					row.PhoneNumber = cols[i]
				}
			}
		} else {
			if len(cols) >= 4 {
				if _, err := uuid.Parse(cols[0]); err == nil {
					row.UUID = cols[0]
					row.Name = cols[1]
					row.ClassName = cols[2]
					row.PhoneNumber = cols[3]
				} else {
					row.Name = cols[0]
					row.ClassName = cols[1]
					row.PhoneNumber = cols[2]
					row.UUID = cols[3]
				}
			} else if len(cols) == 3 {
				row.Name = cols[0]
				row.ClassName = cols[1]
				row.PhoneNumber = cols[2]
			} else {
				continue
			}
		}
		if row.Name == "" || row.ClassName == "" || row.PhoneNumber == "" {
			continue
		}
		if row.UUID == "" {
			row.UUID = uuid.NewString()
		} else if _, err := uuid.Parse(row.UUID); err != nil {
			row.UUID = uuid.NewString()
		}
		parsed = append(parsed, row)
	}
	return parsed, nil
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
		var waEnabled int
		var waProvider, waURL, waDeviceID, waUsername, waPassword, waTemplate, onesenderKey string
		var gdriveEnabled, gdriveDeleteLocal, testimAutoApprove, testimEnabled int
		var gdriveFolderID, gdriveSAJSON string

		_ = db.QueryRow(`SELECT 
			COALESCE(announcement_text, ''),
			COALESCE(wa_enabled, onesender_enabled, 0),
			COALESCE(wa_provider, 'gowa'),
			COALESCE(NULLIF(wa_api_url, ''), NULLIF(onesender_api_url, ''), 'http://127.0.0.1:8054/api/whatsapp/send'),
			COALESCE(NULLIF(wa_device_id, ''), 'Pionir'),
			COALESCE(NULLIF(wa_username, ''), 'admin'),
			COALESCE(NULLIF(wa_password, ''), 'PutihAbu123!'),
			COALESCE(NULLIF(wa_template, ''), NULLIF(onesender_template, ''), ''),
			COALESCE(onesender_api_key, ''),
			COALESCE(gdrive_enabled, 0),
			COALESCE(gdrive_folder_id, ''),
			COALESCE(gdrive_service_account_json, ''),
			COALESCE(gdrive_delete_local, 0),
			COALESCE(testimonials_auto_approve, 1),
			COALESCE(testimonials_enabled, 1)
		FROM settings ORDER BY id DESC LIMIT 1`).Scan(
			&announcement, &waEnabled, &waProvider, &waURL, &waDeviceID, &waUsername, &waPassword, &waTemplate, &onesenderKey,
			&gdriveEnabled, &gdriveFolderID, &gdriveSAJSON, &gdriveDeleteLocal, &testimAutoApprove, &testimEnabled,
		)

		var clientEmail string
		if gdriveSAJSON != "" {
			var sa struct {
				ClientEmail string `json:"client_email"`
			}
			if err := json.Unmarshal([]byte(gdriveSAJSON), &sa); err == nil {
				clientEmail = sa.ClientEmail
			}
		}

		data := AdminSettingsData{
			AdminLayoutData:          adminLayout("Pengaturan | OSIS Admin", "Pengaturan Sistem", "Atur pengumuman, integrasi WhatsApp Gateway (GOWA), dan Google Drive Kiosk.", "admin_settings_content", "settings"),
			Announcement:             announcement,
			OneSenderEnabled:         waEnabled == 1,
			OneSenderURL:             waURL,
			OneSenderKey:             onesenderKey,
			OneSenderTemplate:        waTemplate,
			WAEnabled:                waEnabled == 1,
			WAProvider:               waProvider,
			WAURL:                    waURL,
			WADeviceID:               waDeviceID,
			WAUsername:               waUsername,
			WAPassword:               waPassword,
			WATemplate:               waTemplate,
			GDriveEnabled:            gdriveEnabled == 1,
			GDriveFolderID:           gdriveFolderID,
			GDriveServiceAccountJSON: gdriveSAJSON,
			GDriveClientEmail:        clientEmail,
			GDriveDeleteLocal:        gdriveDeleteLocal == 1,
			TestimonialsAutoApprove:  testimAutoApprove == 1,
			TestimonialsEnabled:      testimEnabled == 1,
		}
		return c.Render(http.StatusOK, "admin_settings.html", data)
	}
}

func AdminSettingsSaveHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		announcement := c.FormValue("announcement")
		waEnabled := c.FormValue("wa_enabled") == "1" || c.FormValue("onesender_enabled") == "1"
		waProvider := c.FormValue("wa_provider")
		if waProvider == "" {
			waProvider = "gowa"
		}
		waURL := strings.TrimSpace(c.FormValue("wa_url"))
		if waURL == "" {
			waURL = strings.TrimSpace(c.FormValue("onesender_url"))
		}
		if waURL == "" {
			waURL = "http://127.0.0.1:8054/api/whatsapp/send"
		}
		waDeviceID := strings.TrimSpace(c.FormValue("wa_device_id"))
		if waDeviceID == "" {
			waDeviceID = "Pionir"
		}
		waUsername := strings.TrimSpace(c.FormValue("wa_username"))
		if waUsername == "" {
			waUsername = "admin"
		}
		waPassword := strings.TrimSpace(c.FormValue("wa_password"))
		if waPassword == "" {
			waPassword = "PutihAbu123!"
		}
		waTemplate := strings.TrimSpace(c.FormValue("wa_template"))
		if waTemplate == "" {
			waTemplate = strings.TrimSpace(c.FormValue("onesender_template"))
		}
		apiKey := strings.TrimSpace(c.FormValue("onesender_key"))

		gdriveEnabled := 0
		if c.FormValue("gdrive_enabled") == "1" {
			gdriveEnabled = 1
		}
		gdriveFolderID := services.SanitizeFolderID(c.FormValue("gdrive_folder_id"))
		gdriveSAJSON := strings.TrimSpace(c.FormValue("gdrive_service_account_json"))
		gdriveDeleteLocal := 0
		if c.FormValue("gdrive_delete_local") == "1" {
			gdriveDeleteLocal = 1
		}
		testimAutoApprove := 0
		if c.FormValue("testimonials_auto_approve") == "1" {
			testimAutoApprove = 1
		}
		testimEnabled := 0
		if c.FormValue("testimonials_enabled") == "1" {
			testimEnabled = 1
		}

		enabledVal := 0
		if waEnabled {
			enabledVal = 1
		}

		var existingID int
		err := db.QueryRow("SELECT id FROM settings ORDER BY id DESC LIMIT 1").Scan(&existingID)

		if err == sql.ErrNoRows {
			_, err = db.Exec(`INSERT INTO settings (
				announcement_text, onesender_enabled, onesender_api_url, onesender_api_key, onesender_template,
				wa_provider, wa_enabled, wa_api_url, wa_device_id, wa_username, wa_password, wa_template,
				gdrive_enabled, gdrive_folder_id, gdrive_service_account_json, gdrive_delete_local,
				testimonials_auto_approve, testimonials_enabled
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				announcement, enabledVal, waURL, apiKey, waTemplate,
				waProvider, enabledVal, waURL, waDeviceID, waUsername, waPassword, waTemplate,
				gdriveEnabled, gdriveFolderID, gdriveSAJSON, gdriveDeleteLocal,
				testimAutoApprove, testimEnabled)
		} else if err == nil {
			_, err = db.Exec(`UPDATE settings SET 
				announcement_text = ?, 
				onesender_enabled = ?, onesender_api_url = ?, onesender_api_key = ?, onesender_template = ?,
				wa_provider = ?, wa_enabled = ?, wa_api_url = ?, wa_device_id = ?, wa_username = ?, wa_password = ?, wa_template = ?,
				gdrive_enabled = ?, gdrive_folder_id = ?, gdrive_service_account_json = ?, gdrive_delete_local = ?,
				testimonials_auto_approve = ?, testimonials_enabled = ?,
				updated_at = CURRENT_TIMESTAMP 
			WHERE id = ?`,
				announcement, enabledVal, waURL, apiKey, waTemplate,
				waProvider, enabledVal, waURL, waDeviceID, waUsername, waPassword, waTemplate,
				gdriveEnabled, gdriveFolderID, gdriveSAJSON, gdriveDeleteLocal,
				testimAutoApprove, testimEnabled, existingID)
		}

		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal menyimpan pengaturan: "+err.Error())
		}
		return c.Redirect(http.StatusSeeOther, "/admin/settings")
	}
}


func AdminTestMessageHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		phone := c.FormValue("phone")
		if phone == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Nomor HP wajib diisi"})
		}

		sender, err := services.NewWhatsAppClient(db)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal membuat client WhatsApp: " + err.Error()})
		}

		testMessage := fmt.Sprintf("Halo! 🙏 Ini adalah pesan uji coba dari Sistem E-Voting OSIS SMK NIBA.\n\n"+
			"Integrasi WhatsApp Gateway (GOWA / Shim) berhasil terhubung dengan baik!\n\n"+
			"Waktu kirim: %s", time.Now().Format("02 Jan 2006 15:04:05 WIB"))

		if err := sender.SendMessage(phone, testMessage); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal mengirim pesan via WhatsApp: " + err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "success", "message": "Pesan uji coba berhasil dikirim ke " + phone})
	}
}

func AdminBroadcastReminderHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		target := c.FormValue("target")
		customMsg := strings.TrimSpace(c.FormValue("message"))

		sender, err := services.NewWhatsAppClient(db)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal inisialisasi client WhatsApp: " + err.Error()})
		}

		query := "SELECT name, class_name, phone_number, uuid FROM voters WHERE phone_number != ''"
		if target == "not_voted" {
			query += " AND has_voted = 0"
		} else if target == "not_present" {
			query += " AND COALESCE(presence_status, 0) = 0"
		} else if target == "teachers" {
			query += " AND (UPPER(class_name) LIKE '%GURU%' OR UPPER(class_name) LIKE '%STAF%' OR UPPER(class_name) LIKE '%STAFF%' OR UPPER(class_name) LIKE '%SATPAM%')"
		} else if target == "teachers_not_voted" {
			query += " AND has_voted = 0 AND (UPPER(class_name) LIKE '%GURU%' OR UPPER(class_name) LIKE '%STAF%' OR UPPER(class_name) LIKE '%STAFF%' OR UPPER(class_name) LIKE '%SATPAM%')"
		} else if target == "students" {
			query += " AND NOT (UPPER(class_name) LIKE '%GURU%' OR UPPER(class_name) LIKE '%STAF%' OR UPPER(class_name) LIKE '%STAFF%' OR UPPER(class_name) LIKE '%SATPAM%')"
		} else if target == "students_not_voted" {
			query += " AND has_voted = 0 AND NOT (UPPER(class_name) LIKE '%GURU%' OR UPPER(class_name) LIKE '%STAF%' OR UPPER(class_name) LIKE '%STAFF%' OR UPPER(class_name) LIKE '%SATPAM%')"
		}

		rows, err := db.Query(query)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal mengambil data pemilih: " + err.Error()})
		}
		defer rows.Close()

		type recipient struct {
			Name      string
			ClassName string
			Phone     string
			UUID      string
		}

		var recipients []recipient
		for rows.Next() {
			var r recipient
			if err := rows.Scan(&r.Name, &r.ClassName, &r.Phone, &r.UUID); err == nil {
				recipients = append(recipients, r)
			}
		}

		if len(recipients) == 0 {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"status":  "empty",
				"message": "Tidak ada pemilih dengan nomor telepon yang memenuhi kriteria target broadcast.",
				"count":   0,
			})
		}

		// Asynchronous background delivery with gentle rate limiting
		go func(list []recipient, templateMsg string, client *services.WhatsAppClient) {
			for _, rec := range list {
				isTeacher := services.IsTeacherOrStaff(rec.ClassName)
				sapaan, yth, panggilan := services.GetSalutation(rec.ClassName)

				msg := templateMsg
				if msg == "" {
					if isTeacher {
						msg = fmt.Sprintf("Yth. Bapak/Ibu %s 🗳️\n\n"+
							"Dengan hormat, kami dari Panitia Pemilihan OSIS SMK NIBA Business School menginformasikan bahwa pemungutan suara digital sedang berlangsung.\n\n"+
							"Kami memohon kesediaan Bapak/Ibu untuk dapat hadir dan menyalurkan hak suara di bilik TPS.\n"+
							"Dukungan dan partisipasi Bapak/Ibu sangat berarti bagi kesuksesan pesta demokrasi serta kemajuan sekolah kita. ✨\n\n"+
							"Salam hormat,\nPanitia Pilketos SMK NIBA", rec.Name)
					} else {
						msg = fmt.Sprintf("Halo %s (%s)! 🗳️\n\n"+
							"Mengingatkan bahwa Pemilihan Ketua & Wakil Ketua OSIS SMK NIBA Business School sedang berlangsung.\n\n"+
							"Gunakan hak suara kamu di bilik TPS sekarang juga.\n"+
							"Suara kamu menentukan masa depan sekolah kita! ✨\n\n"+
							"- Panitia Pilketos SMK NIBA", rec.Name, rec.ClassName)
					}
				} else {
					msg = strings.ReplaceAll(msg, "{sapaan}", sapaan)
					msg = strings.ReplaceAll(msg, "{yth}", yth)
					msg = strings.ReplaceAll(msg, "{panggilan}", panggilan)
					msg = strings.ReplaceAll(msg, "{nama}", rec.Name)
					msg = strings.ReplaceAll(msg, "{kelas}", rec.ClassName)
					msg = strings.ReplaceAll(msg, "{uuid}", rec.UUID)
				}

				_ = client.SendMessage(rec.Phone, msg)
				time.Sleep(350 * time.Millisecond) // Friendly delay between messages
			}
		}(recipients, customMsg, sender)

		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("Broadcast pengingat sedang diproses untuk %d pemilih secara bertahap.", len(recipients)),
			"count":   len(recipients),
		})
	}
}

func AdminResetVotesHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if c.FormValue("confirm") != "1" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Konfirmasi diperlukan"})
		}

		_, err := db.Exec("DELETE FROM votes")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal menghapus suara"})
		}

		_, err = db.Exec("UPDATE voters SET has_voted = 0")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal mereset status voter"})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "success", "message": "Semua suara berhasil direset"})
	}
}

func AdminPresenceScannerHandler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		return c.Render(http.StatusOK, "admin_presence_scanner.html", nil)
	}
}

func AdminMarkAttendanceHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var req struct {
			UUID string `json:"uuid"`
		}
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Payload tidak valid"})
		}
		if req.UUID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "UUID wajib diisi"})
		}

		var id int
		var hasVoted int
		var presenceStatus int
		err := db.QueryRow("SELECT id, has_voted, presence_status FROM voters WHERE uuid = ?", req.UUID).Scan(&id, &hasVoted, &presenceStatus)
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Voter tidak ditemukan"})
		} else if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal query voter"})
		}

		if presenceStatus == 1 {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"status":  "already_marked",
				"message": "Voter sudah ditandai hadir sebelumnya",
				"data": map[string]interface{}{
					"name":     "",
					"class":    "",
					"presence": presenceStatus,
					"voted":    hasVoted,
				},
			})
		}

		_, err = db.Exec("UPDATE voters SET presence_status = 1, attended_at = CURRENT_TIMESTAMP WHERE uuid = ?", req.UUID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal update kehadiran"})
		}

		var name, className string
		db.QueryRow("SELECT name, class_name FROM voters WHERE uuid = ?", req.UUID).Scan(&name, &className)

		statusLabel := "Hadir"
		if hasVoted == 1 {
			statusLabel = "Sudah"
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  "success",
			"message": "Kehadiran berhasil ditandai",
			"data": map[string]interface{}{
				"name":     name,
				"class":    className,
				"presence": 1,
				"voted":    hasVoted,
				"status":   statusLabel,
			},
		})
	}
}

func AdminAttendanceListHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		rows, err := db.Query(`
			SELECT id, uuid, name, class_name, phone_number, has_voted, presence_status, attended_at 
			FROM voters 
			ORDER BY presence_status DESC, attended_at DESC
		`)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memuat data")
		}
		defer rows.Close()

		voters := []VoterView{}
		totalHadir := 0
		totalBelum := 0
		for rows.Next() {
			var v VoterView
			var hasVoted, presenceStatus int
			var attendedAt sql.NullTime
			if err := rows.Scan(&v.ID, &v.UUID, &v.Name, &v.ClassName, &v.PhoneNumber, &hasVoted, &presenceStatus, &attendedAt); err != nil {
				continue
			}
			if hasVoted == 1 {
				v.HasVoted = "Sudah"
			} else {
				v.HasVoted = "Belum"
			}
			v.PresenceStatus = presenceStatus
			if presenceStatus == 1 {
				totalHadir++
			} else {
				totalBelum++
			}
			if attendedAt.Valid {
				v.AttendedAt = attendedAt.Time.Format("15:04:05")
			}
			voters = append(voters, v)
		}

		return c.Render(http.StatusOK, "admin_attendance.html", AdminAttendanceData{
			AdminLayoutData: adminLayout("Kehadiran Pemilih | OSIS Admin", "Kehadiran Pemilih", "Monitoring presensi kehadiran pemilih DPT.", "admin_attendance_content", "attendance"),
			Voters:          voters,
			TotalHadir:      totalHadir,
			TotalBelum:      totalBelum,
			Total:           totalHadir + totalBelum,
		})
	}
}

func AdminBackupHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		candidates := []CandidateView{}
		rows, _ := db.Query("SELECT id, COALESCE(candidate_number, 1), name, class_name, photo_url, vision, mission, program, position FROM candidates ORDER BY position, candidate_number, id")
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var c CandidateView
				rows.Scan(&c.ID, &c.CandidateNumber, &c.Name, &c.ClassName, &c.PhotoURL, &c.Vision, &c.Mission, &c.Program, &c.Position)
				candidates = append(candidates, c)
			}
		}

		voters := []VoterView{}
		rows2, _ := db.Query("SELECT id, uuid, name, class_name, phone_number, has_voted FROM voters ORDER BY id")
		if rows2 != nil {
			defer rows2.Close()
			for rows2.Next() {
				var v VoterView
				var hasVoted int
				rows2.Scan(&v.ID, &v.UUID, &v.Name, &v.ClassName, &v.PhoneNumber, &hasVoted)
				if hasVoted == 1 {
					v.HasVoted = "Sudah"
				} else {
					v.HasVoted = "Belum"
				}
				voters = append(voters, v)
			}
		}

		allVotes := []VoteExport{}
		rows3, _ := db.Query(`
			SELECT v.id, v.masked_uuid, v.chairman_id, c1.name, v.vice_chairman_id, c2.name, v.voted_at 
			FROM votes v 
			JOIN candidates c1 ON v.chairman_id = c1.id 
			JOIN candidates c2 ON v.vice_chairman_id = c2.id 
			ORDER BY v.voted_at DESC
		`)
		if rows3 != nil {
			defer rows3.Close()
			for rows3.Next() {
				var ve VoteExport
				rows3.Scan(&ve.ID, &ve.MaskedUUID, &ve.ChairmanID, &ve.ChairmanName, &ve.ViceChairmanID, &ve.ViceChairmanName, &ve.VotedAt)
				allVotes = append(allVotes, ve)
			}
		}

		data := BackupData{
			Candidates: candidates,
			Voters:     voters,
			Votes:      allVotes,
		}

		c.Response().Header().Set("Content-Disposition", "attachment; filename=backup_evoting_"+time.Now().Format("20060102_150405")+".json")
		return c.JSON(http.StatusOK, data)
	}
}

func AdminReportHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		type ReportRow struct {
			ChairmanName string `json:"chairman_name"`
			ViceName     string `json:"vice_name"`
			VoteCount    int    `json:"vote_count"`
		}

		rows, err := db.Query(`
			SELECT c1.name, c2.name, COUNT(*) as vote_count 
			FROM votes v
			JOIN candidates c1 ON v.chairman_id = c1.id
			JOIN candidates c2 ON v.vice_chairman_id = c2.id
			GROUP BY v.chairman_id, v.vice_chairman_id
			ORDER BY vote_count DESC
		`)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal mengambil data"})
		}
		defer rows.Close()

		results := []ReportRow{}
		for rows.Next() {
			var r ReportRow
			rows.Scan(&r.ChairmanName, &r.ViceName, &r.VoteCount)
			results = append(results, r)
		}

		var totalVoters, totalVotes int
		db.QueryRow("SELECT COUNT(*) FROM voters").Scan(&totalVoters)
		db.QueryRow("SELECT COUNT(*) FROM votes").Scan(&totalVotes)

		c.Response().Header().Set("Content-Disposition", "attachment; filename=laporan_voting_"+time.Now().Format("20060102_150405")+".csv")
		c.Response().Header().Set("Content-Type", "text/csv")

		csv := "No,Kandidat Ketua,Kandidat Wakil,Jumlah Suara\n"
		for i, r := range results {
			csv += fmt.Sprintf("%d,%s,%s,%d\n", i+1, r.ChairmanName, r.ViceName, r.VoteCount)
		}
		csv += fmt.Sprintf("\nTotal Suara,%d dari %d Pemilih", totalVotes, totalVoters)

		return c.String(http.StatusOK, csv)
	}
}
