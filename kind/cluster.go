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
	"slices"
	"time"

	kindv1alpha4 "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
)

// DefaultWaitForReady is the default timeout for waiting for the cluster to be ready.
const DefaultWaitForReady = 120 * time.Second

// ClusterStatus represents the current state of a kind cluster.
type ClusterStatus struct {
	// Name is the cluster name.
	Name string
	// Exists indicates whether the kind cluster exists.
	Exists bool
	// Context is the kubectl context name (kind-{name}).
	Context string
}

// CreateCluster loads config from path and creates the cluster.
func CreateCluster(configPath string) error {
	config, err := LoadClusterConfig(configPath)
	if err != nil {
		return err
	}

	return CreateClusterFromConfig(config)
}

// CreateClusterFromConfig creates a cluster from a parsed kind v1alpha4 Cluster config.
func CreateClusterFromConfig(config *kindv1alpha4.Cluster) error {
	provider := cluster.NewProvider()

	name := config.Name
	if name == "" {
		name = "kind"
	}

	logInfof("Creating kind cluster: %s", name)

	opts := []cluster.CreateOption{
		cluster.CreateWithV1Alpha4Config(config),
		cluster.CreateWithWaitForReady(DefaultWaitForReady),
	}

	err := provider.Create(name, opts...)
	if err != nil {
		return fmt.Errorf("creating kind cluster: %w", err)
	}

	logSuccessf("Cluster %s is ready", name)

	return nil
}

// DeleteCluster deletes the kind cluster by name.
// No-op if cluster doesn't exist (returns nil, not error).
func DeleteCluster(name string) error {
	provider := cluster.NewProvider()

	clusters, err := provider.List()
	if err != nil {
		return fmt.Errorf("listing kind clusters: %w", err)
	}

	found := slices.Contains(clusters, name)

	if !found {
		logWarnf("Cluster '%s' does not exist", name)

		return nil
	}

	logInfof("Deleting kind cluster: %s", name)

	if err := provider.Delete(name, ""); err != nil {
		return fmt.Errorf("deleting cluster: %w", err)
	}

	logSuccessf("Cluster deleted successfully")

	return nil
}

// GetClusterStatus returns the current status of the named cluster.
func GetClusterStatus(name string) (*ClusterStatus, error) {
	provider := cluster.NewProvider()

	status := &ClusterStatus{
		Name:    name,
		Context: "kind-" + name,
	}

	clusters, err := provider.List()
	if err != nil {
		return nil, fmt.Errorf("listing kind clusters: %w", err)
	}

	if slices.Contains(clusters, name) {
		status.Exists = true
	}

	return status, nil
}

// logInfof prints an informational message with a blue [INFO] prefix.
func logInfof(format string, args ...any) {
	_, _ = fmt.Printf("\033[34m[INFO]\033[0m "+format+"\n", args...)
}

// logSuccessf prints a success message with a green [SUCCESS] prefix.
func logSuccessf(format string, args ...any) {
	_, _ = fmt.Printf("\033[32m[SUCCESS]\033[0m "+format+"\n", args...)
}

// logWarnf prints a warning message with a yellow [WARN] prefix.
func logWarnf(format string, args ...any) {
	_, _ = fmt.Printf("\033[33m[WARN]\033[0m "+format+"\n", args...)
}
