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
	"io"
	"sync"
)

// CleanupManager manages cleanup functions for graceful shutdown
type CleanupManager struct {
	mu       sync.Mutex
	cleanups []func()
}

var globalCleanup = &CleanupManager{}

// RegisterCleanup registers a cleanup function to be called on shutdown
func RegisterCleanup(fn func()) {
	globalCleanup.mu.Lock()
	defer globalCleanup.mu.Unlock()
	globalCleanup.cleanups = append(globalCleanup.cleanups, fn)
}

// RegisterCloser registers an io.Closer to be closed on shutdown
func RegisterCloser(closer io.Closer) {
	RegisterCleanup(func() {
		_ = closer.Close()
	})
}

// RunCleanup runs all registered cleanup functions
func RunCleanup() {
	globalCleanup.mu.Lock()
	defer globalCleanup.mu.Unlock()

	// Run cleanups in reverse order (LIFO)
	for i := len(globalCleanup.cleanups) - 1; i >= 0; i-- {
		globalCleanup.cleanups[i]()
	}

	// Clear the cleanup list
	globalCleanup.cleanups = nil
}
