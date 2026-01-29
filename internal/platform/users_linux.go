//go:build linux

// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package platform

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// getAllUsersImpl returns all users on Linux by parsing /etc/passwd
func getAllUsersImpl() ([]User, error) {
	var users []User

	file, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, fmt.Errorf("failed to open /etc/passwd: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) < 7 {
			continue
		}

		username := fields[0]
		uid := fields[2]
		homeDir := fields[5]
		shell := fields[6]

		// Skip system users (typically UID < 1000) and users with nologin/false shells
		// But include root (UID 0) if it has a valid home
		if !isRealUser(uid, shell) {
			continue
		}

		// Verify home directory exists
		if _, err := os.Stat(homeDir); os.IsNotExist(err) {
			continue
		}

		users = append(users, User{
			Username: username,
			HomeDir:  homeDir,
			UID:      uid,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read /etc/passwd: %w", err)
	}

	return users, nil
}

// isRealUser checks if this is a real user account (not a system service)
func isRealUser(uid, shell string) bool {
	// Exclude nologin shells
	nologinShells := []string{"/usr/sbin/nologin", "/sbin/nologin", "/bin/false", "/usr/bin/false"}
	for _, nologin := range nologinShells {
		if shell == nologin {
			return false
		}
	}

	// Include root
	if uid == "0" {
		return true
	}

	// For regular users, UID should be >= 1000 (configurable via /etc/login.defs)
	// We'll use 1000 as the default threshold
	var uidNum int
	fmt.Sscanf(uid, "%d", &uidNum)
	return uidNum >= 1000
}

// getCurrentUserImpl returns the current user on Linux
func getCurrentUserImpl() (*User, error) {
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
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
		Username: u.Username,
		HomeDir:  homeDir,
		UID:      u.Uid,
	}, nil
}
