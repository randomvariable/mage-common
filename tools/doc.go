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

// Package tools provides declarative tool dependency management for Mage projects.
//
// It reads a .tools.yaml configuration file and installs, updates, and runs
// development tools (Go, gem, npx, cargo, uvx, and direct download) with
// platform-specific binary placement and version management.
//
// # Ensure and run a tool (install if needed, then execute)
//
//	result, err := tools.Run(ctx, "golangci-lint", []string{"run", "./..."})
//
// # Ensure all configured tools are installed
//
//	err := tools.EnsureAll(ctx)
//
// # Use as a Mage parameterized dependency
//
//	mg.CtxDeps(ctx, mg.F(tools.Ensure, "golangci-lint"))
//
// # Generate a cache key for CI caching
//
//	key, err := tools.CacheKey(".tools.yaml")
//
// # Update all tools to latest versions
//
//	err := tools.UpdateAll(".tools.yaml")
package tools
