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
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

func TestCargoInstallerRuntimeAvailable(t *testing.T) {
	t.Parallel()

	installer := &CargoInstaller{}

	err := installer.RuntimeAvailable()

	// cargo may or may not be on PATH in CI; verify the error type when absent.
	_, lookErr := exec.LookPath("cargo")
	if lookErr != nil {
		if err == nil {
			t.Fatal("RuntimeAvailable() returned nil, but cargo is not on PATH")
		}

		if !errors.Is(err, ErrRuntimeNotFound) {
			t.Errorf("RuntimeAvailable() error = %v, want wrapped ErrRuntimeNotFound", err)
		}
	} else if err != nil {
		t.Errorf("RuntimeAvailable() error = %v, want nil (cargo is on PATH)", err)
	}
}

func TestCargoInstallerIsInstalled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupFunc  func(t *testing.T, platformDir string)
		tool       v1alpha1.Tool
		wantResult bool
	}{
		{
			name: "not installed returns false",
			setupFunc: func(_ *testing.T, _ string) {
				// No setup: tool does not exist.
			},
			tool: v1alpha1.Tool{
				Name:    "ripgrep",
				Version: ptr.To("v14.1.0"),
			},
			wantResult: false,
		},
		{
			name: "correct version returns true",
			setupFunc: func(t *testing.T, platformDir string) {
				t.Helper()

				versionedPath := filepath.Join(platformDir, "ripgrep-v14.1.0")

				err := os.WriteFile(versionedPath, []byte("fake"), 0o755)
				if err != nil {
					t.Fatalf("write versioned binary: %v", err)
				}

				symlinkPath := filepath.Join(platformDir, "ripgrep")

				err = os.Symlink("ripgrep-v14.1.0", symlinkPath)
				if err != nil {
					t.Fatalf("create symlink: %v", err)
				}
			},
			tool: v1alpha1.Tool{
				Name:    "ripgrep",
				Version: ptr.To("v14.1.0"),
			},
			wantResult: true,
		},
		{
			name: "wrong version returns false",
			setupFunc: func(t *testing.T, platformDir string) {
				t.Helper()

				versionedPath := filepath.Join(platformDir, "ripgrep-v13.0.0")

				err := os.WriteFile(versionedPath, []byte("fake"), 0o755)
				if err != nil {
					t.Fatalf("write versioned binary: %v", err)
				}

				symlinkPath := filepath.Join(platformDir, "ripgrep")

				err = os.Symlink("ripgrep-v13.0.0", symlinkPath)
				if err != nil {
					t.Fatalf("create symlink: %v", err)
				}
			},
			tool: v1alpha1.Tool{
				Name:    "ripgrep",
				Version: ptr.To("v14.1.0"),
			},
			wantResult: false,
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

			tt.setupFunc(t, platformDir)

			installer := &CargoInstaller{}

			got, err := installer.IsInstalled(tt.tool, tmpDir)
			if err != nil {
				t.Fatalf("IsInstalled() error = %v", err)
			}

			if got != tt.wantResult {
				t.Errorf("IsInstalled() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}
