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
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

// DownloadInstaller installs tools by downloading binaries or archives via HTTP.
type DownloadInstaller struct {
	// Client is the HTTP client to use. If nil, http.DefaultClient is used.
	Client *http.Client
}

// Install downloads and installs a tool from a download source.
//
//nolint:cyclop // download installation requires multiple sequential steps
func (d *DownloadInstaller) Install(ctx context.Context, tool v1alpha1.Tool, source *v1alpha1.ToolSource, toolsDir string) error {
	version := ptr.Deref(tool.Version, "")

	// Check if already installed at correct version.
	current, err := IsCurrentVersion(tool.Name, version, toolsDir)
	if err != nil {
		return fmt.Errorf("checking current version: %w", err)
	}

	if current {
		return nil
	}

	platformDir, err := EnsureToolsDir(toolsDir)
	if err != nil {
		return err
	}

	templateData := NewTemplateData(version, source.OsMap, source.ArchMap)

	// Render the download URL.
	downloadURL, err := Render(ptr.Deref(source.URL, ""), &templateData)
	if err != nil {
		return fmt.Errorf("rendering download URL: %w", err)
	}

	// Preserve archive extension so ExtractBinary can detect the format.
	tmpExt := archiveExtension(downloadURL)

	tmpFile, err := os.CreateTemp(platformDir, tool.Name+"-download-*"+tmpExt)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	tmpPath := tmpFile.Name()

	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	checksumHash, checksumExpected, err := d.resolveChecksum(ctx, source, &templateData)
	if err != nil {
		return fmt.Errorf("resolving checksum: %w", err)
	}

	err = d.downloadFile(ctx, downloadURL, tmpFile, checksumHash)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", downloadURL, err)
	}

	_ = tmpFile.Close()

	// Verify checksum if provided.
	if checksumHash != nil {
		actual := hex.EncodeToString(checksumHash.Sum(nil))
		if actual != checksumExpected {
			return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, checksumExpected, actual)
		}
	}

	// Determine binary path within archive.
	binaryPath := tool.Name
	if source.BinaryPath != nil && *source.BinaryPath != "" {
		binaryPath, err = Render(*source.BinaryPath, &templateData)
		if err != nil {
			return fmt.Errorf("rendering binaryPath: %w", err)
		}
	}

	// Extract binary from archive (or treat as raw binary).
	extractedPath := filepath.Join(platformDir, tool.Name+"-extracted")

	defer func() { _ = os.Remove(extractedPath) }()

	err = ExtractBinary(tmpPath, binaryPath, extractedPath)
	if err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}

	// Set version with symlink.
	err = SetVersion(tool.Name, version, toolsDir, extractedPath)
	if err != nil {
		return fmt.Errorf("setting version: %w", err)
	}

	return nil
}

// IsInstalled checks if the download tool is installed at the expected version.
func (d *DownloadInstaller) IsInstalled(tool v1alpha1.Tool, toolsDir string) (bool, error) {
	return IsCurrentVersion(tool.Name, ptr.Deref(tool.Version, ""), toolsDir)
}

// RuntimeAvailable always returns nil since downloads only need HTTP.
func (d *DownloadInstaller) RuntimeAvailable() error {
	return nil
}

func (d *DownloadInstaller) httpClient() *http.Client {
	if d.Client != nil {
		return d.Client
	}

	return http.DefaultClient
}

func (d *DownloadInstaller) downloadFile(ctx context.Context, downloadURL string, dst *os.File, checksumWriter hash.Hash) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	client := d.httpClient()

	// Create a custom transport that rejects HTTPS->HTTP redirects.
	originalTransport := client.Transport
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 {
			prevURL := via[len(via)-1].URL
			if prevURL.Scheme == "https" && req.URL.Scheme == "http" {
				return fmt.Errorf("%w: redirect from HTTPS to HTTP rejected", ErrHTTPSRequired)
			}
		}

		const maxRedirects = 10
		if len(via) >= maxRedirects {
			return ErrTooManyRedirects
		}

		return nil
	}

	// Restore original transport after download.
	defer func() {
		client.Transport = originalTransport
	}()

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d %s", ErrHTTPStatus, resp.StatusCode, resp.Status)
	}

	var writer io.Writer = dst
	if checksumWriter != nil {
		writer = io.MultiWriter(dst, checksumWriter)
	}

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("writing response: %w", err)
	}

	return nil
}

func (d *DownloadInstaller) resolveChecksum(
	ctx context.Context,
	source *v1alpha1.ToolSource,
	templateData *TemplateData,
) (hash.Hash, string, error) {
	if source.Checksum == nil {
		return nil, "", nil
	}

	checksumInline := ptr.Deref(source.Checksum.Inline, "")
	checksumURLStr := ptr.Deref(source.Checksum.URL, "")

	if checksumInline != "" {
		return parseInlineChecksum(checksumInline)
	}

	if checksumURLStr != "" {
		checksumURL, err := Render(checksumURLStr, templateData)
		if err != nil {
			return nil, "", fmt.Errorf("rendering checksum URL: %w", err)
		}

		downloadURL, err := Render(ptr.Deref(source.URL, ""), templateData)
		if err != nil {
			return nil, "", fmt.Errorf("rendering download URL for checksum matching: %w", err)
		}

		return d.fetchChecksumFromURL(ctx, checksumURL, downloadURL)
	}

	return nil, "", nil
}

func parseInlineChecksum(inline string) (hash.Hash, string, error) {
	parts := strings.SplitN(inline, ":", 2) //nolint:mnd // format is algo:hex
	if len(parts) != 2 {                    //nolint:mnd // exactly 2 parts expected
		return nil, "", fmt.Errorf("%w: %q", ErrInvalidChecksum, inline)
	}

	algo := parts[0]
	checksumHex := parts[1]

	h, err := newHashForAlgorithm(algo)
	if err != nil {
		return nil, "", err
	}

	return h, checksumHex, nil
}

func newHashForAlgorithm(algo string) (hash.Hash, error) {
	switch algo {
	case "sha256":
		return sha256.New(), nil
	case "sha384":
		return sha512.New384(), nil
	case "sha512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrWeakChecksum, algo)
	}
}

func (d *DownloadInstaller) fetchChecksumFromURL(
	ctx context.Context,
	checksumURL, downloadURL string,
) (hash.Hash, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, http.NoBody)
	if err != nil {
		return nil, "", fmt.Errorf("creating checksum request: %w", err)
	}

	resp, err := d.httpClient().Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetching checksums: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("%w: fetching checksums: %d", ErrHTTPStatus, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading checksums: %w", err)
	}

	// Parse the download URL to extract the filename for matching.
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return nil, "", fmt.Errorf("parsing download URL: %w", err)
	}

	filename := filepath.Base(parsedURL.Path)

	return matchChecksumLine(string(body), filename)
}

// matchChecksumLine finds the checksum for a given filename. It supports two
// formats:
//   - Bare hash: a single hex digest on one line with no filename (e.g. kubectl.sha256).
//   - GNU coreutils: <hash>  <filename> (e.g. checksums.txt with multiple entries).
func matchChecksumLine(content, filename string) (hash.Hash, string, error) {
	// Collect non-empty lines.
	var lines []string

	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}

	// Single line with a single field: treat as a bare hash.
	if len(lines) == 1 && len(strings.Fields(lines[0])) == 1 {
		return hashFromHex(lines[0])
	}

	checksumHex, err := findChecksumForFile(lines, filename)
	if err != nil {
		return nil, "", err
	}

	return hashFromHex(checksumHex)
}

// findChecksumForFile finds a unique checksum for filename in GNU coreutils format lines.
func findChecksumForFile(lines []string, filename string) (string, error) {
	var matches []string

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 { //nolint:mnd // hash + filename
			continue
		}

		lineFilename := strings.TrimPrefix(parts[len(parts)-1], "*")
		if filepath.Base(lineFilename) == filename {
			matches = append(matches, parts[0])
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("%w: %q", ErrChecksumNotFound, filename)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("%w: %d matches for %q", ErrAmbiguousChecksum, len(matches), filename)
	}

	return matches[0], nil
}

// hashFromHex returns a hash.Hash and the hex string for a given checksum hex,
// detecting the algorithm from the hex length.
func hashFromHex(checksumHex string) (hash.Hash, string, error) {
	algo, err := algorithmFromLength(len(checksumHex))
	if err != nil {
		return nil, "", err
	}

	h, err := newHashForAlgorithm(algo)
	if err != nil {
		return nil, "", err
	}

	return h, checksumHex, nil
}

func algorithmFromLength(hexLen int) (string, error) {
	const (
		sha256HexLen = 64
		sha384HexLen = 96
		sha512HexLen = 128
	)

	switch hexLen {
	case sha256HexLen:
		return "sha256", nil
	case sha384HexLen:
		return "sha384", nil
	case sha512HexLen:
		return "sha512", nil
	default:
		return "", fmt.Errorf("%w: hash length %d", ErrUnknownHashAlgorithm, hexLen)
	}
}

// archiveExtension returns the archive file extension from a URL path,
// handling compound extensions like .tar.gz.
func archiveExtension(downloadURL string) string {
	parsed, err := url.Parse(downloadURL)
	if err != nil {
		return ""
	}

	base := filepath.Base(parsed.Path)

	if strings.HasSuffix(base, ".tar.gz") {
		return ".tar.gz"
	}

	if strings.HasSuffix(base, ".tgz") {
		return ".tgz"
	}

	return filepath.Ext(base)
}
