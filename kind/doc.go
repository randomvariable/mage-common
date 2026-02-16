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

// Package kind provides declarative kind cluster lifecycle management driven
// by a .kind-cluster.yaml manifest containing a standard kind v1alpha4 Cluster
// configuration.
//
// The package is a thin wrapper around the sigs.k8s.io/kind Go API. All cluster
// operations (create, delete, status) are exported Go functions that work
// independently of mage. Thin mage target wrappers live in the kind/targets
// sub-package.
//
// Usage:
//
//	config, err := kind.LoadClusterConfig(".kind-cluster.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	err = kind.CreateClusterFromConfig(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	status, err := kind.GetClusterStatus(config.Name)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Cluster %s: exists=%v\n", status.Name, status.Exists)
//
//	err = kind.DeleteCluster(config.Name)
//	if err != nil {
//	    log.Fatal(err)
//	}
package kind
