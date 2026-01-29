// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package browser

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"hist_scanner/internal/db"
	"hist_scanner/internal/dto"
	"hist_scanner/internal/platform"
)

// FirefoxBrowser implements the Browser interface for Firefox
type FirefoxBrowser struct{}

// NewFirefox creates a Firefox browser scanner
func NewFirefox() *FirefoxBrowser {
	return &FirefoxBrowser{}
}

// Name returns the browser name
func (f *FirefoxBrowser) Name() string {
	return "firefox"
}

// FindProfiles returns all Firefox profiles for a given user
func (f *FirefoxBrowser) FindProfiles(user platform.User) ([]Profile, error) {
	profilesDir := f.getProfilesDir(user)
	if profilesDir == "" {
		return nil, nil
	}

	// Check if profiles directory exists
	if _, err := os.Stat(profilesDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Parse profiles.ini to find profile directories
	profilesIni := filepath.Join(profilesDir, "profiles.ini")
	profiles, err := f.parseProfilesIni(profilesIni, profilesDir)
	if err != nil {
		// Fallback: scan directory for profile folders
		return f.scanForProfiles(profilesDir)
	}

	return profiles, nil
}

// parseProfilesIni parses Firefox's profiles.ini file
func (f *FirefoxBrowser) parseProfilesIni(iniPath, profilesDir string) ([]Profile, error) {
	file, err := os.Open(iniPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var profiles []Profile
	var currentProfile struct {
		name       string
		path       string
		isRelative bool
	}

	scanner := bufio.NewScanner(file)
	inProfile := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// New section
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// Save previous profile if valid
			if inProfile && currentProfile.path != "" {
				profilePath := currentProfile.path
				if currentProfile.isRelative {
					profilePath = filepath.Join(profilesDir, currentProfile.path)
				}

				// Verify places.sqlite exists
				placesPath := filepath.Join(profilePath, "places.sqlite")
				if _, err := os.Stat(placesPath); err == nil {
					name := currentProfile.name
					if name == "" {
						name = filepath.Base(profilePath)
					}
					profiles = append(profiles, Profile{
						Name: name,
						Path: profilePath,
					})
				}
			}

			// Start new section
			section := line[1 : len(line)-1]
			inProfile = strings.HasPrefix(section, "Profile")
			currentProfile = struct {
				name       string
				path       string
				isRelative bool
			}{}
			continue
		}

		if !inProfile {
			continue
		}

		// Parse key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Name":
			currentProfile.name = value
		case "Path":
			currentProfile.path = value
		case "IsRelative":
			currentProfile.isRelative = value == "1"
		}
	}

	// Don't forget last profile
	if inProfile && currentProfile.path != "" {
		profilePath := currentProfile.path
		if currentProfile.isRelative {
			profilePath = filepath.Join(profilesDir, currentProfile.path)
		}

		placesPath := filepath.Join(profilePath, "places.sqlite")
		if _, err := os.Stat(placesPath); err == nil {
			name := currentProfile.name
			if name == "" {
				name = filepath.Base(profilePath)
			}
			profiles = append(profiles, Profile{
				Name: name,
				Path: profilePath,
			})
		}
	}

	return profiles, scanner.Err()
}

// scanForProfiles scans directory for Firefox profile folders (fallback)
func (f *FirefoxBrowser) scanForProfiles(profilesDir string) ([]Profile, error) {
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil, err
	}

	var profiles []Profile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Firefox profiles typically have format: xxxxxxxx.ProfileName
		name := entry.Name()
		profilePath := filepath.Join(profilesDir, name)

		// Check if places.sqlite exists
		placesPath := filepath.Join(profilePath, "places.sqlite")
		if _, err := os.Stat(placesPath); err == nil {
			// Extract profile name from directory name
			displayName := name
			if idx := strings.Index(name, "."); idx != -1 {
				displayName = name[idx+1:]
			}
			profiles = append(profiles, Profile{
				Name: displayName,
				Path: profilePath,
			})
		}
	}

	return profiles, nil
}

// GetHistory extracts history entries from a Firefox profile since the given timestamp
func (f *FirefoxBrowser) GetHistory(profile Profile, sinceTimestamp int64) ([]dto.VisitedSite, error) {
	placesPath := filepath.Join(profile.Path, "places.sqlite")

	database, err := db.Open(placesPath)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	// Firefox stores timestamps as microseconds since Unix epoch
	firefoxTimestamp := sinceTimestamp * 1000 // Convert ms to microseconds

	query := `
		SELECT url, last_visit_date
		FROM moz_places
		WHERE last_visit_date > ?
		  AND last_visit_date IS NOT NULL
		ORDER BY last_visit_date ASC
	`

	rows, err := database.Query(query, firefoxTimestamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []dto.VisitedSite
	for rows.Next() {
		var url string
		var lastVisitDate int64

		if err := rows.Scan(&url, &lastVisitDate); err != nil {
			continue
		}

		// Convert microseconds to milliseconds
		unixMs := lastVisitDate / 1000

		sites = append(sites, dto.VisitedSite{
			URL:       url,
			Timestamp: unixMs,
		})
	}

	return sites, rows.Err()
}

// getProfilesDir returns the Firefox profiles directory for a user
func (f *FirefoxBrowser) getProfilesDir(user platform.User) string {
	switch platform.CurrentOS() {
	case platform.Linux:
		return filepath.Join(user.HomeDir, ".mozilla/firefox")

	case platform.Darwin:
		return filepath.Join(user.HomeDir, "Library/Application Support/Firefox/Profiles")

	case platform.Windows:
		// Firefox uses APPDATA on Windows
		appData := filepath.Join(user.HomeDir, "AppData", "Roaming")
		return filepath.Join(appData, "Mozilla", "Firefox", "Profiles")

	default:
		return ""
	}
}
