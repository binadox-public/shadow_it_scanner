// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package db

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database connection with WAL mode support and copy fallback
type DB struct {
	db       *sql.DB
	path     string
	tempCopy string // non-empty if we're using a temp copy
}

// Open opens a SQLite database, trying WAL mode first, then falling back to copy
func Open(dbPath string) (*DB, error) {
	// First try to open directly with WAL mode
	db, err := openWithWAL(dbPath)
	if err == nil {
		return &DB{db: db, path: dbPath}, nil
	}

	// If that failed (likely locked), copy to temp and open the copy
	tempPath, err := copyToTemp(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to copy database to temp: %w", err)
	}

	db, err = openWithWAL(tempPath)
	if err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("failed to open temp copy: %w", err)
	}

	return &DB{db: db, path: dbPath, tempCopy: tempPath}, nil
}

// openWithWAL opens a SQLite database in WAL mode
func openWithWAL(dbPath string) (*sql.DB, error) {
	// Use immutable mode for read-only access, which helps with locked databases
	dsn := fmt.Sprintf("file:%s?mode=ro&_journal_mode=WAL&_busy_timeout=5000", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// copyToTemp copies the database file to a temporary location
func copyToTemp(dbPath string) (string, error) {
	// Create temp file with same extension
	ext := filepath.Ext(dbPath)
	if ext == "" {
		ext = ".db"
	}

	tempFile, err := os.CreateTemp("", "hist_scanner_*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Open source file
	src, err := os.Open(dbPath)
	if err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return "", fmt.Errorf("failed to open source: %w", err)
	}
	defer src.Close()

	// Copy contents
	_, err = io.Copy(tempFile, src)
	tempFile.Close()
	if err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("failed to copy: %w", err)
	}

	// Also copy WAL and SHM files if they exist (for consistency)
	copyIfExists(dbPath+"-wal", tempPath+"-wal")
	copyIfExists(dbPath+"-shm", tempPath+"-shm")

	return tempPath, nil
}

// copyIfExists copies a file if it exists, ignoring errors
func copyIfExists(src, dst string) {
	srcFile, err := os.Open(src)
	if err != nil {
		return
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return
	}
	defer dstFile.Close()

	io.Copy(dstFile, srcFile)
}

// Close closes the database and cleans up any temp files
func (d *DB) Close() error {
	err := d.db.Close()

	// Clean up temp copy if we made one
	if d.tempCopy != "" {
		os.Remove(d.tempCopy)
		os.Remove(d.tempCopy + "-wal")
		os.Remove(d.tempCopy + "-shm")
	}

	return err
}

// Query executes a query and returns rows
func (d *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.Query(query, args...)
}

// QueryRow executes a query that returns a single row
func (d *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.db.QueryRow(query, args...)
}

// Path returns the original database path
func (d *DB) Path() string {
	return d.path
}

// IsTempCopy returns true if we're using a temporary copy
func (d *DB) IsTempCopy() bool {
	return d.tempCopy != ""
}
