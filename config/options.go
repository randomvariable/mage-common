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

package config

import (
	"fmt"
	"os"
)

// Option is a functional option for configuring the package during initialization.
// Options are applied sequentially during Init() and can return validation errors.
type Option func(*config) error

// WithConfigFile overrides the default config file name.
// The name should be without extension (e.g., "app" for app.yaml).
// Default is "config".
//
// Example:
//
//	config.Init(config.WithConfigFile("myapp"))
//	// Searches for myapp.yaml, myapp.json, myapp.toml, etc.
func WithConfigFile(name string) Option {
	return func(c *config) error {
		if name == "" {
			return ErrEmptyConfigFileName
		}

		c.viper.SetConfigName(name)

		return nil
	}
}

// WithConfigPaths adds additional search paths for config files.
// Paths are checked in the order they are added.
//
// Example:
//
//	config.Init(config.WithConfigPaths("/etc/myapp", "$HOME/.config/myapp"))
func WithConfigPaths(paths ...string) Option {
	return func(c *config) error {
		if len(paths) == 0 {
			return ErrNoPathsProvided
		}

		for _, path := range paths {
			if path == "" {
				return ErrEmptyConfigPath
			}
			// Validate path exists
			_, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					err = fmt.Errorf("%w: %s", ErrConfigPathNotExist, path)

					return err
				}

				err = fmt.Errorf("config: stat path: %w", err)

				return err
			}

			c.viper.AddConfigPath(path)
		}

		return nil
	}
}

// WithEnvPrefix sets the environment variable prefix.
// Environment variables will be read as PREFIX_KEY_NAME.
//
// Example:
//
//	config.Init(config.WithEnvPrefix("APP"))
//	// Reads APP_DATABASE_HOST instead of DATABASE_HOST
func WithEnvPrefix(prefix string) Option {
	return func(c *config) error {
		if prefix == "" {
			return ErrEmptyEnvPrefix
		}

		c.viper.SetEnvPrefix(prefix)

		return nil
	}
}

// WithLogger sets the logger for configuration events.
// The logger must not be nil.
//
// Example:
//
//	logger := slog.Default()
//	config.Init(config.WithLogger(config.NewSlogAdapter(logger)))
func WithLogger(logger Logger) Option {
	return func(c *config) error {
		if logger == nil {
			return ErrNilLogger
		}

		c.logger = logger

		return nil
	}
}

// WithSensitiveKeys marks configuration keys as sensitive for redaction in logs.
//
// Example:
//
//	config.Init(config.WithSensitiveKeys("api.token", "db.password"))
func WithSensitiveKeys(keys ...string) Option {
	return func(c *config) error {
		if len(keys) == 0 {
			return ErrNoSensitiveKeys
		}

		for _, key := range keys {
			if key == "" {
				return ErrEmptySensitiveKey
			}

			c.sensitiveKeys[key] = true
		}

		return nil
	}
}

// WithoutSensitiveKeys removes keys from the sensitive keys list.
//
// Example:
//
//	config.Init(
//		config.WithSensitiveKeys("api.token", "debug.secret"),
//		config.WithoutSensitiveKeys("debug.secret"), // Remove in dev mode
//	)
func WithoutSensitiveKeys(keys ...string) Option {
	return func(c *config) error {
		if len(keys) == 0 {
			return ErrNoKeysProvided
		}

		for _, key := range keys {
			if key == "" {
				return ErrEmptyKey
			}

			delete(c.sensitiveKeys, key)
		}

		return nil
	}
}
