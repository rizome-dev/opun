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
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockCloser struct {
	closed bool
}

func (m *mockCloser) Close() error {
	m.closed = true
	return nil
}

func TestRegisterCleanup(t *testing.T) {
	// Reset global state
	globalCleanup.mu.Lock()
	globalCleanup.cleanups = nil
	globalCleanup.mu.Unlock()

	var called int

	// Register multiple cleanup functions
	RegisterCleanup(func() {
		called += 1
	})

	RegisterCleanup(func() {
		called += 10
	})

	RegisterCleanup(func() {
		called += 100
	})

	// Run cleanup
	RunCleanup()

	// Should be called in LIFO order: 100 + 10 + 1 = 111
	assert.Equal(t, 111, called)

	// Running cleanup again should do nothing
	called = 0
	RunCleanup()
	assert.Equal(t, 0, called)
}

func TestRegisterCloser(t *testing.T) {
	// Reset global state
	globalCleanup.mu.Lock()
	globalCleanup.cleanups = nil
	globalCleanup.mu.Unlock()

	closer1 := &mockCloser{}
	closer2 := &mockCloser{}

	RegisterCloser(closer1)
	RegisterCloser(closer2)

	// Neither should be closed yet
	assert.False(t, closer1.closed)
	assert.False(t, closer2.closed)

	// Run cleanup
	RunCleanup()

	// Both should be closed
	assert.True(t, closer1.closed)
	assert.True(t, closer2.closed)
}

func TestCleanupOrder(t *testing.T) {
	// Reset global state
	globalCleanup.mu.Lock()
	globalCleanup.cleanups = nil
	globalCleanup.mu.Unlock()

	var order []int

	RegisterCleanup(func() {
		order = append(order, 1)
	})

	RegisterCleanup(func() {
		order = append(order, 2)
	})

	RegisterCleanup(func() {
		order = append(order, 3)
	})

	RunCleanup()

	// Should be in reverse order (LIFO)
	assert.Equal(t, []int{3, 2, 1}, order)
}

func TestConcurrentRegister(t *testing.T) {
	// Reset global state
	globalCleanup.mu.Lock()
	globalCleanup.cleanups = nil
	globalCleanup.mu.Unlock()

	done := make(chan bool, 10)

	// Register cleanups concurrently
	for i := 0; i < 10; i++ {
		go func() {
			RegisterCleanup(func() {})
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have registered 10 cleanups
	globalCleanup.mu.Lock()
	count := len(globalCleanup.cleanups)
	globalCleanup.mu.Unlock()

	assert.Equal(t, 10, count)

	// Cleanup
	RunCleanup()
}

// mockReadCloser implements io.ReadCloser for testing
type mockReadCloser struct {
	mockCloser
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func TestRegisterCloserWithReadCloser(t *testing.T) {
	// Reset global state
	globalCleanup.mu.Lock()
	globalCleanup.cleanups = nil
	globalCleanup.mu.Unlock()

	rc := &mockReadCloser{}

	// Should accept any io.Closer
	RegisterCloser(rc)

	assert.False(t, rc.closed)

	RunCleanup()

	assert.True(t, rc.closed)
}
