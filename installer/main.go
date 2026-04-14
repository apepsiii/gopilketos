package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	white  = "\033[37m"
	bold   = "\033[1m"
)

type Config struct {
	AppName    string `yaml:"app_name"`
	Port       string `yaml:"port"`
	Domain     string `yaml:"domain"`
	InstallDir string `yaml:"install_dir"`
	AdminUser  string `yaml:"admin_user"`
	AdminPass  string `yaml:"admin_pass"`
	DbPath     string `yaml:"db_path"`
}

func main() {
	fmt.Println()
	printBanner()
	fmt.Println()

	existingConfig := loadExistingConfig()

	action := selectAction(existingConfig != nil)

	var config *Config

	if action == "update" && existingConfig != nil {
		config = existingConfig
		fmt.Println()
		printInfo("Updating existing installation...")
		fmt.Println()
	} else {
		config = runWizard(existingConfig)
		if config == nil {
			printError("Installation cancelled")
			os.Exit(1)
		}
	}

	fmt.Println()
	if action == "update" {
		printInfo("Starting update process...")
	} else {
		printInfo("Starting installation...")
	}
	fmt.Println()

	if err := install(config, action == "update"); err != nil {
		printError(fmt.Sprintf("Process failed: %v", err))
		fmt.Println()
		os.Exit(1)
	}

	printSuccessBox(config, action == "update")
}

func printBanner() {
	header := `
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║     █████╗ ███████╗███████╗██████╗ ███████╗██╗               ║
║    ██╔══██╗██╔════╝██╔════╝██╔══██╗██╔════╝██║               ║
║    ███████║███████╗███████╗██████╔╝█████╗  ██║               ║
║    ██╔══██║╚════██║╚════██║██╔═══╝ ██╔══╝  ██║               ║
║    ██║  ██║███████║███████║██║     ███████╗███████╗         ║
║    ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝     ╚══════╝╚══════╝         ║
║                                                              ║
║                  E-VOTING SYSTEM INSTALLER                    ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
`
	fmt.Print(cyan + header + reset)
}

func loadExistingConfig() *Config {
	configPath := "/opt/pilketos/config.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil
	}

	return &config
}

func selectAction(hasExisting bool) string {
	reader := bufio.NewReader(os.Stdin)

	if hasExisting {
		fmt.Println("  " + white + "Existing installation detected:" + reset)
		fmt.Println("  " + strings.Repeat("─", 50))
		fmt.Println("    " + cyan + "[1]" + reset + " Update existing installation (keep config)")
		fmt.Println("    " + cyan + "[2]" + reset + " Fresh install (new configuration)")
		fmt.Println("    " + cyan + "[3]" + reset + " Cancel")
		fmt.Println("  " + strings.Repeat("─", 50))
		fmt.Println()

		for {
			fmt.Print("  " + white + "Select option [1-3]: " + reset)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			switch input {
			case "1":
				return "update"
			case "2":
				return "install"
			case "3":
				os.Exit(0)
			}
		}
	}

	return "install"
}

func runWizard(existing *Config) *Config {
	reader := bufio.NewReader(os.Stdin)

	defaults := &Config{
		AppName:    "pilketos",
		Port:       "8024",
		Domain:     "",
		InstallDir: "/opt/pilketos",
		AdminUser:  "admin",
		AdminPass:  generateRandomPassword(),
		DbPath:     "/opt/pilketos/database/evoting.db",
	}

	if existing != nil {
		defaults = existing
	}

	printStep(1, 5, "Domain Configuration")
	if existing != nil {
		fmt.Printf("  Current: %s%s%s\n", cyan, existing.Domain, reset)
	}
	fmt.Print("  " + white + "Domain (e.g., vote.bersekola.app): " + reset)
	domain, _ := reader.ReadString('\n')
	configDomain := strings.TrimSpace(domain)
	if configDomain == "" && existing != nil {
		configDomain = existing.Domain
	}
	if configDomain == "" {
		printError("Domain is required")
		os.Exit(1)
	}

	config := &Config{
		AppName:    defaults.AppName,
		Domain:     configDomain,
		InstallDir: defaults.InstallDir,
		DbPath:     defaults.DbPath,
	}

	printStep(2, 5, "Port Configuration")
	if existing != nil {
		fmt.Printf("  Current: %s%s%s\n", cyan, existing.Port, reset)
	}
	fmt.Printf("  %sPort%s [%s%s%s]: ", white, reset, cyan, defaults.Port, reset)
	port, _ := reader.ReadString('\n')
	port = strings.TrimSpace(port)
	if port != "" {
		config.Port = port
	} else {
		config.Port = defaults.Port
	}

	printStep(3, 5, "Installation Directory")
	if existing != nil {
		fmt.Printf("  Current: %s%s%s\n", cyan, existing.InstallDir, reset)
	}
	fmt.Printf("  %sInstall Directory%s [%s%s%s]: ", white, reset, cyan, defaults.InstallDir, reset)
	dir, _ := reader.ReadString('\n')
	dir = strings.TrimSpace(dir)
	if dir != "" {
		config.InstallDir = strings.TrimSuffix(dir, "/")
		config.DbPath = filepath.Join(config.InstallDir, "database", "evoting.db")
	} else {
		config.InstallDir = defaults.InstallDir
		config.DbPath = defaults.DbPath
	}

	printStep(4, 5, "Admin Credentials")
	if existing != nil {
		fmt.Printf("  Current Username: %s%s%s\n", cyan, existing.AdminUser, reset)
	}
	fmt.Printf("  %sAdmin Username%s [%s%s%s]: ", white, reset, cyan, defaults.AdminUser, reset)
	user, _ := reader.ReadString('\n')
	user = strings.TrimSpace(user)
	if user != "" {
		config.AdminUser = user
	} else {
		config.AdminUser = defaults.AdminUser
	}

	if existing != nil {
		fmt.Printf("  Current Password: %s%s%s\n", cyan, "(hidden)", reset)
		fmt.Printf("  %sAdmin Password%s [%s%s%s]: ", white, reset, cyan, "(keep existing)", reset)
	} else {
		fmt.Printf("  %sAdmin Password%s [%s%s%s]: ", white, reset, cyan, "(auto-generated)", reset)
	}
	fmt.Printf("    %sGenerated: %s%s%s\n", yellow, green, config.AdminPass, reset)
	pass, _ := reader.ReadString('\n')
	pass = strings.TrimSpace(pass)
	if pass != "" {
		config.AdminPass = pass
	} else if existing != nil {
		config.AdminPass = existing.AdminPass
	}

	printStep(5, 5, "Review Configuration")
	fmt.Println()
	fmt.Println("  " + bold + white + "Configuration Summary:" + reset)
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Printf("  %sDomain:%s        %s%s%s\n", white, reset, green, config.Domain, reset)
	fmt.Printf("  %sPort:%s          %s%s%s\n", white, reset, green, config.Port, reset)
	fmt.Printf("  %sInstall Dir:%s   %s%s%s\n", white, reset, green, config.InstallDir, reset)
	fmt.Printf("  %sDatabase:%s      %s%s%s\n", white, reset, green, config.DbPath, reset)
	fmt.Printf("  %sAdmin User:%s    %s%s%s\n", white, reset, green, config.AdminUser, reset)
	fmt.Printf("  %sAdmin Pass:%s    %s%s%s\n", white, reset, green, config.AdminPass, reset)
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Println()

	fmt.Printf("  %sProceed with installation?%s [%sY%s/n]: ", white, reset, green, reset)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm == "n" || confirm == "no" {
		return nil
	}

	return config
}

func install(config *Config, isUpdate bool) error {
	steps := []struct {
		name string
		fn   func(*Config) error
	}{
		{"Creating directories", createDirs},
		{"Stopping existing service", stopService},
		{"Copying application binary", copyFiles},
		{"Generating configuration", generateConfig},
		{"Setting permissions", setPermissions},
		{"Creating systemd service", createSystemdService},
		{"Configuring Nginx", configureNginx},
		{"Reloading systemd daemon", reloadSystemd},
		{"Enabling service", enableService},
		{"Starting service", startService},
		{"Verifying installation", verifyInstall},
	}

	total := len(steps)
	fmt.Println()

	for i, step := range steps {
		printProgress(i+1, total, step.name)

		if err := step.fn(config); err != nil {
			fmt.Println()
			printError(fmt.Sprintf("Failed: %v", err))
			return err
		}

		printSuccess(fmt.Sprintf("%s ✓", step.name))
	}

	fmt.Println()
	return nil
}

func printProgress(current, total int, message string) {
	percent := float64(current) / float64(total) * 100
	barLen := 40
	filled := int(float64(barLen) * percent / 100)

	bar := ""
	for j := 0; j < barLen; j++ {
		if j < filled {
			bar += green + "█" + reset
		} else {
			bar += "░"
		}
	}

	fmt.Printf("  %s[%s]%s %3.0f%% %s", cyan, bar, reset, percent, message)
}

func printSuccess(msg string) {
	fmt.Printf("  %s\n", green+msg+reset)
}

func createDirs(config *Config) error {
	dirs := []string{
		config.InstallDir,
		filepath.Dir(config.DbPath),
		"/var/log",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func stopService(config *Config) error {
	exec.Command("systemctl", "stop", config.AppName).Run()
	return nil
}

func copyFiles(config *Config) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	dstBin := filepath.Join(config.InstallDir, config.AppName)

	srcFile, err := os.Open(exePath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstBin)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return os.Chmod(dstBin, 0755)
}

func generateConfig(config *Config) error {
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	configPath := filepath.Join(config.InstallDir, "config.yaml")
	return os.WriteFile(configPath, yamlData, 0644)
}

func setPermissions(config *Config) error {
	dbDir := filepath.Dir(config.DbPath)
	os.Chmod(dbDir, 0755)
	return nil
}

func createSystemdService(config *Config) error {
	serviceContent := fmt.Sprintf(`[Unit]
Description=Pilketos E-Voting System
After=network.target nginx.service
Wants=nginx.service

[Service]
Type=simple
User=root
WorkingDirectory=%s
ExecStart=%s/%s
Restart=always
RestartSec=5
StandardOutput=append:/var/log/%s.log
StandardError=append:/var/log/%s_error.log

[Install]
WantedBy=multi-user.target
`, config.InstallDir, config.InstallDir, config.AppName, config.AppName, config.AppName)

	servicePath := filepath.Join("/etc/systemd/system", config.AppName+".service")
	return os.WriteFile(servicePath, []byte(serviceContent), 0644)
}

func configureNginx(config *Config) error {
	nginxContent := fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    client_max_body_size 20M;

    location / {
        proxy_pass http://127.0.0.1:%s;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        proxy_read_timeout 300s;
        proxy_connect_timeout 75s;
    }

    location /static/ {
        proxy_pass http://127.0.0.1:%s;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
`, config.Domain, config.Port, config.Port)

	nginxPath := filepath.Join("/etc/nginx/sites-available", config.Domain)
	if err := os.WriteFile(nginxPath, []byte(nginxContent), 0644); err != nil {
		return err
	}

	enabledPath := filepath.Join("/etc/nginx/sites-enabled", config.Domain)
	os.Remove(enabledPath)
	return os.Symlink(nginxPath, enabledPath)
}

func reloadSystemd(config *Config) error {
	out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(out))
	}
	return nil
}

func enableService(config *Config) error {
	out, err := exec.Command("systemctl", "enable", config.AppName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(out))
	}
	return nil
}

func startService(config *Config) error {
	out, err := exec.Command("systemctl", "start", config.AppName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(out))
	}

	exec.Command("systemctl", "reload", "nginx").CombinedOutput()
	return nil
}

func verifyInstall(config *Config) error {
	time.Sleep(2)

	out, err := exec.Command("systemctl", "is-active", config.AppName).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "active") {
		return fmt.Errorf("service not running")
	}

	return nil
}

func printSuccessBox(config *Config, isUpdate bool) {
	actionText := "INSTALLATION COMPLETED!"
	if isUpdate {
		actionText = "UPDATE COMPLETED!"
	}

	box := fmt.Sprintf(`
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║           ██████╗ ███████╗ █████╗ ██████╗ ███╗   ███╗       ║
║           ██╔══██╗██╔════╝██╔══██╗██╔══██╗████╗ ████║       ║
║           ██████╔╝█████╗  ███████║██║  ██║██╔████╔██║       ║
║           ██╔══██╗██╔══╝  ██╔══██║██║  ██║██║╚██╔╝██║       ║
║           ██║  ██║███████╗██║  ██║██████╔╝██║ ╚═╝ ██║       ║
║           ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═════╝ ╚═╝     ╚═╝       ║
║                                                              ║
║                  %s                      ║
║                                                              ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║    %sAccess URLs:%s                                            ║
║      • %shttps://%s%s (or http://)                  ║
║      • %shttps://%s/admin/login%s                        ║
║                                                              ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║    %sAdmin Credentials:%s                                     ║
║      Username: %s%s%s                                      ║
║      Password: %s%s%s                                 ║
║                                                              ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║    %sService Commands:%s                                      ║
║      sudo systemctl start pilketos       (Start)            ║
║      sudo systemctl stop pilketos        (Stop)             ║
║      sudo systemctl restart pilketos     (Restart)          ║
║      sudo systemctl status pilketos      (Status)           ║
║      sudo systemctl enable pilketos      (Auto-start)       ║
║      sudo systemctl disable pilketos     (Disable auto)     ║
║                                                              ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║    %sLog Files:%s                                             ║
║      tail -f /var/log/pilketos.log                          ║
║      tail -f /var/log/pilketos_error.log                    ║
║                                                              ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║    %sConfig & Database:%s                                    ║
║      Config:  %s/opt/pilketos/config.yaml%s                   ║
║      Database: %s/opt/pilketos/database/evoting.db%s          ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
`, actionText, bold+white, reset, cyan, config.Domain, reset, cyan, config.Domain, reset, bold+white, reset, green, config.AdminUser, reset, green, config.AdminPass, reset, bold+white, reset, bold+white, reset, bold+white, reset, cyan, reset, cyan, reset)

	fmt.Println(green + box + reset)
}

func printStep(current, total int, message string) {
	fmt.Printf("\n  %s[%d/%d]%s %s%s%s\n", cyan, current, total, reset, bold, message, reset)
}

func printError(msg string) {
	fmt.Printf("  %s✗ %s%s\n", red+bold, reset, msg)
}

func printInfo(msg string) {
	fmt.Printf("  %sℹ %s%s\n", yellow+bold, reset, msg)
}

func generateRandomPassword() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
