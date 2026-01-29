// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package platform

// GetAllUsers returns all users on the system with home directories
// This is implemented per-platform in users_*.go files
func GetAllUsers() ([]User, error) {
	return getAllUsersImpl()
}

// GetCurrentUser returns the current user
func GetCurrentUser() (*User, error) {
	return getCurrentUserImpl()
}
