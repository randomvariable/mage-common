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
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlserializer "k8s.io/apimachinery/pkg/runtime/serializer/yaml"

	"github.com/randomvariable/mage-common/api/tools/v1alpha1"
	toolsscheme "github.com/randomvariable/mage-common/api/tools/v1alpha1/scheme"
)

// DefaultConfigFile is the default configuration file name.
const DefaultConfigFile = ".tools.yaml"

// LoadToolConfiguration reads, decodes, defaults, and validates a .tools.yaml file.
func LoadToolConfiguration(path string) (*v1alpha1.ToolConfiguration, error) {
	data, err := os.ReadFile(path) //nolint:gosec // File path from trusted configuration
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}

		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	scheme, codecs, err := toolsscheme.NewSchemeAndCodecs()
	if err != nil {
		return nil, fmt.Errorf("creating scheme: %w", err)
	}

	// Detect the GVK from the YAML document.
	mediaType := "application/yaml"
	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), mediaType)

	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrSerializerNotFound, mediaType)
	}

	// Decode using the YAML serializer with GVK detection.
	decoder := codecs.DecoderToVersion(
		yamlserializer.NewDecodingSerializer(info.Serializer),
		schema.GroupVersion{Group: v1alpha1.GroupName, Version: "v1alpha1"},
	)

	obj, _, err := decoder.Decode(data, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConfigInvalid, err)
	}

	config, ok := obj.(*v1alpha1.ToolConfiguration)
	if !ok {
		return nil, fmt.Errorf("%w: decoded object is not a ToolConfiguration", ErrConfigInvalid)
	}

	// Apply defaults.
	scheme.Default(config)

	// Validate.
	err = v1alpha1.ValidateToolConfiguration(config)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConfigInvalid, err)
	}

	return config, nil
}
