//go:build darwin

// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package platform

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

// getAllUsersImpl returns all users on macOS using dscl
func getAllUsersImpl() ([]User, error) {
	var users []User

	// Get list of users via dscl
	cmd := exec.Command("dscl", ".", "-list", "/Users")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list users via dscl: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		username := strings.TrimSpace(scanner.Text())
		if username == "" {
			continue
		}

		// Skip system users (those starting with _)
		if strings.HasPrefix(username, "_") {
			continue
		}

		// Skip common system accounts
		systemUsers := map[string]bool{
			"daemon":  true,
			"nobody":  true,
			"root":    false, // Include root
			"Guest":   true,
		}
		if skip, found := systemUsers[username]; found && skip {
			continue
		}

		// Get user's home directory
		homeDir, err := getUserHomeDir(username)
		if err != nil {
			continue
		}

		// Verify home directory exists
		if _, err := os.Stat(homeDir); os.IsNotExist(err) {
			continue
		}

		// Get UID
		uid, _ := getUserUID(username)

		users = append(users, User{
			Username: username,
			HomeDir:  homeDir,
			UID:      uid,
		})
	}

	return users, nil
}

// getUserHomeDir gets a user's home directory via dscl
func getUserHomeDir(username string) (string, error) {
	cmd := exec.Command("dscl", ".", "-read", "/Users/"+username, "NFSHomeDirectory")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output: "NFSHomeDirectory: /Users/username"
	line := strings.TrimSpace(string(output))
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected dscl output format")
	}

	return strings.TrimSpace(parts[1]), nil
}

// getUserUID gets a user's UID via dscl
func getUserUID(username string) (string, error) {
	cmd := exec.Command("dscl", ".", "-read", "/Users/"+username, "UniqueID")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output: "UniqueID: 501"
	line := strings.TrimSpace(string(output))
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected dscl output format")
	}

	return strings.TrimSpace(parts[1]), nil
}

// getCurrentUserImpl returns the current user on macOS
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
