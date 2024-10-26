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

	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	cfg, err := LoadConfig("../../testdata/config.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 120, cfg.FlushInterval)
	// filter
	filt0 := *cfg.Pipeline.Filter
	assert.Equal(t, "source_ip", filt0[1].Match)
	// enrich
	assert.Equal(t, 4, len(*cfg.Pipeline.Enrich))
	// metrics
	assert.Equal(t, 1, len(cfg.Pipeline.Metrics.Items))
	// extensions
	assert.Equal(t, "/usr/share/GeoIP/", cfg.Extensions["maxmind_asn"]["mmdb_dir"])
	assert.Equal(t, "wan0", cfg.Extensions["interface_mapper"]["1"])
	met1 := cfg.Pipeline.Metrics.Items[0]
	assert.Equal(t, "Traffic detail", met1.Description)
	assert.Equal(t, "proto_name", met1.Labels[1].Value)
	assert.Equal(t, "empty_str", met1.Labels[4].OnMissing)
}
