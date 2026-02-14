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
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	// binaryPerm is the permission mode for extracted binaries (rwxr-xr-x).
	extractedBinaryPerm = 0o755
)

// ExtractBinary extracts a single binary from an archive to the specified output path.
// The archive format is detected by file extension: .tar.gz and .tgz are treated as
// gzipped tarballs, .zip as zip archives, and anything else as a raw binary that is
// copied directly.
//
// Security: all archive entry paths are validated against path traversal attacks
// (Zip Slip), absolute paths, and symlinks pointing outside the extraction context.
func ExtractBinary(archivePath, binaryPath, outputPath string) error {
	err := validatePath(binaryPath)
	if err != nil {
		return err
	}

	switch {
	case strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz"):
		return extractFromTarGz(archivePath, binaryPath, outputPath)
	case strings.HasSuffix(archivePath, ".zip"):
		return extractFromZip(archivePath, binaryPath, outputPath)
	default:
		return copyRawBinary(archivePath, outputPath)
	}
}

// validatePath checks that a path does not contain traversal sequences or absolute components.
func validatePath(p string) error {
	if filepath.IsAbs(p) {
		return fmt.Errorf("%w: absolute path %q", ErrPathTraversal, p)
	}

	if containsTraversal(p) {
		return fmt.Errorf("%w: path %q contains traversal", ErrPathTraversal, p)
	}

	return nil
}

// containsTraversal reports whether a path contains "../" sequences.
func containsTraversal(p string) bool {
	return slices.Contains(strings.Split(filepath.ToSlash(p), "/"), "..")
}

// extractFromTarGz extracts a single file from a gzipped tarball.
func extractFromTarGz(archivePath, binaryPath, outputPath string) error {
	f, err := os.Open(archivePath) //nolint:gosec // File path from trusted configuration
	if err != nil {
		return fmt.Errorf("opening archive %q: %w", archivePath, err)
	}

	defer func() { _ = f.Close() }()

	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating gzip reader for %q: %w", archivePath, err)
	}

	defer func() { _ = gzipReader.Close() }()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%w: %q in %q", ErrBinaryNotFound, binaryPath, archivePath)
		}

		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		err = validateTarEntry(header)
		if err != nil {
			return err
		}

		if header.Name != binaryPath {
			continue
		}

		return writeFile(tarReader, outputPath)
	}
}

// validateTarEntry checks a tar header for path traversal and dangerous symlinks.
func validateTarEntry(header *tar.Header) error {
	err := validatePath(header.Name)
	if err != nil {
		return err
	}

	if header.Typeflag == tar.TypeSymlink {
		if filepath.IsAbs(header.Linkname) {
			return fmt.Errorf("%w: symlink %q points to absolute path %q", ErrPathTraversal, header.Name, header.Linkname)
		}

		// Resolve the symlink target relative to the entry's directory.
		resolved := filepath.Join(filepath.Dir(header.Name), header.Linkname) //nolint:gosec // Symlink target validated by validateTarEntry
		if containsTraversal(resolved) {
			return fmt.Errorf("%w: symlink %q escapes archive via %q", ErrPathTraversal, header.Name, header.Linkname)
		}
	}

	return nil
}

// extractFromZip extracts a single file from a zip archive.
func extractFromZip(archivePath, binaryPath, outputPath string) error {
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("opening zip archive %q: %w", archivePath, err)
	}

	defer func() { _ = zipReader.Close() }()

	for _, entry := range zipReader.File {
		err = validatePath(entry.Name)
		if err != nil {
			return err
		}

		if entry.Name != binaryPath {
			continue
		}

		entryReader, err := entry.Open()
		if err != nil {
			return fmt.Errorf("opening zip entry %q: %w", entry.Name, err)
		}

		writeErr := writeFile(entryReader, outputPath)
		_ = entryReader.Close()

		return writeErr
	}

	return fmt.Errorf("%w: %q in %q", ErrBinaryNotFound, binaryPath, archivePath)
}

// copyRawBinary copies a non-archive file directly to the output path.
func copyRawBinary(srcPath, outputPath string) error {
	src, err := os.Open(srcPath) //nolint:gosec // File path from trusted configuration
	if err != nil {
		return fmt.Errorf("opening source binary %q: %w", srcPath, err)
	}

	defer func() { _ = src.Close() }()

	return writeFile(src, outputPath)
}

// writeFile writes the contents from r to the given path with executable permissions.
func writeFile(r io.Reader, outputPath string) error {
	out, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, extractedBinaryPerm) //nolint:gosec // File path from trusted configuration
	if err != nil {
		return fmt.Errorf("creating output file %q: %w", outputPath, err)
	}

	_, err = io.Copy(out, r)
	if err != nil {
		_ = out.Close()

		return fmt.Errorf("writing to %q: %w", outputPath, err)
	}

	err = out.Close()
	if err != nil {
		return fmt.Errorf("closing output file %q: %w", outputPath, err)
	}

	// Explicitly set permissions because os.OpenFile is subject to the process umask.
	err = os.Chmod(outputPath, extractedBinaryPerm)
	if err != nil {
		return fmt.Errorf("setting permissions on %q: %w", outputPath, err)
	}

	return nil
}
