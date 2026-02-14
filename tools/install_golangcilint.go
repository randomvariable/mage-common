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
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

const (
	// CustomGCLConfigFile is the default filename for golangci-lint custom build configuration.
	CustomGCLConfigFile = ".custom-gcl.yml"
)

// customGCLConfig represents the subset of .custom-gcl.yml we need to parse.
type customGCLConfig struct {
	Name        string `yaml:"name"`
	Destination string `yaml:"destination"`
}

// GolangciLintInstaller installs golangci-lint via "go install" and, if a
// .custom-gcl.yml file exists alongside the module root, builds a custom
// binary with module plugins by running "golangci-lint custom".
type GolangciLintInstaller struct {
	goInstaller GoInstaller
}

// Install installs golangci-lint and optionally builds a custom binary.
func (g *GolangciLintInstaller) Install(ctx context.Context, tool v1alpha1.Tool, source *v1alpha1.ToolSource, toolsDir string) error {
	// Install the base golangci-lint binary via go install.
	err := g.goInstaller.Install(ctx, tool, source, toolsDir)
	if err != nil {
		return fmt.Errorf("installing golangci-lint: %w", err)
	}

	// Build custom binary if config exists.
	if !customGCLConfigExists() {
		return nil
	}

	_, err = RunBinary(ctx, ToolBinaryPath(toolsDir, tool.Name), []string{"custom"})
	if err != nil {
		return fmt.Errorf("building custom golangci-lint: %w", err)
	}

	return nil
}

// IsInstalled checks if golangci-lint is installed at the expected version.
// If a custom config exists, it also checks that the custom binary is present.
func (g *GolangciLintInstaller) IsInstalled(tool v1alpha1.Tool, toolsDir string) (bool, error) {
	baseInstalled, err := g.goInstaller.IsInstalled(tool, toolsDir)
	if err != nil || !baseInstalled {
		return baseInstalled, err
	}

	if !customGCLConfigExists() {
		return true, nil
	}

	cfg, err := loadCustomGCLConfig()
	if err != nil {
		return false, fmt.Errorf("loading custom GCL config: %w", err)
	}

	_, statErr := os.Stat(cfg.binaryPath())

	return statErr == nil, nil
}

// RuntimeAvailable checks that the "go" binary is on PATH.
func (g *GolangciLintInstaller) RuntimeAvailable() error {
	return g.goInstaller.RuntimeAvailable()
}

// EffectiveLinterPath returns the path to the linter binary that should be used.
// If a .custom-gcl.yml exists and the custom binary has been built, it returns
// the custom binary path. Otherwise it returns the standard golangci-lint path
// from the tools directory.
func EffectiveLinterPath(toolsDir string) string {
	cfg, err := loadCustomGCLConfig()
	if err == nil {
		p := cfg.binaryPath()

		_, statErr := os.Stat(p)
		if statErr == nil {
			absPath, absErr := filepath.Abs(p)
			if absErr == nil {
				return absPath
			}

			return p
		}
	}

	return ToolBinaryPath(toolsDir, "golangci-lint")
}

func customGCLConfigExists() bool {
	_, err := os.Stat(CustomGCLConfigFile)

	return err == nil
}

// loadCustomGCLConfig reads and parses .custom-gcl.yml to extract the binary
// name and destination directory.
func loadCustomGCLConfig() (*customGCLConfig, error) {
	data, err := os.ReadFile(CustomGCLConfigFile)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", CustomGCLConfigFile, err)
	}

	var cfg customGCLConfig

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", CustomGCLConfigFile, err)
	}

	if cfg.Name == "" {
		return nil, fmt.Errorf("%s: %w", CustomGCLConfigFile, ErrCustomGCLNameRequired)
	}

	return &cfg, nil
}

// binaryPath returns the path to the custom binary based on the config.
func (c *customGCLConfig) binaryPath() string {
	dest := c.Destination
	if dest == "" {
		dest = "."
	}

	return filepath.Join(dest, c.Name)
}
