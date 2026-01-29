// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package browser

// NewOpera creates an Opera browser scanner
func NewOpera() *ChromiumBrowser {
	return NewChromiumBrowser("opera", ChromiumPaths{
		Linux:          ".config/opera",
		Darwin:         "Library/Application Support/com.operasoftware.Opera",
		Windows:        "Opera Software\\Opera Stable",
		WindowsAppData: true, // Uses APPDATA
	}, false) // No profiles like Chrome
}

// NewOperaGX creates an Opera GX browser scanner
func NewOperaGX() *ChromiumBrowser {
	return NewChromiumBrowser("opera-gx", ChromiumPaths{
		Linux:          "", // Opera GX not available on Linux
		Darwin:         "Library/Application Support/com.operasoftware.OperaGX",
		Windows:        "Opera Software\\Opera GX Stable",
		WindowsAppData: true, // Uses APPDATA
	}, false) // No profiles
}
