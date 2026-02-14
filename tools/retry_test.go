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
	"errors"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

func testBackoff() wait.Backoff {
	return wait.Backoff{
		Duration: 1 * time.Millisecond,
		Factor:   1.0,
		Jitter:   0,
		Steps:    3,
	}
}

func TestRetryWithBackoffSuccessFirst(t *testing.T) {
	t.Parallel()

	calls := 0

	err := RetryWithBackoff(context.Background(), testBackoff(), func(_ context.Context) (bool, error) {
		calls++

		return false, nil // done, no error
	})
	if err != nil {
		t.Errorf("RetryWithBackoff() error = %v, want nil", err)
	}

	if calls != 1 {
		t.Errorf("fn called %d times, want 1", calls)
	}
}

func TestRetryWithBackoffSuccessAfterRetries(t *testing.T) {
	t.Parallel()

	calls := 0

	err := RetryWithBackoff(context.Background(), testBackoff(), func(_ context.Context) (bool, error) {
		calls++
		if calls < 3 {
			return true, errors.New("transient")
		}

		return false, nil // success on 3rd attempt
	})
	if err != nil {
		t.Errorf("RetryWithBackoff() error = %v, want nil", err)
	}

	if calls != 3 {
		t.Errorf("fn called %d times, want 3", calls)
	}
}

func TestRetryWithBackoffAllFail(t *testing.T) {
	t.Parallel()

	errTransient := errors.New("transient failure")
	calls := 0

	err := RetryWithBackoff(context.Background(), testBackoff(), func(_ context.Context) (bool, error) {
		calls++

		return true, errTransient
	})
	if err == nil {
		t.Error("RetryWithBackoff() error = nil, want error")
	}

	if calls != 3 {
		t.Errorf("fn called %d times, want 3", calls)
	}
}

func TestRetryWithBackoffContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := RetryWithBackoff(ctx, testBackoff(), func(_ context.Context) (bool, error) {
		return true, errors.New("should not matter")
	})
	if err == nil {
		t.Error("RetryWithBackoff() error = nil, want context error")
	}
}

func TestDefaultBackoff(t *testing.T) {
	t.Parallel()

	b := DefaultBackoff()
	if b.Steps != 3 {
		t.Errorf("DefaultBackoff().Steps = %d, want 3", b.Steps)
	}

	if b.Duration != 1*time.Second {
		t.Errorf("DefaultBackoff().Duration = %v, want 1s", b.Duration)
	}
}
