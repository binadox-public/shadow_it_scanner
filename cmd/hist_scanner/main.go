// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"hist_scanner/internal/browser"
	"hist_scanner/internal/config"
	"hist_scanner/internal/dto"
	"hist_scanner/internal/installer"
	"hist_scanner/internal/platform"
	"hist_scanner/internal/scanner"
	"hist_scanner/internal/sender"
	"hist_scanner/internal/state"
)

var (
	// Version info (set by ldflags)
	version   = "dev"
	buildTime = "unknown"
	commit    = "unknown"

	// Global flags
	cfgFile     string
	serverURL   string
	apiKey      string
	stateFile   string
	logFile     string
	initialDays int
	chunkSizeKB int
	compress    bool
	timeout     time.Duration
	dryRun      bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}

var rootCmd = &cobra.Command{
	Use:   "hist_scanner",
	Short: "Browser history scanner for security audit",
	Long: `A cross-platform utility that scans browser history from all users
and profiles on a machine and sends the data to a server for security audit.`,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the browser history scan",
	Long:  `Scans browser history from all users and profiles and sends to server.`,
	RunE:  runScan,
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install scanner to system scheduler",
	Long:  `Installs the scanner as a scheduled task (systemd/Task Scheduler/launchd).`,
	RunE:  runInstall,
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove scanner from system scheduler",
	Long:  `Removes the scanner from the system scheduler.`,
	RunE:  runUninstall,
}

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug commands for testing",
	Long:  `Various debug commands for testing individual components.`,
}

var debugUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "List all users on the system",
	RunE:  runDebugUsers,
}

var debugBrowserCmd = &cobra.Command{
	Use:   "browser [name]",
	Short: "Test browser history extraction",
	Args:  cobra.ExactArgs(1),
	RunE:  runDebugBrowser,
}

var debugStateCmd = &cobra.Command{
	Use:   "state",
	Short: "Show state file contents",
	RunE:  runDebugState,
}

var debugSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Test sending data to server",
	RunE:  runDebugSend,
}

// Install command specific flags
var (
	installInterval time.Duration
	installUser     string
)

// Debug command specific flags
var (
	debugUser string
)

func init() {
	// Set version template to include build info
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("hist_scanner version %s (commit: %s, built: %s)\n", version, commit, buildTime))

	// Global flags for all commands
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")

	// Run command flags
	runCmd.Flags().StringVar(&serverURL, "server-url", "", "server endpoint URL")
	runCmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication")
	runCmd.Flags().StringVar(&stateFile, "state-file", "", "path to state file")
	runCmd.Flags().StringVar(&logFile, "log-file", "", "path to log file")
	runCmd.Flags().IntVar(&initialDays, "initial-days", 0, "days of history on first scan (default: 7)")
	runCmd.Flags().IntVar(&chunkSizeKB, "chunk-size-kb", 0, "max compressed chunk size in KB (default: 1024)")
	runCmd.Flags().BoolVar(&compress, "compress", true, "enable gzip compression (default: true)")
	runCmd.Flags().DurationVar(&timeout, "timeout", 0, "HTTP timeout (default: 30s)")
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "scan and dump JSON to stdout instead of sending")

	// Install command flags
	installCmd.Flags().StringVar(&serverURL, "server-url", "", "server endpoint URL")
	installCmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication")
	installCmd.Flags().StringVar(&stateFile, "state-file", "", "path to state file")
	installCmd.Flags().StringVar(&logFile, "log-file", "", "path to log file")
	installCmd.Flags().IntVar(&initialDays, "initial-days", 0, "days of history on first scan")
	installCmd.Flags().IntVar(&chunkSizeKB, "chunk-size-kb", 0, "max compressed chunk size in KB")
	installCmd.Flags().BoolVar(&compress, "compress", true, "enable gzip compression")
	installCmd.Flags().DurationVar(&timeout, "timeout", 0, "HTTP timeout")
	installCmd.Flags().DurationVar(&installInterval, "interval", 24*time.Hour, "scan interval")
	installCmd.Flags().StringVar(&installUser, "user", "", "user to run as (default: root/SYSTEM)")

	// Debug command flags
	debugBrowserCmd.Flags().StringVar(&debugUser, "user", "", "specific user to scan")

	// Build command tree
	debugCmd.AddCommand(debugUsersCmd)
	debugCmd.AddCommand(debugBrowserCmd)
	debugCmd.AddCommand(debugStateCmd)
	debugCmd.AddCommand(debugSendCmd)

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(debugCmd)
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, err
	}

	// Check if compress flag was explicitly set
	compressSet := cmd.Flags().Changed("compress")

	// Apply CLI flags
	cfg.ApplyFlags(serverURL, apiKey, stateFile, logFile, initialDays, chunkSizeKB, compress, compressSet, timeout)

	return cfg, nil
}

func runScan(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !dryRun {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}
	}

	s, err := scanner.New(cfg, dryRun)
	if err != nil {
		return fmt.Errorf("failed to initialize scanner: %w", err)
	}

	result := s.Run()

	// Exit with appropriate code
	if result.ExitCode != scanner.ExitSuccess {
		os.Exit(int(result.ExitCode))
	}

	return nil
}

func runInstall(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	inst, err := installer.New()
	if err != nil {
		return fmt.Errorf("failed to create installer: %w", err)
	}

	paths := installer.GetInstallPaths()

	fmt.Printf("Installing browser history scanner...\n")
	fmt.Printf("  Binary: %s\n", paths.BinaryPath)
	fmt.Printf("  Config: %s\n", paths.ConfigPath)
	fmt.Printf("  Interval: %s\n", installInterval)
	fmt.Printf("  Run as: %s\n", installUser)

	// If config was obtained via auto-discovery, save it to file
	// so scheduled runs don't depend on discovery server availability
	if cfg.WasDiscovered() {
		fmt.Println("  Config source: auto-discovery")
		fmt.Printf("  Saving discovered config to: %s\n", paths.ConfigPath)
		if err := cfg.SaveToFile(paths.ConfigPath); err != nil {
			return fmt.Errorf("failed to save discovered config: %w", err)
		}
	}
	fmt.Println()

	if err := inst.Install(cfg, installInterval, installUser); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Println("Installation complete!")
	fmt.Println("\nThe scanner will run automatically on schedule.")
	fmt.Println("To run manually: hist_scanner run --config", paths.ConfigPath)

	return nil
}

func runUninstall(cmd *cobra.Command, args []string) error {
	inst, err := installer.New()
	if err != nil {
		return fmt.Errorf("failed to create installer: %w", err)
	}

	if !inst.IsInstalled() {
		fmt.Println("Scanner is not installed.")
		return nil
	}

	fmt.Println("Uninstalling browser history scanner...")

	if err := inst.Uninstall(); err != nil {
		return fmt.Errorf("uninstallation failed: %w", err)
	}

	fmt.Println("Uninstallation complete!")
	return nil
}

func runDebugUsers(cmd *cobra.Command, args []string) error {
	fmt.Printf("Platform: %s\n\n", platform.CurrentOS())

	users, err := platform.GetAllUsers()
	if err != nil {
		return fmt.Errorf("failed to enumerate users: %w", err)
	}

	fmt.Printf("Found %d users:\n", len(users))
	for _, u := range users {
		fmt.Printf("  - %s (UID: %s)\n", u.Username, u.UID)
		fmt.Printf("    Home: %s\n", u.HomeDir)
	}

	return nil
}

func runDebugBrowser(cmd *cobra.Command, args []string) error {
	browserName := args[0]

	b := browser.ByName(browserName)
	if b == nil {
		return fmt.Errorf("unknown browser: %s\nSupported: %s", browserName, strings.Join(browser.SupportedBrowserNames(), ", "))
	}

	fmt.Printf("Testing browser: %s\n\n", b.Name())

	// Get users to scan
	var users []platform.User
	if debugUser != "" {
		// Scan specific user
		users = []platform.User{{Username: debugUser, HomeDir: ""}}
		// Try to find the user's home directory
		allUsers, _ := platform.GetAllUsers()
		for _, u := range allUsers {
			if u.Username == debugUser {
				users[0] = u
				break
			}
		}
	} else {
		// Scan all users
		var err error
		users, err = platform.GetAllUsers()
		if err != nil {
			return fmt.Errorf("failed to enumerate users: %w", err)
		}
	}

	totalEntries := 0
	for _, user := range users {
		profiles, err := b.FindProfiles(user)
		if err != nil {
			fmt.Printf("User %s: error finding profiles: %v\n", user.Username, err)
			continue
		}

		if len(profiles) == 0 {
			fmt.Printf("User %s: no profiles found\n", user.Username)
			continue
		}

		fmt.Printf("User %s: found %d profile(s)\n", user.Username, len(profiles))

		for _, profile := range profiles {
			// Get last 7 days of history for demo
			sinceTimestamp := time.Now().AddDate(0, 0, -7).UnixMilli()

			entries, err := b.GetHistory(profile, sinceTimestamp)
			if err != nil {
				fmt.Printf("  Profile %s: error reading history: %v\n", profile.Name, err)
				continue
			}

			fmt.Printf("  Profile %s: %d entries (last 7 days)\n", profile.Name, len(entries))
			totalEntries += len(entries)

			// Show first 5 entries as sample
			for i, entry := range entries {
				if i >= 5 {
					fmt.Printf("    ... and %d more\n", len(entries)-5)
					break
				}
				t := time.UnixMilli(entry.Timestamp)
				// Truncate URL for display
				url := entry.URL
				if len(url) > 60 {
					url = url[:57] + "..."
				}
				fmt.Printf("    %s | %s\n", t.Format("2006-01-02 15:04"), url)
			}
		}
	}

	fmt.Printf("\nTotal entries found: %d\n", totalEntries)
	return nil
}

func runDebugState(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mgr := state.NewManager(cfg.StateFile)
	if err := mgr.Load(); err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	fmt.Printf("State file: %s\n\n", mgr.GetStateFilePath())

	entries := mgr.GetAllEntries()
	if len(entries) == 0 {
		fmt.Println("No state entries found (first run or state cleared)")
		return nil
	}

	fmt.Printf("Found %d entries:\n", len(entries))
	for key, timestamp := range entries {
		t := time.UnixMilli(timestamp)
		fmt.Printf("  %s: %s (%d)\n", key, t.Format("2006-01-02 15:04:05"), timestamp)
	}

	return nil
}

func runDebugSend(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	client := sender.NewClient(cfg.ServerURL, cfg.APIKey, cfg.Timeout, cfg.ChunkSizeKB, cfg.Compress)

	// Send test data
	testPayload := dto.VisitedSitesDTO{
		Principal: dto.NewUserPrincipal("test-user"),
		Source:    cfg.Source,
		VisitedSites: []dto.VisitedSite{
			{URL: "https://example.com/test1", Timestamp: time.Now().Add(-1 * time.Hour).UnixMilli()},
			{URL: "https://example.com/test2", Timestamp: time.Now().Add(-30 * time.Minute).UnixMilli()},
			{URL: "https://example.com/test3", Timestamp: time.Now().UnixMilli()},
		},
	}

	fmt.Printf("Sending test data to %s...\n", cfg.ServerURL)
	fmt.Printf("Payload: %d entries\n", len(testPayload.VisitedSites))

	result, maxTs, err := client.Send(testPayload)
	if err != nil {
		return fmt.Errorf("failed to send: %w", err)
	}

	fmt.Printf("\nResult:\n")
	fmt.Printf("  Chunks sent: %d\n", result.ChunksSent)
	fmt.Printf("  Total sent: %d\n", result.TotalSent)
	fmt.Printf("  Failed: %d\n", result.FailedCount)
	fmt.Printf("  Max timestamp: %d (%s)\n", maxTs, time.UnixMilli(maxTs).Format("2006-01-02 15:04:05"))

	if result.LastError != nil {
		fmt.Printf("  Last error: %v\n", result.LastError)
	}

	return nil
}
