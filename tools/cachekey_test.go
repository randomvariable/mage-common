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
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCacheKeyStable(t *testing.T) {
	t.Parallel()

	path := writeCacheTestConfig(t, "config-v1")

	key1, err := CacheKey(path)
	if err != nil {
		t.Fatalf("CacheKey() error = %v", err)
	}

	key2, err := CacheKey(path)
	if err != nil {
		t.Fatalf("CacheKey() second call error = %v", err)
	}

	if key1 != key2 {
		t.Errorf("CacheKey() not stable: %q != %q", key1, key2)
	}
}

func TestCacheKeyChangesOnVersionBump(t *testing.T) {
	t.Parallel()

	path := writeCacheTestConfig(t, "version: v1.0.0\ntools:\n  - name: foo\n")

	key1, err := CacheKey(path)
	if err != nil {
		t.Fatalf("CacheKey(v1) error = %v", err)
	}

	err = os.WriteFile(path, []byte("version: v2.0.0\ntools:\n  - name: foo\n"), 0o600)
	if err != nil {
		t.Fatalf("rewrite config: %v", err)
	}

	key2, err := CacheKey(path)
	if err != nil {
		t.Fatalf("CacheKey(v2) error = %v", err)
	}

	if key1 == key2 {
		t.Error("CacheKey() should change when config changes")
	}
}

func TestCacheKeyChangesOnToolAdd(t *testing.T) {
	t.Parallel()

	path := writeCacheTestConfig(t, "tools:\n  - name: foo\n")

	key1, err := CacheKey(path)
	if err != nil {
		t.Fatalf("CacheKey(1 tool) error = %v", err)
	}

	err = os.WriteFile(path, []byte("tools:\n  - name: foo\n  - name: bar\n"), 0o600)
	if err != nil {
		t.Fatalf("rewrite config: %v", err)
	}

	key2, err := CacheKey(path)
	if err != nil {
		t.Fatalf("CacheKey(2 tools) error = %v", err)
	}

	if key1 == key2 {
		t.Error("CacheKey() should change when tools are added")
	}
}

func TestCacheKeyMissingFile(t *testing.T) {
	t.Parallel()

	_, err := CacheKey("/nonexistent/.tools.yaml")
	if err == nil {
		t.Fatal("CacheKey() error = nil, want error")
	}

	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("error = %v, want ErrConfigNotFound", err)
	}
}

func writeCacheTestConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, ".tools.yaml")

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("write test config: %v", err)
	}

	return path
}
