// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package browser

// NewEdge creates an Edge browser scanner
func NewEdge() *ChromiumBrowser {
	return NewChromiumBrowser("edge", ChromiumPaths{
		Linux:          ".config/microsoft-edge",
		Darwin:         "Library/Application Support/Microsoft Edge",
		Windows:        "Microsoft\\Edge\\User Data",
		WindowsAppData: false, // Uses LOCALAPPDATA
	}, true) // Has profiles
}
