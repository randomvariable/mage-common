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

package kind

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		yaml       string
		wantErr    bool
		errContain string
		wantName   string
	}{
		{
			name: "minimal valid config",
			yaml: `apiVersion: kind.x-k8s.io/v1alpha4
kind: Cluster
nodes:
  - role: control-plane
`,
			wantName: "kind",
		},
		{
			name: "config with explicit name",
			yaml: `apiVersion: kind.x-k8s.io/v1alpha4
kind: Cluster
name: my-cluster
nodes:
  - role: control-plane
`,
			wantName: "my-cluster",
		},
		{
			name:       "invalid YAML",
			yaml:       `not: valid: yaml: [`,
			wantErr:    true,
			errContain: "config file invalid",
		},
		{
			name: "wrong apiVersion",
			yaml: `apiVersion: wrong.group/v1
kind: Cluster
nodes:
  - role: control-plane
`,
			wantErr:    true,
			errContain: "expected kind.x-k8s.io/v1alpha4",
		},
		{
			name: "wrong kind",
			yaml: `apiVersion: kind.x-k8s.io/v1alpha4
kind: NotACluster
nodes:
  - role: control-plane
`,
			wantErr:    true,
			errContain: "expected kind.x-k8s.io/v1alpha4",
		},
		{
			name: "config with GPU",
			yaml: `apiVersion: kind.x-k8s.io/v1alpha4
kind: Cluster
name: gpu-cluster
nodes:
  - role: control-plane
gpu:
  type: nvidia
`,
			wantName: "gpu-cluster",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			cfg, err := ParseConfig([]byte(tc.yaml))

			if tc.wantErr {
				g.Expect(err).To(HaveOccurred())

				if tc.errContain != "" {
					g.Expect(err.Error()).To(ContainSubstring(tc.errContain))
				}

				return
			}

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(cfg.Name).To(Equal(tc.wantName))
		})
	}
}

func TestLoadClusterConfig(t *testing.T) {
	t.Parallel()

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		_, err := LoadClusterConfig("/nonexistent/.kind-cluster.yaml")
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("config file not found"))
	})

	t.Run("valid file", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		dir := t.TempDir()
		path := filepath.Join(dir, ".kind-cluster.yaml")

		err := os.WriteFile(path, []byte(`apiVersion: kind.x-k8s.io/v1alpha4
kind: Cluster
name: test-cluster
nodes:
  - role: control-plane
`), 0o600)
		g.Expect(err).NotTo(HaveOccurred())

		cfg, err := LoadClusterConfig(path)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cfg.Name).To(Equal("test-cluster"))
	})
}
