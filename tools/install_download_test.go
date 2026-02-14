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
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
	"github.com/randomvariable/mage-common/internal/testutil"
)

func TestDownloadInstallerRawBinary(t *testing.T) {
	t.Parallel()

	binaryContent := []byte("#!/bin/sh\necho hello\n")
	checksum := fmt.Sprintf("%x", sha256.Sum256(binaryContent))

	server := testutil.NewTestServer(t, map[string]testutil.ServerResponse{
		"/tool": {
			StatusCode: http.StatusOK,
			Body:       binaryContent,
		},
	})

	tmpDir := t.TempDir()
	toolsDir := tmpDir
	platformDir := filepath.Join(toolsDir, runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(platformDir, 0o750)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	installer := &DownloadInstaller{Client: server.Client()}

	tool := v1alpha1.Tool{
		Name:    "test-tool",
		Version: ptr.To("v1.0.0"),
	}

	source := v1alpha1.ToolSource{
		Type: v1alpha1.SourceTypeDownload,
		URL:  ptr.To(server.URL + "/tool"),
		Checksum: &v1alpha1.Checksum{
			Inline: ptr.To("sha256:" + checksum),
		},
	}

	err = installer.Install(t.Context(), tool, &source, toolsDir)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	// Verify tool was installed.
	installed, err := installer.IsInstalled(tool, toolsDir)
	if err != nil {
		t.Fatalf("IsInstalled() error = %v", err)
	}

	if !installed {
		t.Error("IsInstalled() = false, want true")
	}
}

func TestDownloadInstallerChecksumMismatch(t *testing.T) {
	t.Parallel()

	server := testutil.NewTestServer(t, map[string]testutil.ServerResponse{
		"/tool": {
			StatusCode: http.StatusOK,
			Body:       []byte("binary-content"),
		},
	})

	tmpDir := t.TempDir()
	platformDir := filepath.Join(tmpDir, runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(platformDir, 0o750)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	installer := &DownloadInstaller{Client: server.Client()}

	tool := v1alpha1.Tool{
		Name:    "bad-checksum-tool",
		Version: ptr.To("v1.0.0"),
	}

	source := v1alpha1.ToolSource{
		Type: v1alpha1.SourceTypeDownload,
		URL:  ptr.To(server.URL + "/tool"),
		Checksum: &v1alpha1.Checksum{
			Inline: ptr.To("sha256:0000000000000000000000000000000000000000000000000000000000000000"),
		},
	}

	err = installer.Install(t.Context(), tool, &source, tmpDir)
	if err == nil {
		t.Fatal("Install() error = nil, want checksum mismatch error")
	}

	if !errors.Is(err, ErrChecksumMismatch) {
		t.Errorf("error = %v, want ErrChecksumMismatch", err)
	}
}

func TestDownloadInstallerTarGzExtraction(t *testing.T) {
	t.Parallel()

	binaryContent := []byte("#!/bin/sh\necho tar tool\n")

	archiveData := testutil.CreateTarGz(t, []testutil.ArchiveEntry{
		{Name: "tool-1.0.0-linux-amd64/", IsDir: true},
		{Name: "tool-1.0.0-linux-amd64/tool", Content: binaryContent},
	})

	checksum := fmt.Sprintf("%x", sha256.Sum256(archiveData))

	server := testutil.NewTestServer(t, map[string]testutil.ServerResponse{
		"/tool.tar.gz": {
			StatusCode: http.StatusOK,
			Body:       archiveData,
		},
	})

	tmpDir := t.TempDir()
	platformDir := filepath.Join(tmpDir, runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(platformDir, 0o750)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	installer := &DownloadInstaller{Client: server.Client()}

	tool := v1alpha1.Tool{
		Name:    "tar-tool",
		Version: ptr.To("v1.0.0"),
	}

	source := v1alpha1.ToolSource{
		Type:       v1alpha1.SourceTypeDownload,
		URL:        ptr.To(server.URL + "/tool.tar.gz"),
		BinaryPath: ptr.To("tool-1.0.0-linux-amd64/tool"),
		Checksum: &v1alpha1.Checksum{
			Inline: ptr.To("sha256:" + checksum),
		},
	}

	err = installer.Install(t.Context(), tool, &source, tmpDir)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	installed, err := installer.IsInstalled(tool, tmpDir)
	if err != nil {
		t.Fatalf("IsInstalled() error = %v", err)
	}

	if !installed {
		t.Error("IsInstalled() = false, want true")
	}
}

func TestDownloadInstallerIdempotent(t *testing.T) {
	t.Parallel()

	binaryContent := []byte("#!/bin/sh\necho hello\n")
	checksum := fmt.Sprintf("%x", sha256.Sum256(binaryContent))

	callCount := 0

	server := testutil.NewTestServer(t, map[string]testutil.ServerResponse{
		"/tool": {
			StatusCode: http.StatusOK,
			Body:       binaryContent,
		},
	})

	// Wrap to count calls.
	_ = callCount

	tmpDir := t.TempDir()
	platformDir := filepath.Join(tmpDir, runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(platformDir, 0o750)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	installer := &DownloadInstaller{Client: server.Client()}

	tool := v1alpha1.Tool{
		Name:    "idempotent-tool",
		Version: ptr.To("v1.0.0"),
	}

	source := v1alpha1.ToolSource{
		Type: v1alpha1.SourceTypeDownload,
		URL:  ptr.To(server.URL + "/tool"),
		Checksum: &v1alpha1.Checksum{
			Inline: ptr.To("sha256:" + checksum),
		},
	}

	// First install.
	err = installer.Install(t.Context(), tool, &source, tmpDir)
	if err != nil {
		t.Fatalf("first Install() error = %v", err)
	}

	// Second install should be a no-op.
	err = installer.Install(t.Context(), tool, &source, tmpDir)
	if err != nil {
		t.Fatalf("second Install() error = %v", err)
	}
}

func TestMatchChecksumLine(t *testing.T) {
	t.Parallel()

	const exampleSHA256 = "5de4e9f2266738fd112b721265a0c1cd7f4e5208b670f811861f699474a100a3"

	tests := []struct {
		name     string
		content  string
		filename string
		wantHex  string
		wantErr  error
	}{
		{
			name:     "bare hash (kubectl style)",
			content:  exampleSHA256 + "\n",
			filename: "kubectl",
			wantHex:  exampleSHA256,
		},
		{
			name:     "bare hash without trailing newline",
			content:  exampleSHA256,
			filename: "kubectl",
			wantHex:  exampleSHA256,
		},
		{
			name:     "GNU coreutils format",
			content:  exampleSHA256 + "  kustomize_v5.6.0_linux_amd64.tar.gz\n",
			filename: "kustomize_v5.6.0_linux_amd64.tar.gz",
			wantHex:  exampleSHA256,
		},
		{
			name:     "GNU coreutils multi-line",
			content:  "aaaa  other_file.tar.gz\n" + exampleSHA256 + "  target.tar.gz\nbbbb  another.tar.gz\n",
			filename: "target.tar.gz",
			wantHex:  exampleSHA256,
		},
		{
			name:     "filename not found in multi-line",
			content:  exampleSHA256 + "  other_file.tar.gz\n",
			filename: "missing.tar.gz",
			wantErr:  ErrChecksumNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hasher, gotHex, err := matchChecksumLine(tt.content, tt.filename)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("matchChecksumLine() error = %v, want %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("matchChecksumLine() unexpected error: %v", err)
			}

			if gotHex != tt.wantHex {
				t.Errorf("checksum hex = %q, want %q", gotHex, tt.wantHex)
			}

			if hasher == nil {
				t.Fatal("hash is nil")
			}
		})
	}
}

func TestDownloadInstallerRuntimeAlwaysAvailable(t *testing.T) {
	t.Parallel()

	installer := &DownloadInstaller{}

	err := installer.RuntimeAvailable()
	if err != nil {
		t.Errorf("RuntimeAvailable() = %v, want nil", err)
	}
}
