//	Copyright 2023 Richard Kosegi
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
	"github.com/rkosegi/ipfix-collector/pkg/public"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInterfaceMapper(t *testing.T) {
	e := getEnricher("interface_mapper")
	assert.NoError(t, e.Start())
	e.Configure(map[string]interface{}{
		"0": "wan0",
		"1": "eth0",
	})
	defer func(e public.Enricher) {
		_ = e.Close()
	}(e)
	f := &public.Flow{}
	f.AddAttr("input_interface", uint32(0))
	f.AddAttr("output_interface", uint32(1))
	e.Enrich(f)
	assert.Equal(t, "wan0", *f.AsString("input_interface_name"))
	assert.Equal(t, "eth0", *f.AsString("output_interface_name"))
}

func TestProtocolName(t *testing.T) {
	e := getEnricher("protocol_name")
	assert.NoError(t, e.Start())
	e.Configure(map[string]interface{}{})
	defer func(e public.Enricher) {
		_ = e.Close()
	}(e)
	f := &public.Flow{}
	f.AddAttr("proto", uint32(1))
	e.Enrich(f)
	assert.Equal(t, "icmp", *f.AsString("proto_name"))
}
