// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"hist_scanner/internal/config"
	"hist_scanner/internal/platform"
)

// Installer handles installation and uninstallation of the scanner
type Installer interface {
	Install(cfg *config.Config, interval time.Duration, runAsUser string) error
	Uninstall() error
	IsInstalled() bool
}

// New creates a platform-specific installer
func New() (Installer, error) {
	return newPlatformInstaller()
}

// InstallPaths contains the installation paths for the current platform
type InstallPaths struct {
	BinaryPath string
	ConfigPath string
}

// GetInstallPaths returns the installation paths for the current platform
func GetInstallPaths() InstallPaths {
	switch platform.CurrentOS() {
	case platform.Linux:
		return InstallPaths{
			BinaryPath: "/usr/local/bin/hist_scanner",
			ConfigPath: "/etc/hist_scanner/config.yaml",
		}
	case platform.Windows:
		programFiles := os.Getenv("PROGRAMFILES")
		if programFiles == "" {
			programFiles = "C:\\Program Files"
		}
		programData := os.Getenv("PROGRAMDATA")
		if programData == "" {
			programData = "C:\\ProgramData"
		}
		return InstallPaths{
			BinaryPath: filepath.Join(programFiles, "hist_scanner", "hist_scanner.exe"),
			ConfigPath: filepath.Join(programData, "hist_scanner", "config.yaml"),
		}
	case platform.Darwin:
		return InstallPaths{
			BinaryPath: "/usr/local/bin/hist_scanner",
			ConfigPath: "/etc/hist_scanner/config.yaml",
		}
	default:
		return InstallPaths{}
	}
}

// CopyBinary copies the current executable to the installation path
func CopyBinary(dstPath string) error {
	// Get current executable path
	srcPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable: %w", err)
	}

	// Resolve symlinks
	srcPath, err = filepath.EvalSymlinks(srcPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Create destination directory
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dstDir, err)
	}

	// Read source binary
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source binary: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(dstPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}

	return nil
}

// WriteConfig writes the configuration file
func WriteConfig(cfg *config.Config, configPath string) error {
	// Create config directory
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Convert to YAML-friendly structure
	configData := struct {
		ServerURL   string `yaml:"server_url"`
		APIKey      string `yaml:"api_key"`
		InitialDays int    `yaml:"initial_days"`
		Timeout     string `yaml:"timeout"`
		ChunkSizeKB int    `yaml:"chunk_size_kb"`
		Compress    bool   `yaml:"compress"`
		StateFile   string `yaml:"state_file,omitempty"`
		LogFile     string `yaml:"log_file,omitempty"`
		Source      string `yaml:"source,omitempty"`
	}{
		ServerURL:   cfg.ServerURL,
		APIKey:      cfg.APIKey,
		InitialDays: cfg.InitialDays,
		Timeout:     cfg.Timeout.String(),
		ChunkSizeKB: cfg.ChunkSizeKB,
		Compress:    cfg.Compress,
		StateFile:   cfg.StateFile,
		LogFile:     cfg.LogFile,
		Source:      cfg.Source,
	}

	data, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with restricted permissions (contains API key)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// RemoveFile removes a file if it exists
func RemoveFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// RemoveDir removes a directory if it's empty
func RemoveDir(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		// Ignore "directory not empty" errors
		return nil
	}
	return nil
}
