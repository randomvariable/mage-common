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
	"regexp"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

// majorVersionSuffix matches Go module major version suffixes like /v2, /v3, etc.
var majorVersionSuffix = regexp.MustCompile(`/v\d+$`)

// GoInstaller installs tools using "go install" or "go build" for local paths.
// It is stateless and safe for concurrent use.
type GoInstaller struct{}

// Install installs the tool described by source. When source.Path is set, the
// tool is built from local source using "go build". Otherwise it is installed
// remotely using "go install". If the tool is already installed at the expected
// version (for remote installs) or the binary is up-to-date (for local builds),
// the call is a no-op. Binaries are built with CGO_ENABLED=0.
func (g *GoInstaller) Install(ctx context.Context, tool v1alpha1.Tool, source *v1alpha1.ToolSource, toolsDir string) error {
	if source.Path != nil && *source.Path != "" {
		return g.buildLocal(ctx, tool, source, toolsDir)
	}

	return g.installRemote(ctx, tool, source, toolsDir)
}

// IsInstalled reports whether the tool is already installed at its expected version.
func (g *GoInstaller) IsInstalled(tool v1alpha1.Tool, toolsDir string) (bool, error) {
	return IsCurrentVersion(tool.Name, ptr.Deref(tool.Version, ""), toolsDir)
}

// RuntimeAvailable checks that the "go" binary is on PATH. It returns
// ErrRuntimeNotFound if the Go toolchain cannot be located.
func (g *GoInstaller) RuntimeAvailable() error {
	_, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("%w: go: %w", ErrRuntimeNotFound, err)
	}

	return nil
}

func (g *GoInstaller) installRemote(ctx context.Context, tool v1alpha1.Tool, source *v1alpha1.ToolSource, toolsDir string) error {
	version := ptr.Deref(tool.Version, "")

	current, err := IsCurrentVersion(tool.Name, version, toolsDir)
	if err != nil {
		return fmt.Errorf("checking current version of %s: %w", tool.Name, err)
	}

	if current {
		return nil
	}

	platformDir, err := EnsureToolsDir(toolsDir)
	if err != nil {
		return fmt.Errorf("ensuring tools directory: %w", err)
	}

	absPlatformDir, err := filepath.Abs(platformDir)
	if err != nil {
		return fmt.Errorf("resolving absolute path for %q: %w", platformDir, err)
	}

	sourceURL := ptr.Deref(source.URL, "")
	pkg := sourceURL + "@" + version

	result, err := RunBinary(ctx, "go", []string{"install", pkg},
		WithEnv("CGO_ENABLED=0", "GOBIN="+absPlatformDir),
		WithCombinedOutput(),
	)
	if err != nil {
		return fmt.Errorf("go install %s: %w\n%s", pkg, err, result.Combined())
	}

	// go install places the binary using the last path element of the module URL.
	// Strip major version suffixes (/v2, /v3) first — Go uses the package name,
	// not the version suffix, for the binary name.
	binaryName := filepath.Base(majorVersionSuffix.ReplaceAllString(sourceURL, ""))
	srcPath := filepath.Join(platformDir, binaryName)

	err = SetVersion(tool.Name, version, toolsDir, srcPath)
	if err != nil {
		return fmt.Errorf("setting version for %s: %w", tool.Name, err)
	}

	return nil
}

func (g *GoInstaller) buildLocal(ctx context.Context, tool v1alpha1.Tool, source *v1alpha1.ToolSource, toolsDir string) error {
	platformDir, err := EnsureToolsDir(toolsDir)
	if err != nil {
		return fmt.Errorf("ensuring tools directory: %w", err)
	}

	outputPath := filepath.Join(platformDir, tool.Name)
	sourcePath := ptr.Deref(source.Path, "")

	// Check if rebuild is needed by comparing binary mtime with source directory mtime.
	if !g.needsRebuild(outputPath, sourcePath) {
		return nil
	}

	result, err := RunBinary(ctx, "go", []string{"build", "-o", outputPath, sourcePath},
		WithEnv("CGO_ENABLED=0"),
		WithCombinedOutput(),
	)
	if err != nil {
		return fmt.Errorf("go build %s: %w\n%s", sourcePath, err, result.Combined())
	}

	return nil
}

// needsRebuild returns true if the binary doesn't exist or is older than any
// .go file in the source directory tree.
func (g *GoInstaller) needsRebuild(binaryPath, srcPath string) bool {
	binInfo, err := os.Stat(binaryPath)
	if err != nil {
		return true // Binary doesn't exist, needs build.
	}

	binMtime := binInfo.ModTime()

	// Walk the source directory looking for .go files newer than the binary.
	needsBuild := false

	_ = filepath.WalkDir(srcPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || needsBuild {
			return filepath.SkipAll
		}

		if d.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil //nolint:nilerr // Skip files we can't stat; continue walking.
		}

		if info.ModTime().After(binMtime) {
			needsBuild = true

			return filepath.SkipAll
		}

		return nil
	})

	return needsBuild
}
