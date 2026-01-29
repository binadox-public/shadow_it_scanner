// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package browser

// NewVivaldi creates a Vivaldi browser scanner
func NewVivaldi() *ChromiumBrowser {
	return NewChromiumBrowser("vivaldi", ChromiumPaths{
		Linux:          ".config/vivaldi",
		Darwin:         "Library/Application Support/Vivaldi",
		Windows:        "Vivaldi\\User Data",
		WindowsAppData: false, // Uses LOCALAPPDATA
	}, true) // Has profiles
}
