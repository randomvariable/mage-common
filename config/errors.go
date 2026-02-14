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

package config

import "errors"

// Sentinel errors for error classification and handling.
var (
	// ErrAlreadyInitialized is returned when Init() is called more than once.
	// This is a harmless error that can be safely ignored in most cases
	// since sync.Once ensures the first call succeeds.
	ErrAlreadyInitialized = errors.New("config: already initialized")

	// ErrInvalidOption is returned when a configuration option fails validation.
	// This indicates a programming error in option arguments (e.g., nil logger,
	// empty key name, non-existent path).
	ErrInvalidOption = errors.New("config: invalid option")

	// ErrConfigParseFailed is returned when the configuration file cannot be parsed.
	// This indicates a syntax error in the YAML file.
	// Check the wrapped error for specific parsing details.
	ErrConfigParseFailed = errors.New("config: configuration file parse failed")

	// ErrEmptyConfigFileName is returned when WithConfigFile is called with an empty string.
	ErrEmptyConfigFileName = errors.New("config: config file name cannot be empty")

	// ErrNoPathsProvided is returned when WithConfigPaths is called with no paths.
	ErrNoPathsProvided = errors.New("config: at least one path must be provided")

	// ErrEmptyConfigPath is returned when WithConfigPaths is called with an empty string path.
	ErrEmptyConfigPath = errors.New("config: config path cannot be empty string")

	// ErrConfigPathNotExist is a base error for non-existent config paths.
	ErrConfigPathNotExist = errors.New("config: config path does not exist")

	// ErrEmptyEnvPrefix is returned when WithEnvPrefix is called with an empty string.
	ErrEmptyEnvPrefix = errors.New("config: env prefix cannot be empty string")

	// ErrNilLogger is returned when WithLogger is called with a nil logger.
	ErrNilLogger = errors.New("config: logger cannot be nil")

	// ErrNoSensitiveKeys is returned when WithSensitiveKeys is called with no keys.
	ErrNoSensitiveKeys = errors.New("config: at least one sensitive key must be provided")

	// ErrEmptySensitiveKey is returned when WithSensitiveKeys is called with an empty key.
	ErrEmptySensitiveKey = errors.New("config: sensitive key cannot be empty string")

	// ErrNoKeysProvided is returned when WithoutSensitiveKeys is called with no keys.
	ErrNoKeysProvided = errors.New("config: at least one key must be provided")

	// ErrEmptyKey is returned when WithoutSensitiveKeys is called with an empty key.
	ErrEmptyKey = errors.New("config: key cannot be empty string")
)
