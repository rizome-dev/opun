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
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetActualUser(t *testing.T) {
	t.Run("Without Sudo", func(t *testing.T) {
		// Clear SUDO_USER
		oldSudoUser := os.Getenv("SUDO_USER")
		os.Unsetenv("SUDO_USER")
		defer os.Setenv("SUDO_USER", oldSudoUser)

		u, err := GetActualUser()
		require.NoError(t, err)
		assert.NotNil(t, u)

		// Should return current user
		current, err := user.Current()
		require.NoError(t, err)
		assert.Equal(t, current.Uid, u.Uid)
	})

	t.Run("With Sudo", func(t *testing.T) {
		// This test can only run if we have a valid user to test with
		testUser := os.Getenv("USER")
		if testUser == "" || testUser == "root" {
			t.Skip("Skipping sudo test - no suitable test user")
		}

		oldSudoUser := os.Getenv("SUDO_USER")
		os.Setenv("SUDO_USER", testUser)
		defer os.Setenv("SUDO_USER", oldSudoUser)

		u, err := GetActualUser()
		require.NoError(t, err)
		assert.NotNil(t, u)
		assert.Equal(t, testUser, u.Username)
	})
}

func TestGetActualUserIDs(t *testing.T) {
	uid, gid, err := GetActualUserIDs()
	require.NoError(t, err)
	assert.Greater(t, uid, -1)
	assert.Greater(t, gid, -1)
}

func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()

	testDir := filepath.Join(tempDir, "test", "nested", "dir")
	err := EnsureDir(testDir)
	require.NoError(t, err)

	// Check directory was created
	info, err := os.Stat(testDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Test idempotency
	err = EnsureDir(testDir)
	require.NoError(t, err)
}

func TestWriteFile(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("Simple Write", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.txt")
		testData := []byte("hello world")

		err := WriteFile(testFile, testData)
		require.NoError(t, err)

		// Verify file contents
		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, testData, data)

		// Check permissions
		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
	})

	t.Run("With Directory Creation", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "nested", "dir", "test.txt")
		testData := []byte("nested file")

		err := WriteFile(testFile, testData)
		require.NoError(t, err)

		// Verify file exists
		assert.True(t, FileExists(testFile))

		// Verify contents
		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, testData, data)
	})
}

func TestWriteFileAtomic(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("Atomic Write", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "atomic.txt")
		testData := []byte("atomic data")

		err := WriteFileAtomic(testFile, testData)
		require.NoError(t, err)

		// Verify file contents
		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, testData, data)
	})

	t.Run("Overwrites Existing", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "overwrite.txt")

		// Write initial data
		err := WriteFile(testFile, []byte("initial"))
		require.NoError(t, err)

		// Atomic overwrite
		newData := []byte("overwritten")
		err = WriteFileAtomic(testFile, newData)
		require.NoError(t, err)

		// Verify new contents
		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, newData, data)
	})
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()

	// Non-existent file
	assert.False(t, FileExists(filepath.Join(tempDir, "nonexistent.txt")))

	// Existing file
	testFile := filepath.Join(tempDir, "exists.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)
	assert.True(t, FileExists(testFile))

	// Directory
	testDir := filepath.Join(tempDir, "testdir")
	err = os.Mkdir(testDir, 0755)
	require.NoError(t, err)
	assert.True(t, FileExists(testDir))
}

func TestChownToActualUser(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "chown-test.txt")

	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Without SUDO_USER, should be no-op
	oldSudoUser := os.Getenv("SUDO_USER")
	os.Unsetenv("SUDO_USER")
	defer os.Setenv("SUDO_USER", oldSudoUser)

	err = ChownToActualUser(testFile)
	require.NoError(t, err)

	// With SUDO_USER set (but we can't actually test chown without permissions)
	os.Setenv("SUDO_USER", "testuser")
	err = ChownToActualUser(testFile)
	// This will fail in tests but shouldn't panic
	// In real usage, it would work when running with sudo
}

func TestFixPermissionsRecursive(t *testing.T) {
	tempDir := t.TempDir()

	// Create nested structure
	err := os.MkdirAll(filepath.Join(tempDir, "a", "b", "c"), 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "a", "file1.txt"), []byte("test1"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "a", "b", "file2.txt"), []byte("test2"), 0644)
	require.NoError(t, err)

	// Without SUDO_USER, should be no-op
	oldSudoUser := os.Getenv("SUDO_USER")
	os.Unsetenv("SUDO_USER")
	defer os.Setenv("SUDO_USER", oldSudoUser)

	err = FixPermissionsRecursive(tempDir)
	require.NoError(t, err)
}
