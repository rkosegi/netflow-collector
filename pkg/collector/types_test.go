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
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

var y = `
---
netflow_endpoint: 0.0.0.0:20000
telemetry_endpoint: 0.0.0.0:20001/metrics
flush_interval: 30
pipeline:
  filter:
    - match: source_ip
      cidr: 192.168.1.0/24
  enrich:
    - interface_mapper
    - maxmind_geoip
  metrics:
    prefix: netflow
    items:
      - name: traffic_by_ip
        description: Traffic by IP address
        labels:
          - name: source
            value: source_ip
            converter: ipv4
          - name: destination
            value: destination_ip
            converter: ipv4
      - name: traffic_by_protocol
        description: Traffic by protocol
        labels:
          - name: source
            value: source_ip
            converter: ipv4
          - name: destination
            value: destination_ip
            converter: ipv4
      - name: traffic_by_country
        description: Traffic by country
        labels:
          - name: source
            value: source_country
            converter: str
            on_missing: empty_str
          - name: destination
            value: destination_country
            converter: str
            on_missing: empty_str
          - name: static_label_example
            value: im-static
            converter: static
extensions:
  maxmind_asn:
    mmdb_dir: /usr/share/GeoIP/
  interface_mapper:
    "1": wan0
    "4": lan3
`

func TestParseConfig(t *testing.T) {
	var cfg config
	assert.NoError(t, yaml.Unmarshal([]byte(y), &cfg))
	assert.Equal(t, 30, cfg.FlushInterval)
	//filter
	filt0 := *cfg.Pipeline.Filter
	assert.Equal(t, "source_ip", filt0[0].Match)
	//enrich
	assert.Equal(t, 2, len(*cfg.Pipeline.Enrich))
	//metrics
	assert.Equal(t, 3, len(cfg.Pipeline.Metrics.Items))
	//extensions
	assert.Equal(t, "/usr/share/GeoIP/", cfg.Extensions["maxmind_asn"]["mmdb_dir"])
	assert.Equal(t, "wan0", cfg.Extensions["interface_mapper"]["1"])
	met1 := cfg.Pipeline.Metrics.Items[2]
	assert.Equal(t, "Traffic by country", met1.Description)
	assert.Equal(t, "destination_country", met1.Labels[1].Value)
	assert.Equal(t, "empty_str", met1.Labels[1].OnMissing)
}
