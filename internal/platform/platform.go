// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package platform

import (
	"runtime"
)

// OS represents the operating system type
type OS string

const (
	Linux   OS = "linux"
	Windows OS = "windows"
	Darwin  OS = "darwin"
)

// User represents a system user with their home directory
type User struct {
	Username string
	HomeDir  string
	UID      string
}

// CurrentOS returns the current operating system
func CurrentOS() OS {
	return OS(runtime.GOOS)
}

// IsSupported checks if the current OS is supported
func IsSupported() bool {
	switch CurrentOS() {
	case Linux, Windows, Darwin:
		return true
	default:
		return false
	}
}
