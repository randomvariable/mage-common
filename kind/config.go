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
	"fmt"
	"os"

	kindv1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/yaml"
)

// DefaultConfigFile is the default configuration file name.
const DefaultConfigFile = ".kind-cluster.yaml"

// LoadClusterConfig reads a kind v1alpha4 Cluster configuration from a YAML file.
func LoadClusterConfig(path string) (*kindv1alpha4.Cluster, error) {
	data, err := os.ReadFile(path) //nolint:gosec // File path from trusted configuration
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}

		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	return ParseConfig(data)
}

// ParseConfig parses YAML bytes into a kind v1alpha4 Cluster.
func ParseConfig(data []byte) (*kindv1alpha4.Cluster, error) {
	cluster := &kindv1alpha4.Cluster{}

	if err := yaml.Unmarshal(data, cluster); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConfigInvalid, err)
	}

	if cluster.Kind != "Cluster" || cluster.APIVersion != "kind.x-k8s.io/v1alpha4" {
		return nil, fmt.Errorf("%w: expected kind.x-k8s.io/v1alpha4 Cluster", ErrConfigInvalid)
	}

	if cluster.Name == "" {
		cluster.Name = "kind"
	}

	return cluster, nil
}
