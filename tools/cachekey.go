// Copyright 2026 Naadir Jeewa
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
)

// CacheKey returns a deterministic cache key derived from the contents of the
// configuration file and the current platform (GOOS + GOARCH).
// The key changes when the config content changes, tools are added/removed,
// versions are bumped, or the platform differs.
func CacheKey(configPath string) (string, error) {
	data, err := os.ReadFile(configPath) //nolint:gosec // File path from trusted configuration
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrConfigNotFound, configPath)
		}

		return "", fmt.Errorf("reading config for cache key: %w", err)
	}

	h := sha256.New()
	_, _ = h.Write(data)
	_, _ = h.Write([]byte(runtime.GOOS))
	_, _ = h.Write([]byte(runtime.GOARCH))

	return hex.EncodeToString(h.Sum(nil)), nil
}
