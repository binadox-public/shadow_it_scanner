//go:build windows

// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package platform

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// getAllUsersImpl returns all users on Windows by scanning C:\Users directory
func getAllUsersImpl() ([]User, error) {
	var users []User

	usersDir := os.Getenv("SYSTEMDRIVE") + "\\Users"
	if usersDir == "\\Users" {
		usersDir = "C:\\Users"
	}

	entries, err := os.ReadDir(usersDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read Users directory: %w", err)
	}

	// System directories to skip
	skipDirs := map[string]bool{
		"Public":        true,
		"Default":       true,
		"Default User":  true,
		"All Users":     true,
		"desktop.ini":   true,
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if skipDirs[name] {
			continue
		}

		// Skip hidden directories
		if strings.HasPrefix(name, ".") {
			continue
		}

		homeDir := filepath.Join(usersDir, name)

		// Verify it's a valid user profile (has NTUSER.DAT)
		ntUserPath := filepath.Join(homeDir, "NTUSER.DAT")
		if _, err := os.Stat(ntUserPath); os.IsNotExist(err) {
			continue
		}

		users = append(users, User{
			Username: name,
			HomeDir:  homeDir,
			UID:      "", // Windows doesn't use numeric UIDs in the same way
		})
	}

	return users, nil
}

// getCurrentUserImpl returns the current user on Windows
func getCurrentUserImpl() (*User, error) {
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// On Windows, Username might be DOMAIN\username, extract just the username
	username := u.Username
	if idx := strings.LastIndex(username, "\\"); idx != -1 {
		username = username[idx+1:]
	}

	// Ensure home directory is absolute
	homeDir := u.HomeDir
	if !filepath.IsAbs(homeDir) {
		homeDir, err = filepath.Abs(homeDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve home directory: %w", err)
		}
	}

	return &User{
		Username: username,
		HomeDir:  homeDir,
		UID:      u.Uid,
	}, nil
}
