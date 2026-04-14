package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"gopilketos/database"
	"gopilketos/handlers"
	"gopkg.in/yaml.v3"
)

const AppVersion = "v1.0.1"

//go:embed views
//go:embed public/css
//go:embed public/js
//go:embed public/images
var embeddedAssets embed.FS

type Config struct {
	AppName    string `yaml:"app_name"`
	Port       string `yaml:"port"`
	Domain     string `yaml:"domain"`
	InstallDir string `yaml:"install_dir"`
	AdminUser  string `yaml:"admin_user"`
	AdminPass  string `yaml:"admin_pass"`
	DbPath     string `yaml:"db_path"`
}

func getViewsFS() fs.FS {
	f, _ := fs.Sub(embeddedAssets, "views")
	return f
}

func getStaticFS() fs.FS {
	f, _ := fs.Sub(embeddedAssets, "public")
	return f
}

func getUploadsDir() string {
	exeDir, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exeDir), "public", "uploads")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	cwd, _ := os.Getwd()
	dir := filepath.Join(cwd, "public", "uploads")
	os.MkdirAll(dir, 0755)
	return dir
}

func loadConfig() *Config {
	config := &Config{
		Port:      "8024",
		DbPath:    "database/evoting.db",
		AdminUser: "admin",
		AdminPass: "admin123",
	}

	exeDir, err := os.Executable()
	if err == nil {
		exeDir = filepath.Dir(exeDir)
	} else {
		exeDir = "."
	}

	cwd, _ := os.Getwd()

	configPaths := []string{
		filepath.Join(exeDir, "config.yaml"),
		"/opt/pilketos/config.yaml",
		filepath.Join(cwd, "config.yaml"),
		"config.yaml",
	}

	for _, path := range configPaths {
		log.Printf("Trying config: %s", path)
		if data, err := os.ReadFile(path); err == nil {
			if err := yaml.Unmarshal(data, config); err == nil {
				if config.Port == "" {
					config.Port = "8024"
				}
				if config.DbPath == "" {
					config.DbPath = "database/evoting.db"
				}
				log.Printf("Config loaded from: %s", path)
				return config
			} else {
				log.Printf("Failed to parse %s: %v", path, err)
			}
		}
	}

	log.Println("No config.yaml found, using defaults")
	return config
}

type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(c *echo.Context, w io.Writer, name string, data interface{}) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	config := loadConfig()

	if config.Port == "" {
		config.Port = "8024"
	}
	if config.DbPath == "" {
		config.DbPath = "database/evoting.db"
	}

	dbDir := filepath.Dir(config.DbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	uploadsDir := getUploadsDir()
	os.MkdirAll(uploadsDir, 0755)

	db := database.InitDB(config.DbPath)
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Migration error: %v", err)
	}

	funcMap := template.FuncMap{
		"eq": func(a, b string) bool { return a == b },
	}
	tmpl := template.New("").Funcs(funcMap)
	viewsFS := getViewsFS()
	tmpl = template.Must(tmpl.ParseFS(viewsFS, "voter/*.html"))
	tmpl = template.Must(tmpl.ParseFS(viewsFS, "admin/*.html"))
	tmpl = template.Must(tmpl.ParseFS(viewsFS, "layouts/*.html"))
	renderer := &TemplateRenderer{templates: tmpl}

	e := echo.New()
	e.Renderer = renderer
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.StaticFS("/static", getStaticFS())

	e.Static("/static/uploads", uploadsDir)

	e.GET("/", handlers.LandingPageHandler(db))
	e.GET("/scanner", handlers.ScannerPageHandler())
	e.POST("/validate-uuid", handlers.ValidateUUIDHandler(db))
	e.GET("/api/candidates", handlers.ListCandidatesHandler(db))
	e.GET("/api/candidate", handlers.GetCandidateHandler(db))

	adminSecret := config.AdminPass + "-session-secret"

	e.GET("/admin/login", handlers.AdminLoginPageHandler(adminSecret, config.AdminUser))
	e.POST("/admin/login", handlers.AdminLoginHandler(config.AdminUser, config.AdminPass, adminSecret))

	adminGroup := e.Group("/admin")
	adminGroup.Use(handlers.AdminSessionMiddleware(adminSecret, config.AdminUser))

	adminGroup.GET("", handlers.AdminDashboardHandler(db))
	adminGroup.GET("/candidates", handlers.AdminCandidatesHandler(db))
	adminGroup.GET("/candidates/new", handlers.AdminCandidateFormHandler(db))
	adminGroup.GET("/candidates/:id/edit", handlers.AdminCandidateFormHandler(db))
	adminGroup.POST("/candidates", handlers.AdminCandidateCreateHandler(db))
	adminGroup.POST("/candidates/:id", handlers.AdminCandidateUpdateHandler(db))
	adminGroup.POST("/candidates/:id/delete", handlers.AdminCandidateDeleteHandler(db))
	adminGroup.GET("/logout", handlers.AdminLogoutHandler())

	adminGroup.GET("/voters", handlers.AdminVotersHandler(db))
	adminGroup.GET("/voters/new", handlers.AdminVoterFormHandler(db))
	adminGroup.GET("/voters/:id/edit", handlers.AdminVoterEditHandler(db))
	adminGroup.GET("/voters/import", handlers.AdminVoterImportPageHandler())
	adminGroup.GET("/voters/cards", handlers.AdminVoterCardsHandler(db))
	adminGroup.POST("/voters/save", handlers.AdminVoterSaveHandler(db))
	adminGroup.POST("/voters/update", handlers.AdminVoterUpdateHandler(db))
	adminGroup.POST("/voters/import", handlers.AdminVoterImportHandler(db))
	adminGroup.POST("/voters/:id/delete", handlers.AdminVoterDeleteHandler(db))
	adminGroup.POST("/voters/bulk-delete", handlers.AdminVotersBulkDeleteHandler(db))

	adminGroup.GET("/logs", handlers.AdminLogsHandler(db))
	adminGroup.GET("/settings", handlers.AdminSettingsHandler(db))
	adminGroup.POST("/settings/save", handlers.AdminSettingsSaveHandler(db))
	adminGroup.POST("/settings/test-message", handlers.AdminTestMessageHandler(db))
	adminGroup.POST("/settings/reset-votes", handlers.AdminResetVotesHandler(db))
	adminGroup.GET("/settings/backup", handlers.AdminBackupHandler(db))
	adminGroup.GET("/settings/report", handlers.AdminReportHandler(db))

	adminGroup.GET("/attendance", handlers.AdminAttendanceListHandler(db))
	adminGroup.GET("/attendance/scanner", handlers.AdminPresenceScannerHandler())
	adminGroup.POST("/attendance/mark", handlers.AdminMarkAttendanceHandler(db))

	e.POST("/submit-vote", handlers.SubmitVoteHandler(db))
	e.GET("/vote", handlers.VoteStep1Handler())
	e.GET("/vote/step2", handlers.VoteStep2Handler())
	e.GET("/vote/confirm", handlers.VoteConfirmationHandler())
	e.GET("/vote/success", handlers.VoteSuccessHandler())

	e.GET("/sw.js", func(c *echo.Context) error {
		return c.Blob(http.StatusOK, "application/javascript", []byte("// no-op service worker\n"))
	})
	e.GET("/favicon.ico", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	fmt.Printf("===========================================\n")
	fmt.Printf("  Pilketos E-Voting System\n")
	fmt.Printf("===========================================\n")
	fmt.Printf("  Port:    %s\n", config.Port)
	fmt.Printf("  Domain:  %s\n", config.Domain)
	fmt.Printf("  DB:      %s\n", config.DbPath)
	fmt.Printf("  Uploads: %s\n", uploadsDir)
	fmt.Printf("  Admin:   %s\n", config.AdminUser)
	fmt.Printf("===========================================\n")
	fmt.Printf("  Server running at :%s\n", config.Port)
	fmt.Printf("  Press Ctrl+C to stop\n")
	fmt.Printf("===========================================\n")

	log.Fatal(e.Start(":" + config.Port))
}
