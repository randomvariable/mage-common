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
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	testValueString = "test-value"
	testEnvValue    = "from-env"
)

// initTestConfig creates an isolated config for parallel testing.
// It uses a fresh viper instance (not global) to avoid test interference.
func initTestConfig(t *testing.T, configDir string, opts ...Option) (*config, error) {
	t.Helper()

	cfg := &config{
		viper:         viper.New(),
		sensitiveKeys: make(map[string]bool),
	}

	if configDir != "" {
		cfg.viper.AddConfigPath(configDir)
	}

	err := cfg.initialize(opts...)

	return cfg, err
}

// initTestConfigWithFlags creates an isolated config with a custom pflag set.
func initTestConfigWithFlags(t *testing.T, configDir string, flagSet *pflag.FlagSet, opts ...Option) (*config, error) {
	t.Helper()

	cfg := &config{
		viper:         viper.New(),
		sensitiveKeys: make(map[string]bool),
		flagSet:       flagSet,
	}

	if configDir != "" {
		cfg.viper.AddConfigPath(configDir)
	}

	err := cfg.initialize(opts...)

	return cfg, err
}

// writeTestConfig writes config.yaml to the given directory.
func writeTestConfig(t *testing.T, dir, content string) {
	t.Helper()

	configFile := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(configFile, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
}

// writeNamedConfig writes a named config file to the given directory.
func writeNamedConfig(t *testing.T, dir, name, content string) {
	t.Helper()

	configFile := filepath.Join(dir, name)

	err := os.WriteFile(configFile, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
}

// setTestEnv sets an environment variable for the duration of the test.
// Uses unique env var names per test to allow safe parallel execution.
// Note: os.Setenv is used instead of t.Setenv because the Go runtime
// forbids combining t.Setenv with t.Parallel() on the same test.
func setTestEnv(t *testing.T, key, value string) {
	t.Helper()

	err := os.Setenv(key, value)
	if err != nil {
		t.Fatalf("os.Setenv(%q) failed: %v", key, err)
	}

	t.Cleanup(func() {
		err := os.Unsetenv(key)
		if err != nil {
			t.Errorf("os.Unsetenv(%q) failed: %v", key, err)
		}
	})
}

func TestInit_BasicUsage(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	writeTestConfig(t, tmpDir, `
test:
  key: "test-value"
  number: 42
`)

	cfg, err := initTestConfig(t, tmpDir)
	if err != nil {
		t.Fatalf("initialize() failed: %v", err)
	}

	if got := cfg.viper.GetString("test.key"); got != testValueString {
		t.Errorf("GetString(\"test.key\") = %q, want %q", got, testValueString)
	}

	if got := cfg.viper.GetInt("test.number"); got != 42 {
		t.Errorf("GetInt(\"test.number\") = %d, want %d", got, 42)
	}
}

func TestInit_MissingConfigFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	_, err := initTestConfig(t, tmpDir)
	if err != nil {
		t.Errorf("initialize() with missing config should not error, got: %v", err)
	}
}

func TestInit_InvalidYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	writeTestConfig(t, tmpDir, `
invalid: yaml: content:
  - broken
    indentation
`)

	_, err := initTestConfig(t, tmpDir)
	if err == nil {
		t.Error("initialize() with invalid YAML should return error")
	}

	if !errors.Is(err, ErrConfigParseFailed) {
		t.Errorf("error should be ErrConfigParseFailed, got: %v", err)
	}
}

func TestInit_Idempotent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	cfg := &config{
		viper:         viper.New(),
		sensitiveKeys: make(map[string]bool),
	}
	cfg.viper.AddConfigPath(tmpDir)

	// First call via initOnce
	cfg.initOnce.Do(func() {
		cfg.initErr = cfg.initialize()
	})

	if cfg.initErr != nil {
		t.Fatalf("First initialize() failed: %v", cfg.initErr)
	}

	// Second call should be no-op (sync.Once guarantees this)
	secondCalled := false

	cfg.initOnce.Do(func() {
		secondCalled = true
	})

	if secondCalled {
		t.Error("Second initOnce.Do should not execute")
	}
}

func TestInit_EnvironmentVariableOverride(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	writeTestConfig(t, tmpDir, `
app:
  name: "from-file"
`)

	// Use unique env prefix to avoid collisions with other parallel tests
	setTestEnv(t, "ENVTEST_APP_NAME", testEnvValue)

	cfg, err := initTestConfig(t, tmpDir, WithEnvPrefix("ENVTEST"))
	if err != nil {
		t.Fatalf("initialize() failed: %v", err)
	}

	// Environment variable should override config file
	if got := cfg.viper.GetString("app.name"); got != testEnvValue {
		t.Errorf("GetString(\"app.name\") = %q, want %q (env should override file)", got, testEnvValue)
	}
}

func TestInit_WithOptions(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	mockLog := &mockLogger{}

	cfg, err := initTestConfig(t, tmpDir,
		WithLogger(mockLog),
		WithSensitiveKeys("password", "token"),
	)
	if err != nil {
		t.Fatalf("initialize() with options failed: %v", err)
	}

	if cfg.logger != mockLog {
		t.Error("Logger was not set correctly")
	}

	if !cfg.sensitiveKeys["password"] {
		t.Error("Sensitive key 'password' was not set")
	}

	if !cfg.sensitiveKeys["token"] {
		t.Error("Sensitive key 'token' was not set")
	}
}

func TestInit_InvalidOption(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	_, err := initTestConfig(t, tmpDir, WithLogger(nil))
	if err == nil {
		t.Error("initialize() with nil logger should return error")
	}

	if !errors.Is(err, ErrInvalidOption) {
		t.Errorf("error should be ErrInvalidOption, got: %v", err)
	}
}

func TestV_BeforeInit(t *testing.T) {
	t.Parallel()

	cfg := &config{sensitiveKeys: make(map[string]bool)}

	defer func() {
		if r := recover(); r == nil {
			t.Error("v() before initialize should panic")
		}
	}()

	cfg.v()
}

func TestV_AfterInit(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	cfg, err := initTestConfig(t, tmpDir)
	if err != nil {
		t.Fatalf("initialize() failed: %v", err)
	}

	viperInstance := cfg.v()
	if viperInstance == nil {
		t.Error("v() should return non-nil Viper instance after initialize")
	}
}

func TestInit_DotNotation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	writeTestConfig(t, tmpDir, `
tools:
  golangci-lint:
    version: "1.62.2"
    enabled: true
  gofumpt:
    version: "0.7.0"
database:
  host: "localhost"
  port: 5432
  credentials:
    username: "admin"
`)

	cfg, err := initTestConfig(t, tmpDir)
	if err != nil {
		t.Fatalf("initialize() failed: %v", err)
	}

	tests := []struct {
		key  string
		want any
	}{
		{"tools.golangci-lint.version", "1.62.2"},
		{"tools.golangci-lint.enabled", true},
		{"tools.gofumpt.version", "0.7.0"},
		{"database.host", "localhost"},
		{"database.port", 5432},
		{"database.credentials.username", "admin"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()

			got := cfg.viper.Get(tt.key)
			if got != tt.want {
				t.Errorf("Get(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestInit_InitOnce(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	cfg := &config{
		viper:         viper.New(),
		sensitiveKeys: make(map[string]bool),
	}
	cfg.viper.AddConfigPath(tmpDir)

	// Run multiple goroutines to test thread safety
	var wg sync.WaitGroup

	const goroutines = 10

	errs := make([]error, goroutines)

	for i := range goroutines {
		wg.Go(func() {
			cfg.initOnce.Do(func() {
				cfg.initErr = cfg.initialize()
			})

			errs[i] = cfg.initErr
		})
	}

	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d got error: %v", i, err)
		}
	}
}
