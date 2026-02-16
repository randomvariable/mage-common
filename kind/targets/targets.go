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

// Package targets provides Mage-importable kind cluster management targets.
//
// Import this package with the mage:import directive to gain kind:create,
// kind:delete, and kind:status targets automatically:
//
//	//mage:import kind
//	_ "github.com/randomvariable/mage-common/kind/targets"
//
// These targets operate on the .kind-cluster.yaml file in the current
// directory, which should contain a standard kind v1alpha4 Cluster configuration.
package targets

import (
	"context"
	"fmt"

	"github.com/randomvariable/mage-common/kind"
)

// Create creates the kind cluster from .kind-cluster.yaml.
func Create(_ context.Context) error {
	err := kind.CreateCluster(kind.DefaultConfigFile)
	if err != nil {
		return fmt.Errorf("creating cluster: %w", err)
	}

	return nil
}

// Delete deletes the kind cluster defined in .kind-cluster.yaml.
func Delete(_ context.Context) error {
	config, err := kind.LoadClusterConfig(kind.DefaultConfigFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	err = kind.DeleteCluster(config.Name)
	if err != nil {
		return fmt.Errorf("deleting cluster: %w", err)
	}

	return nil
}

// Status reports the current state of the kind cluster defined in
// .kind-cluster.yaml.
func Status(_ context.Context) error {
	config, err := kind.LoadClusterConfig(kind.DefaultConfigFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	status, err := kind.GetClusterStatus(config.Name)
	if err != nil {
		return fmt.Errorf("getting cluster status: %w", err)
	}

	if !status.Exists {
		_, _ = fmt.Printf("Cluster %q: not found\n", status.Name)

		return nil
	}

	_, _ = fmt.Printf("Cluster %q:\n", status.Name)
	_, _ = fmt.Printf("  Context:  %s\n", status.Context)
	_, _ = fmt.Printf("  Exists:   %v\n", status.Exists)

	return nil
}
