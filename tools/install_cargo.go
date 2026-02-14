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
	"os/exec"
	"path/filepath"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

// CargoInstaller installs tools using "cargo install". It is stateless and safe
// for concurrent use.
type CargoInstaller struct{}

// Install installs the tool described by source using "cargo install". If the
// tool is already installed at the expected version, the call is a no-op. The
// binary is built into a temporary directory, then moved into the
// platform-specific tools directory and versioned via SetVersion.
func (c *CargoInstaller) Install(ctx context.Context, tool v1alpha1.Tool, source *v1alpha1.ToolSource, toolsDir string) error {
	version := ptr.Deref(tool.Version, "")
	pkg := ptr.Deref(source.Package, "")

	current, err := IsCurrentVersion(tool.Name, version, toolsDir)
	if err != nil {
		return fmt.Errorf("checking current version of %s: %w", tool.Name, err)
	}

	if current {
		return nil
	}

	_, err = EnsureToolsDir(toolsDir)
	if err != nil {
		return fmt.Errorf("ensuring tools directory: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "cargo-install-*")
	if err != nil {
		return fmt.Errorf("creating temp directory for cargo install: %w", err)
	}

	defer func() { _ = os.RemoveAll(tmpDir) }()

	result, err := RunBinary(ctx, "cargo", []string{"install", pkg, "--version", BareVersion(version), "--root", tmpDir},
		WithCombinedOutput(),
	)
	if err != nil {
		return fmt.Errorf("cargo install %s@%s: %w\n%s", pkg, version, err, result.Combined())
	}

	// cargo install places binaries in <root>/bin/<name>.
	srcPath := filepath.Join(tmpDir, "bin", tool.Name)

	err = SetVersion(tool.Name, version, toolsDir, srcPath)
	if err != nil {
		return fmt.Errorf("setting version for %s: %w", tool.Name, err)
	}

	return nil
}

// IsInstalled reports whether the tool is already installed at its expected version.
func (c *CargoInstaller) IsInstalled(tool v1alpha1.Tool, toolsDir string) (bool, error) {
	return IsCurrentVersion(tool.Name, ptr.Deref(tool.Version, ""), toolsDir)
}

// RuntimeAvailable checks that the "cargo" binary is on PATH. It returns
// ErrRuntimeNotFound if the Cargo toolchain cannot be located.
func (c *CargoInstaller) RuntimeAvailable() error {
	_, err := exec.LookPath("cargo")
	if err != nil {
		return fmt.Errorf("%w: cargo: %w", ErrRuntimeNotFound, err)
	}

	return nil
}
