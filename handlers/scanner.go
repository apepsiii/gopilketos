package handlers

import (
	"database/sql"
	"net/http"
	"github.com/labstack/echo/v5"
)

type ScannerPageData struct {
	Error string
}

// Halaman scanner QR
func ScannerPageHandler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		return c.Render(http.StatusOK, "scanner.html", ScannerPageData{})
	}
}

// Endpoint validasi UUID (AJAX)
func ValidateUUIDHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		type req struct { UUID string `json:"uuid"` }
		var r req
		if err := c.Bind(&r); err != nil || r.UUID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "UUID tidak valid"})
		}
		var hasVoted int
		err := db.QueryRow("SELECT has_voted FROM voters WHERE uuid = ?", r.UUID).Scan(&hasVoted)
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "UUID tidak ditemukan"})
		} else if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Terjadi kesalahan"})
		}
		if hasVoted != 0 {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Anda sudah memilih"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
