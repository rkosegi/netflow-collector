//	Copyright 2022 Richard Kosegi
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

package collector

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkosegi/ipfix-collector/pkg/public"
	"time"
)

type metricEntry struct {
	counter *prometheus.CounterVec
	labels  []*labelProcessor
	cleaner *time.Ticker
}

type config struct {
	NetflowEndpoint   string                            `yaml:"netflow_endpoint"`
	TelemetryEndpoint *string                           `yaml:"telemetry_endpoint"`
	Pipeline          pipeline                          `yaml:"pipeline"`
	FlushInterval     int                               `yaml:"flush_interval"`
	Extensions        map[string]map[string]interface{} `yaml:"extensions"`
}

type pipeline struct {
	Filter  *[]flowMatchRule `yaml:"filter,omitempty"`
	Enrich  *[]string        `yaml:"enrich,omitempty"`
	Metrics metricsConfig    `yaml:"metrics"`
}

type metricsConfig struct {
	Prefix string       `yaml:"prefix"`
	Items  []metricSpec `yaml:"items"`
}

type metricSpec struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Labels      []metricLabel `yaml:"labels"`
}

type metricLabel struct {
	Name      string `yaml:"name"`
	Value     string `yaml:"value"`
	OnMissing string `yaml:"on_missing,omitempty"`
	Converter string `yaml:"converter"`
}

type filterFn func(flow *public.Flow) bool

type flowMatchRule struct {
	Match       string  `yaml:"match"`
	Cidr        *string `yaml:"cidr,omitempty"`
	Is          *string `yaml:"is,omitempty"`
	IsUint32    *string `yaml:"isUint32,omitempty"`
	Local2Local *bool   `yaml:"local-to-local"`
}

type flowMatcher struct {
	rule *flowMatchRule
	fn   filterFn
}

type labelProcessor struct {
	attr        string
	applyFn     func(flow *public.Flow) string
	onMissingFn func(flow *public.Flow) string
	converterFn func(interface{}) string
}
