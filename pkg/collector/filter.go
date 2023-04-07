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
	"github.com/rkosegi/ipfix-collector/pkg/public"
	"net"
	"strconv"
)

func getFilterMatcher(rule public.FlowMatchRule) (*FlowMatcher, error) {
	ret := &FlowMatcher{
		rule: &rule,
	}
	fn, err := getFilterFn(&rule)
	if err != nil {
		return nil, err
	}
	ret.fn = fn
	return ret, nil
}

func getCidrFilterFn(rule *public.FlowMatchRule) (FilterFn, error) {
	_, ipnet, err := net.ParseCIDR(*rule.Cidr)
	if err != nil {
		return nil, err
	}
	return func(flow *public.Flow) bool {
		ip := flow.AsIp(rule.Match)
		if ip == nil {
			return false
		}
		return ipnet.Contains(ip)
	}, nil
}

func getL2LFilterFn(rule *public.FlowMatchRule) (FilterFn, error) {
	return func(flow *public.Flow) bool {
		return isLocalIp(flow.AsIp("source_ip")) && isLocalIp(flow.AsIp("destination_ip"))
	}, nil
}

func getIsFilterFn(rule *public.FlowMatchRule) (FilterFn, error) {
	ip := net.ParseIP(*rule.Is)
	return func(flow *public.Flow) bool {
		v := flow.AsIp(rule.Match)
		if v == nil && ip == nil {
			return true
		}
		if v == nil || ip == nil {
			return false
		}
		return ip.Equal(v)
	}, nil
}

func getIsUint32FilterFn(rule *public.FlowMatchRule) (FilterFn, error) {
	i, err := strconv.Atoi(*rule.IsUint32)
	if err != nil {
		return nil, err
	}
	return func(flow *public.Flow) bool {
		v := flow.AsUint32(rule.Match)
		return v != nil && *v == uint32(i)
	}, nil
}

func getFilterFn(rule *public.FlowMatchRule) (FilterFn, error) {
	if rule.Local2Local != nil && *rule.Local2Local {
		return getL2LFilterFn(rule)
	}
	if rule.Cidr != nil {
		return getCidrFilterFn(rule)
	}
	if rule.Is != nil {
		return getIsFilterFn(rule)
	}
	if rule.IsUint32 != nil {
		return getIsUint32FilterFn(rule)
	}
	return nil, nil
}
