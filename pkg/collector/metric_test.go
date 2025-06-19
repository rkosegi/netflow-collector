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
	"testing"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/rkosegi/ipfix-collector/pkg/public"
	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	s := &public.MetricSpec{
		Name:        "test1",
		Description: "Test metric 1",
		Labels: []public.MetricLabel{
			{
				Name:      "source",
				Value:     "source_ip",
				OnMissing: "empty_str",
				Converter: "ipv4",
			},
		},
	}
	m := &metricEntry{}
	m.init("netflow", s, 60)
	f := &public.Flow{}
	f.AddAttr("source_ip", []byte{10, 11, 12, 13})
	f.AddAttr("bytes", uint64(30))
	m.apply(f)

	assert.Equal(t, float64(30), getMetric(t, m, "10.11.12.13"))
}

// This test is a bit awful because it has time.Sleep() in it and takes approx 2 seconds
// This is to verify that metrics expire as expected
func TestMetricExpiration(t *testing.T) {
	s := &public.MetricSpec{
		Name:        "test1",
		Description: "Test metric 1",
		Labels: []public.MetricLabel{
			{
				Name:      "source",
				Value:     "source_ip",
				OnMissing: "empty_str",
				Converter: "ipv4",
			},
		},
	}
	m := &metricEntry{}
	m.init("netflow", s, 1)
	f := &public.Flow{}
	f.AddAttr("source_ip", []byte{10, 11, 12, 13})
	f.AddAttr("bytes", uint64(1))

	// start tests

	// t = 0.0 - add a thing, verify we get the stat back
	assert.Equal(t, 0, countMetrics(m))
	m.apply(f)
	assert.Equal(t, 1, countMetrics(m))
	assert.Equal(t, true, metricExists(m, "10.11.12.13"))
	assert.Equal(t, float64(1), getMetric(t, m, "10.11.12.13"))
	time.Sleep(time.Millisecond * 500)

	// t = 0.5 - first thing should still be validated as we have 1 sec TTL, now add second thing
	m.apply(f)
	assert.Equal(t, 1, countMetrics(m))
	assert.Equal(t, true, metricExists(m, "10.11.12.13"))
	assert.Equal(t, float64(2), getMetric(t, m, "10.11.12.13"))
	time.Sleep(time.Millisecond * 600)

	// t = 1.1 - adding second thing should have extended TTL, so verify we still have both things
	assert.Equal(t, true, metricExists(m, "10.11.12.13"))
	assert.Equal(t, float64(2), getMetric(t, m, "10.11.12.13"))
	time.Sleep(time.Millisecond * 500)

	// t = 1.6 - now it should have expired, verify it has gone
	assert.Equal(t, 0, countMetrics(m))
	assert.Equal(t, false, metricExists(m, "10.11.12.13"))

	// t = 1.6 - add it again, verify counter has reset
	m.apply(f)
	assert.Equal(t, 1, countMetrics(m))
	assert.Equal(t, true, metricExists(m, "10.11.12.13"))
	assert.Equal(t, float64(1), getMetric(t, m, "10.11.12.13"))
}

func metricExists(m *metricEntry, v string) bool {
	return m.metrics.Get(v, ttlcache.WithDisableTouchOnHit[string, prometheus.Counter]()) != nil
}

func getMetric(t *testing.T, m *metricEntry, v string) float64 {
	vv := dto.Metric{}
	assert.NoError(t, (m.metrics.Get(v, ttlcache.WithDisableTouchOnHit[string, prometheus.Counter]()).Value()).Write(&vv))
	return *vv.Counter.Value
}

func countMetrics(m *metricEntry) int {
	return m.metrics.Len()
}
