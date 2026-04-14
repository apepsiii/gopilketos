package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	white  = "\033[37m"
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

	config, err := runWizard()
	if err != nil {
		printError("Installation cancelled")
		os.Exit(1)
	}

	fmt.Println()
	printInfo("Starting installation...")
	fmt.Println()

	if err := install(config); err != nil {
		printError(fmt.Sprintf("Installation failed: %v", err))
		fmt.Println()
		os.Exit(1)
	}

	printSuccessBox(config)
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

func printSuccessBox(config *Config) {
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
║                  INSTALLATION COMPLETED!                      ║
║                                                              ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║    Domain:     %shttps://%s%s                    ║
║    Admin:      %shttps://%s/admin/login%s                  ║
║    Port:       %s%s%s                               ║
║                                                              ║
║    Username:   %s%s%s                                    ║
║    Password:   %s%s%s                               ║
║                                                              ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║    Commands:                                                 ║
║      sudo systemctl start %s                                 ║
║      sudo systemctl enable %s                                ║
║      sudo systemctl status %s                               ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
`, cyan, config.Domain, reset, cyan, config.Domain, reset, cyan, config.Port, reset, cyan, config.AdminUser, reset, cyan, config.AdminPass, reset, config.AppName, config.AppName, config.AppName)
	fmt.Print(green + box + reset)
}

func runWizard() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)
	config := &Config{
		AppName:    "pilketos",
		Port:       "8024",
		Domain:     "",
		InstallDir: "/opt/pilketos",
		AdminUser:  "admin",
		AdminPass:  randomString(12),
		DbPath:     "/opt/pilketos/database/evoting.db",
	}

	printStep(1, 6, "Domain Configuration")
	fmt.Print(white + "  Domain (e.g., vote.bersekola.app): " + reset)
	domain, _ := reader.ReadString('\n')
	config.Domain = strings.TrimSpace(domain)
	if config.Domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	printStep(2, 6, "Port Configuration")
	fmt.Printf("  %sPort%s [%s%s%s]: ", white, reset, cyan, config.Port, reset)
	port, _ := reader.ReadString('\n')
	port = strings.TrimSpace(port)
	if port != "" {
		config.Port = port
	}

	printStep(3, 6, "Installation Directory")
	fmt.Printf("  %sInstall Directory%s [%s%s%s]: ", white, reset, cyan, config.InstallDir, reset)
	dir, _ := reader.ReadString('\n')
	dir = strings.TrimSpace(dir)
	if dir != "" {
		config.InstallDir = strings.TrimSuffix(dir, "/")
		config.DbPath = filepath.Join(config.InstallDir, "database", "evoting.db")
	}

	printStep(4, 6, "Admin Credentials")
	fmt.Printf("  %sAdmin Username%s [%s%s%s]: ", white, reset, cyan, config.AdminUser, reset)
	user, _ := reader.ReadString('\n')
	user = strings.TrimSpace(user)
	if user != "" {
		config.AdminUser = user
	}

	fmt.Print("  " + white + "Admin Password" + reset + " [" + cyan + "(auto-generated)" + reset + "]: ")
	pass, _ := reader.ReadString('\n')
	pass = strings.TrimSpace(pass)
	if pass != "" {
		config.AdminPass = pass
	} else {
		fmt.Printf("    %sGenerated: %s%s\n", cyan, config.AdminPass, reset)
	}

	printStep(5, 6, "Review Configuration")
	fmt.Println()
	fmt.Println("  " + white + "Configuration Summary:" + reset)
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Printf("  %sDomain:%s      %s\n", white, reset, config.Domain)
	fmt.Printf("  %sPort:%s        %s\n", white, reset, config.Port)
	fmt.Printf("  %sInstall Dir:%s %s\n", white, reset, config.InstallDir)
	fmt.Printf("  %sDatabase:%s    %s\n", white, reset, config.DbPath)
	fmt.Printf("  %sAdmin User:%s  %s\n", white, reset, config.AdminUser)
	fmt.Printf("  %sAdmin Pass:%s  %s\n", white, reset, config.AdminPass)
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Println()

	printStep(6, 6, "Starting Installation")
	return config, nil
}

func install(config *Config) error {
	steps := []struct {
		name string
		fn   func(*Config) error
	}{
		{"Create installation directories", createDirs},
		{"Stop existing service (if any)", stopService},
		{"Copy application files", copyFiles},
		{"Generate configuration file", generateConfig},
		{"Set file permissions", setPermissions},
		{"Create systemd service", createSystemdService},
		{"Configure Nginx", configureNginx},
		{"Reload systemd and nginx", reloadServices},
		{"Start application service", startServiceFn},
		{"Verify installation", verifyInstall},
	}

	total := len(steps)
	prefix := ""
	for i, step := range steps {
		percent := float64(i+1) / float64(total) * 100
		barLen := 30
		filled := int(float64(barLen) * float64(i+1) / float64(total))
		bar := ""
		for j := 0; j < barLen; j++ {
			if j < filled {
				bar += green + "█" + reset
			} else {
				bar += "░"
			}
		}
		fmt.Printf("\r  %s[%s] %3d%% %s", prefix, bar, int(percent), step.name)
		prefix = "\r"

		if err := step.fn(config); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}
	fmt.Printf("\r  %s[%s] 100%% Done!%s\n", green, green+"█"+reset, reset)
	return nil
}

func createDirs(config *Config) error {
	dirs := []string{
		config.InstallDir,
		filepath.Dir(config.DbPath),
		"/etc/nginx/sites-available",
		"/etc/nginx/sites-enabled",
		"/etc/systemd/system",
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
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	srcBin := exePath
	dstBin := filepath.Join(config.InstallDir, config.AppName)

	if err := copyFile(srcBin, dstBin); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, info.Mode())
}

func generateConfig(config *Config) error {
	yamlData, err := yaml.Marshal(&Config{
		AppName:    config.AppName,
		Port:       config.Port,
		Domain:     config.Domain,
		InstallDir: config.InstallDir,
		AdminUser:  config.AdminUser,
		AdminPass:  config.AdminPass,
		DbPath:     config.DbPath,
	})
	if err != nil {
		return err
	}

	configPath := filepath.Join(config.InstallDir, "config.yaml")
	return os.WriteFile(configPath, yamlData, 0644)
}

func setPermissions(config *Config) error {
	pilketosPath := filepath.Join(config.InstallDir, config.AppName)
	if err := os.Chmod(pilketosPath, 0755); err != nil {
		return err
	}
	return os.Chown(filepath.Join(config.InstallDir, "database"), 0, 0)
}

func createSystemdService(config *Config) error {
	serviceContent := fmt.Sprintf(`[Unit]
Description=Pilketos E-Voting System
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=%s
ExecStart=%s
Restart=always
RestartSec=5
StandardOutput=append:/var/log/%s.log
StandardError=append:/var/log/%s_error.log

[Install]
WantedBy=multi-user.target
`, config.InstallDir, filepath.Join(config.InstallDir, config.AppName), config.AppName, config.AppName)

	servicePath := filepath.Join("/etc/systemd/system", config.AppName+".service")
	return os.WriteFile(servicePath, []byte(serviceContent), 0644)
}

func configureNginx(config *Config) error {
	nginxContent := fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    location / {
        proxy_pass http://127.0.0.1:%s;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_cache_bypass $http_upgrade;
    }
}
`, config.Domain, config.Port)

	nginxPath := filepath.Join("/etc/nginx/sites-available", config.Domain)
	if err := os.WriteFile(nginxPath, []byte(nginxContent), 0644); err != nil {
		return err
	}

	enabledPath := filepath.Join("/etc/nginx/sites-enabled", config.Domain)
	os.Remove(enabledPath)
	return os.Symlink(nginxPath, enabledPath)
}

func reloadServices(config *Config) error {
	exec.Command("systemctl", "daemon-reload").CombinedOutput()
	exec.Command("systemctl", "reload", "nginx").CombinedOutput()
	return nil
}

func startServiceFn(config *Config) error {
	out, err := exec.Command("systemctl", "enable", config.AppName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("enable failed: %s", string(out))
	}

	out, err = exec.Command("systemctl", "start", config.AppName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("start failed: %s", string(out))
	}

	return nil
}

func verifyInstall(config *Config) error {
	out, err := exec.Command("systemctl", "is-active", config.AppName).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "active") {
		return fmt.Errorf("service not running: %s", string(out))
	}

	out, err = exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost:"+config.Port).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "200") {
		return fmt.Errorf("application not responding: %s", string(out))
	}

	return nil
}

func printStep(current, total int, message string) {
	fmt.Printf("\n  %s[%d/%d]%s %s\n", cyan, current, total, reset, message)
}

func printError(msg string) {
	fmt.Printf("  %s✗%s %s\n", red, reset, msg)
}

func printInfo(msg string) {
	fmt.Printf("  %sℹ%s %s\n", yellow, reset, msg)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
