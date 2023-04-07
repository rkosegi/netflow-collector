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
	dto "github.com/prometheus/client_model/go"
	"github.com/rkosegi/ipfix-collector/pkg/public"
	"github.com/stretchr/testify/assert"
	"testing"
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
	v := &dto.Metric{}
	assert.NoError(t, m.counter.WithLabelValues("10.11.12.13").Write(v))
	assert.Equal(t, float64(30), v.Counter.GetValue())

}
