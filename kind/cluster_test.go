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
	"testing"

	. "github.com/onsi/gomega"
)

func TestClusterStatusDefaults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var status ClusterStatus

	g.Expect(status.Name).To(BeEmpty(), "zero-value Name should be empty")
	g.Expect(status.Exists).To(BeFalse(), "zero-value Exists should be false")
	g.Expect(status.Context).To(BeEmpty(), "zero-value Context should be empty")
}

func TestCreateCluster_InvalidConfigPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configPath string
		errContain string
	}{
		{
			name:       "nonexistent path",
			configPath: "/nonexistent/path/.kind-cluster.yaml",
			errContain: "config file not found",
		},
		{
			name:       "empty path",
			configPath: "",
			errContain: "config file not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			err := CreateCluster(tc.configPath)
			g.Expect(err).To(HaveOccurred())
			g.Expect(err.Error()).To(ContainSubstring(tc.errContain))
		})
	}
}
