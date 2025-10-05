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
	"github.com/rkosegi/ipfix-collector/pkg/public"
)

type enrichHostAlias struct {
	aliases map[string]string
}

func (e *enrichHostAlias) Close() error { return nil }
func (e *enrichHostAlias) Start() error { return nil }

func (e *enrichHostAlias) Configure(cfg map[string]interface{}) {
	e.aliases = map[string]string{}
	if _, ok := cfg["alias_map"]; ok {
		m := cfg["alias_map"].(map[string]interface{})
		for k, v := range m {
			e.aliases[k] = v.(string)
		}
	}
}

func (e *enrichHostAlias) enrichAliasAttr(flow *public.Flow, attr, dest string) {
	if ip := flow.AsIp(attr); ip != nil {
		if alias, ok := e.aliases[ip.String()]; ok {
			flow.AddAttr(dest, alias)
			return
		}
	}
	flow.AddAttr(dest, "unknown")
}

func (e *enrichHostAlias) Enrich(flow *public.Flow) {
	e.enrichAliasAttr(flow, "source_ip", "source_host_alias")
	e.enrichAliasAttr(flow, "destination_ip", "destination_host_alias")
}
