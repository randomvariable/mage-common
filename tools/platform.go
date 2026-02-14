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
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"text/template"
)

// TemplateData holds the context passed to Go text/template when rendering
// URL, binaryPath, and checksum URL templates.
type TemplateData struct {
	// Version is the full version string from config (e.g., "v2.4.0").
	Version string

	// VersionNum is the version with the "v" prefix stripped (e.g., "2.4.0").
	VersionNum string

	// Major is the semver major component (empty if not parseable).
	Major string

	// Minor is the semver minor component (empty if not parseable).
	Minor string

	// Patch is the semver patch component (empty if not parseable).
	Patch string

	// OS is the resolved OS name (after osMap lookup; raw GOOS if no map entry).
	OS string

	// Arch is the resolved architecture name (after archMap lookup; raw GOARCH if no map entry).
	Arch string
}

// NewTemplateData constructs a TemplateData from a version string and platform maps.
// It uses the current runtime GOOS/GOARCH unless overridden by the maps.
func NewTemplateData(version string, osMap, archMap map[string]string) TemplateData {
	return NewTemplateDataForPlatform(version, runtime.GOOS, runtime.GOARCH, osMap, archMap)
}

// NewTemplateDataForPlatform constructs a TemplateData for a specific platform.
func NewTemplateDataForPlatform(version, goos, goarch string, osMap, archMap map[string]string) TemplateData {
	templateData := TemplateData{
		Version:    version,
		VersionNum: strings.TrimPrefix(version, "v"),
		OS:         resolveMap(goos, osMap),
		Arch:       resolveMap(goarch, archMap),
	}

	parseSemver(templateData.VersionNum, &templateData)

	return templateData
}

// Render executes a Go template string with the given TemplateData context.
func Render(tmpl string, data *TemplateData) (string, error) {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer

	err = t.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

func resolveMap(key string, m map[string]string) string {
	if v, ok := m[key]; ok {
		return v
	}

	return key
}

func parseSemver(versionNum string, templateData *TemplateData) {
	// Try to parse as semver: major.minor.patch with optional pre-release.
	parts := strings.SplitN(versionNum, ".", 3) //nolint:mnd // semver has 3 components
	if len(parts) < 3 {                         //nolint:mnd // semver requires 3 parts
		return
	}

	// The patch part may include pre-release info after a hyphen.
	patch := parts[2]
	if idx := strings.IndexByte(patch, '-'); idx >= 0 {
		patch = patch[:idx]
	}

	if idx := strings.IndexByte(patch, '+'); idx >= 0 {
		patch = patch[:idx]
	}

	templateData.Major = parts[0]
	templateData.Minor = parts[1]
	templateData.Patch = patch
}
