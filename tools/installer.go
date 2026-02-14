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
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

// InstallOption configures installation behavior.
type InstallOption func(*installConfig)

type installConfig struct {
	failFast bool
	logger   io.Writer
}

func defaultInstallConfig() *installConfig {
	return &installConfig{
		logger: os.Stderr,
	}
}

// WithFailFast enables strict mode: stop on first tool failure.
func WithFailFast() InstallOption {
	return func(c *installConfig) {
		c.failFast = true
	}
}

// WithInstallLogger sets the writer for installation progress messages.
func WithInstallLogger(w io.Writer) InstallOption {
	return func(c *installConfig) {
		c.logger = w
	}
}

// TypeInstaller is implemented by each install type (go, gem, npx, cargo, uvx, download).
type TypeInstaller interface {
	Install(ctx context.Context, tool v1alpha1.Tool, source *v1alpha1.ToolSource, toolsDir string) error
	IsInstalled(tool v1alpha1.Tool, toolsDir string) (bool, error)
	RuntimeAvailable() error
}

// Installer orchestrates tool installation from a configuration.
type Installer struct {
	Config     *v1alpha1.ToolConfiguration
	Installers map[v1alpha1.SourceType]TypeInstaller
}

// NewInstaller creates an Installer with default TypeInstallers for all source types.
func NewInstaller(config *v1alpha1.ToolConfiguration) *Installer {
	return &Installer{
		Config: config,
		Installers: map[v1alpha1.SourceType]TypeInstaller{
			v1alpha1.SourceTypeGo:           &GoInstaller{},
			v1alpha1.SourceTypeGem:          &GemInstaller{},
			v1alpha1.SourceTypeNpx:          &NpxInstaller{},
			v1alpha1.SourceTypeCargo:        &CargoInstaller{},
			v1alpha1.SourceTypeUvx:          &UvxInstaller{},
			v1alpha1.SourceTypeDownload:     &DownloadInstaller{},
			v1alpha1.SourceTypeGolangciLint: &GolangciLintInstaller{},
		},
	}
}

// InstallAll installs all configured tools. Different install types run
// concurrently; tools within the same type run sequentially.
// By default, continues on failure and collects errors.
func (inst *Installer) InstallAll(ctx context.Context, opts ...InstallOption) error {
	cfg := defaultInstallConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	toolsDir := ptr.Deref(inst.Config.Spec.ToolsDir, v1alpha1.DefaultToolsDir)

	_, err := EnsureToolsDir(toolsDir)
	if err != nil {
		return err
	}

	// Group tools by their first source type for parallel execution.
	type toolWork struct {
		tool   v1alpha1.Tool
		source v1alpha1.ToolSource
	}

	workByType := make(map[v1alpha1.SourceType][]toolWork)

	for i := range inst.Config.Spec.Tools {
		tool := inst.Config.Spec.Tools[i]
		if len(tool.Sources) > 0 {
			// We'll try sources in order within installTool, but group by first source type.
			workByType[tool.Sources[0].Type] = append(workByType[tool.Sources[0].Type], toolWork{
				tool:   tool,
				source: tool.Sources[0],
			})
		}
	}

	// Wrap the logger for thread-safe writes from concurrent goroutines.
	safeLogger := &syncWriter{w: cfg.logger}

	var (
		errMu   sync.Mutex
		allErrs []error
		wg      sync.WaitGroup
	)

	for _, work := range workByType {
		items := work

		wg.Go(func() {
			for i := range items {
				if ctx.Err() != nil {
					return
				}

				_, _ = fmt.Fprintf(safeLogger, "Installing %s...\n", items[i].tool.Name)

				err := inst.installToolWithFallback(ctx, items[i].tool, toolsDir, safeLogger)
				if err != nil {
					errMu.Lock()

					allErrs = append(allErrs, fmt.Errorf("%s: %w", items[i].tool.Name, err))

					failFast := cfg.failFast

					errMu.Unlock()

					if failFast {
						return
					}

					_, _ = fmt.Fprintf(safeLogger, "  FAILED: %s: %v\n", items[i].tool.Name, err)
				} else {
					_, _ = fmt.Fprintf(safeLogger, "  OK: %s\n", items[i].tool.Name)
				}
			}
		})
	}

	wg.Wait()

	if len(allErrs) > 0 {
		return errors.Join(allErrs...)
	}

	return nil
}

// InstallByName installs a single tool by name from the configuration.
func (inst *Installer) InstallByName(ctx context.Context, name string, opts ...InstallOption) error {
	cfg := defaultInstallConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	for i := range inst.Config.Spec.Tools {
		tool := inst.Config.Spec.Tools[i]
		if tool.Name == name {
			toolsDir := ptr.Deref(inst.Config.Spec.ToolsDir, v1alpha1.DefaultToolsDir)

			_, err := EnsureToolsDir(toolsDir)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cfg.logger, "Installing %s...\n", name)

			err = inst.installToolWithFallback(ctx, tool, toolsDir, cfg.logger)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}

			_, _ = fmt.Fprintf(cfg.logger, "  OK: %s\n", name)

			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrToolNotFound, name)
}

func (inst *Installer) installToolWithFallback(
	ctx context.Context,
	tool v1alpha1.Tool,
	toolsDir string,
	logger io.Writer,
) error {
	var sourceErrs []error

	for i := range tool.Sources {
		source := tool.Sources[i]

		installer, ok := inst.Installers[source.Type]
		if !ok {
			sourceErrs = append(sourceErrs, fmt.Errorf("source[%d]: %w: %q", i, ErrUnsupportedSourceType, source.Type))

			continue
		}

		// Check runtime availability.
		err := installer.RuntimeAvailable()
		if err != nil {
			_, _ = fmt.Fprintf(logger, "  source[%d] (%s): runtime not available: %v\n", i, source.Type, err)
			sourceErrs = append(sourceErrs, fmt.Errorf("source[%d] (%s): %w", i, source.Type, err))

			continue
		}

		// Check if already installed.
		installed, err := installer.IsInstalled(tool, toolsDir)
		if err == nil && installed {
			return nil // Already installed, skip.
		}

		// Attempt installation.
		err = installer.Install(ctx, tool, &source, toolsDir)
		if err != nil {
			_, _ = fmt.Fprintf(logger, "  source[%d] (%s): failed: %v\n", i, source.Type, err)
			sourceErrs = append(sourceErrs, fmt.Errorf("source[%d] (%s): %w", i, source.Type, err))

			continue
		}

		return nil // Success.
	}

	return fmt.Errorf("%w: %w", ErrAllSourcesFailed, errors.Join(sourceErrs...))
}

// Ensure installs a single tool by name if not already installed.
// The install is attempted at most once per process; subsequent calls for the
// same tool return the cached result. Designed for use as a Mage parameterized
// dependency:
//
//	mg.CtxDeps(ctx, mg.F(tools.Ensure, "golangci-lint"))
func Ensure(ctx context.Context, name string) error {
	return ensureOnce(name, func() error {
		config, err := LoadToolConfiguration(DefaultConfigFile)
		if err != nil {
			return err
		}

		inst := NewInstaller(config)

		err = inst.InstallByName(ctx, name)
		if err != nil {
			return err
		}

		toolsDir := ptr.Deref(config.Spec.ToolsDir, v1alpha1.DefaultToolsDir)

		return PrependToPath(toolsDir)
	})
}

// EnsureAll installs all tools from .tools.yaml.
// Each tool is installed at most once per process. Designed for use as a Mage
// dependency:
//
//	mg.CtxDeps(ctx, tools.EnsureAll)
func EnsureAll(ctx context.Context) error {
	config, err := LoadToolConfiguration(DefaultConfigFile)
	if err != nil {
		return err
	}

	inst := NewInstaller(config)

	err = inst.InstallAll(ctx)
	if err != nil {
		return err
	}

	// Mark each tool as individually ensured so that subsequent Ensure or
	// Run calls for any tool skip the install.
	for i := range config.Spec.Tools {
		markEnsured(config.Spec.Tools[i].Name)
	}

	toolsDir := ptr.Deref(config.Spec.ToolsDir, v1alpha1.DefaultToolsDir)

	return PrependToPath(toolsDir)
}
