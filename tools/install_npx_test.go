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
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

func TestNpxInstallerInstall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tool        v1alpha1.Tool
		source      v1alpha1.ToolSource
		wantShebang string
		wantExec    string
	}{
		{
			name: "creates shim with correct content",
			tool: v1alpha1.Tool{
				Name:    "commitlint",
				Version: ptr.To("19.6.0"),
			},
			source: v1alpha1.ToolSource{
				Type:    v1alpha1.SourceTypeNpx,
				Package: ptr.To("@commitlint/cli"),
			},
			wantShebang: "#!/bin/sh",
			wantExec:    `exec npx @commitlint/cli@19.6.0 "$@"`,
		},
		{
			name: "creates shim for scoped package with version prefix",
			tool: v1alpha1.Tool{
				Name:    "prettier",
				Version: ptr.To("v3.2.0"),
			},
			source: v1alpha1.ToolSource{
				Type:    v1alpha1.SourceTypeNpx,
				Package: ptr.To("prettier"),
			},
			wantShebang: "#!/bin/sh",
			wantExec:    `exec npx prettier@3.2.0 "$@"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			toolsDir := t.TempDir()
			platformDir := filepath.Join(toolsDir, runtime.GOOS, runtime.GOARCH)

			installer := &NpxInstaller{}

			err := installer.Install(context.Background(), tt.tool, &tt.source, toolsDir)
			if err != nil {
				t.Fatalf("Install() error = %v", err)
			}

			shimPath := filepath.Join(platformDir, tt.tool.Name)

			data, err := os.ReadFile(shimPath)
			if err != nil {
				t.Fatalf("reading shim: %v", err)
			}

			content := string(data)

			if !strings.HasPrefix(content, tt.wantShebang) {
				t.Errorf("shim does not start with shebang %q, got:\n%s", tt.wantShebang, content)
			}

			if !strings.Contains(content, tt.wantExec) {
				t.Errorf("shim does not contain exec line %q, got:\n%s", tt.wantExec, content)
			}

			if !strings.Contains(content, versionMarker(ptr.Deref(tt.tool.Version, ""))) {
				t.Errorf("shim does not contain version marker %q, got:\n%s", versionMarker(ptr.Deref(tt.tool.Version, "")), content)
			}

			// Verify the shim is executable.
			info, err := os.Stat(shimPath)
			if err != nil {
				t.Fatalf("stat shim: %v", err)
			}

			if info.Mode().Perm()&0o111 == 0 {
				t.Errorf("shim is not executable, mode = %v", info.Mode())
			}
		})
	}
}

func TestNpxInstallerInstallSkipsExisting(t *testing.T) {
	t.Parallel()

	toolsDir := t.TempDir()
	platformDir := filepath.Join(toolsDir, runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(platformDir, 0o750)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tool := v1alpha1.Tool{Name: "commitlint", Version: ptr.To("19.6.0")}
	source := v1alpha1.ToolSource{Type: v1alpha1.SourceTypeNpx, Package: ptr.To("@commitlint/cli")}

	installer := &NpxInstaller{}

	// First install.
	err = installer.Install(context.Background(), tool, &source, toolsDir)
	if err != nil {
		t.Fatalf("first Install() error = %v", err)
	}

	shimPath := filepath.Join(platformDir, tool.Name)

	firstStat, err := os.Stat(shimPath)
	if err != nil {
		t.Fatalf("stat after first install: %v", err)
	}

	// Second install should be a no-op (file unchanged).
	err = installer.Install(context.Background(), tool, &source, toolsDir)
	if err != nil {
		t.Fatalf("second Install() error = %v", err)
	}

	secondStat, err := os.Stat(shimPath)
	if err != nil {
		t.Fatalf("stat after second install: %v", err)
	}

	if !firstStat.ModTime().Equal(secondStat.ModTime()) {
		t.Error("shim was rewritten on second install, expected skip")
	}
}

func TestNpxInstallerIsInstalled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		shimContent   string
		toolVersion   string
		wantInstalled bool
	}{
		{
			name:          "correct version returns true",
			shimContent:   shimContent("@commitlint/cli", "19.6.0"),
			toolVersion:   "19.6.0",
			wantInstalled: true,
		},
		{
			name:          "wrong version returns false",
			shimContent:   shimContent("@commitlint/cli", "18.0.0"),
			toolVersion:   "19.6.0",
			wantInstalled: false,
		},
		{
			name:          "empty file returns false",
			shimContent:   "",
			toolVersion:   "19.6.0",
			wantInstalled: false,
		},
		{
			name:          "arbitrary content without marker returns false",
			shimContent:   "#!/bin/sh\necho hello\n",
			toolVersion:   "19.6.0",
			wantInstalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			toolsDir := t.TempDir()
			platformDir := filepath.Join(toolsDir, runtime.GOOS, runtime.GOARCH)

			err := os.MkdirAll(platformDir, 0o750)
			if err != nil {
				t.Fatalf("mkdir: %v", err)
			}

			shimPath := filepath.Join(platformDir, "commitlint")

			err = os.WriteFile(shimPath, []byte(tt.shimContent), shimPerm)
			if err != nil {
				t.Fatalf("write shim: %v", err)
			}

			tool := v1alpha1.Tool{Name: "commitlint", Version: ptr.To(tt.toolVersion)}
			installer := &NpxInstaller{}

			got, err := installer.IsInstalled(tool, toolsDir)
			if err != nil {
				t.Fatalf("IsInstalled() error = %v", err)
			}

			if got != tt.wantInstalled {
				t.Errorf("IsInstalled() = %v, want %v", got, tt.wantInstalled)
			}
		})
	}
}

func TestNpxInstallerIsInstalledNotExists(t *testing.T) {
	t.Parallel()

	toolsDir := t.TempDir()
	platformDir := filepath.Join(toolsDir, runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(platformDir, 0o750)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tool := v1alpha1.Tool{Name: "nonexistent", Version: ptr.To("1.0.0")}
	installer := &NpxInstaller{}

	got, err := installer.IsInstalled(tool, toolsDir)
	if err != nil {
		t.Fatalf("IsInstalled() error = %v", err)
	}

	if got {
		t.Error("IsInstalled() = true for non-existent shim, want false")
	}
}
