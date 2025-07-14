package pty

import (
	"testing"
)

func TestContainsPattern(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		pattern  []byte
		expected bool
	}{
		{
			name:     "exact match",
			data:     []byte("│ > "),
			pattern:  []byte("│ > "),
			expected: true,
		},
		{
			name:     "no match",
			data:     []byte("hello world"),
			pattern:  []byte("│ > "),
			expected: false,
		},
		{
			name:     "partial match at end",
			data:     []byte("some text │ >"),
			pattern:  []byte("│ > "),
			expected: false,
		},
		{
			name:     "match in longer text",
			data:     []byte("Some text before │ > Type your message"),
			pattern:  []byte("│ > "),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsPattern(tt.data, tt.pattern)
			if result != tt.expected {
				t.Errorf("ContainsPattern(%q, %q) = %v, want %v", tt.data, tt.pattern, result, tt.expected)
			}
		})
	}
}