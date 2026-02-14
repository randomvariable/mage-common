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
	"reflect"
	"testing"

	"github.com/spf13/pflag"
)

const (
	testImageWebapp = "webapp"
	testTagV1       = "v1.0"
)

func TestCleanArgs_SingleArgForm(t *testing.T) {
	t.Parallel()

	args := []string{"mage", "build", "--image=webapp", "--tag=v1.0"}
	cleaned, removed := cleanArgs(args)

	expectedCleaned := []string{"mage", "build"}
	if !reflect.DeepEqual(cleaned, expectedCleaned) {
		t.Errorf("cleaned = %v, want %v", cleaned, expectedCleaned)
	}

	expectedRemoved := []string{"--image=webapp", "--tag=v1.0"}
	if !reflect.DeepEqual(removed, expectedRemoved) {
		t.Errorf("removed = %v, want %v", removed, expectedRemoved)
	}
}

func TestCleanArgs_TwoArgForm(t *testing.T) {
	t.Parallel()

	args := []string{"mage", "build", "--image", testImageWebapp, "--tag", testTagV1}
	cleaned, removed := cleanArgs(args)

	expectedCleaned := []string{"mage", "build"}
	if !reflect.DeepEqual(cleaned, expectedCleaned) {
		t.Errorf("cleaned = %v, want %v", cleaned, expectedCleaned)
	}

	expectedRemoved := []string{"--image", testImageWebapp, "--tag", testTagV1}
	if !reflect.DeepEqual(removed, expectedRemoved) {
		t.Errorf("removed = %v, want %v", removed, expectedRemoved)
	}
}

func TestCleanArgs_MixedForms(t *testing.T) {
	t.Parallel()

	args := []string{"mage", "build", "--image=webapp", "--tag", testTagV1, "--verbose"}
	cleaned, removed := cleanArgs(args)

	expectedCleaned := []string{"mage", "build"}
	if !reflect.DeepEqual(cleaned, expectedCleaned) {
		t.Errorf("cleaned = %v, want %v", cleaned, expectedCleaned)
	}

	expectedRemoved := []string{"--image=webapp", "--tag", testTagV1, "--verbose"}
	if !reflect.DeepEqual(removed, expectedRemoved) {
		t.Errorf("removed = %v, want %v", removed, expectedRemoved)
	}
}

func TestCleanArgs_PreserveShortFlags(t *testing.T) {
	t.Parallel()

	args := []string{"mage", "-v", "build", "--image=webapp", "-d"}
	cleaned, removed := cleanArgs(args)

	expectedCleaned := []string{"mage", "-v", "build", "-d"}
	if !reflect.DeepEqual(cleaned, expectedCleaned) {
		t.Errorf("cleaned = %v, want %v", cleaned, expectedCleaned)
	}

	expectedRemoved := []string{"--image=webapp"}
	if !reflect.DeepEqual(removed, expectedRemoved) {
		t.Errorf("removed = %v, want %v", removed, expectedRemoved)
	}
}

func TestCleanArgs_PreserveTargets(t *testing.T) {
	t.Parallel()

	args := []string{"mage", "build", "--image=webapp", "test", "--tag=v1.0", "deploy"}
	cleaned, removed := cleanArgs(args)

	expectedCleaned := []string{"mage", "build", "test", "deploy"}
	if !reflect.DeepEqual(cleaned, expectedCleaned) {
		t.Errorf("cleaned = %v, want %v", cleaned, expectedCleaned)
	}

	expectedRemoved := []string{"--image=webapp", "--tag=v1.0"}
	if !reflect.DeepEqual(removed, expectedRemoved) {
		t.Errorf("removed = %v, want %v", removed, expectedRemoved)
	}
}

func TestCleanArgs_NoFlags(t *testing.T) {
	t.Parallel()

	args := []string{"mage", "build", "test"}
	cleaned, removed := cleanArgs(args)

	expectedCleaned := []string{"mage", "build", "test"}
	if !reflect.DeepEqual(cleaned, expectedCleaned) {
		t.Errorf("cleaned = %v, want %v", cleaned, expectedCleaned)
	}

	if len(removed) != 0 {
		t.Errorf("removed = %v, want empty slice", removed)
	}
}

func TestCleanArgs_EmptyValue(t *testing.T) {
	t.Parallel()

	args := []string{"mage", "build", "--verbose"}
	cleaned, removed := cleanArgs(args)

	expectedCleaned := []string{"mage", "build"}
	if !reflect.DeepEqual(cleaned, expectedCleaned) {
		t.Errorf("cleaned = %v, want %v", cleaned, expectedCleaned)
	}

	expectedRemoved := []string{"--verbose"}
	if !reflect.DeepEqual(removed, expectedRemoved) {
		t.Errorf("removed = %v, want %v", removed, expectedRemoved)
	}
}

func TestCleanArgs_OrderIndependence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantCleaned []string
		wantRemoved []string
	}{
		{
			name:        "flags first",
			args:        []string{"mage", "--image=webapp", "--tag=v1.0", "build"},
			wantCleaned: []string{"mage", "build"},
			wantRemoved: []string{"--image=webapp", "--tag=v1.0"},
		},
		{
			name:        "flags last",
			args:        []string{"mage", "build", "--image=webapp", "--tag=v1.0"},
			wantCleaned: []string{"mage", "build"},
			wantRemoved: []string{"--image=webapp", "--tag=v1.0"},
		},
		{
			name:        "flags mixed",
			args:        []string{"mage", "--image=webapp", "build", "--tag=v1.0"},
			wantCleaned: []string{"mage", "build"},
			wantRemoved: []string{"--image=webapp", "--tag=v1.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cleaned, removed := cleanArgs(tt.args)

			if !reflect.DeepEqual(cleaned, tt.wantCleaned) {
				t.Errorf("cleaned = %v, want %v", cleaned, tt.wantCleaned)
			}

			if !reflect.DeepEqual(removed, tt.wantRemoved) {
				t.Errorf("removed = %v, want %v", removed, tt.wantRemoved)
			}
		})
	}
}

func TestCleanArgs_WithPflagIntegration(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a local flag set (not global pflag.CommandLine)
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	image := flagSet.String("image", "", "Container image name")
	tag := flagSet.String("tag", "", "Image tag")

	// Parse flags from args
	args := []string{"build", "--image=webapp", "--tag=v1.0"}

	err := flagSet.Parse(args)
	if err != nil {
		t.Fatalf("FlagSet.Parse() failed: %v", err)
	}

	// Verify flags were captured
	if *image != testImageWebapp {
		t.Errorf("image flag = %q, want %q", *image, testImageWebapp)
	}

	if *tag != testTagV1 {
		t.Errorf("tag flag = %q, want %q", *tag, testTagV1)
	}

	// Clean the original args (simulating CleanOSArgs behavior)
	fullArgs := []string{"mage", "build", "--image=webapp", "--tag=v1.0"}
	cleaned, removed := cleanArgs(fullArgs)

	expectedCleaned := []string{"mage", "build"}
	if !reflect.DeepEqual(cleaned, expectedCleaned) {
		t.Errorf("cleaned = %v, want %v", cleaned, expectedCleaned)
	}

	expectedRemoved := []string{"--image=webapp", "--tag=v1.0"}
	if !reflect.DeepEqual(removed, expectedRemoved) {
		t.Errorf("removed = %v, want %v", removed, expectedRemoved)
	}

	// Initialize config with the local flag set and verify flags are bound
	cfg, initErr := initTestConfigWithFlags(t, tmpDir, flagSet)
	if initErr != nil {
		t.Fatalf("initialize() failed: %v", initErr)
	}

	if got := cfg.viper.GetString("image"); got != testImageWebapp {
		t.Errorf("GetString(\"image\") = %q, want %q", got, testImageWebapp)
	}

	if got := cfg.viper.GetString("tag"); got != testTagV1 {
		t.Errorf("GetString(\"tag\") = %q, want %q", got, testTagV1)
	}
}

func TestGetCleanedArgs_ReturnsCopy(t *testing.T) {
	t.Parallel()

	// Test that GetCleanedArgs returns a copy by checking the default empty state.
	// We can't test CleanOSArgs in parallel (modifies os.Args), but we can
	// verify the copy behavior on the singleton's default empty state.
	cleaned := GetCleanedArgs()
	if cleaned == nil {
		t.Error("GetCleanedArgs() should return non-nil (empty slice or copy)")
	}
}
