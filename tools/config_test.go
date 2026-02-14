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
	"testing"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

func TestLoadToolConfigurationValid(t *testing.T) {
	t.Parallel()

	yaml := `apiVersion: mage-common.randomvariable.co.uk/v1alpha1
kind: ToolConfiguration
spec:
  toolsDir: hack/bin
  tools:
    - name: controller-gen
      version: v0.18.0
      sources:
        - type: go
          url: sigs.k8s.io/controller-tools/cmd/controller-gen
`
	path := writeTestConfig(t, yaml)

	config, err := LoadToolConfiguration(path)
	if err != nil {
		t.Fatalf("LoadToolConfiguration() error = %v", err)
	}

	if ptr.Deref(config.Spec.ToolsDir, "") != "hack/bin" {
		t.Errorf("ToolsDir = %q, want %q", ptr.Deref(config.Spec.ToolsDir, ""), "hack/bin")
	}

	if len(config.Spec.Tools) != 1 {
		t.Fatalf("len(Tools) = %d, want 1", len(config.Spec.Tools))
	}

	if config.Spec.Tools[0].Name != "controller-gen" {
		t.Errorf("Tools[0].Name = %q, want %q", config.Spec.Tools[0].Name, "controller-gen")
	}
}

func TestLoadToolConfigurationDefaults(t *testing.T) {
	t.Parallel()

	yaml := `apiVersion: mage-common.randomvariable.co.uk/v1alpha1
kind: ToolConfiguration
spec:
  tools:
    - name: controller-gen
      version: v0.18.0
      sources:
        - type: go
          url: sigs.k8s.io/controller-tools/cmd/controller-gen
`
	path := writeTestConfig(t, yaml)

	config, err := LoadToolConfiguration(path)
	if err != nil {
		t.Fatalf("LoadToolConfiguration() error = %v", err)
	}

	if ptr.Deref(config.Spec.ToolsDir, "") != v1alpha1.DefaultToolsDir {
		t.Errorf("ToolsDir = %q, want default %q", ptr.Deref(config.Spec.ToolsDir, ""), v1alpha1.DefaultToolsDir)
	}
}

func TestLoadToolConfigurationMissingFile(t *testing.T) {
	t.Parallel()

	_, err := LoadToolConfiguration("/nonexistent/.tools.yaml")
	if err == nil {
		t.Fatal("LoadToolConfiguration() error = nil, want error")
	}

	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("error = %v, want ErrConfigNotFound", err)
	}
}

func TestLoadToolConfigurationMalformedYAML(t *testing.T) {
	t.Parallel()

	path := writeTestConfig(t, "not: valid: yaml: [[[")

	_, err := LoadToolConfiguration(path)
	if err == nil {
		t.Fatal("LoadToolConfiguration() error = nil, want error")
	}

	if !errors.Is(err, ErrConfigInvalid) {
		t.Errorf("error = %v, want ErrConfigInvalid", err)
	}
}

func TestLoadToolConfigurationValidationFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "duplicate tool names",
			yaml: `apiVersion: mage-common.randomvariable.co.uk/v1alpha1
kind: ToolConfiguration
spec:
  tools:
    - name: tool-a
      version: v1.0.0
      sources:
        - type: go
          url: example.com/tool-a
    - name: tool-a
      version: v2.0.0
      sources:
        - type: go
          url: example.com/tool-a-v2
`,
		},
		{
			name: "missing sources",
			yaml: `apiVersion: mage-common.randomvariable.co.uk/v1alpha1
kind: ToolConfiguration
spec:
  tools:
    - name: tool-a
      version: v1.0.0
      sources: []
`,
		},
		{
			name: "HTTP URL without allowInsecure",
			yaml: `apiVersion: mage-common.randomvariable.co.uk/v1alpha1
kind: ToolConfiguration
spec:
  tools:
    - name: tool-a
      version: v1.0.0
      sources:
        - type: download
          url: "http://example.com/tool.tar.gz"
          skipChecksum: true
`,
		},
		{
			name: "weak checksum algorithm",
			yaml: `apiVersion: mage-common.randomvariable.co.uk/v1alpha1
kind: ToolConfiguration
spec:
  tools:
    - name: tool-a
      version: v1.0.0
      sources:
        - type: download
          url: "https://example.com/tool.tar.gz"
          checksum:
            inline: "md5:abc123"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := writeTestConfig(t, tt.yaml)

			_, err := LoadToolConfiguration(path)
			if err == nil {
				t.Fatal("LoadToolConfiguration() error = nil, want validation error")
			}

			if !errors.Is(err, ErrConfigInvalid) {
				t.Errorf("error = %v, want ErrConfigInvalid", err)
			}
		})
	}
}

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, ".tools.yaml")

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	return path
}
