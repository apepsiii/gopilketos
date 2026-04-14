package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"gopilketos/services"
)

type VoteRequest struct {
	UUID           string `json:"uuid"`
	ChairmanID     int    `json:"chairman_id"`
	ViceChairmanID int    `json:"vice_chairman_id"`
}

type VoteResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func maskUUID(uuid string) string {
	if len(uuid) < 8 {
		return uuid
	}
	return uuid[:8] + strings.Repeat("*", len(uuid)-8)
}

func SubmitVoteHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var req VoteRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(http.StatusBadRequest, VoteResponse{"error", "Payload tidak valid"})
		}
		if req.UUID == "" || req.ChairmanID == 0 || req.ViceChairmanID == 0 {
			return c.JSON(http.StatusBadRequest, VoteResponse{"error", "Data tidak lengkap"})
		}

		tx, err := db.Begin()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, VoteResponse{"error", "Gagal mulai transaksi"})
		}
		defer tx.Rollback()

		var hasVoted int
		var phoneNumber, voterName string
		err = tx.QueryRow("SELECT has_voted, phone_number, name FROM voters WHERE uuid = ?", req.UUID).Scan(&hasVoted, &phoneNumber, &voterName)
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, VoteResponse{"error", "UUID tidak ditemukan"})
		} else if err != nil {
			return c.JSON(http.StatusInternalServerError, VoteResponse{"error", "Gagal query voter"})
		}
		if hasVoted != 0 {
			return c.JSON(http.StatusForbidden, VoteResponse{"error", "Anda sudah memilih"})
		}

		_, err = tx.Exec("UPDATE voters SET has_voted = 1 WHERE uuid = ?", req.UUID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, VoteResponse{"error", "Gagal update status voter"})
		}

		var chairmanPosition, vicePosition string
		err = tx.QueryRow("SELECT position FROM candidates WHERE id = ?", req.ChairmanID).Scan(&chairmanPosition)
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, VoteResponse{"error", "Kandidat ketua tidak ditemukan"})
		} else if err != nil {
			return c.JSON(http.StatusInternalServerError, VoteResponse{"error", "Gagal validasi kandidat"})
		}
		if chairmanPosition != "CHAIRMAN" {
			return c.JSON(http.StatusBadRequest, VoteResponse{"error", "Kandidat ID pertama bukan chairman"})
		}

		err = tx.QueryRow("SELECT position FROM candidates WHERE id = ?", req.ViceChairmanID).Scan(&vicePosition)
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, VoteResponse{"error", "Kandidat wakil tidak ditemukan"})
		} else if err != nil {
			return c.JSON(http.StatusInternalServerError, VoteResponse{"error", "Gagal validasi kandidat"})
		}
		if vicePosition != "VICE_CHAIRMAN" {
			return c.JSON(http.StatusBadRequest, VoteResponse{"error", "Kandidat ID kedua bukan vice chairman"})
		}

		masked := maskUUID(req.UUID)
		_, err = tx.Exec(`INSERT INTO votes (masked_uuid, chairman_id, vice_chairman_id, voted_at) VALUES (?, ?, ?, ?)`, masked, req.ChairmanID, req.ViceChairmanID, time.Now())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, VoteResponse{"error", "Gagal simpan suara"})
		}

		if err := tx.Commit(); err != nil {
			return c.JSON(http.StatusInternalServerError, VoteResponse{"error", "Gagal commit"})
		}

		if phoneNumber != "" {
			go func() {
				sender, err := services.NewOneSenderClient(db)
				if err != nil {
					log.Printf("Gagal membuat OneSender client: %v", err)
					return
				}

				message := fmt.Sprintf(`Halo %s! Terima kasih telah berpartisipasi dalam Pemilihan OSIS SMK NIBA Business School.

Suara Anda telah tercatat dengan aman dan rahasia.

.partisipasimu sangat berarti untuk masa depan sekolah kita. Juntos, kita bisa menciptakan perubahan positif! 💪

"Mari bersama-sama membangun sekolah yang lebih baik."

Terima kasih!`, voterName)

				if err := sender.SendMessage(phoneNumber, message); err != nil {
					log.Printf("Gagal mengirim WA ke %s: %v", phoneNumber, err)
				}
			}()
		}

		return c.JSON(http.StatusOK, VoteResponse{"success", "Pilihan berhasil disimpan"})
	}
}
