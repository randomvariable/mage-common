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
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// config holds the runtime configuration state.
// It is internal and accessed through package-level functions.
type config struct {
	// viper is the Viper instance managing configuration data
	viper *viper.Viper

	// logger is the optional logger for configuration events
	// nil if no logger provided
	logger Logger

	// sensitiveKeys tracks keys marked for redaction in logs
	sensitiveKeys map[string]bool

	// cleanArgs preserves flags removed by CleanOSArgs for debugging
	cleanArgs []string

	// flagSet overrides pflag.CommandLine for testing isolation.
	// nil means use pflag.CommandLine (default production behavior).
	flagSet *pflag.FlagSet

	// initOnce ensures single initialization
	initOnce sync.Once

	// initErr captures the error from the first Init() call
	initErr error
}

// singleton holds the package-level configuration instance.
var singleton *config

// init initializes the package-level singleton.
func init() {
	singleton = &config{
		sensitiveKeys: make(map[string]bool),
	}
}

// Init initializes the global Viper configuration with optional settings.
// This function is idempotent (safe to call multiple times, first call wins).
// It loads config.yaml from the current directory and enables environment variable overrides.
//
// Parameters:
//   - opts: Zero or more configuration options (applied sequentially)
//
// Returns:
//   - error: Non-nil if initialization fails
//
// Errors:
//   - ErrInvalidOption: Option validation failed
//   - ErrConfigParseFailed: Config file exists but contains invalid syntax
//
// Notes:
//   - Missing config file is NOT an error (env vars still work)
//   - Thread-safe: concurrent calls are safe
//   - After Init(), Viper is read-only (do not call viper.Set())
func Init(opts ...Option) error {
	singleton.initOnce.Do(func() {
		singleton.initErr = singleton.initialize(opts...)
	})

	return singleton.initErr
}

// V returns the global Viper instance for reading configuration values.
// This function will panic if called before Init().
//
// Users should access this via the standard viper package import instead:
//
//	import "github.com/spf13/viper"
//	value := viper.GetString("key")
//
// This function exists for testing and advanced use cases only.
func V() *viper.Viper {
	return singleton.v()
}

// v returns the config's Viper instance, panicking if uninitialized.
func (c *config) v() *viper.Viper {
	if c.viper == nil {
		panic("config.Init() must be called before accessing configuration")
	}

	return c.viper
}

// CleanOSArgs removes long flags (--flag) from os.Args to prevent Mage conflicts.
// This function MUST be called AFTER pflag.Parse() and BEFORE Init().
//
// It preserves:
//   - Short flags (-v, -d) for Mage compatibility
//   - Non-flag arguments (binary name, target names)
//
// Returns:
//   - []string: The flags that were removed (for debugging/logging)
//
// Side Effects:
//   - MODIFIES os.Args in-place
//
// Example:
//
//	pflag.String("image", "", "Image name")
//	pflag.Parse()
//	removed := config.CleanOSArgs()
//	config.Init()
func CleanOSArgs() []string {
	newArgs, removed := cleanArgs(os.Args)
	os.Args = newArgs
	singleton.cleanArgs = removed

	return removed
}

// cleanArgs removes long flags (--flag) from an args slice.
// Returns the cleaned args and the removed flags.
// This is a pure function suitable for parallel testing.
func cleanArgs(args []string) (cleaned, removed []string) {
	cleaned = []string{}
	removed = []string{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check if this is a long flag (starts with --)
		if !strings.HasPrefix(arg, "--") {
			// Keep everything else (binary name, targets, short flags)
			cleaned = append(cleaned, arg)

			continue
		}

		// Long flag detected - remove it
		removed = append(removed, arg)

		// Check if flag has '=' (--flag=value)
		if !strings.Contains(arg, "=") {
			// Two argument form: --flag value
			// Capture the value if it exists
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++ // Skip next arg (the value)
				removed = append(removed, args[i])
			}
		}
	}

	return cleaned, removed
}

// GetCleanedArgs returns the flags that were removed by CleanOSArgs.
// Useful for debugging and logging.
func GetCleanedArgs() []string {
	result := make([]string, len(singleton.cleanArgs))
	copy(result, singleton.cleanArgs)

	return result
}

// initialize performs the actual initialization work (called once by sync.Once).
func (c *config) initialize(opts ...Option) error {
	// Use the global Viper instance when not pre-configured (production path).
	// Tests may pre-set c.viper to a fresh viper.New() instance for isolation.
	if c.viper == nil {
		c.viper = viper.GetViper()
	}

	// Set configuration file name (without extension) - default
	c.viper.SetConfigName("config")

	// Add current directory as default search path
	c.viper.AddConfigPath(".")

	// Apply options BEFORE reading config file
	// This allows options to override config name and paths
	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			err = fmt.Errorf("%w: %w", ErrInvalidOption, err)

			return err
		}
	}

	// Enable environment variable binding with key replacement
	// This allows nested keys like "database.host" to be set via DATABASE_HOST
	c.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	c.viper.AutomaticEnv()

	// Bind pflags - use custom flag set if provided, otherwise use global
	flagSet := c.flagSet
	if flagSet == nil {
		flagSet = pflag.CommandLine
	}

	if flagSet.Parsed() {
		err := c.viper.BindPFlags(flagSet)
		if err != nil {
			err = fmt.Errorf("failed to bind command-line flags: %w", err)

			return err
		}
	}

	// Read config file (graceful if missing)
	err := c.viper.ReadInConfig()
	if err != nil {
		// Check if error is due to file not found
		var notFoundErr viper.ConfigFileNotFoundError
		if !errors.As(err, &notFoundErr) && !os.IsNotExist(err) {
			// Config file exists but parse failed
			err = fmt.Errorf("%w: %w", ErrConfigParseFailed, err)

			return err
		}
		// Missing config file is not an error - env vars still work
	}

	return nil
}
