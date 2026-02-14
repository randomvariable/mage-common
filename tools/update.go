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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	goversion "github.com/hashicorp/go-version"
	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

const configFilePerm = 0o600

// RegistryQuerier queries an upstream registry for the latest version of a package.
type RegistryQuerier interface {
	LatestVersion(ctx context.Context, tool v1alpha1.Tool, source *v1alpha1.ToolSource) (string, error)
}

// Updater checks for newer versions and updates the config file.
type Updater struct {
	Config   *v1alpha1.ToolConfiguration
	Queriers map[v1alpha1.SourceType]RegistryQuerier
	Client   *http.Client
	Logger   io.Writer
}

// NewUpdater creates an Updater with default registry queriers.
func NewUpdater(config *v1alpha1.ToolConfiguration) *Updater {
	client := http.DefaultClient

	return &Updater{
		Config: config,
		Queriers: map[v1alpha1.SourceType]RegistryQuerier{
			v1alpha1.SourceTypeGo:       &GoRegistryQuerier{Client: client},
			v1alpha1.SourceTypeNpx:      &NpmRegistryQuerier{Client: client},
			v1alpha1.SourceTypeCargo:    &CratesRegistryQuerier{Client: client},
			v1alpha1.SourceTypeUvx:      &PyPIRegistryQuerier{Client: client},
			v1alpha1.SourceTypeGem:      &RubyGemsRegistryQuerier{Client: client},
			v1alpha1.SourceTypeDownload: &GoRegistryQuerier{Client: client},
		},
		Client: client,
		Logger: os.Stderr,
	}
}

// UpdateAll queries upstream registries for all tools and updates versions in the config file.
func (u *Updater) UpdateAll(ctx context.Context, configPath string) error {
	data, err := os.ReadFile(configPath) //nolint:gosec // File path from trusted configuration
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	content := string(data)

	for i := range u.Config.Spec.Tools {
		tool := u.Config.Spec.Tools[i]

		toolVersion := ptr.Deref(tool.Version, "")

		newVersion, err := u.queryLatestForTool(ctx, tool)
		if err != nil {
			_, _ = fmt.Fprintf(u.Logger, "  %s: failed to query: %v\n", tool.Name, err)

			continue
		}

		if newVersion == "" || newVersion == toolVersion {
			_, _ = fmt.Fprintf(u.Logger, "  %s: up to date (%s)\n", tool.Name, toolVersion)

			continue
		}

		content = replaceToolVersion(content, tool.Name, toolVersion, newVersion)
		_, _ = fmt.Fprintf(u.Logger, "  %s: %s → %s\n", tool.Name, toolVersion, newVersion)
	}

	err = os.WriteFile(configPath, []byte(content), configFilePerm)
	if err != nil {
		return fmt.Errorf("writing updated config: %w", err)
	}

	return nil
}

// UpdateByName queries the upstream registry for a single tool and updates its version.
func (u *Updater) UpdateByName(ctx context.Context, name, configPath string) error {
	for i := range u.Config.Spec.Tools {
		tool := u.Config.Spec.Tools[i]
		if tool.Name != name {
			continue
		}

		toolVersion := ptr.Deref(tool.Version, "")

		newVersion, err := u.queryLatestForTool(ctx, tool)
		if err != nil {
			return fmt.Errorf("querying latest version for %s: %w", name, err)
		}

		if newVersion == "" || newVersion == toolVersion {
			_, _ = fmt.Fprintf(u.Logger, "  %s: up to date (%s)\n", name, toolVersion)

			return nil
		}

		data, err := os.ReadFile(configPath) //nolint:gosec // File path from trusted configuration
		if err != nil {
			return fmt.Errorf("reading config: %w", err)
		}

		content := replaceToolVersion(string(data), tool.Name, toolVersion, newVersion)

		err = os.WriteFile(configPath, []byte(content), configFilePerm)
		if err != nil {
			return fmt.Errorf("writing updated config: %w", err)
		}

		_, _ = fmt.Fprintf(u.Logger, "  %s: %s → %s\n", name, toolVersion, newVersion)

		return nil
	}

	return fmt.Errorf("%w: %s", ErrToolNotFound, name)
}

func (u *Updater) queryLatestForTool(ctx context.Context, tool v1alpha1.Tool) (string, error) {
	// Skip tools pinned to "latest" or local-path-only.
	if ptr.Deref(tool.Version, "") == "latest" {
		return "", nil
	}

	if isLocalOnly(tool) {
		return "", nil
	}

	// Query using the first source's type.
	if len(tool.Sources) == 0 {
		return "", nil
	}

	source := tool.Sources[0]
	querier, ok := u.Queriers[source.Type]

	if !ok {
		return "", fmt.Errorf("%w: %q", ErrNoRegistryQuerier, source.Type)
	}

	version, err := querier.LatestVersion(ctx, tool, &source)
	if err != nil {
		return "", fmt.Errorf("querying %s registry: %w", source.Type, err)
	}

	return version, nil
}

func isLocalOnly(tool v1alpha1.Tool) bool {
	for _, s := range tool.Sources {
		if ptr.Deref(s.Path, "") == "" {
			return false
		}
	}

	return len(tool.Sources) > 0
}

// replaceToolVersion does a targeted replacement of the version for a specific tool
// in the YAML content. It finds the tool by name and replaces the version on the
// following lines.
func replaceToolVersion(content, toolName, oldVersion, newVersion string) string {
	// Pattern: find "name: <toolName>" followed by "version: <oldVersion>" within a few lines.
	pattern := fmt.Sprintf(`(- name: %s\n(?:.*\n){0,2}?\s+version: )%s`,
		regexp.QuoteMeta(toolName), regexp.QuoteMeta(oldVersion))

	re := regexp.MustCompile(pattern)

	return re.ReplaceAllString(content, "${1}"+newVersion)
}

// GoRegistryQuerier queries the Go module proxy for the latest version.
type GoRegistryQuerier struct {
	Client *http.Client
}

// LatestVersion fetches the latest version from the Go module proxy.
func (q *GoRegistryQuerier) LatestVersion(ctx context.Context, tool v1alpha1.Tool, source *v1alpha1.ToolSource) (string, error) {
	moduleURL := ptr.Deref(source.URL, "")
	if moduleURL == "" {
		return "", ErrMissingURL
	}

	// Strip the /cmd/<name> suffix to get the module path.
	modulePath := moduleURL

	proxyURL := fmt.Sprintf("https://proxy.golang.org/%s/@v/list", modulePath)

	body, err := q.fetch(ctx, proxyURL)
	if err != nil {
		return "", err
	}

	return filterLatestVersion(body, ptr.Deref(tool.TagPrefix, ""))
}

func (q *GoRegistryQuerier) fetch(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := q.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", url, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %d from %s", ErrHTTPStatus, resp.StatusCode, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	return string(data), nil
}

// filterLatestVersion finds the highest semver from a newline-separated version list,
// optionally filtering by tag prefix.
func filterLatestVersion(versionList, tagPrefix string) (string, error) {
	var latest *goversion.Version

	var latestRaw string

	for line := range strings.SplitSeq(strings.TrimSpace(versionList), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Apply tag prefix filter.
		versionStr := line
		if tagPrefix != "" {
			if !strings.HasPrefix(line, tagPrefix) {
				continue
			}

			versionStr = strings.TrimPrefix(line, tagPrefix)
		}

		v, err := goversion.NewVersion(versionStr)
		if err != nil {
			continue // Skip unparseable versions.
		}

		// Skip pre-releases.
		if v.Prerelease() != "" {
			continue
		}

		if latest == nil || v.GreaterThan(latest) {
			latest = v
			latestRaw = line
		}
	}

	if latest == nil {
		return "", ErrNoStableVersions
	}

	return latestRaw, nil
}

// NpmRegistryQuerier queries the npm registry.
type NpmRegistryQuerier struct {
	Client *http.Client
}

// LatestVersion fetches the latest version from the npm registry.
func (q *NpmRegistryQuerier) LatestVersion(ctx context.Context, _ v1alpha1.Tool, source *v1alpha1.ToolSource) (string, error) {
	pkg := ptr.Deref(source.Package, "")
	if pkg == "" {
		return "", ErrMissingPackage
	}

	url := fmt.Sprintf("https://registry.npmjs.org/%s/latest", pkg)

	body, err := fetchJSON(ctx, q.Client, url)
	if err != nil {
		return "", err
	}

	version, ok := body["version"].(string)
	if !ok {
		return "", fmt.Errorf("npm: %w", ErrNoVersionInResponse)
	}

	return version, nil
}

// CratesRegistryQuerier queries crates.io.
type CratesRegistryQuerier struct {
	Client *http.Client
}

// LatestVersion fetches the latest version from crates.io.
func (q *CratesRegistryQuerier) LatestVersion(ctx context.Context, _ v1alpha1.Tool, source *v1alpha1.ToolSource) (string, error) {
	pkg := ptr.Deref(source.Package, "")
	if pkg == "" {
		return "", ErrMissingPackage
	}

	url := "https://crates.io/api/v1/crates/" + pkg

	body, err := fetchJSON(ctx, q.Client, url)
	if err != nil {
		return "", err
	}

	crate, ok := body["crate"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("crates.io: %w", ErrUnexpectedResponse)
	}

	version, ok := crate["max_stable_version"].(string)
	if !ok {
		version, ok = crate["max_version"].(string)
		if !ok {
			return "", fmt.Errorf("crates.io: %w", ErrNoVersionInResponse)
		}
	}

	return version, nil
}

// PyPIRegistryQuerier queries the Python Package Index.
type PyPIRegistryQuerier struct {
	Client *http.Client
}

// LatestVersion fetches the latest version from PyPI.
func (q *PyPIRegistryQuerier) LatestVersion(ctx context.Context, _ v1alpha1.Tool, source *v1alpha1.ToolSource) (string, error) {
	pkg := ptr.Deref(source.Package, "")
	if pkg == "" {
		return "", ErrMissingPackage
	}

	url := fmt.Sprintf("https://pypi.org/pypi/%s/json", pkg)

	body, err := fetchJSON(ctx, q.Client, url)
	if err != nil {
		return "", err
	}

	info, ok := body["info"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("pypi: %w", ErrUnexpectedResponse)
	}

	version, ok := info["version"].(string)
	if !ok {
		return "", fmt.Errorf("pypi: %w", ErrNoVersionInResponse)
	}

	return version, nil
}

// RubyGemsRegistryQuerier queries the RubyGems API.
type RubyGemsRegistryQuerier struct {
	Client *http.Client
}

// LatestVersion fetches the latest version from RubyGems.
func (q *RubyGemsRegistryQuerier) LatestVersion(ctx context.Context, _ v1alpha1.Tool, source *v1alpha1.ToolSource) (string, error) {
	pkg := ptr.Deref(source.Package, "")
	if pkg == "" {
		return "", ErrMissingPackage
	}

	url := fmt.Sprintf("https://rubygems.org/api/v1/gems/%s.json", pkg)

	body, err := fetchJSON(ctx, q.Client, url)
	if err != nil {
		return "", err
	}

	version, ok := body["version"].(string)
	if !ok {
		return "", fmt.Errorf("rubygems: %w", ErrNoVersionInResponse)
	}

	return version, nil
}

func fetchJSON(ctx context.Context, client *http.Client, url string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "mage-common/tools")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d from %s", ErrHTTPStatus, resp.StatusCode, url)
	}

	var result map[string]any

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("decoding JSON from %s: %w", url, err)
	}

	return result, nil
}

// UpdateAll is a convenience function that loads config and updates all tools.
func UpdateAll(configPath string) error {
	config, err := LoadToolConfiguration(configPath)
	if err != nil {
		return err
	}

	updater := NewUpdater(config)

	return updater.UpdateAll(context.Background(), configPath)
}

// UpdateByName is a convenience function that loads config and updates a single tool.
func UpdateByName(name, configPath string) error {
	config, err := LoadToolConfiguration(configPath)
	if err != nil {
		return err
	}

	updater := NewUpdater(config)

	return updater.UpdateByName(context.Background(), name, configPath)
}
