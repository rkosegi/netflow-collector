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
	"net"
	"strconv"
	"time"
)

func (m *metricEntry) init(prefix string, spec *metricSpec, flushInterval int) {
	labels := make([]*labelProcessor, 0)
	labelNames := make([]string, 0)
	for _, label := range spec.Labels {
		labelNames = append(labelNames, label.Name)
		lp := &labelProcessor{}
		lp.init(label)
		labels = append(labels, lp)
	}
	m.labels = labels
	m.counter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: prefix,
		Subsystem: "flow",
		Name:      spec.Name,
		Help:      spec.Description,
	}, labelNames)
	m.cleaner = time.NewTicker(time.Second * time.Duration(flushInterval))
	go func() {
		for range m.cleaner.C {
			m.counter.Reset()
		}
	}()
}

func (m *metricEntry) apply(flow *public.Flow) {
	labelValues := make([]string, 0)
	for _, lp := range m.labels {
		labelValues = append(labelValues, lp.apply(flow))
	}
	m.counter.WithLabelValues(labelValues...).Add(float64(flow.Raw("bytes").(uint64)))
}

func (lp *labelProcessor) init(label metricLabel) {
	lp.attr = label.Value
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
