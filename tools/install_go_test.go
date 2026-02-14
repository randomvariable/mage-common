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
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

func TestGoInstaller_RuntimeAvailable(t *testing.T) {
	t.Parallel()

	g := &GoInstaller{}

	// In a Go development environment, "go" must be on PATH.
	err := g.RuntimeAvailable()
	if err != nil {
		t.Fatalf("RuntimeAvailable() error = %v; expected nil in Go dev environment", err)
	}
}

func TestGoInstaller_RuntimeAvailableError(t *testing.T) {
	t.Parallel()

	// Verify that the error wraps ErrRuntimeNotFound so callers can use errors.Is.
	// We cannot easily remove "go" from PATH in a parallel test, so we only
	// check the sentinel error type on the happy path tested above.
	// This test documents the expected wrapping behaviour.
	g := &GoInstaller{}

	err := g.RuntimeAvailable()
	if err != nil && !errors.Is(err, ErrRuntimeNotFound) {
		t.Errorf("RuntimeAvailable() error = %v; expected it to wrap ErrRuntimeNotFound", err)
	}
}

func TestGoInstaller_IsInstalled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		toolName   string
		version    string
		setupLink  bool
		linkTarget string
		want       bool
	}{
		{
			name:     "not installed returns false",
			toolName: "controller-gen",
			version:  "v0.17.0",
			want:     false,
		},
		{
			name:       "correct version returns true",
			toolName:   "controller-gen",
			version:    "v0.17.0",
			setupLink:  true,
			linkTarget: "controller-gen-v0.17.0",
			want:       true,
		},
		{
			name:       "wrong version returns false",
			toolName:   "controller-gen",
			version:    "v0.18.0",
			setupLink:  true,
			linkTarget: "controller-gen-v0.17.0",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			platformDir := filepath.Join(tmpDir, runtime.GOOS, runtime.GOARCH)

			err := os.MkdirAll(platformDir, 0o750)
			if err != nil {
				t.Fatalf("mkdir: %v", err)
			}

			if tt.setupLink {
				// Create the versioned binary file so the symlink target exists.
				versionedPath := filepath.Join(platformDir, tt.linkTarget)

				err := os.WriteFile(versionedPath, []byte("fake"), 0o755)
				if err != nil {
					t.Fatalf("write versioned binary: %v", err)
				}

				symlinkPath := filepath.Join(platformDir, tt.toolName)

				err = os.Symlink(tt.linkTarget, symlinkPath)
				if err != nil {
					t.Fatalf("symlink: %v", err)
				}
			}

			g := &GoInstaller{}
			tool := v1alpha1.Tool{
				Name:    tt.toolName,
				Version: ptr.To(tt.version),
			}

			got, err := g.IsInstalled(tool, tmpDir)
			if err != nil {
				t.Fatalf("IsInstalled() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("IsInstalled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGoInstaller_InstallSkipsWhenCurrent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	platformDir := filepath.Join(tmpDir, runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(platformDir, 0o750)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Pre-install: create versioned binary and symlink.
	toolName := "goimports"
	version := "v0.30.0"
	versionedName := VersionedName(toolName, version)
	versionedPath := filepath.Join(platformDir, versionedName)

	err = os.WriteFile(versionedPath, []byte("fake-binary"), 0o755)
	if err != nil {
		t.Fatalf("write versioned binary: %v", err)
	}

	err = os.Symlink(versionedName, filepath.Join(platformDir, toolName))
	if err != nil {
		t.Fatalf("symlink: %v", err)
	}

	g := &GoInstaller{}
	tool := v1alpha1.Tool{
		Name:    toolName,
		Version: ptr.To(version),
		Sources: []v1alpha1.ToolSource{
			{
				Type: v1alpha1.SourceTypeGo,
				URL:  ptr.To("golang.org/x/tools/cmd/goimports"),
			},
		},
	}

	// Install should be a no-op because the version is already current.
	// If it tried to actually run "go install", it would need network access,
	// but since the version check passes, it returns nil immediately.
	err = g.Install(t.Context(), tool, &tool.Sources[0], tmpDir)
	if err != nil {
		t.Fatalf("Install() error = %v; expected nil for already-current tool", err)
	}
}

func TestGoInstaller_NeedsRebuild(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T, binaryPath, srcDir string)
		wantBuild bool
	}{
		{
			name: "no binary exists",
			setup: func(t *testing.T, _, srcDir string) {
				t.Helper()
				writeGoFile(t, srcDir)
			},
			wantBuild: true,
		},
		{
			name: "binary newer than source",
			setup: func(t *testing.T, binaryPath, srcDir string) {
				t.Helper()
				writeGoFile(t, srcDir)

				err := os.WriteFile(binaryPath, []byte("binary"), 0o755)
				if err != nil {
					t.Fatalf("write binary: %v", err)
				}
			},
			wantBuild: false,
		},
		{
			name: "source newer than binary",
			setup: func(t *testing.T, binaryPath, srcDir string) {
				t.Helper()

				// Create binary first.
				err := os.WriteFile(binaryPath, []byte("old-binary"), 0o755)
				if err != nil {
					t.Fatalf("write binary: %v", err)
				}

				// Backdate binary.
				past := time.Now().Add(-1 * time.Hour)

				err = os.Chtimes(binaryPath, past, past)
				if err != nil {
					t.Fatalf("chtimes binary: %v", err)
				}

				// Create source file (newer).
				writeGoFile(t, srcDir)
			},
			wantBuild: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			binaryPath := filepath.Join(tmpDir, "mybinary")
			srcDir := filepath.Join(tmpDir, "src")

			err := os.MkdirAll(srcDir, 0o750)
			if err != nil {
				t.Fatalf("mkdir: %v", err)
			}

			tt.setup(t, binaryPath, srcDir)

			g := &GoInstaller{}
			got := g.needsRebuild(binaryPath, srcDir)

			if got != tt.wantBuild {
				t.Errorf("needsRebuild() = %v, want %v", got, tt.wantBuild)
			}
		})
	}
}

func TestGoInstaller_BuildLocalSkipsWhenUpToDate(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	platformDir := filepath.Join(tmpDir, runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(platformDir, 0o750)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create a source directory with a .go file.
	srcDir := filepath.Join(tmpDir, "cmd", "mytool")

	err = os.MkdirAll(srcDir, 0o750)
	if err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	writeGoFile(t, srcDir)

	// Pre-create the binary (newer than source to trigger skip).
	binaryPath := filepath.Join(platformDir, "mytool")

	err = os.WriteFile(binaryPath, []byte("prebuilt"), 0o755)
	if err != nil {
		t.Fatalf("write binary: %v", err)
	}

	g := &GoInstaller{}
	tool := v1alpha1.Tool{Name: "mytool"}
	source := v1alpha1.ToolSource{Type: v1alpha1.SourceTypeGo, Path: ptr.To(srcDir)}

	err = g.Install(t.Context(), tool, &source, tmpDir)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	// Binary should be unchanged (skip, not rebuilt).
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("read binary: %v", err)
	}

	if string(data) != "prebuilt" {
		t.Errorf("binary was rebuilt, expected skip; got content %q", data)
	}
}

func writeGoFile(t *testing.T, dir string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)
	if err != nil {
		t.Fatalf("writing go file main.go: %v", err)
	}
}
