// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package browser

import (
	"os"
	"path/filepath"

	"hist_scanner/internal/db"
	"hist_scanner/internal/dto"
	"hist_scanner/internal/platform"
)

// SafariBrowser implements the Browser interface for Safari (macOS only)
type SafariBrowser struct{}

// NewSafari creates a Safari browser scanner
func NewSafari() *SafariBrowser {
	return &SafariBrowser{}
}

// Name returns the browser name
func (s *SafariBrowser) Name() string {
	return "safari"
}

// FindProfiles returns Safari profiles for a given user
// Safari doesn't have multiple profiles, so we return a single "Default" profile
func (s *SafariBrowser) FindProfiles(user platform.User) ([]Profile, error) {
	// Safari is macOS only
	if platform.CurrentOS() != platform.Darwin {
		return nil, nil
	}

	historyPath := filepath.Join(user.HomeDir, "Library/Safari/History.db")

	// Check if History.db exists
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return nil, nil
	}

	return []Profile{
		{
			Name: "Default",
			Path: filepath.Join(user.HomeDir, "Library/Safari"),
		},
	}, nil
}

// GetHistory extracts history entries from Safari since the given timestamp
func (s *SafariBrowser) GetHistory(profile Profile, sinceTimestamp int64) ([]dto.VisitedSite, error) {
	historyPath := filepath.Join(profile.Path, "History.db")

	database, err := db.Open(historyPath)
	if err != nil {
		return nil, err
	}
	defer database.Close()

	// Safari uses "Mac Absolute Time" (seconds since 2001-01-01 00:00:00 UTC)
	// Unix epoch to Mac epoch difference: 978307200 seconds
	var safariTimestamp float64
	if sinceTimestamp > 0 {
		// Convert Unix ms to Safari timestamp (seconds since 2001-01-01)
		safariTimestamp = float64(sinceTimestamp)/1000.0 - 978307200.0
	}

	query := `
		SELECT hi.url, hv.visit_time
		FROM history_visits hv
		JOIN history_items hi ON hv.history_item = hi.id
		WHERE hv.visit_time > ?
		ORDER BY hv.visit_time ASC
	`

	rows, err := database.Query(query, safariTimestamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []dto.VisitedSite
	for rows.Next() {
		var url string
		var visitTime float64

		if err := rows.Scan(&url, &visitTime); err != nil {
			continue
		}

		// Convert Safari timestamp back to Unix milliseconds
		unixMs := int64((visitTime + 978307200.0) * 1000)

		sites = append(sites, dto.VisitedSite{
			URL:       url,
			Timestamp: unixMs,
		})
	}

	return sites, rows.Err()
}
