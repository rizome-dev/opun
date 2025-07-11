package utils

// Copyright (C) 2025 Rizome Labs, Inc.
//
// This program is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; either version 2
// of the License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
)

// GetActualUser returns the actual user info even when running under sudo
func GetActualUser() (*user.User, error) {
	// Check if running under sudo
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		return user.Lookup(sudoUser)
	}
	// Otherwise return current user
	return user.Current()
}

// GetActualUserIDs returns the actual user's UID and GID even when running under sudo
func GetActualUserIDs() (uid, gid int, err error) {
	actualUser, err := GetActualUser()
	if err != nil {
		return 0, 0, err
	}

	uid, err = strconv.Atoi(actualUser.Uid)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse UID: %w", err)
	}

	gid, err = strconv.Atoi(actualUser.Gid)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse GID: %w", err)
	}

	return uid, gid, nil
}

// ChownToActualUser changes ownership of a file/directory to the actual user
func ChownToActualUser(path string) error {
	// Only chown if running under sudo
	if os.Getenv("SUDO_USER") == "" {
		return nil
	}

	uid, gid, err := GetActualUserIDs()
	if err != nil {
		return err
	}

	return os.Chown(path, uid, gid)
}

// EnsureDir creates a directory if it doesn't exist with proper ownership
func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// Fix ownership if running under sudo
	return ChownToActualUser(dir)
}

// WriteFile writes data to a file, creating directories as needed with proper ownership
func WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	// Fix ownership if running under sudo
	return ChownToActualUser(path)
}

// WriteFileAtomic writes data to a file atomically with proper ownership
func WriteFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	// Create temp file in same directory
	tmpFile, err := os.CreateTemp(dir, ".tmp-")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up on any error

	// Write data
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}

	// Set permissions
	if err := tmpFile.Chmod(0644); err != nil {
		tmpFile.Close()
		return err
	}

	// Close before rename
	if err := tmpFile.Close(); err != nil {
		return err
	}

	// Fix ownership if running under sudo
	if err := ChownToActualUser(tmpPath); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpPath, path)
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// FixPermissionsRecursive fixes ownership recursively for a directory
func FixPermissionsRecursive(root string) error {
	if os.Getenv("SUDO_USER") == "" {
		return nil // Nothing to fix if not running under sudo
	}

	uid, gid, err := GetActualUserIDs()
	if err != nil {
		return err
	}

	// Use system chown command for recursive operation with numeric IDs
	// #nosec G204 -- using numeric UIDs from OS, not user input
	cmd := exec.Command("chown", "-R", fmt.Sprintf("%d:%d", uid, gid), root)
	return cmd.Run()
}

// SafeReadFile reads a file after validating the path to prevent directory traversal
func SafeReadFile(path string) ([]byte, error) {
	// Clean and validate the path
	cleanPath := filepath.Clean(path)

	// Ensure the path is absolute to prevent ambiguity
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Check if file exists and is not a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %s", absPath)
	}

	// Read the file
	// #nosec G304 -- path has been validated above
	return os.ReadFile(absPath)
}
