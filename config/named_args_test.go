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
	"testing"

	"github.com/spf13/pflag"
)

func TestNamedArgs_BasicUsage(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a local flag set
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("image", "", "Container image name")
	flagSet.String("tag", "", "Image tag")

	// Parse flags as if called: mage build --image=webapp --tag=v1.0
	err := flagSet.Parse([]string{"build", "--image=webapp", "--tag=v1.0"})
	if err != nil {
		t.Fatalf("FlagSet.Parse() failed: %v", err)
	}

	// Initialize config with the local flag set
	cfg, initErr := initTestConfigWithFlags(t, tmpDir, flagSet)
	if initErr != nil {
		t.Fatalf("initialize() failed: %v", initErr)
	}

	// Verify named arguments are accessible via the config's viper
	if got := cfg.viper.GetString("image"); got != "webapp" {
		t.Errorf("GetString(\"image\") = %q, want %q", got, "webapp")
	}

	if got := cfg.viper.GetString("tag"); got != "v1.0" {
		t.Errorf("GetString(\"tag\") = %q, want %q", got, "v1.0")
	}
}

func TestNamedArgs_Precedence_FlagOverEnv(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Set environment variable with unique prefix to avoid parallel collisions
	setTestEnv(t, "FLAGENV_IMAGE", "env-image")

	// Create flag set and parse with a flag value
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("image", "", "Container image name")

	err := flagSet.Parse([]string{"build", "--image=flag-image"})
	if err != nil {
		t.Fatalf("FlagSet.Parse() failed: %v", err)
	}

	cfg, initErr := initTestConfigWithFlags(t, tmpDir, flagSet, WithEnvPrefix("FLAGENV"))
	if initErr != nil {
		t.Fatalf("initialize() failed: %v", initErr)
	}

	// CLI flag should override environment variable
	if got := cfg.viper.GetString("image"); got != "flag-image" {
		t.Errorf("CLI flag should override env var, got %q, want %q", got, "flag-image")
	}
}

func TestNamedArgs_Precedence_EnvOverFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	writeTestConfig(t, tmpDir, `
image: file-image
tag: file-tag
`)

	// Set environment variable with unique prefix
	setTestEnv(t, "ENVFILE_IMAGE", "env-image")

	// Create flag set but don't provide flag values
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("image", "", "Container image name")
	flagSet.String("tag", "", "Image tag")

	err := flagSet.Parse([]string{"build"})
	if err != nil {
		t.Fatalf("FlagSet.Parse() failed: %v", err)
	}

	cfg, initErr := initTestConfigWithFlags(t, tmpDir, flagSet, WithEnvPrefix("ENVFILE"))
	if initErr != nil {
		t.Fatalf("initialize() failed: %v", initErr)
	}

	// Environment should override config file
	if got := cfg.viper.GetString("image"); got != "env-image" {
		t.Errorf("Env var should override file, got %q, want %q", got, "env-image")
	}

	// Tag should come from file (no env var set for tag)
	if got := cfg.viper.GetString("tag"); got != "file-tag" {
		t.Errorf("Config file value should be used, got %q, want %q", got, "file-tag")
	}
}

func TestNamedArgs_Precedence_FlagOverEnvOverFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	writeTestConfig(t, tmpDir, `
image: file-image
tag: file-tag
push: false
`)

	// Set environment variables with unique prefix
	setTestEnv(t, "ALLPREC_IMAGE", "env-image")
	setTestEnv(t, "ALLPREC_TAG", "env-tag")

	// Create flag set and provide only image flag
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("image", "", "Container image name")
	flagSet.String("tag", "", "Image tag")
	flagSet.Bool("push", false, "Push to registry")

	err := flagSet.Parse([]string{"build", "--image=flag-image"})
	if err != nil {
		t.Fatalf("FlagSet.Parse() failed: %v", err)
	}

	cfg, initErr := initTestConfigWithFlags(t, tmpDir, flagSet, WithEnvPrefix("ALLPREC"))
	if initErr != nil {
		t.Fatalf("initialize() failed: %v", initErr)
	}

	// Verify precedence: flag > env > file
	if got := cfg.viper.GetString("image"); got != "flag-image" {
		t.Errorf("image: flag should win, got %q, want %q", got, "flag-image")
	}

	if got := cfg.viper.GetString("tag"); got != "env-tag" {
		t.Errorf("tag: env should win, got %q, want %q", got, "env-tag")
	}

	if got := cfg.viper.GetBool("push"); got {
		t.Errorf("push: file should win, got %v, want %v", got, false)
	}
}

func TestNamedArgs_OrderIndependence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "flags_first_then_target",
			args: []string{"--image=webapp", "--tag=v1.0", "build"},
		},
		{
			name: "target_then_flags",
			args: []string{"build", "--image=webapp", "--tag=v1.0"},
		},
		{
			name: "interleaved",
			args: []string{"--image=webapp", "build", "--tag=v1.0"},
		},
		{
			name: "reversed_flag_order",
			args: []string{"--tag=v1.0", "--image=webapp", "build"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()

			flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flagSet.String("image", "", "Container image name")
			flagSet.String("tag", "", "Image tag")

			err := flagSet.Parse(tt.args)
			if err != nil {
				t.Fatalf("FlagSet.Parse() failed: %v", err)
			}

			cfg, initErr := initTestConfigWithFlags(t, tmpDir, flagSet)
			if initErr != nil {
				t.Fatalf("initialize() failed: %v", initErr)
			}

			// All orders should produce the same result
			if got := cfg.viper.GetString("image"); got != "webapp" {
				t.Errorf("GetString(\"image\") = %q, want %q", got, "webapp")
			}

			if got := cfg.viper.GetString("tag"); got != "v1.0" {
				t.Errorf("GetString(\"tag\") = %q, want %q", got, "v1.0")
			}
		})
	}
}

func TestNamedArgs_DefaultValues(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Define pflags with default values
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("image", "default-image", "Container image name")
	flagSet.String("tag", "latest", "Image tag")

	// Parse without providing any flags
	err := flagSet.Parse([]string{"build"})
	if err != nil {
		t.Fatalf("FlagSet.Parse() failed: %v", err)
	}

	cfg, initErr := initTestConfigWithFlags(t, tmpDir, flagSet)
	if initErr != nil {
		t.Fatalf("initialize() failed: %v", initErr)
	}

	// Should use pflag default values
	if got := cfg.viper.GetString("image"); got != "default-image" {
		t.Errorf("GetString(\"image\") = %q, want %q", got, "default-image")
	}

	if got := cfg.viper.GetString("tag"); got != "latest" {
		t.Errorf("GetString(\"tag\") = %q, want %q", got, "latest")
	}
}

func TestNamedArgs_BooleanFlags(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.Bool("push", false, "Push to registry")
	flagSet.Bool("verbose", false, "Verbose output")

	err := flagSet.Parse([]string{"build", "--push", "--verbose=false"})
	if err != nil {
		t.Fatalf("FlagSet.Parse() failed: %v", err)
	}

	cfg, initErr := initTestConfigWithFlags(t, tmpDir, flagSet)
	if initErr != nil {
		t.Fatalf("initialize() failed: %v", initErr)
	}

	// --push without value should be true
	if got := cfg.viper.GetBool("push"); !got {
		t.Errorf("GetBool(\"push\") = %v, want %v", got, true)
	}

	// --verbose=false should be false
	if got := cfg.viper.GetBool("verbose"); got {
		t.Errorf("GetBool(\"verbose\") = %v, want %v", got, false)
	}
}
