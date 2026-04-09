
package main

import (
	"html/template"
	"io"
	"log"
	"path/filepath"
	"net/http"
	"os"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"gopilketos/database"
	"gopilketos/handlers"
)

// TemplateRenderer implements echo.Renderer
type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(c *echo.Context, w io.Writer, name string, data interface{}) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	// Init DB
	dbPath := filepath.Join("database", "evoting.db")
	db := database.InitDB(dbPath)
	defer db.Close()

	// Migrate tables
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Migration error: %v", err)
	}

	// Load templates
	funcMap := template.FuncMap{
		"eq": func(a, b string) bool { return a == b },
	}
	tmpl := template.New("").Funcs(funcMap)
	tmpl = template.Must(tmpl.ParseGlob(filepath.ToSlash(filepath.Join("views", "voter", "*.html"))))
	tmpl = template.Must(tmpl.ParseGlob(filepath.ToSlash(filepath.Join("views", "admin", "*.html"))))
	tmpl = template.Must(tmpl.ParseGlob(filepath.ToSlash(filepath.Join("views", "layouts", "*.html"))))
	renderer := &TemplateRenderer{templates: tmpl}

	e := echo.New()
	e.Renderer = renderer
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	// Static files
	e.Static("/static", "public")

	// Landing page
	e.GET("/", handlers.LandingPageHandler(db))
	// Scanner page
	e.GET("/scanner", handlers.ScannerPageHandler())
	// Endpoint validasi UUID
	e.POST("/validate-uuid", handlers.ValidateUUIDHandler(db))
	// Candidate API
	e.GET("/api/candidates", handlers.ListCandidatesHandler(db))
	e.GET("/api/candidate", handlers.GetCandidateHandler(db))

	adminUser := os.Getenv("ADMIN_USER")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := os.Getenv("ADMIN_PASS")
	if adminPass == "" {
		adminPass = "admin123"
	}
	adminSecret := os.Getenv("ADMIN_SESSION_SECRET")
	if adminSecret == "" {
		adminSecret = "pilketos-session-secret"
	}

	e.GET("/admin/login", handlers.AdminLoginPageHandler(adminSecret, adminUser))
	e.POST("/admin/login", handlers.AdminLoginHandler(adminUser, adminPass, adminSecret))

	adminGroup := e.Group("/admin")
	adminGroup.Use(handlers.AdminSessionMiddleware(adminSecret, adminUser))

	adminGroup.GET("", handlers.AdminDashboardHandler(db))
	adminGroup.GET("/candidates", handlers.AdminCandidatesHandler(db))
	adminGroup.GET("/candidates/new", handlers.AdminCandidateFormHandler(db))
	adminGroup.GET("/candidates/edit", handlers.AdminCandidateFormHandler(db))
	adminGroup.POST("/candidates/save", handlers.AdminCandidateSaveHandler(db))
	adminGroup.POST("/candidates/:id/delete", handlers.AdminCandidateDeleteHandler(db))
	adminGroup.GET("/logout", handlers.AdminLogoutHandler())

	adminGroup.GET("/voters", handlers.AdminVotersHandler(db))
	adminGroup.GET("/voters/new", handlers.AdminVoterFormHandler())
	adminGroup.POST("/voters/save", handlers.AdminVoterSaveHandler(db))
	adminGroup.POST("/voters/:id/delete", handlers.AdminVoterDeleteHandler(db))

	adminGroup.GET("/logs", handlers.AdminLogsHandler(db))
	adminGroup.GET("/settings", handlers.AdminSettingsHandler(db))
	adminGroup.POST("/settings/save", handlers.AdminSettingsSaveHandler(db))
	// Vote endpoints
	e.POST("/submit-vote", handlers.SubmitVoteHandler(db))
	e.GET("/vote", handlers.VoteStep1Handler())
	e.GET("/vote/step2", handlers.VoteStep2Handler())
	e.GET("/vote/confirm", handlers.VoteConfirmationHandler())
	e.GET("/vote/success", handlers.VoteSuccessHandler())
	// Serve blank service worker file to avoid browser 404 noise when not using sw.js
	e.GET("/sw.js", func(c *echo.Context) error {
		return c.Blob(http.StatusOK, "application/javascript", []byte("// no-op service worker\n"))
	})
	e.GET("/favicon.ico", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	log.Println("Server running at :8080")
	log.Fatal(e.Start(":8080"))
}
