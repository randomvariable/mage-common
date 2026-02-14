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

// GemInstaller installs tools via gem install.
type GemInstaller struct{}

// Install runs gem install to place the tool binary at the pinned version
// directly into the platform-specific tools directory.
func (g *GemInstaller) Install(ctx context.Context, tool v1alpha1.Tool, source *v1alpha1.ToolSource, toolsDir string) error {
	platformDir := PlatformToolsDir(toolsDir)

	_, err := EnsureToolsDir(toolsDir)
	if err != nil {
		return fmt.Errorf("ensuring tools directory: %w", err)
	}

	pkg := ptr.Deref(source.Package, "")
	version := ptr.Deref(tool.Version, "")

	args := []string{"install", pkg, "--bindir", platformDir, "--no-document"}
	if version != "" {
		args = append(args, "--version", BareVersion(version))
	}

	_, err = RunBinary(ctx, "gem", args)
	if err != nil {
		return fmt.Errorf("gem install %s: %w", pkg, err)
	}

	return nil
}

// IsInstalled checks whether a gem executable exists for the tool in the
// platform-specific tools directory.
func (g *GemInstaller) IsInstalled(tool v1alpha1.Tool, toolsDir string) (bool, error) {
	platformDir := PlatformToolsDir(toolsDir)
	binPath := filepath.Join(platformDir, tool.Name)

	info, err := os.Stat(binPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("checking gem binary %q: %w", binPath, err)
	}

	if info.IsDir() {
		return false, fmt.Errorf("%w: %q", ErrUnexpectedDirectory, binPath)
	}

	return true, nil
}

// RuntimeAvailable checks that gem is available on PATH.
// Returns ErrRuntimeNotFound if missing.
func (g *GemInstaller) RuntimeAvailable() error {
	_, err := exec.LookPath("gem")
	if err != nil {
		return fmt.Errorf("%w: gem", ErrRuntimeNotFound)
	}

	return nil
}
