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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// VersionedName returns the versioned binary name: <name>-<version>.
func VersionedName(name, version string) string {
	return name + "-" + version
}

// BareVersion strips a leading "v" prefix from a version string.
// Package managers like cargo, gem, pip/uvx, and npm expect bare semver
// (e.g., "1.32.0") while .tools.yaml stores versions with the Go/Git
// convention "v1.32.0". Internal version tracking (symlinks, shim markers)
// should use the original version string; only strip when invoking external
// package manager commands.
func BareVersion(version string) string {
	return strings.TrimPrefix(version, "v")
}

// IsCurrentVersion checks if the symlink for a tool points to the expected
// versioned binary.
func IsCurrentVersion(toolName, expectedVersion, toolsDir string) (bool, error) {
	dir := PlatformToolsDir(toolsDir)
	symlinkPath := filepath.Join(dir, toolName)

	target, err := os.Readlink(symlinkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("reading symlink %q: %w", symlinkPath, err)
	}

	expectedTarget := VersionedName(toolName, expectedVersion)

	return filepath.Base(target) == expectedTarget, nil
}

// SetVersion creates the versioned binary (by renaming src) and atomically
// updates the symlink to point to it.
func SetVersion(toolName, version, toolsDir, srcPath string) error {
	dir := PlatformToolsDir(toolsDir)
	versionedPath := filepath.Join(dir, VersionedName(toolName, version))

	err := moveFile(srcPath, versionedPath)
	if err != nil {
		return fmt.Errorf("moving binary to versioned path: %w", err)
	}

	const binaryPerm = 0o755

	err = os.Chmod(versionedPath, binaryPerm)
	if err != nil {
		return fmt.Errorf("setting binary permissions: %w", err)
	}

	symlinkPath := filepath.Join(dir, toolName)

	// Atomic symlink update: create temp symlink then rename.
	tmpLink := symlinkPath + ".tmp"

	// Clean up any leftover temp symlink.
	_ = os.Remove(tmpLink)

	err = os.Symlink(VersionedName(toolName, version), tmpLink)
	if err != nil {
		return fmt.Errorf("creating temp symlink: %w", err)
	}

	err = os.Rename(tmpLink, symlinkPath)
	if err != nil {
		_ = os.Remove(tmpLink)

		return fmt.Errorf("atomic symlink rename: %w", err)
	}

	return nil
}

// moveFile moves src to dst, falling back to copy+delete when os.Rename
// fails with EXDEV (cross-device link). This happens when src lives on a
// different filesystem mount (e.g. /tmp) than dst.
func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) || !errors.Is(linkErr.Err, syscall.EXDEV) {
		return fmt.Errorf("renaming file: %w", err)
	}

	return copyAndRemove(src, dst)
}

func copyAndRemove(src, dst string) error {
	srcFile, err := os.Open(filepath.Clean(src))
	if err != nil {
		return fmt.Errorf("opening source: %w", err)
	}

	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(filepath.Clean(dst))
	if err != nil {
		return fmt.Errorf("creating destination: %w", err)
	}

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		_ = dstFile.Close()
		_ = os.Remove(dst)

		return fmt.Errorf("copying file: %w", err)
	}

	if err = dstFile.Close(); err != nil {
		_ = os.Remove(dst)

		return fmt.Errorf("closing destination: %w", err)
	}

	if err = os.Remove(src); err != nil {
		return fmt.Errorf("removing source: %w", err)
	}

	return nil
}
