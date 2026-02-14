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
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPlatformToolsDir(t *testing.T) {
	t.Parallel()

	got := PlatformToolsDir("hack/bin")
	want := filepath.Join("hack", "bin", runtime.GOOS, runtime.GOARCH)

	if got != want {
		t.Errorf("PlatformToolsDir() = %q, want %q", got, want)
	}
}

func TestEnsureToolsDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	toolsDir := filepath.Join(tmpDir, "hack", "bin")

	dir, err := EnsureToolsDir(toolsDir)
	if err != nil {
		t.Fatalf("EnsureToolsDir() error = %v", err)
	}

	want := filepath.Join(toolsDir, runtime.GOOS, runtime.GOARCH)
	if dir != want {
		t.Errorf("EnsureToolsDir() = %q, want %q", dir, want)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat tools dir: %v", err)
	}

	if !info.IsDir() {
		t.Error("tools dir is not a directory")
	}
}

func TestEnsureToolsDirIdempotent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	toolsDir := filepath.Join(tmpDir, "hack", "bin")

	dir1, err := EnsureToolsDir(toolsDir)
	if err != nil {
		t.Fatalf("first EnsureToolsDir() error = %v", err)
	}

	dir2, err := EnsureToolsDir(toolsDir)
	if err != nil {
		t.Fatalf("second EnsureToolsDir() error = %v", err)
	}

	if dir1 != dir2 {
		t.Errorf("EnsureToolsDir() not idempotent: %q != %q", dir1, dir2)
	}
}

func TestToolBinaryPath(t *testing.T) {
	t.Parallel()

	got := ToolBinaryPath("hack/bin", "golangci-lint")
	want := filepath.Join("hack", "bin", runtime.GOOS, runtime.GOARCH, "golangci-lint")

	if got != want {
		t.Errorf("ToolBinaryPath() = %q, want %q", got, want)
	}
}
