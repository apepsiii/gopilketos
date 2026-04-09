package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"
)

type CandidateAPIResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	ClassName string `json:"class_name"`
	PhotoURL  string `json:"photo_url"`
	Vision    string `json:"vision"`
	Mission   string `json:"mission"`
	Program   string `json:"program"`
}

func ListCandidatesHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		position := c.QueryParam("position")
		if position != "CHAIRMAN" && position != "VICE_CHAIRMAN" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Posisi kandidat tidak valid"})
		}

		rows, err := db.Query("SELECT id, name, class_name, photo_url, vision, mission, program FROM candidates WHERE position = ? ORDER BY id", position)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal mengambil kandidat"})
		}
		defer rows.Close()

		candidates := []CandidateAPIResponse{}
		for rows.Next() {
			var candidate CandidateAPIResponse
			if err := rows.Scan(&candidate.ID, &candidate.Name, &candidate.ClassName, &candidate.PhotoURL, &candidate.Vision, &candidate.Mission, &candidate.Program); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal memproses kandidat"})
			}
			candidates = append(candidates, candidate)
		}

		return c.JSON(http.StatusOK, candidates)
	}
}

func GetCandidateHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		idStr := c.QueryParam("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "ID kandidat tidak valid"})
		}

		var candidate CandidateAPIResponse
		err = db.QueryRow("SELECT id, name, class_name, photo_url, vision, mission, program FROM candidates WHERE id = ?", id).Scan(
			&candidate.ID,
			&candidate.Name,
			&candidate.ClassName,
			&candidate.PhotoURL,
			&candidate.Vision,
			&candidate.Mission,
			&candidate.Program,
		)
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Kandidat tidak ditemukan"})
		} else if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal mengambil kandidat"})
		}

		return c.JSON(http.StatusOK, candidate)
	}
}
