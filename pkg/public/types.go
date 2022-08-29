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

package public

import (
	"io"
	"net"
)

type Collector interface {
	Run() error
}

type Flow struct {
	attrs map[string]interface{}
}

type Enricher interface {
	io.Closer
	Configure(map[string]interface{})
	Start() error
	Enrich(*Flow)
}

func (f *Flow) AddAttr(attr string, v interface{}) {
	if f.attrs == nil {
		f.attrs = make(map[string]interface{}, 0)
	}
	f.attrs[attr] = v
}

func (f *Flow) AsIp(attr string) net.IP {
	if v, ok := f.attrs[attr]; ok {
		b := v.([]byte)
		return net.IPv4(b[0], b[1], b[2], b[3])
	}
	return nil
}

func (f *Flow) AsString(attr string) *string {
	if v, ok := f.attrs[attr]; ok {
		x := v.(string)
		return &x
	}
	return nil
}

func (f *Flow) Raw(attr string) interface{} {
	return f.attrs[attr]
}

func (f *Flow) AsUint32(attr string) *uint32 {
	if v, ok := f.attrs[attr]; ok {
		x := v.(uint32)
		return &x
	}
	return nil
}
