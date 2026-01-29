// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package browser

import (
	"hist_scanner/internal/dto"
	"hist_scanner/internal/platform"
)

// Profile represents a browser profile
type Profile struct {
	Name string // Profile name (e.g., "Default", "Profile 1")
	Path string // Full path to profile directory
}

// Browser defines the interface for all browser implementations
type Browser interface {
	// Name returns the browser name (e.g., "chrome", "firefox")
	Name() string

	// FindProfiles returns all profiles for a given user
	FindProfiles(user platform.User) ([]Profile, error)

	// GetHistory extracts history entries from a profile since the given timestamp
	// timestamp is in Unix milliseconds, 0 means get all history
	GetHistory(profile Profile, sinceTimestamp int64) ([]dto.VisitedSite, error)
}

// All returns all supported browsers
func All() []Browser {
	return []Browser{
		NewChrome(),
		NewEdge(),
		NewOpera(),
		NewOperaGX(),
		NewVivaldi(),
		NewFirefox(),
		NewSafari(),
	}
}

// ByName returns a browser by name, or nil if not found
func ByName(name string) Browser {
	for _, b := range All() {
		if b.Name() == name {
			return b
		}
	}
	return nil
}

// SupportedBrowserNames returns names of all supported browsers
func SupportedBrowserNames() []string {
	browsers := All()
	names := make([]string, len(browsers))
	for i, b := range browsers {
		names[i] = b.Name()
	}
	return names
}
