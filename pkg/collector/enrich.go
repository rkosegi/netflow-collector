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
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/jellydator/ttlcache/v3"
	"github.com/oschwald/geoip2-golang"
	"github.com/rkosegi/ipfix-collector/pkg/public"
)

var (
	localCidrsStr = []string{
		"0.0.0.0/8,10.0.0.0/8,100.64.0.0/10,127.0.0.0/8",
		"169.254.0.0/16,172.16.0.0/12,192.0.0.0/24,192.0.2.0/24",
		"192.88.99.0/24,192.168.0.0/16,198.18.0.0/15,198.51.100.0/24",
		"203.0.113.0/24,224.0.0.0/4,233.252.0.0/24,240.0.0.0/4,255.255.255.255/32",
	}
	enrichers = map[string]public.Enricher{
		"maxmind_country":  &maxmindCountry{},
		"maxmind_asn":      &maxmindAsn{},
		"interface_mapper": &interfaceName{},
		"protocol_name":    &protocolName{},
		"reverse_dns":      &reverseDNS{},
	}
	localCidrs []*net.IPNet
)

func init() {
	localCidrs = make([]*net.IPNet, 0)
	for _, s := range localCidrsStr {
		for _, ips := range strings.Split(s, ",") {
			_, ipnet, err := net.ParseCIDR(ips)
			if err == nil {
				localCidrs = append(localCidrs, ipnet)
			}
		}
	}
}

func getEnricher(name string) public.Enricher {
	return enrichers[name]
}

type maxmindCountry struct {
	logger log.Logger
	isOpen bool
	dir    string
	db     *geoip2.Reader
}

func (m *maxmindCountry) Configure(cfg map[string]interface{}) {
	if dir, ok := cfg["mmdb_dir"]; !ok {
		m.dir = "/usr/share/GeoIP"
	} else {
		m.dir = dir.(string)
	}
	m.logger = log.With(baseLogger, "component", "geoip_country")
	m.logger.Log("msg", fmt.Sprintf("using directory %s for Country GeoIP", m.dir))
}

func (m *maxmindCountry) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func isLocalIp(addr net.IP) bool {
	for _, cidr := range localCidrs {
		if cidr.Contains(addr) {
			return true
		}
	}
	return false
}

func (m *maxmindCountry) Enrich(flow *public.Flow) {
	if m.isOpen {
		sourceIp := flow.AsIp("source_ip")
		destIp := flow.AsIp("destination_ip")
		if isLocalIp(sourceIp) {
			flow.AddAttr("source_country", "local")
		} else {
			country, _ := m.db.Country(sourceIp)
			if country != nil {
				if len(country.Country.IsoCode) == 0 {
					country.Country.IsoCode = "Unknown"
				}
				flow.AddAttr("source_country", country.Country.IsoCode)
			}
		}
		if isLocalIp(destIp) {
			flow.AddAttr("destination_country", "local")
		} else {
			country, _ := m.db.Country(flow.AsIp("destination_ip"))
			if country != nil {
				if len(country.Country.IsoCode) == 0 {
					country.Country.IsoCode = "Unknown"
				}
				flow.AddAttr("destination_country", country.Country.IsoCode)
			}
		}
	}
}

func (m *maxmindCountry) Start() error {
	db, err := geoip2.Open(fmt.Sprintf("%s/GeoLite2-Country.mmdb", m.dir))
	if err != nil {
		return err
	}
	m.isOpen = true
	m.db = db
	return nil
}

type interfaceName struct {
	mapping map[string]string
}

func (i *interfaceName) Close() error {
	return nil
}

func (i *interfaceName) Configure(cfg map[string]interface{}) {
	i.mapping = map[string]string{}
	for k, v := range cfg {
		i.mapping[k] = v.(string)
	}
}

func (i *interfaceName) Start() error {
	return nil
}

func (i *interfaceName) Enrich(flow *public.Flow) {
	ii := flow.AsUint32("input_interface")
	if ii != nil {
		if name, ok := i.mapping[strconv.FormatUint(uint64(*ii), 10)]; ok {
			flow.AddAttr("input_interface_name", name)
		}
	}

	ii = flow.AsUint32("output_interface")
	if ii != nil {
		if name, ok := i.mapping[strconv.FormatUint(uint64(*ii), 10)]; ok {
			flow.AddAttr("output_interface_name", name)
		}
	}
}

type protocolName struct {
}

func (p *protocolName) Close() error {
	return nil
}

func (p *protocolName) Configure(_ map[string]interface{}) {
	// not used in this enricher
}

func (p *protocolName) Start() error {
	return nil
}

func (p *protocolName) Enrich(flow *public.Flow) {
	protoName := ""
	proto := flow.AsUint32("proto")
	switch *proto {
	case 0x01:
		protoName = "icmp"

	case 0x02:
		protoName = "igmp"

	case 0x06:
		protoName = "tcp"

	case 0x11:
		protoName = "udp"

	default:
		protoName = fmt.Sprintf("other (%d)", *proto)
	}
	flow.AddAttr("proto_name", protoName)
}

type maxmindAsn struct {
	logger log.Logger
	isOpen bool
	dir    string
	db     *geoip2.Reader
}

func (m *maxmindAsn) Configure(cfg map[string]interface{}) {
	if dir, ok := cfg["mmdb_dir"]; !ok {
		m.dir = "/usr/share/GeoIP"
	} else {
		m.dir = dir.(string)
	}
	m.logger = log.With(baseLogger, "component", "geoip_asn")
	m.logger.Log("msg", fmt.Sprintf("using directory %s for ASN GeoIP", m.dir))
}

func (m *maxmindAsn) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *maxmindAsn) Start() error {
	db, err := geoip2.Open(fmt.Sprintf("%s/GeoLite2-ASN.mmdb", m.dir))
	if err != nil {
		return err
	}
	m.isOpen = true
	m.db = db
	return nil
}

func (m *maxmindAsn) Enrich(flow *public.Flow) {
	if m.isOpen {
		for _, dir := range []string{"source", "destination"} {
			ip := flow.AsIp(dir + "_ip")
			if !isLocalIp(ip) {
				asn, _ := m.db.ASN(ip)
				if asn != nil {
					if len(asn.AutonomousSystemOrganization) > 0 {
						flow.AddAttr(dir+"_asn_org", asn.AutonomousSystemOrganization)
						flow.AddAttr(dir+"_asn_num", asn.AutonomousSystemNumber)
					}
				}
			}
		}
	}
}

type reverseDNS struct {
	ttl   time.Duration
	cache *ttlcache.Cache[string, string]

	tailPiHole    bool
	piHoleResults *ttlcache.Cache[string, string]
}

func (m *reverseDNS) Configure(cfg map[string]interface{}) {
	tailPiholeObject, ok := cfg["tail_pihole"]
	if ok {
		m.tailPiHole, ok = tailPiholeObject.(bool)
		if !ok {
			panic("tail_pihole (if specified) must be a boolean, e.g. true (or false)")
		}
	}

	durationResult, ok := cfg["cache_duration"]
	if !ok {
		// if not found, set default
		m.ttl = time.Hour
		return
	}

	durationString, ok := durationResult.(string)
	if !ok {
		panic("cache_duration (if specified) must be a Go duration string, e.g. 1h")
	}

	var err error
	m.ttl, err = time.ParseDuration(durationString)
	if err != nil {
		panic("cache_duration (if specified) must be a Go duration string, e.g. 1h")
	}
}

func (m *reverseDNS) populateCacheWithPiHoleEntries(logger log.Logger, msgs io.Reader) {
	// cache to hold the results
	m.piHoleResults = ttlcache.New(
		ttlcache.WithTTL[string, string](m.ttl), // IP to name
	)
	go m.piHoleResults.Start()

	// cache to hold the DNS masq query session
	dnsMasqCache := ttlcache.New(
		ttlcache.WithTTL[string, string](time.Minute), // session ID to original query
	)
	go dnsMasqCache.Start()

	scanner := bufio.NewScanner(msgs)
	for scanner.Scan() {
		bits := strings.Split(scanner.Text(), " ")
		if len(bits) < 7 {
			continue
		}
		sessionId := bits[1]
		action := bits[3]
		switch {
		case strings.HasPrefix(action, "query"):
			dnsMasqCache.Set(sessionId, bits[4], ttlcache.DefaultTTL)
		case action == "cached", action == "reply":
			resultIP := bits[6]
			if bits[5] == "is" && resultIP != "<CNAME>" {
				origQuery := dnsMasqCache.Get(sessionId)
				if origQuery != nil {
					m.piHoleResults.Set(resultIP, origQuery.Value(), ttlcache.DefaultTTL)
					logger.Log("tph-query", origQuery.Value(), "tph-result", resultIP)
				}
			}
		}
	}
}

func (m *reverseDNS) Close() error {
	return nil
}

func (m *reverseDNS) reverseLookup(ip net.IP) string {
	if isLocalIp(ip) {
		return "local"
	}
	s := ip.String()
	if m.tailPiHole {
		piHoleResult := m.piHoleResults.Get(s)
		if piHoleResult != nil {
			return piHoleResult.Value()
		}
	}
	return m.cache.Get(s).Value()
}

func (m *reverseDNS) Enrich(flow *public.Flow) {
	flow.AddAttr("source_dns", m.reverseLookup(flow.AsIp("source_ip")))
	flow.AddAttr("destination_dns", m.reverseLookup(flow.AsIp("destination_ip")))
}

func (m *reverseDNS) Start() error {
	logger := log.With(baseLogger, "component", "reverse_dns")
	ctx := context.Background()
	m.cache = ttlcache.New(
		ttlcache.WithTTL[string, string](m.ttl),
		ttlcache.WithDisableTouchOnHit[string, string](),
		ttlcache.WithLoader(ttlcache.LoaderFunc[string, string](
			func(c *ttlcache.Cache[string, string], key string) *ttlcache.Item[string, string] {
				logger.Log("lookup", key)
				result := "unknown"
				names, err := net.DefaultResolver.LookupAddr(ctx, key)
				if err == nil && len(names) != 0 {
					result = strings.TrimRight(names[0], ".")
				}
				logger.Log("result", result)
				return c.Set(key, result, ttlcache.DefaultTTL)
			},
		)),
	)
	go m.cache.Start()

	if m.tailPiHole {
		logger.Log("tailing", "pihole")
		tph := exec.CommandContext(ctx, "pihole", "-t")
		stdout, err := tph.StdoutPipe()
		if err != nil {
			return err
		}
		err = tph.Start()
		if err != nil {
			return err
		}
		go m.populateCacheWithPiHoleEntries(logger, stdout)
	}

	return nil
}
