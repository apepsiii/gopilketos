package handlers

import (
	"net/http"
	"github.com/labstack/echo/v5"
)

type VotingPageData struct {
	UUID string
}

// Step 1: Pilih Ketua
func VoteStep1Handler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		uuid := c.QueryParam("uuid")
		if uuid == "" {
			return c.Redirect(http.StatusSeeOther, "/scanner")
		}
		return c.Render(http.StatusOK, "vote_step1.html", VotingPageData{UUID: uuid})
	}
}

// Step 2: Pilih Wakil
func VoteStep2Handler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		uuid := c.QueryParam("uuid")
		if uuid == "" {
			return c.Redirect(http.StatusSeeOther, "/scanner")
		}
		return c.Render(http.StatusOK, "vote_step2.html", VotingPageData{UUID: uuid})
	}
}

// Step 3: Konfirmasi
func VoteConfirmationHandler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		uuid := c.QueryParam("uuid")
		if uuid == "" {
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
