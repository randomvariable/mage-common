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

// Package testutil provides test helpers for the tools package.
package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ServerResponse configures the response for a test HTTP server route.
type ServerResponse struct {
	// StatusCode is the HTTP status code to return.
	StatusCode int

	// Body is the response body.
	Body []byte

	// ContentType overrides the Content-Type header.
	ContentType string

	// Delay adds a delay before responding (for timeout testing).
	Delay time.Duration

	// RedirectTo sends a redirect to this URL.
	RedirectTo string
}

// NewTestServer creates a test HTTP server with configurable routes.
// Routes is a map of URL path to ServerResponse.
// The server is automatically cleaned up when the test completes.
func NewTestServer(t *testing.T, routes map[string]ServerResponse) *httptest.Server {
	t.Helper()

	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		resp, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(writer, r)

			return
		}

		if resp.Delay > 0 {
			time.Sleep(resp.Delay)
		}

		if resp.RedirectTo != "" {
			http.Redirect(writer, r, resp.RedirectTo, http.StatusFound)

			return
		}

		if resp.ContentType != "" {
			writer.Header().Set("Content-Type", resp.ContentType)
		}

		if resp.StatusCode != 0 {
			writer.WriteHeader(resp.StatusCode)
		}

		if resp.Body != nil {
			_, _ = writer.Write(resp.Body)
		}
	}))

	t.Cleanup(server.Close)

	return server
}
