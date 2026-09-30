package handlers

import (
	"database/sql"
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
	"gopilketos/services"
)

type TestimonialView struct {
	ID           int
	Name         string
	ClassName    string
	Message      string
	VideoPath    string
	VideoType    string
	GDriveFileID string
	GDriveURL    string
	SyncStatus   string
	SyncError    string
	IsApproved   bool
	CreatedAt    string
}

type AdminTestimonialsData struct {
	AdminLayoutData
	Testimonials []TestimonialView
	TotalCount   int
	SyncedCount  int
	LocalCount   int
	FailedCount  int
	GDriveActive bool
}

type KioskPageData struct {
	Title        string
	AutoApprove  bool
	GDriveActive bool
}

// KioskPageHandler serves the interactive standing booth / kiosk recording page
func KioskPageHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		gdriveClient, _ := services.NewGDriveClient(db)
		cfg := gdriveClient.GetConfig()

		data := KioskPageData{
			Title:        "Video Booth Kesan & Pesan - Pilketos SMK NIBA",
			AutoApprove:  cfg.AutoApprove,
			GDriveActive: cfg.Enabled,
		}
		return c.Render(http.StatusOK, "kiosk.html", data)
	}
}

// UploadTestimonialHandler handles video recording uploads from kiosk or public modal
func UploadTestimonialHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		name := strings.TrimSpace(c.FormValue("name"))
		className := strings.TrimSpace(c.FormValue("class_name"))
		message := strings.TrimSpace(c.FormValue("message"))

		if name == "" || className == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Nama dan kelas wajib diisi.",
			})
		}

		file, err := c.FormFile("video")
		if err != nil || file == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "File video tidak ditemukan dalam request.",
			})
		}

		// Maximum 60MB for video
		const maxVideoSize = 60 * 1024 * 1024
		if file.Size > maxVideoSize {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Ukuran video melebihi batas 60MB.",
			})
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext == "" || (ext != ".mp4" && ext != ".webm" && ext != ".mov" && ext != ".mkv") {
			ext = ".webm" // Default browser recording extension
		}

		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Gagal membaca file video: " + err.Error(),
			})
		}
		defer src.Close()

		filename := fmt.Sprintf("%s%s", uuid.NewString(), ext)
		uploadDir := services.FindVideosUploadDir()
		destPath := filepath.Join(uploadDir, filename)

		out, err := os.Create(destPath)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Gagal menyimpan video ke disk server: " + err.Error(),
			})
		}
		defer out.Close()

		if _, err := io.Copy(out, src); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Gagal menulis file video: " + err.Error(),
			})
		}

		localWebPath := "/static/uploads/videos/" + filename
		mimeType := "video/webm"
		if ext == ".mp4" {
			mimeType = "video/mp4"
		}

		gdriveClient, _ := services.NewGDriveClient(db)
		cfg := gdriveClient.GetConfig()

		isApproved := 1
		if !cfg.AutoApprove {
			isApproved = 0
		}

		res, err := db.Exec(`INSERT INTO testimonials 
			(name, class_name, message, video_path, video_type, sync_status, is_approved) 
			VALUES (?, ?, ?, ?, ?, 'local', ?)`,
			name, className, message, localWebPath, mimeType, isApproved,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Gagal mencatat video ke database: " + err.Error(),
			})
		}

		insertedID, _ := res.LastInsertId()

		// Trigger background Google Drive upload if enabled
		if cfg.Enabled {
			go func(id int64, filePath, fname, mtype string, client *services.GDriveClient, database *sql.DB) {
				_, _ = database.Exec("UPDATE testimonials SET sync_status = 'syncing' WHERE id = ?", id)
				fileID, previewURL, uploadErr := client.UploadVideo(filePath, fname, mtype)
				if uploadErr != nil {
					_, _ = database.Exec("UPDATE testimonials SET sync_status = 'failed', sync_error = ? WHERE id = ?", uploadErr.Error(), id)
				} else {
					_, _ = database.Exec("UPDATE testimonials SET sync_status = 'synced', gdrive_file_id = ?, gdrive_url = ?, sync_error = '' WHERE id = ?", fileID, previewURL, id)
				}
			}(insertedID, destPath, filename, mimeType, gdriveClient, db)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":     "success",
			"message":    "Video kesan & pesan berhasil diunggah!",
			"id":         insertedID,
			"video_path": localWebPath,
		})
	}
}

// AdminTestimonialsHandler displays the testimonials management page in admin panel
func AdminTestimonialsHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		rows, err := db.Query(`SELECT 
			id, name, class_name, COALESCE(message, ''), 
			COALESCE(video_path, ''), COALESCE(video_type, 'video/webm'), 
			COALESCE(gdrive_file_id, ''), COALESCE(gdrive_url, ''), 
			COALESCE(sync_status, 'local'), COALESCE(sync_error, ''), 
			COALESCE(is_approved, 1), 
			COALESCE(created_at, '') 
			FROM testimonials ORDER BY id DESC`)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Gagal memuat data video: "+err.Error())
		}
		defer rows.Close()

		var list []TestimonialView
		syncedCount, localCount, failedCount := 0, 0, 0

		for rows.Next() {
			var t TestimonialView
			var approvedInt int
			var createdAt time.Time
			if err := rows.Scan(
				&t.ID, &t.Name, &t.ClassName, &t.Message,
				&t.VideoPath, &t.VideoType,
				&t.GDriveFileID, &t.GDriveURL,
				&t.SyncStatus, &t.SyncError,
				&approvedInt, &createdAt,
			); err == nil {
				t.IsApproved = approvedInt == 1
				t.CreatedAt = createdAt.Format("02 Jan 2006 15:04 WIB")
				list = append(list, t)

				switch t.SyncStatus {
				case "synced":
					syncedCount++
				case "failed":
					failedCount++
				default:
					localCount++
				}
			}
		}

		gdriveClient, _ := services.NewGDriveClient(db)
		cfg := gdriveClient.GetConfig()

		return c.Render(http.StatusOK, "admin_testimonials.html", AdminTestimonialsData{
			AdminLayoutData: adminLayout("Video Ucapan Kiosk | OSIS Admin", "Video Ucapan Kiosk", "Kelola video ucapan, kesan, dan pesan siswa dari bilik kiosk.", "admin_testimonials_content", "testimonials"),
			Testimonials:    list,
			TotalCount:      len(list),
			SyncedCount:     syncedCount,
			LocalCount:      localCount,
			FailedCount:     failedCount,
			GDriveActive:    cfg.Enabled,
		})
	}
}

// AdminTestimonialToggleHandler toggles the approval/publish status of a testimonial
func AdminTestimonialToggleHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "ID video tidak valid"})
		}

		_, err = db.Exec("UPDATE testimonials SET is_approved = 1 - COALESCE(is_approved, 1) WHERE id = ?", id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal memperbarui status: " + err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "success", "message": "Status moderasi berhasil diubah"})
	}
}

// AdminTestimonialDeleteHandler deletes a video and its local file
func AdminTestimonialDeleteHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			return c.Redirect(http.StatusSeeOther, "/admin/testimonials")
		}

		var videoPath string
		_ = db.QueryRow("SELECT video_path FROM testimonials WHERE id = ?", id).Scan(&videoPath)

		_, _ = db.Exec("DELETE FROM testimonials WHERE id = ?", id)

		// Remove local file if present
		if videoPath != "" && strings.HasPrefix(videoPath, "/static/uploads/videos/") {
			filename := strings.TrimPrefix(videoPath, "/static/uploads/videos/")
			fullPath := filepath.Join(services.FindVideosUploadDir(), filename)
			_ = os.Remove(fullPath)
		}

		return c.Redirect(http.StatusSeeOther, "/admin/testimonials")
	}
}

// AdminTestimonialSyncHandler manually triggers Google Drive sync for a single video
func AdminTestimonialSyncHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "ID tidak valid"})
		}

		var (
			videoPath string
			videoType string
		)
		err = db.QueryRow("SELECT video_path, video_type FROM testimonials WHERE id = ?", id).Scan(&videoPath, &videoType)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Data video tidak ditemukan"})
		}

		gdriveClient, err := services.NewGDriveClient(db)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal inisialisasi GDrive: " + err.Error()})
		}
		if !gdriveClient.GetConfig().Enabled {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Integrasi Google Drive belum diaktifkan di Pengaturan"})
		}

		filename := filepath.Base(videoPath)
		localFullPath := filepath.Join(services.FindVideosUploadDir(), filename)

		if _, err := os.Stat(localFullPath); os.IsNotExist(err) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "File lokal tidak ditemukan di disk server (mungkin sudah terhapus)"})
		}

		_, _ = db.Exec("UPDATE testimonials SET sync_status = 'syncing' WHERE id = ?", id)

		fileID, previewURL, uploadErr := gdriveClient.UploadVideo(localFullPath, filename, videoType)
		if uploadErr != nil {
			_, _ = db.Exec("UPDATE testimonials SET sync_status = 'failed', sync_error = ? WHERE id = ?", uploadErr.Error(), id)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal sinkron ke Google Drive: " + uploadErr.Error()})
		}

		_, _ = db.Exec("UPDATE testimonials SET sync_status = 'synced', gdrive_file_id = ?, gdrive_url = ?, sync_error = '' WHERE id = ?", fileID, previewURL, id)

		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":     "success",
			"message":    "Video berhasil disinkronkan ke Google Drive!",
			"gdrive_url": previewURL,
		})
	}
}

// AdminTestimonialsSyncAllHandler triggers background sync for all un-synced videos
func AdminTestimonialsSyncAllHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		gdriveClient, err := services.NewGDriveClient(db)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal inisialisasi GDrive: " + err.Error()})
		}
		if !gdriveClient.GetConfig().Enabled {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Integrasi Google Drive belum diaktifkan di Pengaturan"})
		}

		rows, err := db.Query("SELECT id, video_path, video_type FROM testimonials WHERE sync_status != 'synced'")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal membaca database: " + err.Error()})
		}
		defer rows.Close()

		type item struct {
			id        int
			videoPath string
			videoType string
		}
		var queue []item
		for rows.Next() {
			var it item
			if err := rows.Scan(&it.id, &it.videoPath, &it.videoType); err == nil {
				queue = append(queue, it)
			}
		}

		if len(queue) == 0 {
			return c.JSON(http.StatusOK, map[string]string{
				"status":  "empty",
				"message": "Semua video sudah tersinkronkan ke Google Drive!",
			})
		}

		// Asynchronously process upload queue
		go func(items []item, client *services.GDriveClient, database *sql.DB) {
			for _, it := range items {
				filename := filepath.Base(it.videoPath)
				localFullPath := filepath.Join(services.FindVideosUploadDir(), filename)

				if _, err := os.Stat(localFullPath); err == nil {
					_, _ = database.Exec("UPDATE testimonials SET sync_status = 'syncing' WHERE id = ?", it.id)
					fileID, previewURL, uploadErr := client.UploadVideo(localFullPath, filename, it.videoType)
					if uploadErr != nil {
						_, _ = database.Exec("UPDATE testimonials SET sync_status = 'failed', sync_error = ? WHERE id = ?", uploadErr.Error(), it.id)
					} else {
						_, _ = database.Exec("UPDATE testimonials SET sync_status = 'synced', gdrive_file_id = ?, gdrive_url = ?, sync_error = '' WHERE id = ?", fileID, previewURL, it.id)
					}
				}
				time.Sleep(500 * time.Millisecond) // Friendly delay between uploads
			}
		}(queue, gdriveClient, db)

		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("Proses sinkronisasi massal dimulai untuk %d video.", len(queue)),
			"count":   len(queue),
		})
	}
}

// AdminTestGDriveHandler tests connection with Google Drive Service Account
func AdminTestGDriveHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		client, err := services.NewGDriveClient(db)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal inisialisasi client: " + err.Error()})
		}

		if err := client.TestConnection(); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Uji koneksi Google Drive gagal: " + err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status":  "success",
			"message": "Koneksi Google Drive Service Account berhasil diverifikasi!",
		})
	}
}
