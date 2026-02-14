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

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"text/template"

	"k8s.io/utils/ptr"
)

// Sentinel errors for the v1alpha1 validation package.
var (
	// ErrValidationFailed is returned when tool configuration validation fails.
	ErrValidationFailed = errors.New("validation failed")

	// errHTTPSRequired is returned when a URL uses HTTP instead of HTTPS.
	errHTTPSRequired = errors.New("must use HTTPS, got HTTP URL")
)

// ValidateToolConfiguration validates a ToolConfiguration and returns an error
// describing all validation failures, or nil if valid.
func ValidateToolConfiguration(config *ToolConfiguration) error {
	var errs []string

	if len(config.Spec.Tools) == 0 {
		errs = append(errs, "spec.tools: at least one tool must be defined")
	}

	toolsDir := ptr.Deref(config.Spec.ToolsDir, "")

	if filepath.IsAbs(toolsDir) {
		errs = append(errs, fmt.Sprintf("spec.toolsDir: must be a relative path, got %q", toolsDir))
	}

	if strings.Contains(toolsDir, "..") {
		errs = append(errs, fmt.Sprintf("spec.toolsDir: must not contain path traversal, got %q", toolsDir))
	}

	names := make(map[string]struct{}, len(config.Spec.Tools))

	for i := range config.Spec.Tools {
		tool := &config.Spec.Tools[i]
		prefix := fmt.Sprintf("spec.tools[%d](%s)", i, tool.Name)

		if tool.Name == "" {
			errs = append(errs, fmt.Sprintf("spec.tools[%d].name: must not be empty", i))
		}

		if _, exists := names[tool.Name]; exists {
			errs = append(errs, fmt.Sprintf("%s.name: duplicate tool name %q", prefix, tool.Name))
		}

		names[tool.Name] = struct{}{}

		if len(tool.Sources) == 0 {
			errs = append(errs, prefix+".sources: at least one source must be defined")
		}

		allLocal := allSourcesLocal(tool.Sources)
		if ptr.Deref(tool.Version, "") == "" && !allLocal {
			errs = append(errs, prefix+".version: required unless all sources are local path type")
		}

		for j := range tool.Sources {
			sourceErrs := validateToolSource(&tool.Sources[j], fmt.Sprintf("%s.sources[%d]", prefix, j))
			errs = append(errs, sourceErrs...)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w:\n  %s", ErrValidationFailed, strings.Join(errs, "\n  "))
	}

	return nil
}

func allSourcesLocal(sources []ToolSource) bool {
	for i := range sources {
		if ptr.Deref(sources[i].Path, "") == "" {
			return false
		}
	}

	return len(sources) > 0
}

func validateToolSource(source *ToolSource, prefix string) []string {
	errs := make([]string, 0, 5) //nolint:mnd // one slot per validation call below

	errs = append(errs, validateSourceMutualExclusion(source, prefix)...)
	errs = append(errs, validateSourceURLSecurity(source, prefix)...)
	errs = append(errs, validateSourceChecksum(source, prefix)...)
	errs = append(errs, validateSourceTemplates(source, prefix)...)
	errs = append(errs, validateSourceBinaryPath(source, prefix)...)

	return errs
}

//nolint:cyclop // validation switch covers all source types
func validateSourceMutualExclusion(source *ToolSource, prefix string) []string {
	var errs []string

	fieldCount := countNonEmpty(source.URL, source.Package, source.Path)

	sourceURL := ptr.Deref(source.URL, "")
	sourcePackage := ptr.Deref(source.Package, "")
	sourcePath := ptr.Deref(source.Path, "")

	switch source.Type {
	case SourceTypeGo, SourceTypeGolangciLint:
		if sourceURL == "" && sourcePath == "" {
			errs = append(errs, fmt.Sprintf("%s: %s type requires url or path", prefix, source.Type))
		}

		if sourcePackage != "" {
			errs = append(errs, fmt.Sprintf("%s: %s type must not set package", prefix, source.Type))
		}
	case SourceTypeGem, SourceTypeNpx, SourceTypeCargo, SourceTypeUvx:
		if sourcePackage == "" {
			errs = append(errs, fmt.Sprintf("%s: %s type requires package", prefix, source.Type))
		}

		if sourceURL != "" || sourcePath != "" {
			errs = append(errs, fmt.Sprintf("%s: %s type must only set package", prefix, source.Type))
		}
	case SourceTypeDownload:
		if sourceURL == "" {
			errs = append(errs, prefix+": download type requires url")
		}

		if sourcePackage != "" || sourcePath != "" {
			errs = append(errs, prefix+": download type must only set url")
		}
	default:
		errs = append(errs, fmt.Sprintf("%s.type: unsupported source type %q", prefix, source.Type))
	}

	if fieldCount == 0 {
		errs = append(errs, prefix+": at least one of url, package, or path must be set")
	}

	return errs
}

func validateSourceURLSecurity(source *ToolSource, prefix string) []string {
	var errs []string

	sourceURL := ptr.Deref(source.URL, "")
	allowInsecure := ptr.Deref(source.AllowInsecure, false)

	if sourceURL != "" && source.Type == SourceTypeDownload && !allowInsecure {
		err := validateHTTPS(sourceURL, prefix+".url")
		if err != nil {
			errs = append(errs, err.Error())
		}
	}

	if source.Checksum != nil && ptr.Deref(source.Checksum.URL, "") != "" && !allowInsecure {
		err := validateHTTPS(ptr.Deref(source.Checksum.URL, ""), prefix+".checksum.url")
		if err != nil {
			errs = append(errs, err.Error())
		}
	}

	return errs
}

func validateHTTPS(rawURL, prefix string) error {
	// Skip template-only URLs that can't be parsed as-is.
	if strings.Contains(rawURL, "{{") {
		// Extract scheme if present before template expansion.
		if strings.HasPrefix(rawURL, "http://") {
			return fmt.Errorf("%w: %s: got HTTP URL", errHTTPSRequired, prefix)
		}

		return nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%s: invalid URL %q: %w", prefix, rawURL, err)
	}

	if parsed.Scheme == "http" {
		return fmt.Errorf("%w: %s: HTTP URL %q", errHTTPSRequired, prefix, rawURL)
	}

	return nil
}

func validateSourceChecksum(source *ToolSource, prefix string) []string {
	var errs []string

	if source.Type == SourceTypeDownload && !ptr.Deref(source.SkipChecksum, false) && source.Checksum == nil {
		errs = append(errs, prefix+": download type requires checksum (set skipChecksum: true to override)")
	}

	if source.Checksum != nil {
		errs = append(errs, validateChecksum(source.Checksum, prefix+".checksum")...)
	}

	return errs
}

func validateChecksum(checksum *Checksum, prefix string) []string {
	var errs []string

	checksumURL := ptr.Deref(checksum.URL, "")
	checksumInline := ptr.Deref(checksum.Inline, "")

	if checksumURL == "" && checksumInline == "" {
		errs = append(errs, prefix+": exactly one of url or inline must be set")

		return errs
	}

	if checksumURL != "" && checksumInline != "" {
		errs = append(errs, prefix+": exactly one of url or inline must be set, got both")

		return errs
	}

	if checksumInline != "" {
		parts := strings.SplitN(checksumInline, ":", 2) //nolint:mnd // format is algo:hex

		const expectedParts = 2
		if len(parts) != expectedParts {
			errs = append(errs, prefix+".inline: must be in <algorithm>:<hex> format")

			return errs
		}

		algo := parts[0]

		switch algo {
		case "sha256", "sha384", "sha512":
			// Valid algorithm.
		case "md5", "sha1":
			errs = append(errs, fmt.Sprintf(
				"%s.inline: weak algorithm %q rejected; use sha256, sha384, or sha512", prefix, algo))
		default:
			errs = append(errs, fmt.Sprintf(
				"%s.inline: unsupported algorithm %q; use sha256, sha384, or sha512", prefix, algo))
		}
	}

	return errs
}

func validateSourceTemplates(source *ToolSource, prefix string) []string {
	var errs []string

	sourceURL := ptr.Deref(source.URL, "")
	binaryPath := ptr.Deref(source.BinaryPath, "")

	if sourceURL != "" && strings.Contains(sourceURL, "{{") {
		_, err := template.New("url").Parse(sourceURL)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s.url: invalid Go template: %v", prefix, err))
		}
	}

	if binaryPath != "" && strings.Contains(binaryPath, "{{") {
		_, err := template.New("binaryPath").Parse(binaryPath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s.binaryPath: invalid Go template: %v", prefix, err))
		}
	}

	if source.Checksum != nil {
		checksumURL := ptr.Deref(source.Checksum.URL, "")
		if checksumURL != "" && strings.Contains(checksumURL, "{{") {
			_, err := template.New("checksumURL").Parse(checksumURL)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s.checksum.url: invalid Go template: %v", prefix, err))
			}
		}
	}

	return errs
}

func validateSourceBinaryPath(source *ToolSource, prefix string) []string {
	var errs []string

	binaryPath := ptr.Deref(source.BinaryPath, "")

	if binaryPath != "" && !strings.Contains(binaryPath, "{{") {
		if strings.Contains(binaryPath, "..") {
			errs = append(errs, prefix+".binaryPath: must not contain path traversal sequence")
		}

		if filepath.IsAbs(binaryPath) {
			errs = append(errs, prefix+".binaryPath: must be a relative path within the archive")
		}
	}

	return errs
}

func countNonEmpty(values ...*string) int {
	count := 0

	for _, v := range values {
		if v != nil && *v != "" {
			count++
		}
	}

	return count
}
