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

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// SourceType defines the installation mechanism for a tool.
// +kubebuilder:validation:Enum=go;gem;npx;cargo;uvx;download;golangci-lint
type SourceType string

const (
	// SourceTypeGo installs via go install.
	SourceTypeGo SourceType = "go"

	// SourceTypeGem installs via Bundler binstubs.
	SourceTypeGem SourceType = "gem"

	// SourceTypeNpx installs via npx with pinned version.
	SourceTypeNpx SourceType = "npx"

	// SourceTypeCargo installs via cargo install.
	SourceTypeCargo SourceType = "cargo"

	// SourceTypeUvx installs via uvx with pinned version.
	SourceTypeUvx SourceType = "uvx"

	// SourceTypeDownload installs via HTTP download with optional archive extraction.
	SourceTypeDownload SourceType = "download"

	// SourceTypeGolangciLint installs golangci-lint via go install and builds a custom
	// binary with module plugins if .custom-gcl.yml is present.
	SourceTypeGolangciLint SourceType = "golangci-lint"
)

// ToolConfiguration is the top-level configuration object loaded from .tools.yaml.
//
// +kubebuilder:object:root=true
type ToolConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// spec contains tool definitions and global settings.
	// +required
	Spec ToolConfigurationSpec `json:"spec,omitzero"`
}

// ToolConfigurationSpec defines the desired state of tool installation.
type ToolConfigurationSpec struct {
	// toolsDir is the base directory for installed binaries, relative to the project root.
	// +optional
	ToolsDir *string `json:"toolsDir,omitempty"`

	// tools is the list of tool definitions to install.
	// +required
	Tools []Tool `json:"tools,omitempty"`
}

// Tool defines a single tool entry with its name, version, and installation sources.
type Tool struct {
	// name is the binary name and unique identity key across the configuration.
	// +required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`

	// version is the pinned version (semver tag). Omit for local-only tools.
	// +optional
	Version *string `json:"version,omitempty"`

	// tagPrefix is the Git tag prefix for version discovery (e.g., "kustomize/").
	// +optional
	TagPrefix *string `json:"tagPrefix,omitempty"`

	// sources is the priority-ordered list of installation sources.
	// The installer tries each source in order until one succeeds.
	// +required
	Sources []ToolSource `json:"sources,omitempty"`
}

// ToolSource defines a single installation method for a tool.
type ToolSource struct {
	// type is the install type: go, gem, npx, cargo, uvx, or download.
	// +required
	Type SourceType `json:"type,omitempty"`

	// url is the Go module URL or download URL template.
	// Required for go and download types. Supports Go template tags.
	// +optional
	URL *string `json:"url,omitempty"`

	// package is the package name.
	// Required for gem, npx, cargo, and uvx types.
	// +optional
	Package *string `json:"package,omitempty"`

	// path is the local path for in-tree tools, relative to the Go module root.
	// Required for local builds.
	// +optional
	Path *string `json:"path,omitempty"`

	// binaryPath is the path to the binary within an archive (download type).
	// Supports Go template tags. Defaults to the tool name at the archive root.
	// +optional
	BinaryPath *string `json:"binaryPath,omitempty"`

	// checksum provides integrity verification for download-type sources.
	// +optional
	Checksum *Checksum `json:"checksum,omitempty"`

	// osMap maps GOOS values to URL OS strings for download-type sources.
	// When absent, the raw GOOS value is used.
	// +optional
	OsMap map[string]string `json:"osMap,omitempty"`

	// archMap maps GOARCH values to URL architecture strings for download-type sources.
	// When absent, the raw GOARCH value is used.
	// +optional
	ArchMap map[string]string `json:"archMap,omitempty"`

	// overrides provides per OS-architecture field overrides for download-type sources.
	// +optional
	Overrides []PlatformOverride `json:"overrides,omitempty"`

	// allowInsecure permits HTTP URLs for this source (download type only).
	// When false (default), non-HTTPS URLs are rejected at validation.
	// +optional
	AllowInsecure *bool `json:"allowInsecure,omitempty"`

	// skipChecksum skips the checksum requirement for this download source.
	// When false (default), a checksum field is required for download-type sources.
	// +optional
	SkipChecksum *bool `json:"skipChecksum,omitempty"`
}

// PlatformOverride allows overriding download source fields for specific OS-architecture combinations.
type PlatformOverride struct {
	// os is the GOOS value to match.
	// +required
	// +kubebuilder:validation:MinLength=1
	OS string `json:"os,omitempty"`

	// arch is the GOARCH value to match.
	// +required
	// +kubebuilder:validation:MinLength=1
	Arch string `json:"arch,omitempty"`

	// url overrides the URL template for this platform.
	// +optional
	URL *string `json:"url,omitempty"`

	// binaryPath overrides the binary path in the archive for this platform.
	// +optional
	BinaryPath *string `json:"binaryPath,omitempty"`

	// checksum overrides the checksum for this platform.
	// +optional
	Checksum *Checksum `json:"checksum,omitempty"`
}

// Checksum provides integrity verification for downloaded files.
type Checksum struct {
	// url points to a checksums file (template tags supported).
	// The file must use GNU coreutils format (<hash>  <filename>).
	// +optional
	URL *string `json:"url,omitempty"`

	// inline is a hash in <algorithm>:<hex> format.
	// Accepted algorithms: sha256, sha384, sha512.
	// +optional
	Inline *string `json:"inline,omitempty"`
}
