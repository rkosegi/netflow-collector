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
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkosegi/ipfix-collector/pkg/public"
)

func (m *metricEntry) init(prefix string, spec *public.MetricSpec, flushInterval int) {
	labels := make([]*labelProcessor, 0)
	labelNames := make([]string, 0)
	for _, label := range spec.Labels {
		labelNames = append(labelNames, label.Name)
		lp := &labelProcessor{}
		lp.init(label)
		labels = append(labels, lp)
	}
	m.labels = labels
	m.opts = prometheus.CounterOpts{
		Namespace: prefix,
		Subsystem: "flow",
		Name:      spec.Name,
		Help:      spec.Description,
	}
	m.counter = prometheus.NewCounterVec(m.opts, labelNames)
	m.metrics = ttlcache.New(
		ttlcache.WithTTL[string, prometheus.Counter](time.Duration(flushInterval) * time.Second),
	)
	go m.metrics.Start()
}

func (m *metricEntry) Collect(ch chan<- prometheus.Metric) {
	m.metrics.Range(func(item *ttlcache.Item[string, prometheus.Counter]) bool {
		ch <- item.Value()
		return true
	})
}

func (m *metricEntry) Describe(ch chan<- *prometheus.Desc) {
	m.counter.Describe(ch)
}

func (m *metricEntry) apply(flow *public.Flow) {
	labelValues := make([]string, 0)
	for _, lp := range m.labels {
		labelValues = append(labelValues, lp.apply(flow))
	}
	m.metrics.Get(strings.Join(labelValues, "|"), ttlcache.WithLoader(ttlcache.LoaderFunc[string, prometheus.Counter](
		func(c *ttlcache.Cache[string, prometheus.Counter], key string) *ttlcache.Item[string, prometheus.Counter] {
			opts := m.opts
			opts.ConstLabels = make(prometheus.Labels)
			for i, lp := range m.labels {
				opts.ConstLabels[lp.name] = labelValues[i]
			}
			return c.Set(key, prometheus.NewCounter(opts), ttlcache.DefaultTTL)
		},
	))).Value().Add(float64(flow.Raw("bytes").(uint64)))
}

func (lp *labelProcessor) init(label public.MetricLabel) {
	lp.attr = label.Value
	lp.name = label.Name
	lp.applyFn = lp.apply
	lp.onMissingFn = func(flow *public.Flow) string {
		return ""
	}

	switch label.Converter {
	case "ipv4":
		lp.converterFn = func(v interface{}) string {
			data := v.([]byte)
			return net.IPv4(data[0], data[1], data[2], data[3]).String()
		}

	case "str":
		lp.converterFn = func(v interface{}) string {
			return v.(string)
		}

	case "uint32":
		lp.converterFn = func(v interface{}) string {
			return strconv.FormatUint(uint64(v.(uint32)), 10)
		}

	case "uint64":
		lp.converterFn = func(v interface{}) string {
			return strconv.FormatUint(v.(uint64), 10)
		}

	case "static":
		lp.applyFn = func(flow *public.Flow) string {
			return label.Value
		}
	}
}

func (lp *labelProcessor) apply(flow *public.Flow) string {
	if attr := flow.Raw(lp.attr); attr != nil {
		return lp.converterFn(attr)
	} else {
		return lp.onMissingFn(flow)
	}
}
