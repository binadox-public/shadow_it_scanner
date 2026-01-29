// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package browser

import (
	"os"
	"path/filepath"
	"strings"

	"hist_scanner/internal/db"
	"hist_scanner/internal/dto"
	"hist_scanner/internal/platform"
)

// ChromiumPaths defines paths for a Chromium-based browser on each platform
type ChromiumPaths struct {
	Linux   string // Path relative to home dir on Linux
	Darwin  string // Path relative to home dir on macOS
	Windows string // Path relative to appropriate Windows folder
	// WindowsAppData indicates if Windows path is relative to APPDATA (true) or LOCALAPPDATA (false)
	WindowsAppData bool
}

// ChromiumBrowser is a base implementation for Chromium-based browsers
type ChromiumBrowser struct {
	name  string
	paths ChromiumPaths
	// hasProfiles indicates if this browser supports multiple profiles
	// Opera doesn't have profiles like Chrome does
	hasProfiles bool
}

// NewChromiumBrowser creates a new Chromium-based browser
func NewChromiumBrowser(name string, paths ChromiumPaths, hasProfiles bool) *ChromiumBrowser {
	return &ChromiumBrowser{
		name:        name,
		paths:       paths,
		hasProfiles: hasProfiles,
	}
}

// Name returns the browser name
func (c *ChromiumBrowser) Name() string {
	return c.name
}

// FindProfiles returns all profiles for a given user
func (c *ChromiumBrowser) FindProfiles(user platform.User) ([]Profile, error) {
	baseDir := c.getBaseDir(user)
	if baseDir == "" {
		return nil, nil
	}

	// Check if base directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil, nil
	}

	var profiles []Profile

	if c.hasProfiles {
		// Look for Default and Profile N directories
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			return nil, nil
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			name := entry.Name()
			// Check for "Default" or "Profile N" directories
			if name == "Default" || strings.HasPrefix(name, "Profile ") {
				profilePath := filepath.Join(baseDir, name)
				historyPath := filepath.Join(profilePath, "History")

				// Only include if History file exists
				if _, err := os.Stat(historyPath); err == nil {
					profiles = append(profiles, Profile{
						Name: name,
						Path: profilePath,
					})
				}
			}
		}
	} else {
		// No profiles - check if History file exists in base dir
		historyPath := filepath.Join(baseDir, "History")
		if _, err := os.Stat(historyPath); err == nil {
			profiles = append(profiles, Profile{
				Name: "Default",
				Path: baseDir,
			})
		}
	}

	return profiles, nil
}

// GetHistory extracts history entries from a profile since the given timestamp
func (c *ChromiumBrowser) GetHistory(profile Profile, sinceTimestamp int64) ([]dto.VisitedSite, error) {
	historyPath := filepath.Join(profile.Path, "History")

	database, err := db.Open(historyPath)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	// Convert Unix milliseconds to Chromium timestamp (microseconds since 1601-01-01)
	// Chromium epoch: 1601-01-01 00:00:00 UTC
	// Unix epoch: 1970-01-01 00:00:00 UTC
	// Difference: 11644473600 seconds
	var chromiumTimestamp int64
	if sinceTimestamp > 0 {
		// Convert ms to microseconds, then add epoch difference
		chromiumTimestamp = (sinceTimestamp * 1000) + (11644473600 * 1000000)
	}

	query := `
		SELECT url, last_visit_time
		FROM urls
		WHERE last_visit_time > ?
		ORDER BY last_visit_time ASC
	`

	rows, err := database.Query(query, chromiumTimestamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []dto.VisitedSite
	for rows.Next() {
		var url string
		var lastVisitTime int64

		if err := rows.Scan(&url, &lastVisitTime); err != nil {
			continue
		}

		// Convert Chromium timestamp back to Unix milliseconds
		unixMs := (lastVisitTime - (11644473600 * 1000000)) / 1000

		sites = append(sites, dto.VisitedSite{
			URL:       url,
			Timestamp: unixMs,
		})
	}

	return sites, rows.Err()
}

// getBaseDir returns the base directory for browser data
func (c *ChromiumBrowser) getBaseDir(user platform.User) string {
	switch platform.CurrentOS() {
	case platform.Linux:
		if c.paths.Linux == "" {
			return ""
		}
		return filepath.Join(user.HomeDir, c.paths.Linux)

	case platform.Darwin:
		if c.paths.Darwin == "" {
			return ""
		}
		return filepath.Join(user.HomeDir, c.paths.Darwin)

	case platform.Windows:
		if c.paths.Windows == "" {
			return ""
		}
		var baseEnv string
		if c.paths.WindowsAppData {
			baseEnv = "APPDATA"
		} else {
			baseEnv = "LOCALAPPDATA"
		}
		// For non-current users, construct the path manually
		if user.HomeDir != "" {
			var appDataPath string
			if c.paths.WindowsAppData {
				appDataPath = filepath.Join(user.HomeDir, "AppData", "Roaming")
			} else {
				appDataPath = filepath.Join(user.HomeDir, "AppData", "Local")
			}
			return filepath.Join(appDataPath, c.paths.Windows)
		}
		// Fallback for current user
		base := os.Getenv(baseEnv)
		if base == "" {
			return ""
		}
		return filepath.Join(base, c.paths.Windows)

	default:
		return ""
	}
}
