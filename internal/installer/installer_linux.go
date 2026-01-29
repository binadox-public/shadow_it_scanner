//go:build linux

// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"

	"hist_scanner/internal/config"
)

// newPlatformInstaller creates the Linux installer
func newPlatformInstaller() (Installer, error) {
	return &LinuxInstaller{}, nil
}

const (
	systemdServicePath = "/etc/systemd/system/hist_scanner.service"
	systemdTimerPath   = "/etc/systemd/system/hist_scanner.timer"
)

// LinuxInstaller handles installation on Linux using systemd
type LinuxInstaller struct{}

const serviceTemplate = `[Unit]
Description=Browser History Scanner
After=network.target

[Service]
Type=oneshot
ExecStart={{.BinaryPath}} run --config {{.ConfigPath}}
User={{.User}}
`

const timerTemplate = `[Unit]
Description=Run Browser History Scanner periodically

[Timer]
OnBootSec=5min
OnUnitActiveSec={{.Interval}}
Persistent=true

[Install]
WantedBy=timers.target
`

// Install installs the scanner as a systemd service
func (i *LinuxInstaller) Install(cfg *config.Config, interval time.Duration, runAsUser string) error {
	// Check for root
	if os.Getuid() != 0 {
		return fmt.Errorf("installation requires root privileges")
	}

	paths := GetInstallPaths()

	// Default user to root
	if runAsUser == "" {
		runAsUser = "root"
	}

	// Copy binary
	if err := CopyBinary(paths.BinaryPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Write config
	if err := WriteConfig(cfg, paths.ConfigPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Generate and write service file
	serviceData := struct {
		BinaryPath string
		ConfigPath string
		User       string
	}{
		BinaryPath: paths.BinaryPath,
		ConfigPath: paths.ConfigPath,
		User:       runAsUser,
	}

	serviceTmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse service template: %w", err)
	}

	serviceFile, err := os.Create(systemdServicePath)
	if err != nil {
		return fmt.Errorf("failed to create service file: %w", err)
	}
	defer serviceFile.Close()

	if err := serviceTmpl.Execute(serviceFile, serviceData); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Generate and write timer file
	timerData := struct {
		Interval string
	}{
		Interval: formatDuration(interval),
	}

	timerTmpl, err := template.New("timer").Parse(timerTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse timer template: %w", err)
	}

	timerFile, err := os.Create(systemdTimerPath)
	if err != nil {
		return fmt.Errorf("failed to create timer file: %w", err)
	}
	defer timerFile.Close()

	if err := timerTmpl.Execute(timerFile, timerData); err != nil {
		return fmt.Errorf("failed to write timer file: %w", err)
	}

	// Reload systemd and enable timer
	commands := [][]string{
		{"systemctl", "daemon-reload"},
		{"systemctl", "enable", "hist_scanner.timer"},
		{"systemctl", "start", "hist_scanner.timer"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to run %v: %w\n%s", args, err, output)
		}
	}

	return nil
}

// Uninstall removes the scanner from systemd
func (i *LinuxInstaller) Uninstall() error {
	// Check for root
	if os.Getuid() != 0 {
		return fmt.Errorf("uninstallation requires root privileges")
	}

	paths := GetInstallPaths()

	// Stop and disable timer
	commands := [][]string{
		{"systemctl", "stop", "hist_scanner.timer"},
		{"systemctl", "disable", "hist_scanner.timer"},
		{"systemctl", "daemon-reload"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Run() // Ignore errors
	}

	// Remove files
	RemoveFile(systemdTimerPath)
	RemoveFile(systemdServicePath)
	RemoveFile(paths.BinaryPath)
	RemoveFile(paths.ConfigPath)
	RemoveDir(filepath.Dir(paths.ConfigPath))

	return nil
}

// IsInstalled checks if the scanner is installed
func (i *LinuxInstaller) IsInstalled() bool {
	_, err := os.Stat(systemdTimerPath)
	return err == nil
}

// formatDuration formats a duration for systemd (e.g., "24h" -> "24h", "6h" -> "6h")
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	if hours >= 24 && hours%24 == 0 {
		return fmt.Sprintf("%dd", hours/24)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(d.Minutes())
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
