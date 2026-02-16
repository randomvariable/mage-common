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

//go:build mage

package main

import (
	"context"
	"fmt"

	"github.com/magefile/mage/mg"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/randomvariable/mage-common/config"
	//mage:import kind
	_ "github.com/randomvariable/mage-common/kind/targets"
	magetools "github.com/randomvariable/mage-common/tools"
	//mage:import tools
	_ "github.com/randomvariable/mage-common/tools/targets"
)

// init registers named argument flags and initializes pflag/config integration.
// Imported target packages (tools/targets) register their own flags in their init(),
// which runs before this one. After all flags are registered, Parse() and CleanOSArgs()
// prepare os.Args for Mage.
func init() {
	pflag.String("packages", "./...", "Go packages to lint")
	pflag.Parse()
	config.CleanOSArgs()
}

// Generate groups code generation targets.
type Generate mg.Namespace

// Deepcopy runs controller-gen to generate DeepCopy methods for API types.
func (Generate) Deepcopy(ctx context.Context) error {
	_, err := magetools.Run(ctx, "controller-gen", []string{
		"object", "paths=./api/...",
	})
	if err != nil {
		return fmt.Errorf("running controller-gen: %w", err)
	}

	return nil
}

// Defaults runs defaulter-gen to generate defaulting functions for API types.
func (Generate) Defaults(ctx context.Context) error {
	_, err := magetools.Run(ctx, "defaulter-gen", []string{
		"--output-file", "zz_generated.defaults.go",
		"./api/tools/v1alpha1",
	})
	if err != nil {
		return fmt.Errorf("running defaulter-gen: %w", err)
	}

	return nil
}

// All runs all code generators.
func (Generate) All(ctx context.Context) error {
	mg.CtxDeps(ctx, Generate.Deepcopy, Generate.Defaults)

	return nil
}

// Lint groups linting targets.
type Lint mg.Namespace

// Run runs golangci-lint (with kube-api-linter if configured) on all packages.
// Override the default package pattern with --packages=./cmd/...
func (Lint) Run(ctx context.Context) error {
	err := config.Init()
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	packages := viper.GetString("packages")

	_, err = magetools.Run(ctx, "golangci-lint", []string{"run", packages})
	if err != nil {
		return fmt.Errorf("running golangci-lint: %w", err)
	}

	return nil
}

// All runs all linting checks.
func (Lint) All(ctx context.Context) error {
	mg.CtxDeps(ctx, Lint.Run)

	return nil
}

// Fix runs golangci-lint with --fix on all packages.
// Override the default package pattern with --packages=./cmd/...
func (Lint) Fix(ctx context.Context) error {
	err := config.Init()
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	packages := viper.GetString("packages")

	_, err = magetools.Run(ctx, "golangci-lint", []string{"run", "--fix", packages})
	if err != nil {
		return fmt.Errorf("running golangci-lint --fix: %w", err)
	}

	return nil
}

// Test groups testing targets.
type Test mg.Namespace

// Run runs tests via gotestsum with race detection, coverage, and JUnit output.
func (Test) Run(ctx context.Context) error {
	err := config.Init()
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	packages := viper.GetString("packages")

	_, err = magetools.Run(ctx, "gotestsum", []string{
		"--junitfile", "junit.xml",
		"--", "-race",
		"-coverprofile=coverage.out",
		"-covermode=atomic",
		"-coverpkg=" + packages,
		packages,
	})
	if err != nil {
		return fmt.Errorf("running tests: %w", err)
	}

	return nil
}

// All runs tests and checks coverage.
func (Test) All(ctx context.Context) error {
	mg.SerialCtxDeps(ctx, Test.Run, Test.Coverage)

	return nil
}

// Coverage checks test coverage against thresholds in .testcoverage.yml.
func (Test) Coverage(ctx context.Context) error {
	_, err := magetools.Run(ctx, "go-test-coverage", []string{
		"--config", ".testcoverage.yml",
	})
	if err != nil {
		return fmt.Errorf("checking test coverage: %w", err)
	}

	return nil
}

// Security groups security scanning targets.
type Security mg.Namespace

// OsvScan runs osv-scanner to check for known vulnerabilities in dependencies.
func (Security) OsvScan(ctx context.Context) error {
	_, err := magetools.Run(ctx, "osv-scanner", []string{"scan", "--recursive", "./"})
	if err != nil {
		return fmt.Errorf("running osv-scanner: %w", err)
	}

	return nil
}

// All runs all security scans.
func (Security) All(ctx context.Context) error {
	mg.CtxDeps(ctx, Security.OsvScan)

	return nil
}

// ModVerify verifies that dependencies in go.sum are correct.
func ModVerify(ctx context.Context) error {
	_, err := magetools.RunBinary(ctx, "go", []string{"mod", "verify"})
	if err != nil {
		return fmt.Errorf("verifying go modules: %w", err)
	}

	return nil
}

// Verify runs module verification, code generation, tests, and linting.
func Verify(ctx context.Context) error {
	mg.SerialCtxDeps(ctx, ModVerify, Generate.All, Test.All, Lint.All, Security.All)

	return nil
}
