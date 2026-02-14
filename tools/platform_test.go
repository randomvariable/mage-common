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
	"testing"
)

func TestNewTemplateDataForPlatform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		goos     string
		goarch   string
		osMap    map[string]string
		archMap  map[string]string
		wantData TemplateData
	}{
		{
			name:    "v-prefixed semver",
			version: "v2.4.0",
			goos:    "linux",
			goarch:  "amd64",
			wantData: TemplateData{
				Version:    "v2.4.0",
				VersionNum: "2.4.0",
				Major:      "2",
				Minor:      "4",
				Patch:      "0",
				OS:         "linux",
				Arch:       "amd64",
			},
		},
		{
			name:    "bare semver without v prefix",
			version: "1.32.0",
			goos:    "darwin",
			goarch:  "arm64",
			wantData: TemplateData{
				Version:    "1.32.0",
				VersionNum: "1.32.0",
				Major:      "1",
				Minor:      "32",
				Patch:      "0",
				OS:         "darwin",
				Arch:       "arm64",
			},
		},
		{
			name:    "non-semver version (latest)",
			version: "latest",
			goos:    "linux",
			goarch:  "amd64",
			wantData: TemplateData{
				Version:    "latest",
				VersionNum: "latest",
				Major:      "",
				Minor:      "",
				Patch:      "",
				OS:         "linux",
				Arch:       "amd64",
			},
		},
		{
			name:    "OS and arch mapping",
			version: "v1.0.0",
			goos:    "darwin",
			goarch:  "amd64",
			osMap:   map[string]string{"darwin": "macOS"},
			archMap: map[string]string{"amd64": "x86_64"},
			wantData: TemplateData{
				Version:    "v1.0.0",
				VersionNum: "1.0.0",
				Major:      "1",
				Minor:      "0",
				Patch:      "0",
				OS:         "macOS",
				Arch:       "x86_64",
			},
		},
		{
			name:    "OS map with no matching entry uses raw GOOS",
			version: "v1.0.0",
			goos:    "windows",
			goarch:  "amd64",
			osMap:   map[string]string{"darwin": "macOS"},
			wantData: TemplateData{
				Version:    "v1.0.0",
				VersionNum: "1.0.0",
				Major:      "1",
				Minor:      "0",
				Patch:      "0",
				OS:         "windows",
				Arch:       "amd64",
			},
		},
		{
			name:    "semver with pre-release",
			version: "v1.2.3-beta.1",
			goos:    "linux",
			goarch:  "arm64",
			wantData: TemplateData{
				Version:    "v1.2.3-beta.1",
				VersionNum: "1.2.3-beta.1",
				Major:      "1",
				Minor:      "2",
				Patch:      "3",
				OS:         "linux",
				Arch:       "arm64",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewTemplateDataForPlatform(tt.version, tt.goos, tt.goarch, tt.osMap, tt.archMap)

			if got != tt.wantData {
				t.Errorf("NewTemplateDataForPlatform() =\n  %+v\nwant\n  %+v", got, tt.wantData)
			}
		})
	}
}

func TestRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tmpl    string
		data    TemplateData
		want    string
		wantErr bool
	}{
		{
			name: "full URL template",
			tmpl: "https://example.com/releases/{{.Version}}/tool-{{.VersionNum}}-{{.OS}}-{{.Arch}}.tar.gz",
			data: TemplateData{
				Version:    "v2.4.0",
				VersionNum: "2.4.0",
				OS:         "linux",
				Arch:       "amd64",
			},
			want: "https://example.com/releases/v2.4.0/tool-2.4.0-linux-amd64.tar.gz",
		},
		{
			name: "binary path template",
			tmpl: "tool-{{.VersionNum}}-{{.OS}}-{{.Arch}}/tool",
			data: TemplateData{
				VersionNum: "2.4.0",
				OS:         "linux",
				Arch:       "amd64",
			},
			want: "tool-2.4.0-linux-amd64/tool",
		},
		{
			name: "major minor version template",
			tmpl: "https://example.com/v{{.Major}}/tool-{{.Major}}.{{.Minor}}.tar.gz",
			data: TemplateData{
				Major: "2",
				Minor: "4",
			},
			want: "https://example.com/v2/tool-2.4.tar.gz",
		},
		{
			name:    "invalid template",
			tmpl:    "{{.Invalid",
			data:    TemplateData{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Render(tt.tmpl, &tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}
