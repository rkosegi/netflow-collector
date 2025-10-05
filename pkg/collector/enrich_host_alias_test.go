//	Copyright 2025 Richard Kosegi
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

	"github.com/rkosegi/ipfix-collector/pkg/public"
	"github.com/stretchr/testify/assert"
)

func TestEnrichHostAlias(t *testing.T) {
	e := getEnricher("host_alias")
	assert.NoError(t, e.Start())
	e.Configure(map[string]interface{}{
		"alias_map": map[string]interface{}{
			"192.168.0.1": "gateway",
		},
	})
	defer func(e public.Enricher) {
		_ = e.Close()
	}(e)
	f := &public.Flow{}
	f.AddAttr("source_ip", []byte{192, 168, 0, 1})
	e.Enrich(f)
	assert.Equal(t, "gateway", *f.AsString("source_host_alias"))

	f = &public.Flow{}
	f.AddAttr("source_ip", []byte{192, 168, 0, 10})
	e.Enrich(f)

	assert.Equal(t, "unknown", *f.AsString("source_host_alias"))
}
