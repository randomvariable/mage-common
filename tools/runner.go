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

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

// RunResult holds the outcome of a tool execution.
type RunResult struct {
	// ExitCode is the process exit code (0 = success).
	ExitCode int

	// Stdout contains captured stdout (if WithStdout() was used).
	Stdout []byte

	// Stderr contains captured stderr (if WithStderr() was used).
	Stderr []byte

	// Duration is the wall-clock execution time.
	Duration time.Duration

	combined []byte
}

// Combined returns interleaved stdout+stderr if WithCombinedOutput() was used.
func (r *RunResult) Combined() []byte {
	return r.combined
}

// Run ensures a tool is installed, then executes it by name with arguments.
// It loads the tool configuration from .tools.yaml, installs the tool if not
// already present (at most once per process), resolves the binary path
// (checking for tool-specific overrides such as a custom golangci-lint build),
// and runs the binary.
func Run(ctx context.Context, toolName string, args []string, opts ...RunOption) (*RunResult, error) {
	err := Ensure(ctx, toolName)
	if err != nil {
		return nil, err
	}

	config, err := LoadToolConfiguration(DefaultConfigFile)
	if err != nil {
		return nil, fmt.Errorf("loading tool configuration: %w", err)
	}

	toolsDir := ptr.Deref(config.Spec.ToolsDir, v1alpha1.DefaultToolsDir)
	binaryPath := resolveBinaryPath(toolsDir, toolName)

	return RunBinary(ctx, binaryPath, args, opts...)
}

// resolveBinaryPath returns the effective binary path for a tool, checking for
// tool-specific overrides (e.g. a custom golangci-lint binary from .custom-gcl.yml)
// before falling back to the standard platform tools directory.
func resolveBinaryPath(toolsDir, toolName string) string {
	if toolName == "golangci-lint" {
		return EffectiveLinterPath(toolsDir)
	}

	return ToolBinaryPath(toolsDir, toolName)
}

// RunBinary executes an arbitrary binary path with arguments.
// It prints the command line (with secrets redacted), streams output in real
// time, and returns the result with exit code, captured output, and duration.
// Use this when the binary is not in the standard tools directory layout
// (e.g. a custom golangci-lint build).
func RunBinary(ctx context.Context, binaryPath string, args []string, opts ...RunOption) (*RunResult, error) {
	cfg := defaultRunConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Print the command line with secrets redacted.
	logWriter := cfg.logger
	if logWriter == nil {
		logWriter = os.Stderr
	}

	cmdLine := redactSecrets(formatVerboseCommand(binaryPath, args, cfg.dir, cfg.env), cfg.secrets)
	_, _ = fmt.Fprintf(logWriter, "$ %s\n", cmdLine)

	cmd := exec.CommandContext(ctx, binaryPath, args...)

	cmd.Env = append(os.Environ(), cfg.env...)

	if cfg.dir != "" {
		cmd.Dir = cfg.dir
	}

	result := &RunResult{}

	// Set up output capture and streaming.
	// By default, stdout/stderr pass through to os.Stdout/os.Stderr directly
	// so tools can detect a TTY and produce colorised output. Capture buffers
	// are only tee'd in via io.MultiWriter when explicitly requested.
	var stdoutBuf, stderrBuf, combinedBuf bytes.Buffer

	// The combined buffer is written by both the stdout and stderr pipe
	// goroutines inside exec.Cmd, so it must be synchronised.
	var combinedWriter io.Writer
	if cfg.combinedOutput {
		combinedWriter = &syncWriter{w: &combinedBuf}
	}

	cmd.Stdout = buildOutputWriter(os.Stdout, cfg.captureStdout, &stdoutBuf, combinedWriter)
	cmd.Stderr = buildOutputWriter(os.Stderr, cfg.captureStderr, &stderrBuf, combinedWriter)

	start := time.Now()
	err := cmd.Run()
	result.Duration = time.Since(start)

	if cfg.captureStdout {
		result.Stdout = stdoutBuf.Bytes()
	}

	if cfg.captureStderr {
		result.Stderr = stderrBuf.Bytes()
	}

	if cfg.combinedOutput {
		result.combined = combinedBuf.Bytes()
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()

			if cfg.failOnNonZero {
				return result, fmt.Errorf("%s exited with code %d: %w", binaryPath, result.ExitCode, err)
			}

			return result, nil
		}

		return result, fmt.Errorf("running %s: %w", binaryPath, err)
	}

	return result, nil
}

// formatVerboseCommand builds a copy-pasteable shell command line showing the
// working directory (when changed), added environment variables, binary path,
// and arguments.
func formatVerboseCommand(binaryPath string, args []string, dir string, env []string) string {
	parts := make([]string, 0, len(env)+len(args)+2) //nolint:mnd // binary + optional cd prefix + env + args

	// Working directory — only shown when explicitly changed via WithDir.
	if dir != "" {
		abs, absErr := filepath.Abs(dir)
		if absErr == nil {
			dir = abs
		}

		parts = append(parts, "cd "+dir+" &&")
	}

	// Added environment variables as inline shell prefix.
	parts = append(parts, env...)

	// Prefer a relative path from cwd when the binary is inside the project
	// tree (no leading ".."). Fall back to the absolute path otherwise.
	parts = append(parts, resolveDisplayPath(binaryPath))

	// Arguments.
	parts = append(parts, args...)

	return strings.Join(parts, " ")
}

// resolveDisplayPath returns a human-friendly path for the binary.
// Bare command names (e.g. "go", "cargo") are returned as-is since they are
// resolved via PATH by exec.Command. If the binary is inside the current
// working directory tree, it returns the relative path (e.g.
// "./hack/bin/linux/amd64/tool"). If the relative path escapes the tree
// (starts with ".."), it returns the absolute path.
func resolveDisplayPath(binaryPath string) string {
	// Bare command names have no directory component — display as-is.
	if !strings.Contains(binaryPath, string(filepath.Separator)) {
		return binaryPath
	}

	abs, err := filepath.Abs(binaryPath)
	if err != nil {
		return binaryPath
	}

	cwd, err := os.Getwd()
	if err != nil {
		return abs
	}

	rel, err := filepath.Rel(cwd, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return abs
	}

	return "./" + rel
}

func redactSecrets(cmdLine string, secrets []string) string {
	for _, secret := range secrets {
		if secret != "" {
			cmdLine = strings.ReplaceAll(cmdLine, secret, "***")
		}
	}

	return cmdLine
}

// buildOutputWriter returns an io.Writer for a command's stdout or stderr.
// When no capture is requested it returns stream unchanged, preserving TTY
// attributes so that child processes can detect a terminal and emit colour.
// When capture or combined buffers are needed, it tees through io.MultiWriter.
func buildOutputWriter(stream io.Writer, capture bool, captureBuf *bytes.Buffer, combinedWriter io.Writer) io.Writer {
	if !capture && combinedWriter == nil {
		return stream
	}

	writers := []io.Writer{stream}

	if capture {
		writers = append(writers, captureBuf)
	}

	if combinedWriter != nil {
		writers = append(writers, combinedWriter)
	}

	return io.MultiWriter(writers...)
}

// syncWriter serialises concurrent writes via a mutex.
// Used to protect the shared combined-output buffer that receives writes from
// both the stdout and stderr pipe goroutines inside exec.Cmd.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (sw *syncWriter) Write(p []byte) (int, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	return sw.w.Write(p) //nolint:wrapcheck // implementing io.Writer; wrapping would break MultiWriter semantics
}
