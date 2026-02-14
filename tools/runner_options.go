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

package tools

import "io"

// RunOption configures the behavior of a tool execution.
type RunOption func(*runConfig)

type runConfig struct {
	captureStdout  bool
	captureStderr  bool
	combinedOutput bool
	env            []string
	dir            string
	secrets        []string
	failOnNonZero  bool
	logger         io.Writer
}

func defaultRunConfig() *runConfig {
	return &runConfig{
		failOnNonZero: true,
	}
}

// WithStdout enables capturing stdout into RunResult.
func WithStdout() RunOption {
	return func(c *runConfig) {
		c.captureStdout = true
	}
}

// WithStderr enables capturing stderr into RunResult.
func WithStderr() RunOption {
	return func(c *runConfig) {
		c.captureStderr = true
	}
}

// WithCombinedOutput enables capturing interleaved stdout+stderr.
func WithCombinedOutput() RunOption {
	return func(c *runConfig) {
		c.combinedOutput = true
	}
}

// WithEnv adds environment variables in KEY=VALUE format.
func WithEnv(env ...string) RunOption {
	return func(c *runConfig) {
		c.env = append(c.env, env...)
	}
}

// WithDir sets the working directory for the command.
func WithDir(dir string) RunOption {
	return func(c *runConfig) {
		c.dir = dir
	}
}

// WithSecrets registers values to redact from the printed command line.
func WithSecrets(secrets ...string) RunOption {
	return func(c *runConfig) {
		c.secrets = append(c.secrets, secrets...)
	}
}

// WithoutFailOnNonZero suppresses the error on non-zero exit codes.
func WithoutFailOnNonZero() RunOption {
	return func(c *runConfig) {
		c.failOnNonZero = false
	}
}

// WithLogger overrides the output destination for command logging.
func WithLogger(w io.Writer) RunOption {
	return func(c *runConfig) {
		c.logger = w
	}
}
