// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package browser

// NewChrome creates a Chrome browser scanner
func NewChrome() *ChromiumBrowser {
	return NewChromiumBrowser("chrome", ChromiumPaths{
		Linux:          ".config/google-chrome",
		Darwin:         "Library/Application Support/Google/Chrome",
		Windows:        "Google\\Chrome\\User Data",
		WindowsAppData: false, // Uses LOCALAPPDATA
	}, true) // Has profiles
}
