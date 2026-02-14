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
	"strings"
	"testing"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

func TestGemInstallerIsInstalled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string)
		tool      v1alpha1.Tool
		want      bool
		wantErr   bool
		errSubstr string
	}{
		{
			name: "binstub exists",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				platformDir := filepath.Join(dir, runtime.GOOS, runtime.GOARCH)

				err := os.MkdirAll(platformDir, 0o750)
				if err != nil {
					t.Fatalf("creating platform dir: %v", err)
				}

				err = os.WriteFile(filepath.Join(platformDir, "mdl"), []byte("#!/usr/bin/env ruby\n"), 0o755)
				if err != nil {
					t.Fatalf("creating fake binstub: %v", err)
				}
			},
			tool: v1alpha1.Tool{Name: "mdl", Version: ptr.To("0.13.0")},
			want: true,
		},
		{
			name:  "binstub does not exist",
			setup: func(_ *testing.T, _ string) {},
			tool:  v1alpha1.Tool{Name: "mdl", Version: ptr.To("0.13.0")},
			want:  false,
		},
		{
			name: "path is a directory not a file",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				binstubDir := filepath.Join(dir, runtime.GOOS, runtime.GOARCH, "mdl")

				err := os.MkdirAll(binstubDir, 0o750)
				if err != nil {
					t.Fatalf("creating directory at binstub path: %v", err)
				}
			},
			tool:      v1alpha1.Tool{Name: "mdl", Version: ptr.To("0.13.0")},
			want:      false,
			wantErr:   true,
			errSubstr: "directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			installer := &GemInstaller{}

			got, err := installer.IsInstalled(tt.tool, tmpDir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("IsInstalled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGemInstallerRuntimeAvailable(t *testing.T) {
	t.Parallel()

	installer := &GemInstaller{}
	err := installer.RuntimeAvailable()

	// We verify the behaviour matches the system state.
	gemFound := checkBinaryOnPath("gem")

	if gemFound {
		if err != nil {
			t.Errorf("RuntimeAvailable() returned error when gem is on PATH: %v", err)
		}
	} else {
		if err == nil {
			t.Error("RuntimeAvailable() returned nil when gem is not on PATH")
		}

		if !errors.Is(err, ErrRuntimeNotFound) {
			t.Errorf("RuntimeAvailable() error = %v, want wrapping %v", err, ErrRuntimeNotFound)
		}
	}
}

// checkBinaryOnPath checks if a binary is available on the system PATH.
func checkBinaryOnPath(name string) bool {
	_, err := exec.LookPath(name)

	return err == nil
}
