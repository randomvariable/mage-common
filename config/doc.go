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

// Package config provides standardized Viper-based configuration loading
// for Mage projects with support for graceful config.yaml loading,
// environment variable overrides, and named command-line arguments.
//
// # Logger Integration
//
// Provide structured logging for configuration events:
//
//	import (
//	    "log/slog"
//	    "github.com/randomvariable/mage-common/config"
//	)
//
//	func main() {
//	    logger := slog.Default()
//	    err := config.Init(
//	        config.WithLogger(config.NewSlogAdapter(logger)),
//	        config.WithSensitiveKeys("api.token", "db.password"),
//	    )
//	    if err != nil {
//	        panic(err)
//	    }
//	}
//
// # Configuration Precedence
//
// Values are loaded in the following order (highest to lowest priority):
//  1. Command-line flags (via pflag binding)
//  2. Environment variables (automatic with optional prefix)
//  3. Configuration file (config.yaml by default)
//  4. Default values
//
// # Thread Safety
//
// Init() uses sync.Once internally to ensure single initialization.
// After Init() completes, all Viper reads are safe for concurrent access.
// Do not call viper.Set() after Init() - configuration is read-only.
//
// # Import Compatibility
//
// This package follows import-safe design:
//   - No init() side effects
//   - Explicit Init() call required
//   - No hardcoded file paths
//   - Works in any project structure
package config
