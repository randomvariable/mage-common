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
	"strings"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

// shimPerm is the permission for generated shim scripts (rwxr-xr-x).
const shimPerm = 0o755

// UvxInstaller installs tools by creating a shell shim that delegates to
// "uvx" with a pinned package version. It is stateless and safe for
// concurrent use.
type UvxInstaller struct{}

// Install creates a shim script at <platformToolsDir>/<tool.Name> that
// invokes uvx with the pinned package version. If the shim already exists
// and contains the correct version string, the call is a no-op.
//
// When tool.Name differs from source.Package, the shim uses
// "uvx --from <package>==<version> <tool.Name>" so that uvx runs the
// correct executable from the package.
func (u *UvxInstaller) Install(_ context.Context, tool v1alpha1.Tool, source *v1alpha1.ToolSource, toolsDir string) error {
	installed, err := u.IsInstalled(tool, toolsDir)
	if err != nil {
		return fmt.Errorf("checking if %s is installed: %w", tool.Name, err)
	}

	if installed {
		return nil
	}

	platformDir, err := EnsureToolsDir(toolsDir)
	if err != nil {
		return fmt.Errorf("ensuring tools directory: %w", err)
	}

	shimPath := filepath.Join(platformDir, tool.Name)
	shimContent := shimScript(tool.Name, ptr.Deref(source.Package, ""), ptr.Deref(tool.Version, ""))

	err = os.WriteFile(shimPath, []byte(shimContent), shimPerm)
	if err != nil {
		return fmt.Errorf("writing shim for %s: %w", tool.Name, err)
	}

	return nil
}

// IsInstalled checks whether the shim script exists and contains the
// expected version string for the tool.
func (u *UvxInstaller) IsInstalled(tool v1alpha1.Tool, toolsDir string) (bool, error) {
	platformDir := PlatformToolsDir(toolsDir)
	shimPath := filepath.Join(platformDir, tool.Name)

	data, err := os.ReadFile(shimPath) //nolint:gosec // File path from trusted configuration
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("reading shim %q: %w", shimPath, err)
	}

	expected := "==" + ptr.Deref(tool.Version, "")

	return strings.Contains(string(data), expected), nil
}

// RuntimeAvailable checks that "uvx" is available on PATH. It returns
// ErrRuntimeNotFound if uvx cannot be located.
func (u *UvxInstaller) RuntimeAvailable() error {
	_, err := exec.LookPath("uvx")
	if err != nil {
		return fmt.Errorf("%w: uvx: %w", ErrRuntimeNotFound, err)
	}

	return nil
}

// shimScript returns the content of a shell shim that delegates to uvx.
// When the tool name differs from the package name, it uses --from to
// specify the package and runs the tool by name.
func shimScript(toolName, pkg, version string) string {
	pinned := pkg + "==" + BareVersion(version)
	if toolName != "" && toolName != pkg {
		return "#!/bin/sh\nexec uvx --from " + pinned + " " + toolName + " \"$@\"\n"
	}

	return "#!/bin/sh\nexec uvx " + pinned + " \"$@\"\n"
}
