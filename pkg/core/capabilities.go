package core

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

import "strings"

func MatchesCapabilities(agentCaps []string, task SubAgentTask) bool {
	if len(agentCaps) == 0 {
		return true
	}

	caps := make(map[string]struct{}, len(agentCaps))
	for _, c := range agentCaps {
		caps[strings.ToLower(c)] = struct{}{}
	}

	for key := range task.Context {
		if _, ok := caps[strings.ToLower(key)]; ok {
			return true
		}
	}

	if list, ok := task.Context["capabilities"].([]string); ok {
		for _, c := range list {
			if _, ok := caps[strings.ToLower(c)]; ok {
				return true
			}
		}
	}

	if typ, ok := task.Context["type"].(string); ok {
		if _, ok := caps[strings.ToLower(typ)]; ok {
			return true
		}
	}

	return false
}
