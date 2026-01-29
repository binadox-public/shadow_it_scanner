# hist_scanner

A cross-platform browser history scanner for enterprise security auditing.

Copyright (c) 2025 [Binadox](https://binadox.com) - Licensed under zlib license.

## Overview

`hist_scanner` is a lightweight utility that scans browser history from all users and profiles on a machine and sends the data to a central server for security audit purposes. It supports multiple browsers and operating systems, with automatic scheduling via system services.

## Features

- **Multi-browser support**: Chrome, Edge, Firefox, Safari, Opera, Opera GX, Vivaldi
- **Cross-platform**: Linux, macOS, Windows
- **Multi-user scanning**: Scans all users on the system
- **Multi-profile support**: Detects and scans all browser profiles
- **Incremental scanning**: Only sends new history since last scan
- **Gzip compression**: Reduces bandwidth with automatic fallback
- **Size-based chunking**: Splits large payloads for reliable transmission
- **Self-registration**: Installs as systemd timer, launchd, or Task Scheduler
- **Static binaries**: No dependencies, easy deployment

## Quick Start

### Download

Download the appropriate binary for your platform from the releases page.

### Basic Usage

```bash
# Scan and send to server
hist_scanner run --server-url https://audit.example.com/api/history --api-key YOUR_API_KEY

# Dry run - scan and print JSON to stdout (no server required)
hist_scanner run --dry-run

# Install as scheduled service (runs daily)
sudo hist_scanner install --server-url https://audit.example.com/api/history --api-key YOUR_API_KEY
```

## Installation

### From Binary

1. Download the binary for your platform
2. Make it executable (Linux/macOS): `chmod +x hist_scanner-*`
3. Move to a location in your PATH or run directly

### From Source

```bash
# Build for current platform
make build

# Build for all platforms
make all

# Create release archives
make release
```

### System Installation

The `install` command copies the binary to a system location and registers it with the OS scheduler:

```bash
# Linux/macOS
sudo hist_scanner install \
  --server-url https://audit.example.com/api/history \
  --api-key YOUR_API_KEY

# Windows (run as Administrator)
hist_scanner.exe install ^
  --server-url https://audit.example.com/api/history ^
  --api-key YOUR_API_KEY
```

#### Installation Paths

| Platform | Binary | Config |
|----------|--------|--------|
| Linux | `/usr/local/bin/hist_scanner` | `/etc/hist_scanner/config.yaml` |
| macOS | `/usr/local/bin/hist_scanner` | `/etc/hist_scanner/config.yaml` |
| Windows | `C:\Program Files\hist_scanner\hist_scanner.exe` | `C:\ProgramData\hist_scanner\config.yaml` |

#### Scheduler Integration

| Platform | Scheduler | Service Name |
|----------|-----------|--------------|
| Linux | systemd | `hist_scanner.timer` / `hist_scanner.service` |
| macOS | launchd | `com.binadox.hist_scanner.plist` |
| Windows | Task Scheduler | `hist_scanner` |

### Uninstallation

```bash
# Linux/macOS
sudo hist_scanner uninstall

# Windows (run as Administrator)
hist_scanner.exe uninstall
```

## Configuration

### Command Line Flags

#### Run Command

| Flag | Description | Default |
|------|-------------|---------|
| `--server-url` | Server endpoint URL | (required) |
| `--api-key` | API key for authentication | (required) |
| `--config` | Path to config file | (none) |
| `--state-file` | Path to state file | (auto-detected) |
| `--log-file` | Path to log file | (no logging) |
| `--initial-days` | Days of history on first scan | 7 |
| `--chunk-size-kb` | Max compressed chunk size in KB | 1024 |
| `--compress` | Enable gzip compression | true |
| `--timeout` | HTTP timeout | 30s |
| `--dry-run` | Dump JSON to stdout instead of sending | false |

#### Install Command

All `run` flags plus:

| Flag | Description | Default |
|------|-------------|---------|
| `--interval` | Scan interval | 24h |
| `--user` | User to run as | root/SYSTEM |

### Config File

Create a YAML config file to avoid passing flags on every run:

```yaml
# /etc/hist_scanner/config.yaml
server_url: https://audit.example.com/api/history
api_key: your-api-key-here
initial_days: 7
timeout: 30s
chunk_size_kb: 1024
compress: true
state_file: /var/lib/hist_scanner/state.json
log_file: /var/log/hist_scanner.log
```

Then run with:

```bash
hist_scanner run --config /etc/hist_scanner/config.yaml
```

### Environment Variables

All config options can be set via environment variables with the `HIST_SCANNER_` prefix:

```bash
export HIST_SCANNER_SERVER_URL=https://audit.example.com/api/history
export HIST_SCANNER_API_KEY=your-api-key
hist_scanner run
```

### Configuration Priority

1. Command line flags (highest)
2. Environment variables
3. Config file
4. Defaults (lowest)

## Supported Browsers

| Browser | Linux | macOS | Windows |
|---------|-------|-------|---------|
| Google Chrome | Yes | Yes | Yes |
| Microsoft Edge | Yes | Yes | Yes |
| Mozilla Firefox | Yes | Yes | Yes |
| Apple Safari | - | Yes | - |
| Opera | Yes | Yes | Yes |
| Opera GX | Yes | Yes | Yes |
| Vivaldi | Yes | Yes | Yes |

## State Management

The scanner tracks the last scan timestamp per user/browser/profile to enable incremental scanning. State file locations (in order of preference):

1. Explicitly set via `--state-file` or config
2. Central system location:
   - Linux/macOS: `/var/lib/hist_scanner/state.json`
   - Windows: `C:\ProgramData\hist_scanner\state.json`
3. Temp directory fallback

## Debug Commands

Use debug commands to troubleshoot issues:

```bash
# List all users on the system
hist_scanner debug users

# Test browser history extraction
hist_scanner debug browser chrome
hist_scanner debug browser firefox
hist_scanner debug browser safari

# Show state file contents
hist_scanner debug state --config /path/to/config.yaml

# Test sending data to server
hist_scanner debug send --config /path/to/config.yaml
```

## API Integration

### Request Format

The scanner sends POST requests with JSON payload:

```json
{
  "principal": {
    "name": "username",
    "kind": "USERNAME"
  },
  "source": "hist_scanner",
  "visitedSites": [
    {
      "url": "https://example.com/page",
      "timestamp": 1702300800000
    }
  ]
}
```

### Headers

| Header | Value |
|--------|-------|
| `Content-Type` | `application/json` |
| `Content-Encoding` | `gzip` (if compression enabled) |
| `X-API-Key` | API key from config |

### Response

The server should return HTTP 200 on success. If the server returns HTTP 415 (Unsupported Media Type) when compression is enabled, the scanner automatically retries without compression.

### Chunking

Large payloads are automatically split into chunks based on compressed size (default 1MB). Each chunk is sent as a separate request.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success - all browsers/profiles scanned and sent |
| 1 | Partial failure - some browsers/profiles failed |
| 2 | Complete failure - nothing sent |

## Logging

By default, the scanner runs silently. Enable logging with:

```bash
hist_scanner run --log-file /var/log/hist_scanner.log ...
```

Log output includes:
- Scan start/end timestamps
- Users and profiles scanned
- Entry counts per profile
- Errors and warnings

## Security Considerations

- The config file contains the API key and should have restricted permissions (0600)
- Run as root/SYSTEM to access all users' browser history
- Browser databases are accessed read-only
- Locked databases (browser running) are copied to temp for safe access

## Troubleshooting

### "Permission denied" errors

Run as root/Administrator to access all users' history:

```bash
sudo hist_scanner run --dry-run
```

### "Database is locked" errors

The scanner automatically copies locked databases to temp. If issues persist, close the browser and retry.

### No history found

1. Verify the browser is installed and has been used
2. Check the user has a valid home directory
3. Use debug commands to verify:
   ```bash
   hist_scanner debug users
   hist_scanner debug browser chrome
   ```

### Server connection issues

1. Verify server URL is correct and reachable
2. Check API key is valid
3. Test with debug send command:
   ```bash
   hist_scanner debug send --server-url URL --api-key KEY
   ```

## Building from Source

### Requirements

- Go 1.21 or later

### Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build for current platform |
| `make all` | Build for all platforms |
| `make test` | Run tests |
| `make fmt` | Format code |
| `make tidy` | Tidy go.mod |
| `make clean` | Remove build artifacts |
| `make install` | Install to /usr/local/bin |
| `make uninstall` | Remove from /usr/local/bin |
| `make version` | Show version info |
| `make release` | Build all platforms and create archives |

### Cross-Compilation

The Makefile supports cross-compilation for:
- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64
- windows/amd64

All binaries are statically linked (CGO_ENABLED=0) with no external dependencies.

### Version Info

Version information is embedded at build time:

```bash
make VERSION=1.0.0 build
./hist_scanner --version
```

## License

This software is licensed under the zlib license. See [LICENSE](LICENSE) file for details.
