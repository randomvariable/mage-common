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
	"fmt"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected error
	}{
		{
			name:     "ErrAlreadyInitialized identity",
			err:      ErrAlreadyInitialized,
			expected: ErrAlreadyInitialized,
		},
		{
			name:     "ErrInvalidOption identity",
			err:      ErrInvalidOption,
			expected: ErrInvalidOption,
		},
		{
			name:     "ErrConfigParseFailed identity",
			err:      ErrConfigParseFailed,
			expected: ErrConfigParseFailed,
		},
		{
			name:     "Wrapped ErrAlreadyInitialized",
			err:      fmt.Errorf("initialization error: %w", ErrAlreadyInitialized),
			expected: ErrAlreadyInitialized,
		},
		{
			name:     "Wrapped ErrInvalidOption",
			err:      fmt.Errorf("option validation failed: %w", ErrInvalidOption),
			expected: ErrInvalidOption,
		},
		{
			name:     "Wrapped ErrConfigParseFailed",
			err:      fmt.Errorf("YAML parse error: %w", ErrConfigParseFailed),
			expected: ErrConfigParseFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if !errors.Is(tt.err, tt.expected) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tt.err, tt.expected)
			}
		})
	}
}

func TestSentinelErrorMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{
			name:    "ErrAlreadyInitialized message",
			err:     ErrAlreadyInitialized,
			wantMsg: "config: already initialized",
		},
		{
			name:    "ErrInvalidOption message",
			err:     ErrInvalidOption,
			wantMsg: "config: invalid option",
		},
		{
			name:    "ErrConfigParseFailed message",
			err:     ErrConfigParseFailed,
			wantMsg: "config: configuration file parse failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

func TestErrorDistinction(t *testing.T) {
	t.Parallel()

	// Verify errors are distinct from each other
	errs := []error{ErrAlreadyInitialized, ErrInvalidOption, ErrConfigParseFailed}

	for i, err1 := range errs {
		for j, err2 := range errs {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("Errors at index %d and %d are not distinct: %v == %v", i, j, err1, err2)
			}
		}
	}
}
