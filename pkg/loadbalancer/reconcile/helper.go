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

package reconcile

import (
	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	lbadapters "github.com/open-telemetry/opentelemetry-operator/pkg/loadbalancer/adapters"
)

func checkMode(mode v1alpha1.Mode, lbMode v1alpha1.LbMode) bool {
	deploy := false
	if mode == v1alpha1.ModeStatefulSet && len(lbMode) > 0 {
		deploy = true
	}
	return deploy
}

func checkConfig(params Params) (map[interface{}]interface{}, error) {
	config, err := adapters.ConfigFromString(params.Instance.Spec.Config)
	if err != nil {
		return nil, err
	}

	promConfig, err := lbadapters.ConfigToPromConfig(config)
	if err != nil {
		return nil, err
	}

	return promConfig, nil
}
