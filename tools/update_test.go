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
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

// fakeQuerier returns a predetermined version from LatestVersion.
type fakeQuerier struct {
	version string
	err     error
}

func (f *fakeQuerier) LatestVersion(_ context.Context, _ v1alpha1.Tool, _ *v1alpha1.ToolSource) (string, error) {
	return f.version, f.err
}

func TestFilterLatestVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		versionList string
		tagPrefix   string
		want        string
		wantErr     bool
	}{
		{
			name:        "simple semver list",
			versionList: "v0.17.0\nv0.18.0\nv0.16.0\n",
			want:        "v0.18.0",
		},
		{
			name:        "skips pre-releases",
			versionList: "v1.0.0\nv1.1.0-rc1\nv1.0.1\n",
			want:        "v1.0.1",
		},
		{
			name:        "tag prefix filtering",
			versionList: "kustomize/v5.6.0\nkustomize/v5.7.1\ncmd/config/v0.15.0\n",
			tagPrefix:   "kustomize/",
			want:        "kustomize/v5.7.1",
		},
		{
			name:        "empty list",
			versionList: "",
			wantErr:     true,
		},
		{
			name:        "no stable versions",
			versionList: "v1.0.0-alpha.1\nv1.0.0-beta.2\n",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := filterLatestVersion(tt.versionList, tt.tagPrefix)
			if tt.wantErr {
				if err == nil {
					t.Fatal("filterLatestVersion() error = nil, want error")
				}

				return
			}

			if err != nil {
				t.Fatalf("filterLatestVersion() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("filterLatestVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReplaceToolVersion(t *testing.T) {
	t.Parallel()

	content := `spec:
  tools:
    - name: controller-gen
      version: v0.17.0
      sources:
        - type: go
    - name: golangci-lint
      version: v2.3.0
      sources:
        - type: download
`

	got := replaceToolVersion(content, "controller-gen", "v0.17.0", "v0.18.0")

	if !strings.Contains(got, "version: v0.18.0") {
		t.Errorf("version not replaced; got:\n%s", got)
	}

	// golangci-lint should be unchanged.
	if !strings.Contains(got, "version: v2.3.0") {
		t.Errorf("wrong tool version modified; got:\n%s", got)
	}
}

func TestUpdaterUpdateAll(t *testing.T) {
	t.Parallel()

	configContent := `apiVersion: mage-common.randomvariable.co.uk/v1alpha1
kind: ToolConfiguration
spec:
  tools:
    - name: controller-gen
      version: v0.17.0
      sources:
        - type: go
          url: sigs.k8s.io/controller-tools/cmd/controller-gen
    - name: ruff
      version: 0.4.8
      sources:
        - type: uvx
          package: ruff
`

	configPath := writeUpdateTestConfig(t, configContent)

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			Tools: []v1alpha1.Tool{
				{
					Name:    "controller-gen",
					Version: ptr.To("v0.17.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("sigs.k8s.io/controller-tools/cmd/controller-gen")},
					},
				},
				{
					Name:    "ruff",
					Version: ptr.To("0.4.8"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeUvx, Package: ptr.To("ruff")},
					},
				},
			},
		},
	}

	var logBuf bytes.Buffer

	updater := &Updater{
		Config: config,
		Queriers: map[v1alpha1.SourceType]RegistryQuerier{
			v1alpha1.SourceTypeGo:  &fakeQuerier{version: "v0.18.0"},
			v1alpha1.SourceTypeUvx: &fakeQuerier{version: "0.5.0"},
		},
		Logger: &logBuf,
	}

	err := updater.UpdateAll(context.Background(), configPath)
	if err != nil {
		t.Fatalf("UpdateAll() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading updated config: %v", err)
	}

	got := string(data)

	if !strings.Contains(got, "version: v0.18.0") {
		t.Errorf("controller-gen version not updated; got:\n%s", got)
	}

	if !strings.Contains(got, "version: 0.5.0") {
		t.Errorf("ruff version not updated; got:\n%s", got)
	}
}

func TestUpdaterSkipsLatest(t *testing.T) {
	t.Parallel()

	configContent := `apiVersion: mage-common.randomvariable.co.uk/v1alpha1
kind: ToolConfiguration
spec:
  tools:
    - name: my-tool
      version: latest
      sources:
        - type: go
          url: example.com/cmd/tool
`

	configPath := writeUpdateTestConfig(t, configContent)

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			Tools: []v1alpha1.Tool{
				{
					Name:    "my-tool",
					Version: ptr.To("latest"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("example.com/cmd/tool")},
					},
				},
			},
		},
	}

	querier := &fakeQuerier{version: "v1.0.0"}

	var logBuf bytes.Buffer

	updater := &Updater{
		Config: config,
		Queriers: map[v1alpha1.SourceType]RegistryQuerier{
			v1alpha1.SourceTypeGo: querier,
		},
		Logger: &logBuf,
	}

	err := updater.UpdateAll(context.Background(), configPath)
	if err != nil {
		t.Fatalf("UpdateAll() error = %v", err)
	}

	// File should be unchanged — "latest" tools are skipped.
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	if !strings.Contains(string(data), "version: latest") {
		t.Error("latest version was modified, should have been skipped")
	}
}

func TestUpdaterSkipsLocalOnly(t *testing.T) {
	t.Parallel()

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			Tools: []v1alpha1.Tool{
				{
					Name: "custom-gen",
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, Path: ptr.To("./cmd/custom-gen")},
					},
				},
			},
		},
	}

	configPath := writeUpdateTestConfig(t, "dummy content\n")

	var logBuf bytes.Buffer

	updater := &Updater{
		Config:   config,
		Queriers: map[v1alpha1.SourceType]RegistryQuerier{},
		Logger:   &logBuf,
	}

	err := updater.UpdateAll(context.Background(), configPath)
	if err != nil {
		t.Fatalf("UpdateAll() error = %v", err)
	}
}

func TestUpdaterUpdateByName(t *testing.T) {
	t.Parallel()

	configContent := `apiVersion: mage-common.randomvariable.co.uk/v1alpha1
kind: ToolConfiguration
spec:
  tools:
    - name: controller-gen
      version: v0.17.0
      sources:
        - type: go
          url: sigs.k8s.io/controller-tools/cmd/controller-gen
`

	configPath := writeUpdateTestConfig(t, configContent)

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			Tools: []v1alpha1.Tool{
				{
					Name:    "controller-gen",
					Version: ptr.To("v0.17.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("sigs.k8s.io/controller-tools/cmd/controller-gen")},
					},
				},
			},
		},
	}

	var logBuf bytes.Buffer

	updater := &Updater{
		Config: config,
		Queriers: map[v1alpha1.SourceType]RegistryQuerier{
			v1alpha1.SourceTypeGo: &fakeQuerier{version: "v0.18.0"},
		},
		Logger: &logBuf,
	}

	err := updater.UpdateByName(context.Background(), "controller-gen", configPath)
	if err != nil {
		t.Fatalf("UpdateByName() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	if !strings.Contains(string(data), "version: v0.18.0") {
		t.Errorf("version not updated; got:\n%s", string(data))
	}
}

func TestUpdaterUpdateByNameNotFound(t *testing.T) {
	t.Parallel()

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			Tools: []v1alpha1.Tool{},
		},
	}

	var logBuf bytes.Buffer

	updater := &Updater{
		Config:   config,
		Queriers: map[v1alpha1.SourceType]RegistryQuerier{},
		Logger:   &logBuf,
	}

	err := updater.UpdateByName(context.Background(), "nonexistent", "/dev/null")

	if !errors.Is(err, ErrToolNotFound) {
		t.Errorf("error = %v, want ErrToolNotFound", err)
	}
}

func TestUpdaterRegistryFailureContinues(t *testing.T) {
	t.Parallel()

	configContent := `apiVersion: mage-common.randomvariable.co.uk/v1alpha1
kind: ToolConfiguration
spec:
  tools:
    - name: fail-tool
      version: v1.0.0
      sources:
        - type: go
          url: example.com/cmd/fail
    - name: ok-tool
      version: v1.0.0
      sources:
        - type: npx
          package: ok-tool
`

	configPath := writeUpdateTestConfig(t, configContent)

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			Tools: []v1alpha1.Tool{
				{
					Name:    "fail-tool",
					Version: ptr.To("v1.0.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("example.com/cmd/fail")},
					},
				},
				{
					Name:    "ok-tool",
					Version: ptr.To("v1.0.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeNpx, Package: ptr.To("ok-tool")},
					},
				},
			},
		},
	}

	var logBuf bytes.Buffer

	updater := &Updater{
		Config: config,
		Queriers: map[v1alpha1.SourceType]RegistryQuerier{
			v1alpha1.SourceTypeGo:  &fakeQuerier{err: errors.New("network timeout")},
			v1alpha1.SourceTypeNpx: &fakeQuerier{version: "v2.0.0"},
		},
		Logger: &logBuf,
	}

	err := updater.UpdateAll(context.Background(), configPath)
	if err != nil {
		t.Fatalf("UpdateAll() error = %v; want nil (continues on failure)", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	got := string(data)

	// ok-tool should be updated despite fail-tool's failure.
	if !strings.Contains(got, "version: v2.0.0") {
		t.Errorf("ok-tool version not updated; got:\n%s", got)
	}

	// Verify failure was logged.
	if !strings.Contains(logBuf.String(), "failed to query") {
		t.Errorf("failure not logged; log = %q", logBuf.String())
	}
}

func writeUpdateTestConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, ".tools.yaml")

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	return path
}
