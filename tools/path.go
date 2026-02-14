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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

const (
	// toolsDirPerm is the permission for the tools directory (rwxr-x---).
	toolsDirPerm = 0o750
)

// PlatformToolsDir returns the platform-specific tools directory path:
// <toolsDir>/<GOOS>/<GOARCH>/.
func PlatformToolsDir(toolsDir string) string {
	return filepath.Join(toolsDir, runtime.GOOS, runtime.GOARCH)
}

// EnsureToolsDir creates the platform-specific tools directory with
// restricted permissions if it does not already exist.
func EnsureToolsDir(toolsDir string) (string, error) {
	dir := PlatformToolsDir(toolsDir)

	err := os.MkdirAll(dir, toolsDirPerm)
	if err != nil {
		return "", fmt.Errorf("creating tools directory %q: %w", dir, err)
	}

	return dir, nil
}

// PrependToPath adds the platform-specific tools directory to the front
// of the process PATH environment variable. It is idempotent: if the
// directory is already on PATH, it returns immediately.
func PrependToPath(toolsDir string) error {
	if IsOnPath(toolsDir) {
		return nil
	}

	dir := PlatformToolsDir(toolsDir)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving absolute path for %q: %w", dir, err)
	}

	current := os.Getenv("PATH")

	err = os.Setenv("PATH", absDir+string(os.PathListSeparator)+current)
	if err != nil {
		return fmt.Errorf("setting PATH: %w", err)
	}

	return nil
}

// ToolBinaryPath returns the full path to a tool binary within the
// platform-specific tools directory.
func ToolBinaryPath(toolsDir, toolName string) string {
	return filepath.Join(PlatformToolsDir(toolsDir), toolName)
}

// IsOnPath checks if the platform-specific tools directory is already on PATH.
func IsOnPath(toolsDir string) bool {
	dir := PlatformToolsDir(toolsDir)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}

	return slices.Contains(strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)), absDir)
}
