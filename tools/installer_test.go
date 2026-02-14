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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"k8s.io/utils/ptr"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
)

// fakeInstaller is a test double implementing TypeInstaller with configurable behavior.
type fakeInstaller struct {
	installFunc          func(ctx context.Context, tool v1alpha1.Tool, source *v1alpha1.ToolSource, toolsDir string) error
	isInstalledFunc      func(tool v1alpha1.Tool, toolsDir string) (bool, error)
	runtimeAvailableFunc func() error
	installCount         atomic.Int32
}

func (f *fakeInstaller) Install(ctx context.Context, tool v1alpha1.Tool, source *v1alpha1.ToolSource, toolsDir string) error {
	f.installCount.Add(1)

	if f.installFunc != nil {
		return f.installFunc(ctx, tool, source, toolsDir)
	}

	return nil
}

func (f *fakeInstaller) IsInstalled(tool v1alpha1.Tool, toolsDir string) (bool, error) {
	if f.isInstalledFunc != nil {
		return f.isInstalledFunc(tool, toolsDir)
	}

	return false, nil
}

func (f *fakeInstaller) RuntimeAvailable() error {
	if f.runtimeAvailableFunc != nil {
		return f.runtimeAvailableFunc()
	}

	return nil
}

func newTestToolsDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	platformDir := filepath.Join(dir, runtime.GOOS, runtime.GOARCH)

	err := os.MkdirAll(platformDir, 0o750)
	if err != nil {
		t.Fatalf("creating test tools dir: %v", err)
	}

	return dir
}

func TestInstallerInstallAllMixedTypes(t *testing.T) {
	t.Parallel()

	toolsDir := newTestToolsDir(t)

	goInstaller := &fakeInstaller{}
	npxInstaller := &fakeInstaller{}

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			ToolsDir: ptr.To(toolsDir),
			Tools: []v1alpha1.Tool{
				{
					Name:    "controller-gen",
					Version: ptr.To("v0.17.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("sigs.k8s.io/controller-tools/cmd/controller-gen")},
					},
				},
				{
					Name:    "commitlint",
					Version: ptr.To("19.6.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeNpx, Package: ptr.To("@commitlint/cli")},
					},
				},
			},
		},
	}

	inst := &Installer{
		Config: config,
		Installers: map[v1alpha1.SourceType]TypeInstaller{
			v1alpha1.SourceTypeGo:  goInstaller,
			v1alpha1.SourceTypeNpx: npxInstaller,
		},
	}

	var logBuf bytes.Buffer

	err := inst.InstallAll(context.Background(), WithInstallLogger(&logBuf))
	if err != nil {
		t.Fatalf("InstallAll() error = %v", err)
	}

	if goInstaller.installCount.Load() != 1 {
		t.Errorf("go installer called %d times, want 1", goInstaller.installCount.Load())
	}

	if npxInstaller.installCount.Load() != 1 {
		t.Errorf("npx installer called %d times, want 1", npxInstaller.installCount.Load())
	}
}

func TestInstallerInstallByNameFound(t *testing.T) {
	t.Parallel()

	toolsDir := newTestToolsDir(t)
	goInstaller := &fakeInstaller{}

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			ToolsDir: ptr.To(toolsDir),
			Tools: []v1alpha1.Tool{
				{
					Name:    "controller-gen",
					Version: ptr.To("v0.17.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("sigs.k8s.io/controller-tools/cmd/controller-gen")},
					},
				},
				{
					Name:    "goimports",
					Version: ptr.To("v0.30.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("golang.org/x/tools/cmd/goimports")},
					},
				},
			},
		},
	}

	inst := &Installer{
		Config: config,
		Installers: map[v1alpha1.SourceType]TypeInstaller{
			v1alpha1.SourceTypeGo: goInstaller,
		},
	}

	var logBuf bytes.Buffer

	err := inst.InstallByName(context.Background(), "goimports", WithInstallLogger(&logBuf))
	if err != nil {
		t.Fatalf("InstallByName() error = %v", err)
	}

	// Only goimports should have been installed, not controller-gen.
	if goInstaller.installCount.Load() != 1 {
		t.Errorf("go installer called %d times, want 1", goInstaller.installCount.Load())
	}
}

func TestInstallerInstallByNameNotFound(t *testing.T) {
	t.Parallel()

	toolsDir := newTestToolsDir(t)

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			ToolsDir: ptr.To(toolsDir),
			Tools: []v1alpha1.Tool{
				{
					Name:    "controller-gen",
					Version: ptr.To("v0.17.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("sigs.k8s.io/controller-tools/cmd/controller-gen")},
					},
				},
			},
		},
	}

	inst := &Installer{
		Config:     config,
		Installers: map[v1alpha1.SourceType]TypeInstaller{},
	}

	var logBuf bytes.Buffer

	err := inst.InstallByName(context.Background(), "nonexistent-tool", WithInstallLogger(&logBuf))
	if err == nil {
		t.Fatal("InstallByName() error = nil, want ErrToolNotFound")
	}

	if !errors.Is(err, ErrToolNotFound) {
		t.Errorf("error = %v, want ErrToolNotFound", err)
	}
}

func TestInstallerMultiSourceFallback(t *testing.T) {
	t.Parallel()

	toolsDir := newTestToolsDir(t)

	failInstaller := &fakeInstaller{
		installFunc: func(_ context.Context, _ v1alpha1.Tool, _ *v1alpha1.ToolSource, _ string) error {
			return errors.New("simulated download failure")
		},
	}
	successInstaller := &fakeInstaller{}

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			ToolsDir: ptr.To(toolsDir),
			Tools: []v1alpha1.Tool{
				{
					Name:    "mytool",
					Version: ptr.To("v1.0.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeDownload, URL: ptr.To("https://example.com/mytool")},
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("example.com/cmd/mytool")},
					},
				},
			},
		},
	}

	inst := &Installer{
		Config: config,
		Installers: map[v1alpha1.SourceType]TypeInstaller{
			v1alpha1.SourceTypeDownload: failInstaller,
			v1alpha1.SourceTypeGo:       successInstaller,
		},
	}

	var logBuf bytes.Buffer

	err := inst.InstallByName(context.Background(), "mytool", WithInstallLogger(&logBuf))
	if err != nil {
		t.Fatalf("InstallByName() error = %v; expected fallback to succeed", err)
	}

	if failInstaller.installCount.Load() != 1 {
		t.Errorf("fail installer called %d times, want 1", failInstaller.installCount.Load())
	}

	if successInstaller.installCount.Load() != 1 {
		t.Errorf("success installer called %d times, want 1", successInstaller.installCount.Load())
	}

	// Verify the first failure was logged.
	if !strings.Contains(logBuf.String(), "failed") {
		t.Errorf("log should contain failure from first source, got %q", logBuf.String())
	}
}

func TestInstallerAllSourcesFailed(t *testing.T) {
	t.Parallel()

	toolsDir := newTestToolsDir(t)

	failInstaller := &fakeInstaller{
		installFunc: func(_ context.Context, _ v1alpha1.Tool, _ *v1alpha1.ToolSource, _ string) error {
			return errors.New("simulated failure")
		},
	}

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			ToolsDir: ptr.To(toolsDir),
			Tools: []v1alpha1.Tool{
				{
					Name:    "doomed-tool",
					Version: ptr.To("v1.0.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeDownload, URL: ptr.To("https://example.com/fail1")},
						{Type: v1alpha1.SourceTypeDownload, URL: ptr.To("https://example.com/fail2")},
					},
				},
			},
		},
	}

	inst := &Installer{
		Config: config,
		Installers: map[v1alpha1.SourceType]TypeInstaller{
			v1alpha1.SourceTypeDownload: failInstaller,
		},
	}

	var logBuf bytes.Buffer

	err := inst.InstallByName(context.Background(), "doomed-tool", WithInstallLogger(&logBuf))
	if err == nil {
		t.Fatal("InstallByName() error = nil, want ErrAllSourcesFailed")
	}

	if !errors.Is(err, ErrAllSourcesFailed) {
		t.Errorf("error = %v, want ErrAllSourcesFailed", err)
	}
}

func TestInstallerIdempotentSkip(t *testing.T) {
	t.Parallel()

	toolsDir := newTestToolsDir(t)

	goInstaller := &fakeInstaller{
		isInstalledFunc: func(_ v1alpha1.Tool, _ string) (bool, error) {
			return true, nil // Already installed.
		},
	}

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			ToolsDir: ptr.To(toolsDir),
			Tools: []v1alpha1.Tool{
				{
					Name:    "already-installed",
					Version: ptr.To("v1.0.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("example.com/cmd/tool")},
					},
				},
			},
		},
	}

	inst := &Installer{
		Config: config,
		Installers: map[v1alpha1.SourceType]TypeInstaller{
			v1alpha1.SourceTypeGo: goInstaller,
		},
	}

	var logBuf bytes.Buffer

	err := inst.InstallAll(context.Background(), WithInstallLogger(&logBuf))
	if err != nil {
		t.Fatalf("InstallAll() error = %v", err)
	}

	// Install should not have been called because IsInstalled returned true.
	if goInstaller.installCount.Load() != 0 {
		t.Errorf("installer called %d times, want 0 (should skip already-installed)", goInstaller.installCount.Load())
	}
}

func TestInstallerFailFast(t *testing.T) {
	t.Parallel()

	toolsDir := newTestToolsDir(t)

	var (
		callOrder   []string
		callOrderMu = make(chan struct{}, 1)
	)

	// Both tools use the same source type so they run sequentially within that type group.
	failInstaller := &fakeInstaller{
		installFunc: func(_ context.Context, tool v1alpha1.Tool, _ *v1alpha1.ToolSource, _ string) error {
			callOrderMu <- struct{}{}

			callOrder = append(callOrder, tool.Name)

			<-callOrderMu

			if tool.Name == "fail-first" {
				return errors.New("first tool fails")
			}

			return nil
		},
	}

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			ToolsDir: ptr.To(toolsDir),
			Tools: []v1alpha1.Tool{
				{
					Name:    "fail-first",
					Version: ptr.To("v1.0.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("example.com/cmd/fail")},
					},
				},
				{
					Name:    "should-skip",
					Version: ptr.To("v1.0.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("example.com/cmd/skip")},
					},
				},
			},
		},
	}

	inst := &Installer{
		Config: config,
		Installers: map[v1alpha1.SourceType]TypeInstaller{
			v1alpha1.SourceTypeGo: failInstaller,
		},
	}

	var logBuf bytes.Buffer

	err := inst.InstallAll(context.Background(), WithFailFast(), WithInstallLogger(&logBuf))
	if err == nil {
		t.Fatal("InstallAll() error = nil, want error in fail-fast mode")
	}

	// In fail-fast mode the second tool should not have been installed.
	callOrderMu <- struct{}{}

	defer func() { <-callOrderMu }()

	if len(callOrder) > 1 {
		t.Errorf("fail-fast should stop after first failure, but %d tools were attempted: %v", len(callOrder), callOrder)
	}
}

func TestInstallerRuntimeNotFound(t *testing.T) {
	t.Parallel()

	toolsDir := newTestToolsDir(t)

	noRuntimeInstaller := &fakeInstaller{
		runtimeAvailableFunc: func() error {
			return fmt.Errorf("%w: cargo: not found", ErrRuntimeNotFound)
		},
	}

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			ToolsDir: ptr.To(toolsDir),
			Tools: []v1alpha1.Tool{
				{
					Name:    "ripgrep",
					Version: ptr.To("v14.1.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeCargo, Package: ptr.To("ripgrep")},
					},
				},
			},
		},
	}

	inst := &Installer{
		Config: config,
		Installers: map[v1alpha1.SourceType]TypeInstaller{
			v1alpha1.SourceTypeCargo: noRuntimeInstaller,
		},
	}

	var logBuf bytes.Buffer

	err := inst.InstallByName(context.Background(), "ripgrep", WithInstallLogger(&logBuf))
	if err == nil {
		t.Fatal("InstallByName() error = nil, want ErrAllSourcesFailed wrapping ErrRuntimeNotFound")
	}

	if !errors.Is(err, ErrAllSourcesFailed) {
		t.Errorf("error = %v, want ErrAllSourcesFailed", err)
	}

	// The runtime-not-found should appear in the error chain.
	if !strings.Contains(err.Error(), "runtime not found") {
		t.Errorf("error should mention runtime not found, got %v", err)
	}

	// Install should never have been called.
	if noRuntimeInstaller.installCount.Load() != 0 {
		t.Errorf("installer called %d times, want 0 (runtime not available)", noRuntimeInstaller.installCount.Load())
	}
}

func TestInstallerParallelExecution(t *testing.T) {
	t.Parallel()

	toolsDir := newTestToolsDir(t)

	// Track concurrent execution: each installer signals when it starts and waits
	// for the other to also start, proving they ran in parallel.
	goStarted := make(chan struct{})
	npxStarted := make(chan struct{})

	goInstaller := &fakeInstaller{
		installFunc: func(_ context.Context, _ v1alpha1.Tool, _ *v1alpha1.ToolSource, _ string) error {
			close(goStarted)
			<-npxStarted // Wait for npx to also start.

			return nil
		},
	}

	npxInstaller := &fakeInstaller{
		installFunc: func(_ context.Context, _ v1alpha1.Tool, _ *v1alpha1.ToolSource, _ string) error {
			close(npxStarted)
			<-goStarted // Wait for go to also start.

			return nil
		},
	}

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			ToolsDir: ptr.To(toolsDir),
			Tools: []v1alpha1.Tool{
				{
					Name:    "controller-gen",
					Version: ptr.To("v0.17.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("sigs.k8s.io/controller-tools/cmd/controller-gen")},
					},
				},
				{
					Name:    "commitlint",
					Version: ptr.To("19.6.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeNpx, Package: ptr.To("@commitlint/cli")},
					},
				},
			},
		},
	}

	inst := &Installer{
		Config: config,
		Installers: map[v1alpha1.SourceType]TypeInstaller{
			v1alpha1.SourceTypeGo:  goInstaller,
			v1alpha1.SourceTypeNpx: npxInstaller,
		},
	}

	var logBuf bytes.Buffer

	// If this deadlocks, parallel execution is broken — the test will time out.
	err := inst.InstallAll(context.Background(), WithInstallLogger(&logBuf))
	if err != nil {
		t.Fatalf("InstallAll() error = %v", err)
	}
}

func TestInstallerContinueOnFailure(t *testing.T) {
	t.Parallel()

	toolsDir := newTestToolsDir(t)

	goInstaller := &fakeInstaller{
		installFunc: func(_ context.Context, tool v1alpha1.Tool, _ *v1alpha1.ToolSource, _ string) error {
			if tool.Name == "fail-tool" {
				return errors.New("this tool fails")
			}

			return nil
		},
	}

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			ToolsDir: ptr.To(toolsDir),
			Tools: []v1alpha1.Tool{
				{
					Name:    "fail-tool",
					Version: ptr.To("v1.0.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("example.com/cmd/fail")},
					},
				},
				{
					Name:    "ok-tool",
					Version: ptr.To("v1.0.0"),
					Sources: []v1alpha1.ToolSource{
						{Type: v1alpha1.SourceTypeGo, URL: ptr.To("example.com/cmd/ok")},
					},
				},
			},
		},
	}

	inst := &Installer{
		Config: config,
		Installers: map[v1alpha1.SourceType]TypeInstaller{
			v1alpha1.SourceTypeGo: goInstaller,
		},
	}

	var logBuf bytes.Buffer

	// Default mode: continue on failure and collect errors.
	err := inst.InstallAll(context.Background(), WithInstallLogger(&logBuf))
	if err == nil {
		t.Fatal("InstallAll() error = nil, want collected error")
	}

	// Both tools should have been attempted.
	if goInstaller.installCount.Load() != 2 {
		t.Errorf("installer called %d times, want 2 (continue on failure)", goInstaller.installCount.Load())
	}

	if !strings.Contains(err.Error(), "fail-tool") {
		t.Errorf("error should mention fail-tool, got %v", err)
	}
}

func TestInstallerInstallAllNoTools(t *testing.T) {
	t.Parallel()

	toolsDir := newTestToolsDir(t)

	config := &v1alpha1.ToolConfiguration{
		Spec: v1alpha1.ToolConfigurationSpec{
			ToolsDir: ptr.To(toolsDir),
			Tools:    []v1alpha1.Tool{},
		},
	}

	inst := &Installer{
		Config:     config,
		Installers: map[v1alpha1.SourceType]TypeInstaller{},
	}

	var logBuf bytes.Buffer

	err := inst.InstallAll(context.Background(), WithInstallLogger(&logBuf))
	if err != nil {
		t.Fatalf("InstallAll() with no tools error = %v, want nil", err)
	}
}
