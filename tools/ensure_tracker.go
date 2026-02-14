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

import "sync"

// ensureEntry tracks a single tool's ensure-once state.
type ensureEntry struct {
	once sync.Once
	err  error
}

// ensuredTools tracks which tools have been ensured during this process.
// keyed by tool name, values are *ensureEntry.
var ensuredTools sync.Map //nolint:gochecknoglobals // process-lifetime tracker

// ensureOnce runs fn at most once for the given tool name within a process.
// Concurrent callers for the same tool block until the first completes.
func ensureOnce(name string, fn func() error) error {
	val, _ := ensuredTools.LoadOrStore(name, &ensureEntry{})

	entry := val.(*ensureEntry) //nolint:forcetypeassert,revive // LoadOrStore always stores *ensureEntry
	entry.once.Do(func() {
		entry.err = fn()
	})

	return entry.err
}

// markEnsured records a tool as already ensured without running any install.
// This is used by EnsureAll to mark individual tools after a bulk install.
func markEnsured(name string) {
	val, _ := ensuredTools.LoadOrStore(name, &ensureEntry{})

	entry := val.(*ensureEntry) //nolint:forcetypeassert,revive // LoadOrStore always stores *ensureEntry
	entry.once.Do(func() {})
}

// ResetEnsureTracker clears all tracked state. Exported for testing only.
func ResetEnsureTracker() {
	ensuredTools = sync.Map{}
}
