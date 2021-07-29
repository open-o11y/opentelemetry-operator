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
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/http"
	_ "github.com/prometheus/prometheus/discovery/install"
	"gopkg.in/yaml.v2"
)

type Config struct {
	PromConfig *promconfig.Config `yaml:"config"`
}

func ReplaceConfig(otelcol v1alpha1.OpenTelemetryCollector) (string, error) {
	if !CheckEnabled(otelcol) {
		return otelcol.Spec.Config, nil
	}
	config, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return "", err
	}

	promCfgMap, err := ConfigToPromConfig(config)
	if err != nil {
		return "", err
	}

	promCfg, err := yaml.Marshal(map[string]interface{}{
		"config": promCfgMap,
	})
	if err != nil {
		return "", err
	}

	var cfg Config
	if err = yaml.UnmarshalStrict(promCfg, &cfg); err != nil {
		return "", fmt.Errorf("error unmarshaling YAML: %w", err)
	}

	for i := range cfg.PromConfig.ScrapeConfigs {
		cfg.PromConfig.ScrapeConfigs[i].ServiceDiscoveryConfigs = discovery.Configs{
			&http.SDConfig{
				URL: fmt.Sprintf("https://%s:443/jobs/%s/targets?collector_id=$POD_NAME", naming.TAService(otelcol), cfg.PromConfig.ScrapeConfigs[i].JobName),
			},
		}
	}

	updPromCfg, err := yaml.Marshal(cfg.PromConfig)
	if err != nil {
		return "", err
	}

	updPromCfgMap := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(updPromCfg, &updPromCfgMap); err != nil {
		return "", err
	}

	config["receivers"].(map[interface{}]interface{})["prometheus"].(map[interface{}]interface{})["config"] = updPromCfgMap

	out, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(out), nil
}