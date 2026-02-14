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
	"context"
	"sync"
	"testing"
)

// mockLogger is a test double that records all log calls for verification.
// Thread-safe for use in parallel tests.
type mockLogger struct {
	mu     sync.Mutex
	events []logEvent
}

// logEvent captures a single log call for test assertions.
type logEvent struct {
	level Level
	msg   string
	attrs []any
}

// Log implements the Logger interface.
func (ml *mockLogger) Log(_ context.Context, level Level, msg string, attrs ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	ml.events = append(ml.events, logEvent{
		level: level,
		msg:   msg,
		attrs: append([]any(nil), attrs...), // Copy attrs to prevent mutation
	})
}

// Events returns a copy of logged events for safe inspection.
func (ml *mockLogger) Events() []logEvent {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	events := make([]logEvent, len(ml.events))
	copy(events, ml.events)

	return events
}

// Reset clears all logged events.
func (ml *mockLogger) Reset() {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	ml.events = nil
}

// Count returns the number of logged events.
func (ml *mockLogger) Count() int {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	return len(ml.events)
}

func TestMockLogger(t *testing.T) {
	t.Parallel()

	mock := &mockLogger{}

	// Test logging
	ctx := context.Background()
	mock.Log(ctx, LevelInfo, "test message", "key1", "value1", "key2", 42)

	if got := mock.Count(); got != 1 {
		t.Errorf("Count() = %d, want 1", got)
	}

	events := mock.Events()
	if len(events) != 1 {
		t.Fatalf("Events() length = %d, want 1", len(events))
	}

	event := events[0]
	if event.level != LevelInfo {
		t.Errorf("level = %v, want %v", event.level, LevelInfo)
	}

	if event.msg != "test message" {
		t.Errorf("msg = %q, want %q", event.msg, "test message")
	}

	if len(event.attrs) != 4 {
		t.Errorf("attrs length = %d, want 4", len(event.attrs))
	}

	// Test reset
	mock.Reset()

	if got := mock.Count(); got != 0 {
		t.Errorf("Count() after Reset() = %d, want 0", got)
	}
}

func TestLoggerInterface(t *testing.T) {
	t.Parallel()

	var _ Logger = (*mockLogger)(nil) // Compile-time interface check
}

func TestLevelString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
