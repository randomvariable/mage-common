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

// Package targets provides Mage-importable tool management targets.
//
// Import this package with the mage:import directive to gain tools:verify,
// tools:install, and tools:ensure targets automatically:
//
//	//mage:import tools
//	_ "github.com/randomvariable/mage-common/tools/targets"
//
// Named arguments are supported via the config package. The consuming magefile
// must call pflag.Parse() and config.CleanOSArgs() in its init() function:
//
//	func init() {
//	    pflag.Parse()
//	    config.CleanOSArgs()
//	}
//
// The ensure target reads the tool name from the --tool flag:
//
//	mage tools:ensure --tool=golangci-lint
//
// Config verification runs automatically as a dependency of Install and Ensure.
// It can also be invoked standalone via "mage tools:verify" for CI pre-flight
// checks.
package targets

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
	"github.com/randomvariable/mage-common/config"
	"github.com/randomvariable/mage-common/tools"
)

// ErrToolFlagRequired is returned when the --tool flag is not provided for the ensure target.
var ErrToolFlagRequired = errors.New("tool name required")

// init registers named argument flags for all targets in this package.
// Flag registration is a safe init-time operation (no I/O), consistent
// with Constitution Principle V.
func init() {
	pflag.String("tool", "", "Tool name for the ensure target")
}

// Verify validates the .tools.yaml configuration file.
// This runs automatically as a dependency of Install and Ensure;
// it can also be invoked standalone for CI pre-flight checks.
func Verify() error {
	_, err := tools.LoadToolConfiguration(tools.DefaultConfigFile)
	if err != nil {
		return fmt.Errorf("validating %s: %w", tools.DefaultConfigFile, err)
	}

	return nil
}

// Install installs all configured tools from .tools.yaml.
func Install(ctx context.Context) error {
	mg.Deps(Verify)

	err := tools.EnsureAll(ctx)
	if err != nil {
		return fmt.Errorf("installing tools: %w", err)
	}

	return nil
}

// Ensure installs a single tool by name from .tools.yaml.
// The tool name is read from the --tool flag or TOOL environment variable.
//
// Usage:
//
//	mage tools:ensure --tool=golangci-lint
//	TOOL=golangci-lint mage tools:ensure
func Ensure(ctx context.Context) error {
	mg.Deps(Verify)

	err := config.Init()
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	name := viper.GetString("tool")
	if name == "" {
		return fmt.Errorf("%w\n\n%s", ErrToolFlagRequired, ensureUsageHelp())
	}

	err = tools.Ensure(ctx, name)
	if err != nil {
		if errors.Is(err, tools.ErrToolNotFound) {
			return fmt.Errorf("%w: %q\n\n%s", tools.ErrToolNotFound, name, ensureUsageHelp())
		}

		return fmt.Errorf("ensuring %s: %w", name, err)
	}

	return nil
}

// ensureUsageHelp returns a help string listing usage and available tools.
func ensureUsageHelp() string {
	var b strings.Builder

	_, _ = fmt.Fprint(&b, "Usage:\n")
	_, _ = fmt.Fprint(&b, "  mage tools:ensure --tool=<name>\n")
	_, _ = fmt.Fprint(&b, "  TOOL=<name> mage tools:ensure\n")

	cfg, err := tools.LoadToolConfiguration(tools.DefaultConfigFile)
	if err != nil {
		return b.String()
	}

	_, _ = fmt.Fprint(&b, "\nAvailable tools:\n")

	for i := range cfg.Spec.Tools {
		_, _ = fmt.Fprintf(&b, "  %s %s\n", cfg.Spec.Tools[i].Name, toolAnnotation(&cfg.Spec.Tools[i]))
	}

	return b.String()
}

// toolAnnotation returns a parenthesized annotation for a tool: version, local, or empty.
func toolAnnotation(tool *v1alpha1.Tool) string {
	if tool.Version != nil {
		return "(" + ptr.Deref(tool.Version, "") + ")"
	}

	for i := range tool.Sources {
		if ptr.Deref(tool.Sources[i].Path, "") != "" {
			return "(local)"
		}
	}

	return ""
}
