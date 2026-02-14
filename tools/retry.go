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
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	defaultBackoffFactor = 2.0
	defaultBackoffJitter = 0.1
	defaultBackoffSteps  = 3
	defaultBackoffCap    = 10 * time.Second
)

// DefaultBackoff returns the default exponential backoff configuration
// for transient failures: 1s initial, factor 2.0, jitter 0.1, 3 steps, 10s cap.
func DefaultBackoff() wait.Backoff {
	return wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   defaultBackoffFactor,
		Jitter:   defaultBackoffJitter,
		Steps:    defaultBackoffSteps,
		Cap:      defaultBackoffCap,
	}
}

// RetryWithBackoff executes fn with exponential backoff. The function fn should
// return true to retry, false to stop. The error from the last attempt is returned.
func RetryWithBackoff(ctx context.Context, backoff wait.Backoff, fn func(ctx context.Context) (bool, error)) error {
	var lastErr error

	err := wait.ExponentialBackoffWithContext(ctx, backoff, func(ctx context.Context) (bool, error) {
		retry, fnErr := fn(ctx)
		if fnErr != nil {
			lastErr = fnErr
		}

		if retry {
			// Return false to indicate "not done yet, retry".
			return false, nil
		}

		// Return true to indicate "done, stop retrying".
		return true, fnErr
	})
	if err != nil && lastErr != nil {
		return lastErr
	}

	if err != nil {
		return fmt.Errorf("retry exhausted: %w", err)
	}

	return nil
}
