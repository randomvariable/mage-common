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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/randomvariable/mage-common/internal/testutil"
)

func TestExtractBinary(t *testing.T) { //nolint:gocognit,cyclop // table-driven test with setup
	t.Parallel()

	binaryContent := []byte("#!/bin/sh\necho hello\n")

	tests := []struct {
		name        string
		archiveName string
		entries     []testutil.ArchiveEntry
		binaryPath  string
		createFn    func(t *testing.T, entries []testutil.ArchiveEntry) []byte
		wantErr     error
		wantContent []byte
		rawContent  []byte // non-nil means write this directly instead of using createFn
	}{
		{
			name:        "tar.gz extraction from subdirectory",
			archiveName: "tool.tar.gz",
			entries: []testutil.ArchiveEntry{
				{Name: "tool/", IsDir: true},
				{Name: "tool/bin/mytool", Content: binaryContent},
				{Name: "tool/README.md", Content: []byte("readme")},
			},
			binaryPath:  "tool/bin/mytool",
			createFn:    testutil.CreateTarGz,
			wantContent: binaryContent,
		},
		{
			name:        "tgz extraction",
			archiveName: "tool.tgz",
			entries: []testutil.ArchiveEntry{
				{Name: "mytool", Content: binaryContent},
			},
			binaryPath:  "mytool",
			createFn:    testutil.CreateTarGz,
			wantContent: binaryContent,
		},
		{
			name:        "zip extraction",
			archiveName: "tool.zip",
			entries: []testutil.ArchiveEntry{
				{Name: "dist/", IsDir: true},
				{Name: "dist/mytool", Content: binaryContent},
			},
			binaryPath:  "dist/mytool",
			createFn:    testutil.CreateZip,
			wantContent: binaryContent,
		},
		{
			name:        "zip slip attack with traversal entry",
			archiveName: "evil.tar.gz",
			entries: []testutil.ArchiveEntry{
				{Name: "../../../etc/passwd", Content: []byte("root:x:0:0")},
			},
			binaryPath: "anything",
			createFn:   testutil.CreateTarGz,
			wantErr:    ErrPathTraversal,
		},
		{
			name:        "zip slip attack in zip archive",
			archiveName: "evil.zip",
			entries: []testutil.ArchiveEntry{
				{Name: "../../../etc/passwd", Content: []byte("root:x:0:0")},
			},
			binaryPath: "anything",
			createFn:   testutil.CreateZip,
			wantErr:    ErrPathTraversal,
		},
		{
			name:        "symlink attack pointing outside archive",
			archiveName: "symlink-attack.tar.gz",
			entries: []testutil.ArchiveEntry{
				{Name: "evil-link", IsSymlink: true, LinkTarget: "/etc/passwd"},
			},
			binaryPath: "evil-link",
			createFn:   testutil.CreateTarGz,
			wantErr:    ErrPathTraversal,
		},
		{
			name:        "symlink attack with relative traversal",
			archiveName: "symlink-relative.tar.gz",
			entries: []testutil.ArchiveEntry{
				{Name: "sub/link", IsSymlink: true, LinkTarget: "../../etc/shadow"},
			},
			binaryPath: "sub/link",
			createFn:   testutil.CreateTarGz,
			wantErr:    ErrPathTraversal,
		},
		{
			name:        "absolute path entry in tar",
			archiveName: "absolute.tar.gz",
			entries: []testutil.ArchiveEntry{
				{Name: "/usr/bin/evil", Content: []byte("bad")},
			},
			binaryPath: "/usr/bin/evil",
			createFn:   testutil.CreateTarGz,
			wantErr:    ErrPathTraversal,
		},
		{
			name:        "binaryPath with traversal",
			archiveName: "normal.tar.gz",
			entries: []testutil.ArchiveEntry{
				{Name: "tool", Content: binaryContent},
			},
			binaryPath: "../etc/passwd",
			createFn:   testutil.CreateTarGz,
			wantErr:    ErrPathTraversal,
		},
		{
			name:        "missing binary in tar.gz",
			archiveName: "missing.tar.gz",
			entries: []testutil.ArchiveEntry{
				{Name: "other-tool", Content: []byte("not what you want")},
			},
			binaryPath: "mytool",
			createFn:   testutil.CreateTarGz,
		},
		{
			name:        "missing binary in zip",
			archiveName: "missing.zip",
			entries: []testutil.ArchiveEntry{
				{Name: "other-tool", Content: []byte("not what you want")},
			},
			binaryPath: "mytool",
			createFn:   testutil.CreateZip,
		},
		{
			name:        "raw binary copy for non-archive file",
			archiveName: "mytool",
			binaryPath:  "mytool",
			rawContent:  binaryContent,
			wantContent: binaryContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			archivePath := filepath.Join(tmpDir, tt.archiveName)
			outputPath := filepath.Join(tmpDir, "output-binary")

			// Write the archive or raw file to disk.
			var data []byte
			if tt.rawContent != nil {
				data = tt.rawContent
			} else {
				data = tt.createFn(t, tt.entries)
			}

			err := os.WriteFile(archivePath, data, 0o644)
			if err != nil {
				t.Fatalf("writing test archive: %v", err)
			}

			err = ExtractBinary(archivePath, tt.binaryPath, outputPath)

			// Check error expectations.
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ExtractBinary() error = nil, want %v", tt.wantErr)
				}

				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ExtractBinary() error = %v, want %v", err, tt.wantErr)
				}

				return
			}

			// For "missing binary" tests, we expect a non-nil error but not a sentinel.
			if tt.wantContent == nil {
				if err == nil {
					t.Fatal("ExtractBinary() error = nil, want error for missing binary")
				}

				return
			}

			if err != nil {
				t.Fatalf("ExtractBinary() unexpected error = %v", err)
			}

			// Verify output file content.
			got, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("reading output file: %v", err)
			}

			if !bytes.Equal(got, tt.wantContent) {
				t.Errorf("output content = %q, want %q", got, tt.wantContent)
			}

			// Verify file permissions.
			info, err := os.Stat(outputPath)
			if err != nil {
				t.Fatalf("stat output file: %v", err)
			}

			wantPerm := os.FileMode(0o755)
			if info.Mode().Perm() != wantPerm {
				t.Errorf("output permissions = %o, want %o", info.Mode().Perm(), wantPerm)
			}
		})
	}
}
