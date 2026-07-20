//go:build e2e

package e2e

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
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/rizome-dev/opun/internal/headless"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeadlessSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping headless smoke test in short mode")
	}
	if os.Getenv("OPUN_E2E_LIVE") != "1" {
		t.Skip("Set OPUN_E2E_LIVE=1 to run live provider smoke tests (spends API credits)")
	}

	providers := []string{"claude", "gemini", "qwen"}
	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			if _, err := exec.LookPath(provider); err != nil {
				t.Skipf("%s not installed", provider)
			}

			runner, err := headless.NewRunner(provider)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			result, err := runner.Run(ctx, headless.Request{
				Prompt: "Reply with exactly: OK",
			})
			require.NoError(t, err)
			assert.NotEmpty(t, result.Output)
			t.Logf("%s output: %s", provider, result.Output)
		})
	}
}
