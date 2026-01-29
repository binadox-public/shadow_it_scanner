// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"hist_scanner/internal/platform"
)

// Manager handles state persistence for scan timestamps
type Manager struct {
	stateFile string
	data      map[string]int64 // key: "user/browser/profile", value: last timestamp (Unix ms)
	mu        sync.RWMutex
}

// stateFileName is the hidden file name for per-profile state
const stateFileName = ".hist_scanner_state"

// NewManager creates a new state manager
// If stateFile is empty, uses automatic location resolution
func NewManager(stateFile string) *Manager {
	return &Manager{
		stateFile: stateFile,
		data:      make(map[string]int64),
	}
}

// Load loads state from file
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := m.resolveStatePath()
	if path == "" {
		// No state file found, start fresh
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No state yet, start fresh
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, &m.data); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	m.stateFile = path
	return nil
}

// Save persists state to file
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path := m.stateFile
	if path == "" {
		path = m.findWritablePath()
		if path == "" {
			// Can't write anywhere, silently continue
			return nil
		}
		m.stateFile = path
	}

	data, err := json.MarshalIndent(m.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// GetLastTimestamp returns the last scan timestamp for a user/browser/profile
func (m *Manager) GetLastTimestamp(username, browserName, profileName string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := makeKey(username, browserName, profileName)
	return m.data[key]
}

// SetLastTimestamp sets the last scan timestamp for a user/browser/profile
func (m *Manager) SetLastTimestamp(username, browserName, profileName string, timestamp int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := makeKey(username, browserName, profileName)
	m.data[key] = timestamp
}

// makeKey creates a state key from user/browser/profile
func makeKey(username, browserName, profileName string) string {
	return fmt.Sprintf("%s/%s/%s", username, browserName, profileName)
}

// resolveStatePath finds an existing state file
func (m *Manager) resolveStatePath() string {
	// 1. Explicit path from config/flag
	if m.stateFile != "" {
		if _, err := os.Stat(m.stateFile); err == nil {
			return m.stateFile
		}
	}

	// 2. Central config location
	centralPath := getCentralStatePath()
	if centralPath != "" {
		if _, err := os.Stat(centralPath); err == nil {
			return centralPath
		}
	}

	// 3. Temp location
	tempPath := getTempStatePath()
	if _, err := os.Stat(tempPath); err == nil {
		return tempPath
	}

	return ""
}

// findWritablePath finds a location where we can write state
func (m *Manager) findWritablePath() string {
	// 1. Explicit path from config/flag
	if m.stateFile != "" {
		return m.stateFile
	}

	// 2. Central config location
	centralPath := getCentralStatePath()
	if centralPath != "" && canWrite(filepath.Dir(centralPath)) {
		return centralPath
	}

	// 3. Temp location
	return getTempStatePath()
}

// getCentralStatePath returns the central state file path for the current OS
func getCentralStatePath() string {
	switch platform.CurrentOS() {
	case platform.Linux:
		// Check if running as root
		if os.Getuid() == 0 {
			return "/var/lib/hist_scanner/state.json"
		}
		home, _ := os.UserHomeDir()
		if home != "" {
			return filepath.Join(home, ".config/hist_scanner/state.json")
		}

	case platform.Windows:
		programData := os.Getenv("PROGRAMDATA")
		if programData == "" {
			programData = "C:\\ProgramData"
		}
		return filepath.Join(programData, "hist_scanner", "state.json")

	case platform.Darwin:
		home, _ := os.UserHomeDir()
		if home != "" {
			return filepath.Join(home, "Library/Application Support/hist_scanner/state.json")
		}
	}

	return ""
}

// getTempStatePath returns the temp state file path
func getTempStatePath() string {
	return filepath.Join(os.TempDir(), "hist_scanner_state.json")
}

// canWrite checks if we can write to a directory
func canWrite(dir string) bool {
	// Try to create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}

	// Try to create a temp file
	testFile := filepath.Join(dir, ".write_test")
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(testFile)
	return true
}

// GetStateFilePath returns the current state file path
func (m *Manager) GetStateFilePath() string {
	return m.stateFile
}

// GetAllEntries returns all state entries (for debugging)
func (m *Manager) GetAllEntries() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]int64, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result
}
