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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/onsi/gomega"
)

func setupTestBinary(t *testing.T, name, script string) string {
	t.Helper()

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, name)

	err := os.WriteFile(binaryPath, []byte(script), 0o755)
	if err != nil {
		t.Fatalf("write tool: %v", err)
	}

	return binaryPath
}

// runBinaryRetry wraps RunBinary with retry logic for transient "text file busy"
// errors that occur on Linux when executing a recently-written file.
func runBinaryRetry(ctx context.Context, t *testing.T, binaryPath string, args []string, opts ...RunOption) (*RunResult, error) {
	t.Helper()

	g := gomega.NewWithT(t)

	var (
		result *RunResult
		runErr error
	)

	g.Eventually(func() bool {
		result, runErr = RunBinary(ctx, binaryPath, args, opts...)
		if runErr != nil && strings.Contains(runErr.Error(), "text file busy") {
			return false
		}

		return true
	}, "5s", "100ms").Should(gomega.BeTrue(), "timed out waiting for binary to become available")

	return result, runErr
}

func TestRunBinarySuccess(t *testing.T) {
	t.Parallel()

	binaryPath := setupTestBinary(t, "echo-tool", "#!/bin/sh\necho hello world\n")

	var logBuf bytes.Buffer

	result, err := runBinaryRetry(context.Background(), t, binaryPath, nil,
		WithStdout(),
		WithLogger(&logBuf),
	)
	if err != nil {
		t.Fatalf("RunBinary() error = %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	if !strings.Contains(string(result.Stdout), "hello world") {
		t.Errorf("Stdout = %q, want to contain 'hello world'", result.Stdout)
	}

	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}

	// Check command line was logged.
	if !strings.Contains(logBuf.String(), "echo-tool") {
		t.Errorf("log output should contain tool name, got %q", logBuf.String())
	}
}

func TestRunBinarySecretsRedaction(t *testing.T) {
	t.Parallel()

	binaryPath := setupTestBinary(t, "secret-tool", "#!/bin/sh\necho ok\n")

	var logBuf bytes.Buffer

	_, err := runBinaryRetry(context.Background(), t, binaryPath, []string{"--token", "mysecretvalue"},
		WithLogger(&logBuf),
		WithSecrets("mysecretvalue"),
	)
	if err != nil {
		t.Fatalf("RunBinary() error = %v", err)
	}

	logOutput := logBuf.String()
	if strings.Contains(logOutput, "mysecretvalue") {
		t.Errorf("log output contains secret: %q", logOutput)
	}

	if !strings.Contains(logOutput, "***") {
		t.Errorf("log output should contain redacted placeholder '***', got %q", logOutput)
	}
}

func TestRunBinaryNonZeroExit(t *testing.T) {
	t.Parallel()

	binaryPath := setupTestBinary(t, "fail-tool", "#!/bin/sh\nexit 42\n")

	var logBuf bytes.Buffer

	result, err := runBinaryRetry(context.Background(), t, binaryPath, nil,
		WithLogger(&logBuf),
	)
	if err == nil {
		t.Fatal("RunBinary() error = nil, want non-zero exit error")
	}

	if result.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", result.ExitCode)
	}
}

func TestRunBinaryWithoutFailOnNonZero(t *testing.T) {
	t.Parallel()

	binaryPath := setupTestBinary(t, "soft-fail", "#!/bin/sh\nexit 1\n")

	var logBuf bytes.Buffer

	result, err := runBinaryRetry(context.Background(), t, binaryPath, nil,
		WithLogger(&logBuf),
		WithoutFailOnNonZero(),
	)
	if err != nil {
		t.Fatalf("RunBinary() error = %v, want nil with WithoutFailOnNonZero", err)
	}

	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
}

func TestRunBinaryStderrCapture(t *testing.T) {
	t.Parallel()

	binaryPath := setupTestBinary(t, "stderr-tool", "#!/bin/sh\necho error-output >&2\n")

	var logBuf bytes.Buffer

	result, err := runBinaryRetry(context.Background(), t, binaryPath, nil,
		WithStderr(),
		WithLogger(&logBuf),
	)
	if err != nil {
		t.Fatalf("RunBinary() error = %v", err)
	}

	if !strings.Contains(string(result.Stderr), "error-output") {
		t.Errorf("Stderr = %q, want to contain 'error-output'", result.Stderr)
	}
}

func TestRunBinaryCombinedOutput(t *testing.T) {
	t.Parallel()

	binaryPath := setupTestBinary(t, "combined-tool", "#!/bin/sh\necho stdout-line\necho stderr-line >&2\n")

	var logBuf bytes.Buffer

	result, err := runBinaryRetry(context.Background(), t, binaryPath, nil,
		WithCombinedOutput(),
		WithLogger(&logBuf),
	)
	if err != nil {
		t.Fatalf("RunBinary() error = %v", err)
	}

	combined := string(result.Combined())
	if !strings.Contains(combined, "stdout-line") || !strings.Contains(combined, "stderr-line") {
		t.Errorf("Combined() = %q, want both stdout and stderr", combined)
	}
}

func TestRunBinaryWithEnvAndDir(t *testing.T) {
	t.Parallel()

	binaryPath := setupTestBinary(t, "env-tool", "#!/bin/sh\necho \"MY_VAR=$MY_VAR\"\npwd\n")

	tmpWorkDir := t.TempDir()

	var logBuf bytes.Buffer

	result, err := runBinaryRetry(context.Background(), t, binaryPath, nil,
		WithStdout(),
		WithLogger(&logBuf),
		WithEnv("MY_VAR=test_value"),
		WithDir(tmpWorkDir),
	)
	if err != nil {
		t.Fatalf("RunBinary() error = %v", err)
	}

	output := string(result.Stdout)
	if !strings.Contains(output, "MY_VAR=test_value") {
		t.Errorf("output should contain env var, got %q", output)
	}

	if !strings.Contains(output, tmpWorkDir) {
		t.Errorf("output should contain working dir %q, got %q", tmpWorkDir, output)
	}
}

//nolint:paralleltest // changes working directory
func TestRunResolvesToolBinaryPath(t *testing.T) {
	ResetEnsureTracker()

	// Build a temp dir with a relative tools layout and .tools.yaml.
	tmpDir := t.TempDir()

	relToolsDir := "hack/bin"
	platformDir := filepath.Join(tmpDir, relToolsDir, runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(platformDir, 0o750)
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create versioned binary and symlink (matching the version scheme used by GoInstaller).
	versionedName := "my-tool-v1.0.0"

	err = os.WriteFile(filepath.Join(platformDir, versionedName), []byte("#!/bin/sh\necho resolved\n"), 0o755)
	if err != nil {
		t.Fatalf("write tool: %v", err)
	}

	err = os.Symlink(versionedName, filepath.Join(platformDir, "my-tool"))
	if err != nil {
		t.Fatalf("symlink: %v", err)
	}

	configContent := "apiVersion: mage-common.randomvariable.co.uk/v1alpha1\nkind: ToolConfiguration\nspec:\n  toolsDir: " + relToolsDir + "\n  tools:\n    - name: my-tool\n      version: v1.0.0\n      sources:\n        - type: go\n          url: example.com/my-tool\n"

	err = os.WriteFile(filepath.Join(tmpDir, DefaultConfigFile), []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Change to the temp dir so Run finds the config and relative toolsDir.
	t.Chdir(tmpDir)

	g := gomega.NewWithT(t)

	var result *RunResult

	// Use Eventually to handle transient "text file busy" on the symlinked binary.
	g.Eventually(func() bool {
		// Reset tracker so a cached "text file busy" error doesn't stick across retries.
		ResetEnsureTracker()

		var runErr error

		result, runErr = Run(context.Background(), "my-tool", nil,
			WithStdout(),
			WithLogger(&bytes.Buffer{}),
		)
		if runErr != nil && strings.Contains(runErr.Error(), "text file busy") {
			return false
		}

		if runErr != nil {
			t.Fatalf("Run() error = %v", runErr)
		}

		return true
	}, "5s", "100ms").Should(gomega.BeTrue())

	if !strings.Contains(string(result.Stdout), "resolved") {
		t.Errorf("Stdout = %q, want to contain 'resolved'", result.Stdout)
	}
}

func TestRunBinaryVerboseLog(t *testing.T) {
	t.Parallel()

	binaryPath := setupTestBinary(t, "verbose-tool", "#!/bin/sh\necho ok\n")

	workDir := t.TempDir()

	var logBuf bytes.Buffer

	_, err := runBinaryRetry(context.Background(), t, binaryPath, []string{"--flag", "value"},
		WithLogger(&logBuf),
		WithEnv("EXTRA_VAR=hello"),
		WithDir(workDir),
	)
	if err != nil {
		t.Fatalf("RunBinary() error = %v", err)
	}

	logOutput := logBuf.String()

	// Should contain the working directory in copy-pasteable format.
	if !strings.Contains(logOutput, "cd "+workDir) {
		t.Errorf("log should contain working directory, got %q", logOutput)
	}

	// Should contain the added env var.
	if !strings.Contains(logOutput, "EXTRA_VAR=hello") {
		t.Errorf("log should contain added env var, got %q", logOutput)
	}

	// Should contain the full absolute path to the binary.
	absPath, _ := filepath.Abs(binaryPath)
	if !strings.Contains(logOutput, absPath) {
		t.Errorf("log should contain absolute binary path %q, got %q", absPath, logOutput)
	}

	// Should contain the arguments.
	if !strings.Contains(logOutput, "--flag value") {
		t.Errorf("log should contain arguments, got %q", logOutput)
	}
}

func TestRunBinaryEnvVarSecretRedaction(t *testing.T) {
	t.Parallel()

	binaryPath := setupTestBinary(t, "env-secret-tool", "#!/bin/sh\necho ok\n")

	var logBuf bytes.Buffer

	_, err := runBinaryRetry(context.Background(), t, binaryPath, nil,
		WithLogger(&logBuf),
		WithEnv("API_KEY=supersecret"),
		WithSecrets("supersecret"),
	)
	if err != nil {
		t.Fatalf("RunBinary() error = %v", err)
	}

	logOutput := logBuf.String()

	if strings.Contains(logOutput, "supersecret") {
		t.Errorf("log should not contain secret value, got %q", logOutput)
	}

	if !strings.Contains(logOutput, "API_KEY=***") {
		t.Errorf("log should contain redacted env var, got %q", logOutput)
	}
}
