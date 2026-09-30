package handlers

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v5"
)

type VotingPageData struct {
	UUID string
}

func isVoterEligible(db *sql.DB, uuid string) bool {
	if uuid == "" {
		return false
	}
	var hasVoted int
	err := db.QueryRow("SELECT has_voted FROM voters WHERE uuid = ?", uuid).Scan(&hasVoted)
	return err == nil && hasVoted == 0
}

// Step 1: Pilih Ketua
func VoteStep1Handler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		uuid := c.QueryParam("uuid")
		if !isVoterEligible(db, uuid) {
			return c.Redirect(http.StatusSeeOther, "/scanner")
		}
		return c.Render(http.StatusOK, "vote_step1.html", VotingPageData{UUID: uuid})
	}
}

// Step 2: Pilih Wakil
func VoteStep2Handler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		uuid := c.QueryParam("uuid")
		if !isVoterEligible(db, uuid) {
			return c.Redirect(http.StatusSeeOther, "/scanner")
		}
		return c.Render(http.StatusOK, "vote_step2.html", VotingPageData{UUID: uuid})
	}
}

// Step 3: Konfirmasi
func VoteConfirmationHandler(db *sql.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		uuid := c.QueryParam("uuid")
		if !isVoterEligible(db, uuid) {
			return c.Redirect(http.StatusSeeOther, "/scanner")
		}
		return c.Render(http.StatusOK, "vote_confirmation.html", VotingPageData{UUID: uuid})
	}
}

// Sukses
func VoteSuccessHandler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		return c.Render(http.StatusOK, "success.html", nil)
	}
}
