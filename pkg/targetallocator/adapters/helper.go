// Copyright The OpenTelemetry Authors
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

package adapters

import (
	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
)

func CheckEnabled(otelcol v1alpha1.OpenTelemetryCollector) bool {
	return otelcol.Spec.Mode == v1alpha1.ModeStatefulSet && otelcol.Spec.TargetAllocator.Enabled
}

func CheckConfig(otelcol v1alpha1.OpenTelemetryCollector) (map[interface{}]interface{}, error) {
	config, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return nil, err
	}

	return ConfigToPromConfig(config)
}
