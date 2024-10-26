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

	"github.com/rkosegi/ipfix-collector/pkg/public"
	"github.com/stretchr/testify/assert"
)

func TestCidrFn(t *testing.T) {
	subnet1 := "192.168.1.0/24"
	flow1 := &public.Flow{}
	flow1.AddAttr("source_ip", []byte{192, 168, 1, 14})
	flow2 := &public.Flow{}
	flow2.AddAttr("source_ip", []byte{10, 11, 12, 13})
	fn, err := getFilterFn(&public.FlowMatchRule{
		Match: "source_ip",
		Cidr:  &subnet1,
	})
	assert.NoError(t, err)
	assert.True(t, fn(flow1))
	assert.False(t, fn(flow2))
}

func TestIsFn(t *testing.T) {
	ip := "192.168.1.14"
	flow1 := &public.Flow{}
	flow1.AddAttr("source_ip", []byte{192, 168, 1, 14})
	flow2 := &public.Flow{}
	flow2.AddAttr("source_ip", []byte{10, 11, 12, 13})
	fn, err := getFilterFn(&public.FlowMatchRule{
		Match: "source_ip",
		Is:    &ip,
	})
	assert.NoError(t, err)
	assert.True(t, fn(flow1))
	assert.False(t, fn(flow2))
}
