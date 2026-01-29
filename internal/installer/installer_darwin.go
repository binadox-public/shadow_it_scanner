//go:build darwin

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

// newPlatformInstaller creates the macOS installer
func newPlatformInstaller() (Installer, error) {
	return &DarwinInstaller{}, nil
}

const (
	launchdPlistPath = "/Library/LaunchDaemons/com.binadox.hist_scanner.plist"
	launchdLabel     = "com.binadox.hist_scanner"
)

// DarwinInstaller handles installation on macOS using launchd
type DarwinInstaller struct{}

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>run</string>
        <string>--config</string>
        <string>{{.ConfigPath}}</string>
    </array>
    <key>StartInterval</key>
    <integer>{{.IntervalSeconds}}</integer>
    <key>RunAtLoad</key>
    <true/>
    <key>UserName</key>
    <string>{{.User}}</string>
</dict>
</plist>
`

// Install installs the scanner as a launchd service
func (i *DarwinInstaller) Install(cfg *config.Config, interval time.Duration, runAsUser string) error {
	// Check for root
	if os.Getuid() != 0 {
		return fmt.Errorf("installation requires root privileges (run with sudo)")
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

	// Generate and write plist file
	plistData := struct {
		Label           string
		BinaryPath      string
		ConfigPath      string
		IntervalSeconds int
		User            string
	}{
		Label:           launchdLabel,
		BinaryPath:      paths.BinaryPath,
		ConfigPath:      paths.ConfigPath,
		IntervalSeconds: int(interval.Seconds()),
		User:            runAsUser,
	}

	plistTmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse plist template: %w", err)
	}

	plistFile, err := os.Create(launchdPlistPath)
	if err != nil {
		return fmt.Errorf("failed to create plist file: %w", err)
	}
	defer plistFile.Close()

	if err := plistTmpl.Execute(plistFile, plistData); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	// Set correct permissions
	if err := os.Chmod(launchdPlistPath, 0644); err != nil {
		return fmt.Errorf("failed to set plist permissions: %w", err)
	}

	// Load the service
	cmd := exec.Command("launchctl", "load", launchdPlistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to load launchd service: %w\n%s", err, output)
	}

	return nil
}

// Uninstall removes the scanner from launchd
func (i *DarwinInstaller) Uninstall() error {
	// Check for root
	if os.Getuid() != 0 {
		return fmt.Errorf("uninstallation requires root privileges (run with sudo)")
	}

	paths := GetInstallPaths()

	// Unload the service
	exec.Command("launchctl", "unload", launchdPlistPath).Run()

	// Remove files
	RemoveFile(launchdPlistPath)
	RemoveFile(paths.BinaryPath)
	RemoveFile(paths.ConfigPath)
	RemoveDir(filepath.Dir(paths.ConfigPath))

	return nil
}

// IsInstalled checks if the scanner is installed
func (i *DarwinInstaller) IsInstalled() bool {
	_, err := os.Stat(launchdPlistPath)
	return err == nil
}
