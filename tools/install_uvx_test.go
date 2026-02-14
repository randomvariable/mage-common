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
	"testing"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

func TestUvxInstaller_Install(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tool        v1alpha1.Tool
		source      v1alpha1.ToolSource
		wantContent string
	}{
		{
			name: "creates shim with correct content",
			tool: v1alpha1.Tool{
				Name:    "ruff",
				Version: ptr.To("0.4.8"),
			},
			source: v1alpha1.ToolSource{
				Type:    v1alpha1.SourceTypeUvx,
				Package: ptr.To("ruff"),
			},
			wantContent: "#!/bin/sh\nexec uvx ruff==0.4.8 \"$@\"\n",
		},
		{
			name: "package differs from tool name",
			tool: v1alpha1.Tool{
				Name:    "yamllint",
				Version: ptr.To("1.35.1"),
			},
			source: v1alpha1.ToolSource{
				Type:    v1alpha1.SourceTypeUvx,
				Package: ptr.To("yamllint"),
			},
			wantContent: "#!/bin/sh\nexec uvx yamllint==1.35.1 \"$@\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			toolsDir := t.TempDir()
			installer := &UvxInstaller{}

			err := installer.Install(context.Background(), tt.tool, &tt.source, toolsDir)
			if err != nil {
				t.Fatalf("Install() error = %v", err)
			}

			shimPath := filepath.Join(toolsDir, runtime.GOOS, runtime.GOARCH, tt.tool.Name)

			data, err := os.ReadFile(shimPath)
			if err != nil {
				t.Fatalf("reading shim: %v", err)
			}

			if got := string(data); got != tt.wantContent {
				t.Errorf("shim content = %q, want %q", got, tt.wantContent)
			}

			// Verify the shim is executable.
			info, err := os.Stat(shimPath)
			if err != nil {
				t.Fatalf("stat shim: %v", err)
			}

			if info.Mode().Perm()&0o111 == 0 {
				t.Error("shim is not executable")
			}
		})
	}
}

func TestUvxInstaller_InstallIdempotent(t *testing.T) {
	t.Parallel()

	toolsDir := t.TempDir()
	installer := &UvxInstaller{}

	tool := v1alpha1.Tool{Name: "ruff", Version: ptr.To("0.4.8")}
	source := v1alpha1.ToolSource{Type: v1alpha1.SourceTypeUvx, Package: ptr.To("ruff")}

	// Install twice; second call should be a no-op.
	err := installer.Install(context.Background(), tool, &source, toolsDir)
	if err != nil {
		t.Fatalf("first Install() error = %v", err)
	}

	err = installer.Install(context.Background(), tool, &source, toolsDir)
	if err != nil {
		t.Fatalf("second Install() error = %v", err)
	}

	shimPath := filepath.Join(toolsDir, runtime.GOOS, runtime.GOARCH, tool.Name)

	data, err := os.ReadFile(shimPath)
	if err != nil {
		t.Fatalf("reading shim: %v", err)
	}

	want := "#!/bin/sh\nexec uvx ruff==0.4.8 \"$@\"\n"
	if got := string(data); got != want {
		t.Errorf("shim content after idempotent install = %q, want %q", got, want)
	}
}

func TestUvxInstaller_IsInstalled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shimContent string
		tool        v1alpha1.Tool
		wantInstall bool
	}{
		{
			name:        "correct version installed",
			shimContent: "#!/bin/sh\nexec uvx ruff==0.4.8 \"$@\"\n",
			tool:        v1alpha1.Tool{Name: "ruff", Version: ptr.To("0.4.8")},
			wantInstall: true,
		},
		{
			name:        "wrong version installed",
			shimContent: "#!/bin/sh\nexec uvx ruff==0.3.0 \"$@\"\n",
			tool:        v1alpha1.Tool{Name: "ruff", Version: ptr.To("0.4.8")},
			wantInstall: false,
		},
		{
			name:        "shim does not exist",
			shimContent: "",
			tool:        v1alpha1.Tool{Name: "ruff", Version: ptr.To("0.4.8")},
			wantInstall: false,
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

			if tt.shimContent != "" {
				shimPath := filepath.Join(platformDir, tt.tool.Name)

				err = os.WriteFile(shimPath, []byte(tt.shimContent), shimPerm)
				if err != nil {
					t.Fatalf("writing shim: %v", err)
				}
			}

			installer := &UvxInstaller{}

			got, err := installer.IsInstalled(tt.tool, toolsDir)
			if err != nil {
				t.Fatalf("IsInstalled() error = %v", err)
			}

			if got != tt.wantInstall {
				t.Errorf("IsInstalled() = %v, want %v", got, tt.wantInstall)
			}
		})
	}
}
