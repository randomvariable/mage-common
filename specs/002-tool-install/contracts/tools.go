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

// Package contracts defines the Go interfaces for the tool installation system.
// These are design contracts, not runnable code.
package contracts

import (
	"context"
	"time"
)

// --- Configuration Loading ---

// ConfigLoader loads and validates a ToolConfiguration from a file path.
type ConfigLoader interface {
	// Load reads, decodes, defaults, and validates a .tools.yaml file.
	// Returns the typed configuration or a validation error.
	Load(path string) (*ToolConfiguration, error)
}

// ToolConfiguration is a placeholder for the K8s API Machinery typed config.
// Actual implementation lives in api/tools/v1alpha1/types.go.
type ToolConfiguration struct{}

// --- Installation ---

// Installer orchestrates tool installation from a configuration.
type Installer interface {
	// InstallAll installs all configured tools. Different install types run
	// concurrently; tools within the same type run sequentially.
	// By default, continues on failure and collects errors.
	InstallAll(ctx context.Context, opts ...InstallOption) error

	// InstallByName installs a single tool by name from the configuration.
	InstallByName(ctx context.Context, name string, opts ...InstallOption) error
}

// InstallOption configures installation behaviour.
type InstallOption func(*installConfig)

// WithFailFast enables strict mode: stop on first tool failure.
// func WithFailFast() InstallOption

// --- Individual Install Types ---

// TypeInstaller is implemented by each install type (go, gem, npx, cargo, uvx, download).
type TypeInstaller interface {
	// Install installs a single tool using this type's mechanism.
	// Returns nil if already installed at the correct version (idempotent).
	Install(ctx context.Context, tool Tool, source ToolSource, toolsDir string) error

	// IsInstalled checks if the tool is already installed at the expected version.
	IsInstalled(tool Tool, toolsDir string) (bool, error)

	// RuntimeAvailable checks if the required runtime is on PATH.
	RuntimeAvailable() error
}

// Tool and ToolSource are placeholders for the typed API objects.
type (
	Tool       struct{}
	ToolSource struct{}
)

// --- Download ---

// Downloader handles HTTP downloads with checksum verification and retry.
type Downloader interface {
	// Download fetches a URL to a local path, verifying checksum if provided.
	// Retries transient failures with exponential backoff (up to 3 retries).
	Download(ctx context.Context, url, outputPath string, checksum *Checksum) error
}

// Checksum is a placeholder for the typed API object.
type Checksum struct{}

// ArchiveExtractor extracts a named binary from an archive.
type ArchiveExtractor interface {
	// Extract locates and extracts the target binary from an archive file.
	// Supports tar.gz and zip formats, detected by file extension.
	Extract(archivePath, binaryPath, outputPath string) error
}

// --- Version Management ---

// VersionChecker manages version detection and symlink state.
type VersionChecker interface {
	// IsCurrentVersion returns true if the installed version matches expected.
	IsCurrentVersion(toolName, expectedVersion, toolsDir string) (bool, error)

	// SetVersion creates/updates the versioned binary + symlink.
	SetVersion(toolName, version, toolsDir string) error
}

// --- Update ---

// Updater queries upstream registries for latest versions and updates config.
type Updater interface {
	// UpdateAll queries all tools' registries and updates the config file.
	UpdateAll(ctx context.Context, configPath string) error

	// UpdateByName queries a single tool's registry and updates the config file.
	UpdateByName(ctx context.Context, configPath, name string) error
}

// RegistryQuerier is implemented per-ecosystem for version discovery.
type RegistryQuerier interface {
	// LatestVersion returns the latest available version for a tool.
	LatestVersion(ctx context.Context, tool Tool) (string, error)
}

// --- Cache Key ---

// CacheKeyGenerator produces deterministic cache keys for CI caching.
type CacheKeyGenerator interface {
	// CacheKey returns a deterministic hash of the config content + platform.
	CacheKey(configPath string) (string, error)
}

// --- Tool Runner ---

// Runner executes installed tools with logging, streaming, and capture.
type Runner interface {
	// Run executes a tool by name with arguments.
	// Always prints the command line (with secrets redacted) and streams output.
	// Returns the result with exit code, captured output, and duration.
	Run(ctx context.Context, toolName string, args []string, opts ...RunOption) (*RunResult, error)
}

// RunResult holds the outcome of a tool execution.
type RunResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	Duration time.Duration
}

// Combined returns interleaved stdout+stderr if WithCombinedOutput() was used.
func (r *RunResult) Combined() []byte { return nil }

// RunOption configures a tool run.
type RunOption func(*runConfig)

// Placeholder config structs for functional options.
type (
	installConfig struct{}
	runConfig     struct{}
)

// --- PATH Management ---

// PathManager provides tools directory path and PATH manipulation.
type PathManager interface {
	// ToolsDir returns the platform-specific tools directory path.
	ToolsDir() string

	// PrependToPath adds the tools directory to the process PATH.
	PrependToPath() error
}

// --- Template Rendering ---

// TemplateData holds the context passed to Go text/template for URL,
// binaryPath, and checksum URL rendering. See data-model.md for details.
type TemplateData struct {
	Version    string // Full version from config (e.g., "v2.4.0")
	VersionNum string // Version with "v" prefix stripped (e.g., "2.4.0")
	Major      string // Semver major component (empty if not parseable)
	Minor      string // Semver minor component (empty if not parseable)
	Patch      string // Semver patch component (empty if not parseable)
	OS         string // Resolved OS name (after osMap lookup)
	Arch       string // Resolved architecture name (after archMap lookup)
}

// TemplateRenderer resolves URL and binaryPath templates with platform values.
type TemplateRenderer interface {
	// Render executes a Go template with the full TemplateData context.
	// The version string is parsed for semver components; OS and Arch are
	// resolved through osMap/archMap before rendering.
	Render(tmpl string, version, goos, goarch string, osMap, archMap map[string]string) (string, error)
}

// --- Logging ---

// Logger is the interface for installation-level structured logging.
// Compatible with the config package's Logger interface.
// Used by Installer and Updater for progress/error messages.
type Logger interface {
	// Infof logs an informational message.
	Infof(format string, args ...any)

	// Warnf logs a warning message.
	Warnf(format string, args ...any)

	// Errorf logs an error message.
	Errorf(format string, args ...any)
}

// Note: The tool runner (Runner) uses io.Writer for output streaming
// via WithLogger(io.Writer), not the Logger interface. The runner streams
// raw bytes (stdout/stderr) which is a different concern from structured
// installation logging.
