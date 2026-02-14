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

import "errors"

// Sentinel errors for the tools package.
var (
	// ErrToolNotFound is returned when a tool name is not present in the configuration.
	ErrToolNotFound = errors.New("tools: tool not found in configuration")

	// ErrToolNotInstalled is returned when trying to run a tool that has not been installed.
	ErrToolNotInstalled = errors.New("tools: tool is not installed")

	// ErrConfigNotFound is returned when the .tools.yaml configuration file cannot be found.
	ErrConfigNotFound = errors.New("tools: configuration file not found")

	// ErrConfigInvalid is returned when the configuration file fails validation.
	ErrConfigInvalid = errors.New("tools: configuration is invalid")

	// ErrRuntimeNotFound is returned when a required runtime (go, gem, npx, cargo, uvx) is not on PATH.
	ErrRuntimeNotFound = errors.New("tools: required runtime not found on PATH")

	// ErrChecksumMismatch is returned when a downloaded file's checksum does not match the expected value.
	ErrChecksumMismatch = errors.New("tools: checksum mismatch")

	// ErrAllSourcesFailed is returned when all configured sources for a tool have failed.
	ErrAllSourcesFailed = errors.New("tools: all sources failed")

	// ErrHTTPSRequired is returned when a download URL uses HTTP instead of HTTPS.
	ErrHTTPSRequired = errors.New("tools: HTTPS required for download URLs")

	// ErrWeakChecksum is returned when a checksum uses a weak algorithm (md5, sha1).
	ErrWeakChecksum = errors.New("tools: weak checksum algorithm")

	// ErrPathTraversal is returned when a path contains traversal sequences (../).
	ErrPathTraversal = errors.New("tools: path traversal detected")

	// ErrUnsupportedArchive is returned when an archive format is not supported.
	ErrUnsupportedArchive = errors.New("tools: unsupported archive format")

	// ErrBinaryNotFound is returned when a binary is not found in an archive.
	ErrBinaryNotFound = errors.New("tools: binary not found in archive")

	// ErrHTTPStatus is returned when an HTTP request receives an unexpected status code.
	ErrHTTPStatus = errors.New("tools: unexpected HTTP status")

	// ErrTooManyRedirects is returned when an HTTP request encounters too many redirects.
	ErrTooManyRedirects = errors.New("tools: too many redirects")

	// ErrInvalidChecksum is returned when a checksum format is invalid.
	ErrInvalidChecksum = errors.New("tools: invalid checksum format")

	// ErrChecksumNotFound is returned when no matching checksum is found in a checksums file.
	ErrChecksumNotFound = errors.New("tools: no checksum found for file")

	// ErrAmbiguousChecksum is returned when multiple checksums match the same filename.
	ErrAmbiguousChecksum = errors.New("tools: ambiguous checksum match")

	// ErrUnknownHashAlgorithm is returned when a hash algorithm cannot be determined.
	ErrUnknownHashAlgorithm = errors.New("tools: unknown hash algorithm")

	// ErrMissingURL is returned when a source has no URL configured.
	ErrMissingURL = errors.New("tools: source has no URL")

	// ErrMissingPackage is returned when a source has no package configured.
	ErrMissingPackage = errors.New("tools: source has no package")

	// ErrNoStableVersions is returned when no stable versions are found in a registry.
	ErrNoStableVersions = errors.New("tools: no stable versions found")

	// ErrNoVersionInResponse is returned when a registry response contains no version field.
	ErrNoVersionInResponse = errors.New("tools: no version field in response")

	// ErrUnexpectedResponse is returned when a registry response has an unexpected format.
	ErrUnexpectedResponse = errors.New("tools: unexpected response format")

	// ErrNoRegistryQuerier is returned when no registry querier exists for a source type.
	ErrNoRegistryQuerier = errors.New("tools: no registry querier for source type")

	// ErrUnsupportedSourceType is returned when a source type is not supported.
	ErrUnsupportedSourceType = errors.New("tools: unsupported source type")

	// ErrUnexpectedDirectory is returned when a directory is found where a file was expected.
	ErrUnexpectedDirectory = errors.New("tools: expected file but found directory")

	// ErrSerializerNotFound is returned when no serializer exists for a media type.
	ErrSerializerNotFound = errors.New("tools: no serializer for media type")

	// ErrCustomGCLNameRequired is returned when the .custom-gcl.yml name field is empty.
	ErrCustomGCLNameRequired = errors.New("tools: custom GCL config name field is required")

	// ErrValidationFailed is returned when tool configuration validation fails.
	ErrValidationFailed = errors.New("tools: validation failed")
)
