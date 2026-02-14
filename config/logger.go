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

import "context"

// Level represents a logging level.
type Level int

const (
	// LevelDebug is for detailed debugging information.
	LevelDebug Level = iota
	// LevelInfo is for informational messages.
	LevelInfo
	// LevelWarn is for warning messages.
	LevelWarn
	// LevelError is for error messages.
	LevelError
)

// String returns the string representation of the log level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger is the minimal logging contract for adapter compatibility.
// This interface allows integration with popular Go logging libraries
// (slog, zerolog, zap) via adapters.
//
// Contract:
//   - attrs must contain an even number of elements (key-value pairs)
//   - ctx enables trace propagation for observability systems
//   - Implementations must handle nil ctx gracefully
//   - Implementations must handle odd-length attrs by appending a default value
type Logger interface {
	// Log emits a log message at the specified level with structured attributes.
	// ctx enables context propagation for distributed tracing.
	// attrs are key-value pairs (e.g., "user", "alice", "action", "login").
	Log(ctx context.Context, level Level, msg string, attrs ...any)
}
