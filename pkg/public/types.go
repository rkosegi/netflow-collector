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

// AddAttr adds or updates attribute value
func (f *Flow) AddAttr(attr string, v interface{}) {
	if f.attrs == nil {
		f.attrs = make(map[string]interface{}, 0)
	}
	f.attrs[attr] = v
}

// AsIp attempts to get attribute value as net.IP
func (f *Flow) AsIp(attr string) net.IP {
	if v, ok := f.attrs[attr]; ok {
		b := v.([]byte)
		return net.IPv4(b[0], b[1], b[2], b[3])
	}
	return nil
}

// AsString attempts to get attribute value as string
func (f *Flow) AsString(attr string) *string {
	if v, ok := f.attrs[attr]; ok {
		x := v.(string)
		return &x
	}
	return nil
}

// Raw attempts to get attribute value
func (f *Flow) Raw(attr string) interface{} {
	return f.attrs[attr]
}

// AsUint32 attempts to get attribute value as uint32
func (f *Flow) AsUint32(attr string) *uint32 {
	if v, ok := f.attrs[attr]; ok {
		x := v.(uint32)
		return &x
	}
	return nil
}

type Config struct {
	NetflowEndpoint   string                            `yaml:"netflow_endpoint"`
	TelemetryEndpoint *string                           `yaml:"telemetry_endpoint"`
	Pipeline          Pipeline                          `yaml:"pipeline"`
	FlushInterval     int                               `yaml:"flush_interval"`
	Extensions        map[string]map[string]interface{} `yaml:"extensions"`
}

type Pipeline struct {
	Filter  *[]FlowMatchRule `yaml:"filter,omitempty"`
	Enrich  *[]string        `yaml:"enrich,omitempty"`
	Metrics MetricsConfig    `yaml:"metrics"`
}

type MetricsConfig struct {
	Prefix string       `yaml:"prefix"`
	Items  []MetricSpec `yaml:"items"`
}

type MetricSpec struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Labels      []MetricLabel `yaml:"labels"`
}

type MetricLabel struct {
	Name      string `yaml:"name"`
	Value     string `yaml:"value"`
	OnMissing string `yaml:"on_missing,omitempty"`
	Converter string `yaml:"converter"`
}

type FlowMatchRule struct {
	Match       string  `yaml:"match"`
	Cidr        *string `yaml:"cidr,omitempty"`
	Is          *string `yaml:"is,omitempty"`
	IsUint32    *string `yaml:"isUint32,omitempty"`
	Local2Local *bool   `yaml:"local-to-local"`
}
