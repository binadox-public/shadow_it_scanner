// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Discovery configuration constants
const (
	// DiscoveryURL is the endpoint for auto-discovery configuration.
	// The hostname "binadox.config" must be resolvable via DNS or /etc/hosts.
	//
	// Example setup:
	//   echo "192.168.1.100 binadox.config" >> /etc/hosts
	//
	// The discovery server should respond with JSON:
	//   {"url": "https://server.example.com/api/1/organizations/discovery/store-events", "token": "..."}
	//
	// Note: The scanner will append "/visited-sites" to the URL automatically.
	DiscoveryURL = "http://binadox.config:3000"

	// DiscoveryTimeout is the maximum time to wait for discovery response.
	// Kept short to avoid delaying startup if discovery server is unavailable.
	DiscoveryTimeout = 2 * time.Second

	// VisitedSitesEndpoint is appended to the discovered URL
	VisitedSitesEndpoint = "/visited-sites"
)

// discoveryResponse represents the JSON response from the discovery server
type discoveryResponse struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// DiscoveryResult contains the configuration obtained from auto-discovery
type DiscoveryResult struct {
	ServerURL string
	APIKey    string
}

// Discover attempts to fetch configuration from the discovery server.
// Returns nil if discovery fails or server is unavailable.
//
// The discovery server must be accessible at http://binadox.config:3000
// and return a JSON response with "url" and "token" fields.
func Discover() *DiscoveryResult {
	client := &http.Client{
		Timeout: DiscoveryTimeout,
	}

	resp, err := client.Get(DiscoveryURL)
	if err != nil {
		// Discovery server unavailable - this is expected in many deployments
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var discovery discoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil
	}

	if discovery.URL == "" || discovery.Token == "" {
		return nil
	}

	// Append /visited-sites endpoint to the URL
	serverURL := strings.TrimSuffix(discovery.URL, "/") + VisitedSitesEndpoint

	return &DiscoveryResult{
		ServerURL: serverURL,
		APIKey:    discovery.Token,
	}
}

// FormatDiscoveryDocs returns documentation string for discovery setup
func FormatDiscoveryDocs() string {
	return fmt.Sprintf(`Auto-Discovery Configuration
============================
The scanner can automatically discover configuration from a discovery server.

Requirements:
  1. The hostname "binadox.config" must resolve to the discovery server IP.
     Add to /etc/hosts (Linux/macOS) or C:\Windows\System32\drivers\etc\hosts (Windows):
       192.168.1.100 binadox.config

  2. The discovery server must listen on port 3000 and respond to GET / with:
       {"url": "https://your-server/api/1/organizations/discovery/store-events", "token": "your-api-token"}

Discovery URL: %s
Timeout: %s

Priority (highest to lowest):
  1. CLI flags (--server-url, --api-key)
  2. Environment variables (HIST_SCANNER_SERVER_URL, HIST_SCANNER_API_KEY)
  3. Config file (--config)
  4. Auto-discovery
`, DiscoveryURL, DiscoveryTimeout)
}
