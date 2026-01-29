//go:build windows

// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"hist_scanner/internal/config"
)

// newPlatformInstaller creates the Windows installer
func newPlatformInstaller() (Installer, error) {
	return &WindowsInstaller{}, nil
}

const taskName = "BrowserHistoryScanner"

// WindowsInstaller handles installation on Windows using Task Scheduler
type WindowsInstaller struct{}

// Install installs the scanner as a scheduled task
func (i *WindowsInstaller) Install(cfg *config.Config, interval time.Duration, runAsUser string) error {
	paths := GetInstallPaths()

	// Default user to SYSTEM
	if runAsUser == "" {
		runAsUser = "SYSTEM"
	}

	// Create directories
	binaryDir := filepath.Dir(paths.BinaryPath)
	if err := os.MkdirAll(binaryDir, 0755); err != nil {
		return fmt.Errorf("failed to create binary directory: %w", err)
	}

	// Copy binary
	if err := CopyBinary(paths.BinaryPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Write config
	if err := WriteConfig(cfg, paths.ConfigPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Delete existing task if present
	exec.Command("schtasks", "/delete", "/tn", taskName, "/f").Run()

	// Create scheduled task
	// Build command for task
	taskCmd := fmt.Sprintf(`"%s" run --config "%s"`, paths.BinaryPath, paths.ConfigPath)

	// Calculate repetition interval
	intervalMinutes := int(interval.Minutes())
	if intervalMinutes < 1 {
		intervalMinutes = 1
	}

	// Use schtasks to create the task
	// For intervals > 24h, use daily schedule
	// For intervals <= 24h, use repetition
	var args []string
	if interval >= 24*time.Hour {
		days := int(interval.Hours() / 24)
		if days < 1 {
			days = 1
		}
		args = []string{
			"/create",
			"/tn", taskName,
			"/tr", taskCmd,
			"/sc", "daily",
			"/mo", strconv.Itoa(days),
			"/ru", runAsUser,
			"/rl", "HIGHEST",
			"/f",
		}
	} else {
		args = []string{
			"/create",
			"/tn", taskName,
			"/tr", taskCmd,
			"/sc", "minute",
			"/mo", strconv.Itoa(intervalMinutes),
			"/ru", runAsUser,
			"/rl", "HIGHEST",
			"/f",
		}
	}

	cmd := exec.Command("schtasks", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create scheduled task: %w\n%s", err, output)
	}

	return nil
}

// Uninstall removes the scanner from Task Scheduler
func (i *WindowsInstaller) Uninstall() error {
	paths := GetInstallPaths()

	// Delete scheduled task
	cmd := exec.Command("schtasks", "/delete", "/tn", taskName, "/f")
	cmd.Run() // Ignore errors

	// Remove files
	RemoveFile(paths.BinaryPath)
	RemoveFile(paths.ConfigPath)
	RemoveDir(filepath.Dir(paths.BinaryPath))
	RemoveDir(filepath.Dir(paths.ConfigPath))

	return nil
}

// IsInstalled checks if the scanner is installed
func (i *WindowsInstaller) IsInstalled() bool {
	cmd := exec.Command("schtasks", "/query", "/tn", taskName)
	return cmd.Run() == nil
}
