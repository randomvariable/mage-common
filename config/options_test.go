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

import (
	"errors"
	"path/filepath"
	"testing"
)

// runOptionValidationTests runs table-driven tests for option validation.
// It handles the common pattern of testing an option with valid/invalid inputs
// and checking for ErrInvalidOption wrapping.
func runOptionValidationTests(t *testing.T, tests []optionValidationCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()

			_, err := initTestConfig(t, tmpDir, tt.opt)

			if tt.wantError {
				if err == nil {
					t.Errorf("option should return error for case %q", tt.name)
				}

				if !errors.Is(err, ErrInvalidOption) {
					t.Errorf("Error should be ErrInvalidOption, got: %v", err)
				}
			} else if err != nil {
				t.Errorf("option unexpected error for case %q: %v", tt.name, err)
			}
		})
	}
}

// optionValidationCase is a test case for option validation.
type optionValidationCase struct {
	name      string
	opt       Option
	wantError bool
}

func TestWithConfigFile(t *testing.T) {
	t.Parallel()

	tests := []optionValidationCase{
		{
			name:      "valid filename",
			opt:       WithConfigFile("myconfig"),
			wantError: false,
		},
		{
			name:      "empty string",
			opt:       WithConfigFile(""),
			wantError: true,
		},
		{
			name:      "filename with extension",
			opt:       WithConfigFile("config.yaml"),
			wantError: false,
		},
	}

	runOptionValidationTests(t, tests)
}

func TestWithConfigFile_LoadsCustomFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	writeNamedConfig(t, tmpDir, "custom.yaml", `
custom:
  loaded: true
`)

	cfg, err := initTestConfig(t, tmpDir, WithConfigFile("custom"))
	if err != nil {
		t.Fatalf("initialize() with custom config failed: %v", err)
	}

	if got := cfg.viper.GetBool("custom.loaded"); !got {
		t.Error("Custom config file was not loaded")
	}
}

func TestWithConfigPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		makePaths func(t *testing.T) []string
		wantError bool
	}{
		{
			name: "valid existing paths",
			makePaths: func(t *testing.T) []string {
				t.Helper()

				return []string{t.TempDir(), t.TempDir()}
			},
			wantError: false,
		},
		{
			name: "non-existent path",
			makePaths: func(_ *testing.T) []string {
				return []string{"/nonexistent/path/12345"}
			},
			wantError: true,
		},
		{
			name: "empty path string",
			makePaths: func(_ *testing.T) []string {
				return []string{""}
			},
			wantError: true,
		},
		{
			name: "empty paths slice",
			makePaths: func(_ *testing.T) []string {
				return []string{}
			},
			wantError: true,
		},
		{
			name: "mixed valid and empty",
			makePaths: func(t *testing.T) []string {
				t.Helper()

				return []string{t.TempDir(), ""}
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			paths := tt.makePaths(t)
			tmpDir := t.TempDir()

			_, err := initTestConfig(t, tmpDir, WithConfigPaths(paths...))

			if tt.wantError {
				if err == nil {
					t.Errorf("WithConfigPaths(%v) should return error", paths)
				}

				if !errors.Is(err, ErrInvalidOption) {
					t.Errorf("Error should be ErrInvalidOption, got: %v", err)
				}
			} else if err != nil {
				t.Errorf("WithConfigPaths(%v) unexpected error: %v", paths, err)
			}
		})
	}
}

func TestWithConfigPaths_SearchOrder(t *testing.T) {
	t.Parallel()

	// Create two directories with different config files
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	writeTestConfig(t, dir1, "priority: first\n")
	writeTestConfig(t, dir2, "priority: second\n")

	// dir1 should be searched first (added first via option)
	cfg, err := initTestConfig(t, "", WithConfigPaths(dir1, dir2))
	if err != nil {
		t.Fatalf("initialize() failed: %v", err)
	}

	if got := cfg.viper.GetString("priority"); got != "first" {
		t.Errorf("Config search order incorrect, got %q, want %q", got, "first")
	}
}

func TestWithEnvPrefix(t *testing.T) {
	t.Parallel()

	tests := []optionValidationCase{
		{
			name:      "valid prefix",
			opt:       WithEnvPrefix("MYAPP"),
			wantError: false,
		},
		{
			name:      "empty string",
			opt:       WithEnvPrefix(""),
			wantError: true,
		},
		{
			name:      "prefix with underscore",
			opt:       WithEnvPrefix("MY_APP"),
			wantError: false,
		},
	}

	runOptionValidationTests(t, tests)
}

func TestWithEnvPrefix_EnvironmentBinding(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	setTestEnv(t, "PFXTEST_DATABASE_HOST", "prod-db.example.com")

	cfg, err := initTestConfig(t, tmpDir, WithEnvPrefix("PFXTEST"))
	if err != nil {
		t.Fatalf("initialize() with env prefix failed: %v", err)
	}

	if got := cfg.viper.GetString("database.host"); got != "prod-db.example.com" {
		t.Errorf("Environment variable with prefix not bound correctly, got %q", got)
	}
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	mockLog := &mockLogger{}

	cfg, err := initTestConfig(t, tmpDir, WithLogger(mockLog))
	if err != nil {
		t.Fatalf("WithLogger() failed: %v", err)
	}

	if cfg.logger != mockLog {
		t.Error("Logger was not set correctly")
	}
}

func TestWithLogger_Nil(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	_, err := initTestConfig(t, tmpDir, WithLogger(nil))
	if err == nil {
		t.Error("WithLogger(nil) should return error")
	}

	if !errors.Is(err, ErrInvalidOption) {
		t.Errorf("Error should be ErrInvalidOption, got: %v", err)
	}
}

func TestWithSensitiveKeys(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	cfg, err := initTestConfig(t, tmpDir, WithSensitiveKeys("password", "api-key", "secret"))
	if err != nil {
		t.Fatalf("WithSensitiveKeys() failed: %v", err)
	}

	expectedKeys := []string{"password", "api-key", "secret"}
	for _, key := range expectedKeys {
		if !cfg.sensitiveKeys[key] {
			t.Errorf("Sensitive key %q was not set", key)
		}
	}
}

func TestWithSensitiveKeys_Empty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		keys []string
	}{
		{"no keys", []string{}},
		{"empty string key", []string{""}},
		{"mixed with empty", []string{"valid", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()

			_, err := initTestConfig(t, tmpDir, WithSensitiveKeys(tt.keys...))
			if err == nil {
				t.Error("WithSensitiveKeys with invalid keys should return error")
			}

			if !errors.Is(err, ErrInvalidOption) {
				t.Errorf("Error should be ErrInvalidOption, got: %v", err)
			}
		})
	}
}

func TestWithoutSensitiveKeys(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	cfg, err := initTestConfig(t, tmpDir,
		WithSensitiveKeys("password", "api-key", "secret", "token"),
		WithoutSensitiveKeys("api-key", "secret"),
	)
	if err != nil {
		t.Fatalf("initialize() failed: %v", err)
	}

	// Check remaining keys
	if !cfg.sensitiveKeys["password"] {
		t.Error("'password' should still be sensitive")
	}

	if !cfg.sensitiveKeys["token"] {
		t.Error("'token' should still be sensitive")
	}

	// Check removed keys
	if cfg.sensitiveKeys["api-key"] {
		t.Error("'api-key' should have been removed")
	}

	if cfg.sensitiveKeys["secret"] {
		t.Error("'secret' should have been removed")
	}
}

func TestWithoutSensitiveKeys_Empty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		keys []string
	}{
		{"no keys", []string{}},
		{"empty string key", []string{""}},
		{"mixed with empty", []string{"valid", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()

			_, err := initTestConfig(t, tmpDir, WithoutSensitiveKeys(tt.keys...))
			if err == nil {
				t.Error("WithoutSensitiveKeys with invalid keys should return error")
			}

			if !errors.Is(err, ErrInvalidOption) {
				t.Errorf("Error should be ErrInvalidOption, got: %v", err)
			}
		})
	}
}

func TestMultipleOptions(t *testing.T) {
	t.Parallel()

	configDir := t.TempDir()
	writeNamedConfig(t, configDir, "app.yaml", `
database:
  host: "localhost"
`)

	tmpDir := t.TempDir()
	mockLog := &mockLogger{}

	cfg, err := initTestConfig(t, tmpDir,
		WithConfigFile("app"),
		WithConfigPaths(configDir),
		WithEnvPrefix("TESTAPP"),
		WithLogger(mockLog),
		WithSensitiveKeys("password", "token"),
	)
	if err != nil {
		t.Fatalf("initialize() with multiple options failed: %v", err)
	}

	// Verify config was loaded from configDir
	if got := cfg.viper.GetString("database.host"); got != "localhost" {
		t.Errorf("Config not loaded correctly, got %q", got)
	}

	if cfg.logger != mockLog {
		t.Error("Logger was not set")
	}

	if !cfg.sensitiveKeys["password"] || !cfg.sensitiveKeys["token"] {
		t.Error("Sensitive keys were not set correctly")
	}
}

func TestMultipleOptions_ConfigFromSearchPath(t *testing.T) {
	t.Parallel()

	configDir := t.TempDir()
	writeNamedConfig(t, configDir, "app.yaml", `
database:
  host: "localhost"
  port: 5432
`)

	cfg, err := initTestConfig(t, "",
		WithConfigFile("app"),
		WithConfigPaths(configDir),
	)
	if err != nil {
		t.Fatalf("initialize() failed: %v", err)
	}

	if got := cfg.viper.GetString("database.host"); got != "localhost" {
		t.Errorf("database.host = %q, want %q", got, "localhost")
	}

	if got := cfg.viper.GetInt("database.port"); got != 5432 {
		t.Errorf("database.port = %d, want %d", got, 5432)
	}
}

func TestWithConfigPaths_FileInSecondPath(t *testing.T) {
	t.Parallel()

	dir1 := t.TempDir() // Empty - no config file
	dir2 := t.TempDir()

	writeTestConfig(t, dir2, `
source: "dir2"
`)

	cfg, err := initTestConfig(t, "",
		WithConfigPaths(dir1, dir2),
	)
	if err != nil {
		t.Fatalf("initialize() failed: %v", err)
	}

	if got := cfg.viper.GetString("source"); got != "dir2" {
		t.Errorf("source = %q, want %q", got, "dir2")
	}
}

func TestWithConfigPaths_AbsolutePaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	absPath, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("filepath.Abs() failed: %v", err)
	}

	writeTestConfig(t, absPath, `
absolute: true
`)

	cfg, initErr := initTestConfig(t, "", WithConfigPaths(absPath))
	if initErr != nil {
		t.Fatalf("initialize() failed: %v", initErr)
	}

	if got := cfg.viper.GetBool("absolute"); !got {
		t.Error("Config from absolute path not loaded")
	}
}
