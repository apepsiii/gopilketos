package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"log"

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

		// Validasi UUID dan status
		var hasVoted int
		var phoneNumber string
		err = tx.QueryRow("SELECT has_voted, phone_number FROM voters WHERE uuid = ?", req.UUID).Scan(&hasVoted, &phoneNumber)
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, VoteResponse{"error", "UUID tidak ditemukan"})
		} else if err != nil {
			return c.JSON(http.StatusInternalServerError, VoteResponse{"error", "Gagal query voter"})
		}
		if hasVoted != 0 {
			return c.JSON(http.StatusForbidden, VoteResponse{"error", "Anda sudah memilih"})
		}

		// Update status voter
		_, err = tx.Exec("UPDATE voters SET has_voted = 1 WHERE uuid = ?", req.UUID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, VoteResponse{"error", "Gagal update status voter"})
		}

		// Insert ke votes
		masked := maskUUID(req.UUID)
		_, err = tx.Exec(`INSERT INTO votes (masked_uuid, chairman_id, vice_chairman_id, voted_at) VALUES (?, ?, ?, ?)`, masked, req.ChairmanID, req.ViceChairmanID, time.Now())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, VoteResponse{"error", "Gagal simpan suara"})
		}

		if err := tx.Commit(); err != nil {
			return c.JSON(http.StatusInternalServerError, VoteResponse{"error", "Gagal commit"})
		}

		// TODO: Trigger goroutine kirim WA (OneSender)

		if phoneNumber != "" {
			sender := services.NewOneSenderClient()
			message := "Terima kasih atas partisipasi Anda dalam pemilihan OSIS. Suara Anda telah tercatat."
			go func(phone string) {
				if err := sender.SendMessage(phone, message); err != nil {
					log.Printf("Gagal mengirim WA ke %s: %v", phone, err)
				}
			}(phoneNumber)
		}

		return c.JSON(http.StatusOK, VoteResponse{"success", "Pilihan berhasil disimpan"})
	}
}
