// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package sender

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"hist_scanner/internal/dto"
)

// Client handles HTTP communication with the server
type Client struct {
	serverURL    string
	apiKey       string
	httpClient   *http.Client
	maxChunkSize int  // Max compressed chunk size in bytes
	compress     bool // Whether to use gzip compression
}

// NewClient creates a new HTTP client for sending history data
// maxChunkSizeKB is the maximum compressed chunk size in kilobytes
func NewClient(serverURL, apiKey string, timeout time.Duration, maxChunkSizeKB int, compress bool) *Client {
	return &Client{
		serverURL: serverURL,
		apiKey:    apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxChunkSize: maxChunkSizeKB * 1024, // Convert to bytes
		compress:     compress,
	}
}

// SendResult contains the result of a send operation
type SendResult struct {
	TotalSent      int   // Total entries successfully sent
	ChunksSent     int   // Number of chunks sent
	LastError      error // Last error encountered (if any)
	FailedCount    int   // Number of entries that failed to send
	BytesSent      int64 // Total bytes sent (compressed if enabled)
	BytesOriginal  int64 // Total bytes before compression
}

// Send sends visited sites to the server, chunking by compressed size
// Returns the maximum timestamp of successfully sent entries (for state update)
func (c *Client) Send(payload dto.VisitedSitesDTO) (*SendResult, int64, error) {
	result := &SendResult{}

	if len(payload.VisitedSites) == 0 {
		return result, 0, nil
	}

	var maxTimestamp int64

	// Build chunks based on compressed size
	chunks := c.buildChunks(payload)

	for _, chunk := range chunks {
		bytesSent, bytesOriginal, err := c.sendChunk(chunk)
		if err != nil {
			result.LastError = err
			result.FailedCount += len(chunk.VisitedSites)
			// Continue trying other chunks
			continue
		}

		result.TotalSent += len(chunk.VisitedSites)
		result.ChunksSent++
		result.BytesSent += bytesSent
		result.BytesOriginal += bytesOriginal

		// Track max timestamp from successful sends
		for _, site := range chunk.VisitedSites {
			if site.Timestamp > maxTimestamp {
				maxTimestamp = site.Timestamp
			}
		}
	}

	if result.TotalSent == 0 && result.LastError != nil {
		return result, 0, result.LastError
	}

	return result, maxTimestamp, nil
}

// buildChunks splits the payload into chunks based on compressed size
func (c *Client) buildChunks(payload dto.VisitedSitesDTO) []dto.VisitedSitesDTO {
	var chunks []dto.VisitedSitesDTO
	var currentSites []dto.VisitedSite
	var currentSize int

	for _, site := range payload.VisitedSites {
		// Estimate size of this entry (JSON overhead + data)
		// Approximate: {"url":"...","timestamp":1234567890123}
		entrySize := len(site.URL) + 40 // URL + JSON overhead + timestamp

		// If adding this entry would exceed limit, start new chunk
		// Use compression ratio estimate of ~0.3 for gzip on JSON
		estimatedCompressedSize := currentSize
		if c.compress {
			estimatedCompressedSize = int(float64(currentSize) * 0.3)
		}

		if len(currentSites) > 0 && estimatedCompressedSize+entrySize > c.maxChunkSize {
			// Save current chunk
			chunks = append(chunks, dto.VisitedSitesDTO{
				Principal:    payload.Principal,
				Source:       payload.Source,
				VisitedSites: currentSites,
			})
			currentSites = nil
			currentSize = 0
		}

		currentSites = append(currentSites, site)
		currentSize += entrySize
	}

	// Don't forget the last chunk
	if len(currentSites) > 0 {
		chunks = append(chunks, dto.VisitedSitesDTO{
			Principal:    payload.Principal,
			Source:       payload.Source,
			VisitedSites: currentSites,
		})
	}

	return chunks
}

// sendChunk sends a single chunk to the server
// Returns (bytesSent, bytesOriginal, error)
func (c *Client) sendChunk(payload dto.VisitedSitesDTO) (int64, int64, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal payload: %w", err)
	}

	bytesOriginal := int64(len(data))

	if c.compress {
		// Try with gzip first
		bytesSent, err := c.sendWithGzip(data)
		if err == nil {
			return bytesSent, bytesOriginal, nil
		}

		// If server rejected gzip (415 Unsupported Media Type), retry without compression
		if isUnsupportedMediaType(err) {
			bytesSent, err = c.sendRaw(data)
			return bytesSent, bytesOriginal, err
		}

		return 0, bytesOriginal, err
	}

	bytesSent, err := c.sendRaw(data)
	return bytesSent, bytesOriginal, err
}

// sendWithGzip sends gzip-compressed data
func (c *Client) sendWithGzip(data []byte) (int64, error) {
	var compressed bytes.Buffer
	gzWriter, err := gzip.NewWriterLevel(&compressed, gzip.DefaultCompression)
	if err != nil {
		return 0, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	if _, err := gzWriter.Write(data); err != nil {
		return 0, fmt.Errorf("failed to write gzip data: %w", err)
	}

	if err := gzWriter.Close(); err != nil {
		return 0, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.serverURL, &compressed)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Authorization", "ProxyToken "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, &httpError{statusCode: resp.StatusCode}
	}

	return int64(compressed.Len()), nil
}

// sendRaw sends uncompressed data
func (c *Client) sendRaw(data []byte) (int64, error) {
	req, err := http.NewRequest(http.MethodPost, c.serverURL, bytes.NewReader(data))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "ProxyToken "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, &httpError{statusCode: resp.StatusCode}
	}

	return int64(len(data)), nil
}

// httpError represents an HTTP error with status code
type httpError struct {
	statusCode int
}

func (e *httpError) Error() string {
	return fmt.Sprintf("server returned status %d", e.statusCode)
}

// isUnsupportedMediaType checks if error is 415 Unsupported Media Type
func isUnsupportedMediaType(err error) bool {
	if httpErr, ok := err.(*httpError); ok {
		return httpErr.statusCode == http.StatusUnsupportedMediaType
	}
	return false
}

// TestConnection tests if the server is reachable
func (c *Client) TestConnection() error {
	// Send empty payload to test connection
	testPayload := dto.VisitedSitesDTO{
		Principal: dto.PrincipalDTO{
			Name: "test",
			Kind: dto.KindUsername,
		},
		VisitedSites: []dto.VisitedSite{},
		Source:       "test",
	}

	_, _, err := c.sendChunk(testPayload)
	return err
}
