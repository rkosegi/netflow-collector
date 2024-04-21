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
	"os"

	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkosegi/ipfix-collector/pkg/public"
	"gopkg.in/yaml.v3"
)

type metricEntry struct {
	counter *prometheus.CounterVec
	labels  []*labelProcessor
	metrics *ttlcache.Cache[string, prometheus.Counter]
}

type FilterFn func(flow *public.Flow) bool

type FlowMatcher struct {
	rule *public.FlowMatchRule
	fn   FilterFn
}

type labelProcessor struct {
	attr        string
	applyFn     func(flow *public.Flow) string
	onMissingFn func(flow *public.Flow) string
	converterFn func(interface{}) string
}

func LoadConfig(file string) (*public.Config, error) {
	var cfg public.Config
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
