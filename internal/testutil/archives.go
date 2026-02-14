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

package testutil

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"testing"
)

const testBinaryPerm = 0o755

// ArchiveEntry describes a single entry in a test archive.
type ArchiveEntry struct {
	// Name is the file path within the archive.
	Name string

	// Content is the file content.
	Content []byte

	// IsDir marks this entry as a directory.
	IsDir bool

	// IsSymlink marks this entry as a symlink (tar only).
	IsSymlink bool

	// LinkTarget is the symlink target path.
	LinkTarget string
}

// CreateTarGz builds an in-memory tar.gz archive from the given entries.
func CreateTarGz(t *testing.T, entries []ArchiveEntry) []byte {
	t.Helper()

	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, entry := range entries {
		header := &tar.Header{
			Name: entry.Name,
			Mode: testBinaryPerm,
		}

		switch {
		case entry.IsDir:
			header.Typeflag = tar.TypeDir
		case entry.IsSymlink:
			header.Typeflag = tar.TypeSymlink
			header.Linkname = entry.LinkTarget
		default:
			header.Typeflag = tar.TypeReg
			header.Size = int64(len(entry.Content))
		}

		err := tarWriter.WriteHeader(header)
		if err != nil {
			t.Fatalf("writing tar header for %q: %v", entry.Name, err)
		}

		if !entry.IsDir && !entry.IsSymlink && len(entry.Content) > 0 {
			_, err = tarWriter.Write(entry.Content)
			if err != nil {
				t.Fatalf("writing tar content for %q: %v", entry.Name, err)
			}
		}
	}

	err := tarWriter.Close()
	if err != nil {
		t.Fatalf("closing tar writer: %v", err)
	}

	err = gzipWriter.Close()
	if err != nil {
		t.Fatalf("closing gzip writer: %v", err)
	}

	return buf.Bytes()
}

// CreateZip builds an in-memory zip archive from the given entries.
func CreateZip(t *testing.T, entries []ArchiveEntry) []byte {
	t.Helper()

	var buf bytes.Buffer

	zipWriter := zip.NewWriter(&buf)

	for _, entry := range entries {
		if entry.IsDir {
			// Zip directories end with a slash.
			name := entry.Name
			if name[len(name)-1] != '/' {
				name += "/"
			}

			_, err := zipWriter.Create(name)
			if err != nil {
				t.Fatalf("creating zip dir %q: %v", name, err)
			}

			continue
		}

		writer, err := zipWriter.Create(entry.Name)
		if err != nil {
			t.Fatalf("creating zip entry %q: %v", entry.Name, err)
		}

		_, err = writer.Write(entry.Content)
		if err != nil {
			t.Fatalf("writing zip content for %q: %v", entry.Name, err)
		}
	}

	err := zipWriter.Close()
	if err != nil {
		t.Fatalf("closing zip writer: %v", err)
	}

	return buf.Bytes()
}
