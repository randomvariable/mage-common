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
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBareVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"v1.32.0", "1.32.0"},
		{"1.32.0", "1.32.0"},
		{"v0.4.8", "0.4.8"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := BareVersion(tt.input); got != tt.want {
			t.Errorf("BareVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCopyAndRemove(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcPath := filepath.Join(srcDir, "binary")
	dstPath := filepath.Join(dstDir, "binary")

	content := []byte("fake-binary-content")

	err := os.WriteFile(srcPath, content, 0o755)
	if err != nil {
		t.Fatalf("write source: %v", err)
	}

	err = copyAndRemove(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyAndRemove() error = %v", err)
	}

	// Source should be removed.
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Error("source file still exists after copyAndRemove")
	}

	// Destination should have the same content.
	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("reading destination: %v", err)
	}

	if !bytes.Equal(got, content) {
		t.Errorf("destination content = %q, want %q", got, content)
	}
}

func TestVersionedName(t *testing.T) {
	t.Parallel()

	got := VersionedName("golangci-lint", "v2.4.0")
	want := "golangci-lint-v2.4.0"

	if got != want {
		t.Errorf("VersionedName() = %q, want %q", got, want)
	}
}

func TestSetVersionAndIsCurrentVersion(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	toolsDir := tmpDir
	platformDir := filepath.Join(toolsDir, runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(platformDir, 0o750)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create a fake binary.
	srcPath := filepath.Join(platformDir, "golangci-lint.download")

	err = os.WriteFile(srcPath, []byte("fake-binary"), 0o644)
	if err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	// Set version v2.4.0.
	err = SetVersion("golangci-lint", "v2.4.0", toolsDir, srcPath)
	if err != nil {
		t.Fatalf("SetVersion() error = %v", err)
	}

	// Verify symlink exists and points to versioned binary.
	current, err := IsCurrentVersion("golangci-lint", "v2.4.0", toolsDir)
	if err != nil {
		t.Fatalf("IsCurrentVersion() error = %v", err)
	}

	if !current {
		t.Error("IsCurrentVersion() = false, want true")
	}

	// Different version should return false.
	current, err = IsCurrentVersion("golangci-lint", "v2.5.0", toolsDir)
	if err != nil {
		t.Fatalf("IsCurrentVersion(v2.5.0) error = %v", err)
	}

	if current {
		t.Error("IsCurrentVersion(v2.5.0) = true, want false")
	}
}

func TestIsCurrentVersionNotInstalled(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	platformDir := filepath.Join(tmpDir, runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(platformDir, 0o750)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	current, err := IsCurrentVersion("nonexistent", "v1.0.0", tmpDir)
	if err != nil {
		t.Fatalf("IsCurrentVersion() error = %v", err)
	}

	if current {
		t.Error("IsCurrentVersion() = true for non-existent tool, want false")
	}
}

func TestSetVersionUpdate(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	toolsDir := tmpDir
	platformDir := filepath.Join(toolsDir, runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(platformDir, 0o750)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Install v1.0.0.
	src1 := filepath.Join(platformDir, "tool.dl1")

	err = os.WriteFile(src1, []byte("v1"), 0o644)
	if err != nil {
		t.Fatalf("write v1: %v", err)
	}

	err = SetVersion("tool", "v1.0.0", toolsDir, src1)
	if err != nil {
		t.Fatalf("SetVersion(v1) error = %v", err)
	}

	// Update to v2.0.0.
	src2 := filepath.Join(platformDir, "tool.dl2")

	err = os.WriteFile(src2, []byte("v2"), 0o644)
	if err != nil {
		t.Fatalf("write v2: %v", err)
	}

	err = SetVersion("tool", "v2.0.0", toolsDir, src2)
	if err != nil {
		t.Fatalf("SetVersion(v2) error = %v", err)
	}

	// Should now point to v2.
	current, err := IsCurrentVersion("tool", "v2.0.0", toolsDir)
	if err != nil {
		t.Fatalf("IsCurrentVersion(v2) error = %v", err)
	}

	if !current {
		t.Error("IsCurrentVersion(v2) = false, want true")
	}

	// v1 should no longer be current.
	current, err = IsCurrentVersion("tool", "v1.0.0", toolsDir)
	if err != nil {
		t.Fatalf("IsCurrentVersion(v1) error = %v", err)
	}

	if current {
		t.Error("IsCurrentVersion(v1) = true after update, want false")
	}
}
