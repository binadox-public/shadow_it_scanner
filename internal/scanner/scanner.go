// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package scanner

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"hist_scanner/internal/browser"
	"hist_scanner/internal/config"
	"hist_scanner/internal/dto"
	"hist_scanner/internal/platform"
	"hist_scanner/internal/sender"
	"hist_scanner/internal/state"
)

// ExitCode represents the scanner exit status
type ExitCode int

const (
	ExitSuccess         ExitCode = 0 // All browsers/profiles scanned and sent
	ExitPartialFailure  ExitCode = 1 // Some browsers/profiles failed
	ExitCompleteFailure ExitCode = 2 // Nothing sent
)

// Scanner orchestrates the browser history scanning process
type Scanner struct {
	cfg    *config.Config
	state  *state.Manager
	client *sender.Client
	logger *log.Logger
	dryRun bool
}

// ScanResult contains the results of a scan operation
type ScanResult struct {
	UsersScanned    int
	ProfilesScanned int
	EntriesSent     int
	Errors          []string
	ExitCode        ExitCode
}

// New creates a new Scanner instance
func New(cfg *config.Config, dryRun bool) (*Scanner, error) {
	// Set up logger
	var logWriter io.Writer = io.Discard
	if cfg.LogFile != "" {
		if strings.EqualFold(cfg.LogFile, "STDERR") {
			logWriter = os.Stderr
		} else {
			f, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, fmt.Errorf("failed to open log file: %w", err)
			}
			logWriter = f
		}

	}

	logger := log.New(logWriter, "[hist_scanner] ", log.LstdFlags)

	// Initialize state manager
	stateMgr := state.NewManager(cfg.StateFile)
	if err := stateMgr.Load(); err != nil {
		logger.Printf("Warning: failed to load state: %v", err)
	}

	// Initialize HTTP client (nil if dry run)
	var client *sender.Client
	if !dryRun {
		client = sender.NewClient(cfg.ServerURL, cfg.APIKey, cfg.Timeout, cfg.ChunkSizeKB, cfg.Compress)
	}

	return &Scanner{
		cfg:    cfg,
		state:  stateMgr,
		client: client,
		logger: logger,
		dryRun: dryRun,
	}, nil
}

// Run executes the full scan process
func (s *Scanner) Run() *ScanResult {
	result := &ScanResult{}

	s.logger.Println("Starting browser history scan")

	// Get all users
	users, err := platform.GetAllUsers()
	if err != nil {
		s.logger.Printf("Error: failed to enumerate users: %v", err)
		result.Errors = append(result.Errors, fmt.Sprintf("user enumeration failed: %v", err))
		result.ExitCode = ExitCompleteFailure
		return result
	}

	if len(users) == 0 {
		s.logger.Println("No users found")
		result.ExitCode = ExitCompleteFailure
		return result
	}

	s.logger.Printf("Found %d users to scan", len(users))

	// Get all browsers
	browsers := browser.All()

	successCount := 0
	failureCount := 0

	// Scan each user
	for _, user := range users {
		result.UsersScanned++
		s.logger.Printf("Scanning user: %s", user.Username)

		// Scan each browser for this user
		for _, b := range browsers {
			profiles, err := b.FindProfiles(user)
			if err != nil {
				s.logger.Printf("Error finding %s profiles for %s: %v", b.Name(), user.Username, err)
				continue
			}

			if len(profiles) == 0 {
				continue
			}

			// Scan each profile
			for _, profile := range profiles {
				result.ProfilesScanned++

				sent, err := s.scanProfile(user, b, profile)
				if err != nil {
					failureCount++
					errMsg := fmt.Sprintf("%s/%s/%s: %v", user.Username, b.Name(), profile.Name, err)
					result.Errors = append(result.Errors, errMsg)
					s.logger.Printf("Error: %s", errMsg)
					continue
				}

				result.EntriesSent += sent
				if sent > 0 {
					successCount++
				}
			}
		}
	}

	// Save state
	if err := s.state.Save(); err != nil {
		s.logger.Printf("Warning: failed to save state: %v", err)
	}

	// Determine exit code
	if successCount == 0 && failureCount > 0 {
		result.ExitCode = ExitCompleteFailure
	} else if failureCount > 0 {
		result.ExitCode = ExitPartialFailure
	} else {
		result.ExitCode = ExitSuccess
	}

	s.logger.Printf("Scan complete: %d entries sent, %d errors", result.EntriesSent, len(result.Errors))

	return result
}

// scanProfile scans a single browser profile and sends the results
func (s *Scanner) scanProfile(user platform.User, b browser.Browser, profile browser.Profile) (int, error) {
	// Get last scan timestamp
	lastTimestamp := s.state.GetLastTimestamp(user.Username, b.Name(), profile.Name)

	// If no previous scan, use initial_days config
	if lastTimestamp == 0 {
		lastTimestamp = time.Now().AddDate(0, 0, -s.cfg.InitialDays).UnixMilli()
	}

	// Get history since last scan
	entries, err := b.GetHistory(profile, lastTimestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to get history: %w", err)
	}

	if len(entries) == 0 {
		return 0, nil
	}

	s.logger.Printf("  %s/%s: %d new entries", b.Name(), profile.Name, len(entries))

	// Create principal
	principal := dto.NewUserPrincipal(user.Username)
	if user.Username == "" {
		// Fallback to IP if username unknown
		principal = dto.NewIPPrincipal(getLocalIP())
	}

	// Create payload
	payload := dto.VisitedSitesDTO{
		Principal:    principal,
		Source:       s.cfg.Source,
		VisitedSites: entries,
	}

	if s.dryRun {
		// In dry run, dump JSON to stdout
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return 0, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return len(entries), nil
	}

	// Send to server
	result, maxTimestamp, err := s.client.Send(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to send: %w", err)
	}

	// Update state with the max timestamp of sent entries
	if maxTimestamp > 0 {
		s.state.SetLastTimestamp(user.Username, b.Name(), profile.Name, maxTimestamp)
	}

	return result.TotalSent, nil
}

// getLocalIP returns the local IP address with hostname fallback
func getLocalIP() string {
	var ip string

	// Try interface addresses first
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ip = ipnet.IP.String()
					break
				}
			}
		}
	}

	// Try getting outbound IP if no interface IP found
	if ip == "" {
		conn, err := net.Dial("udp", "8.8.8.8:80")
		if err == nil {
			defer conn.Close()
			localAddr := conn.LocalAddr().(*net.UDPAddr)
			ip = localAddr.IP.String()
		}
	}

	// Get hostname
	hostname, _ := os.Hostname()

	// Return ip/hostname or just hostname as fallback
	if ip != "" && hostname != "" {
		return ip + "/" + hostname
	} else if ip != "" {
		return ip
	} else if hostname != "" {
		return hostname
	}
	return "unknown"
}
