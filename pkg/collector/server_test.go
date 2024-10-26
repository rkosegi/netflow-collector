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
	"fmt"
	"net"
	"os"
	"testing"

	flowprotob "github.com/cloudflare/goflow/v3/pb"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	dto "github.com/prometheus/client_model/go"
	"github.com/rkosegi/ipfix-collector/pkg/public"
	"github.com/stretchr/testify/assert"
)

func strPtr(str string) *string {
	return &str
}

func genMockmmdb(path string, t *testing.T) {
	db, err := mmdbwriter.New(mmdbwriter.Options{
		RecordSize:   24,
		DatabaseType: "GeoLite2-ASN",
	})
	if err != nil {
		t.Fatalf("unable to create new MMDB: %v", err)
	}

	err = db.Insert(&net.IPNet{
		IP:   net.IP{8, 8, 8, 8},
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}, mmdbtype.Map{
		"autonomous_system_number":       mmdbtype.Uint32(15169),
		"autonomous_system_organization": mmdbtype.String("Google LLC"),
	})
	if err != nil {
		t.Fatalf("unable to insert mock data: %v", err)
	}
	f, err := os.OpenFile(fmt.Sprintf("%s/%s", path, "GeoLite2-ASN.mmdb"), os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.FileMode(0o600))
	if err != nil {
		t.Fatalf("unable to open file for writing %v", err)
	}
	_, err = db.WriteTo(f)
	if err != nil {
		t.Fatalf("unable to write %v", err)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	db, err = mmdbwriter.New(mmdbwriter.Options{
		RecordSize:   24,
		DatabaseType: "GeoLite2-Country",
	})
	if err != nil {
		t.Fatalf("unable to create new MMDB: %v", err)
	}

	err = db.Insert(&net.IPNet{
		IP:   net.IP{8, 8, 8, 8},
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}, mmdbtype.Map{
		"continent": mmdbtype.Map{
			"code":       mmdbtype.String("NA"),
			"geoname_id": mmdbtype.Int32(6255149),
			"names": mmdbtype.Map{
				"en": mmdbtype.String("North America"),
			},
		},
		"registered_country": mmdbtype.Map{
			"geoname_id": mmdbtype.Int32(6252001),
			"iso_code":   mmdbtype.String("US"),
			"names": mmdbtype.Map{
				"en": mmdbtype.String("USA"),
			},
		},
		"country": mmdbtype.Map{
			"geoname_id": mmdbtype.Int32(6252001),
			"iso_code":   mmdbtype.String("US"),
			"names": mmdbtype.Map{
				"en": mmdbtype.String("USA"),
			},
		},
	})

	f2, err := os.OpenFile(fmt.Sprintf("%s/%s", path, "GeoLite2-Country.mmdb"), os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.FileMode(0o600))
	if err != nil {
		t.Fatalf("unable to open file for writing %v", err)
	}
	_, err = db.WriteTo(f2)
	if err != nil {
		t.Fatalf("unable to write %v", err)
	}
	defer func() {
		err = f2.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

}

func getFreePort(proto string, t *testing.T) int {
	if proto == "udp" {
		addr, err := net.ResolveUDPAddr(proto, "127.0.0.1:0")
		if err != nil {
			t.Fatalf("unable to resolve loopback udp address: %v", err)
		}
		l, err := net.ListenUDP(proto, addr)
		if err != nil {
			t.Fatalf("unable to listen on udp address: %v", err)
		}
		defer func(l *net.UDPConn) {
			_ = l.Close()
		}(l)
		return l.LocalAddr().(*net.UDPAddr).Port
	} else {
		addr, err := net.ResolveTCPAddr(proto, "127.0.0.1:0")
		if err != nil {
			t.Fatalf("unable to resolve loopback tcp address: %v", err)
		}
		l, err := net.ListenTCP(proto, addr)
		if err != nil {
			t.Fatalf("unable to listen on tcp address: %v", err)
		}
		defer func(l *net.TCPListener) {
			_ = l.Close()
		}(l)
		return l.Addr().(*net.TCPAddr).Port
	}
}

func TestServer(t *testing.T) {
	f, err := os.CreateTemp("", "nf.*.yaml")
	assert.NoError(t, err)
	defer func() {
		_ = os.Remove(f.Name())
	}()
	cfg, err := LoadConfig("../../testdata/config.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	cfg.TelemetryEndpoint = strPtr(fmt.Sprintf("0.0.0.0:%d", getFreePort("tcp", t)))
	cfg.NetflowEndpoint = fmt.Sprintf("0.0.0.0:%d", getFreePort("udp", t))
	*cfg.Pipeline.Filter = append(*cfg.Pipeline.Filter, public.FlowMatchRule{
		IsUint32: strPtr("10"),
		Match:    "source_as",
	})

	d, err := os.MkdirTemp("", "geoip")
	assert.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(d)
	}()
	cfg.Extensions["maxmind_asn"]["mmdb_dir"] = d
	cfg.Extensions["maxmind_country"]["mmdb_dir"] = d
	genMockmmdb(d, t)

	c := New(cfg, baseLogger)
	go func() {
		_ = c.Run()
	}()
	c.(*col).waitUntilReady()
	c.(*col).Publish([]*flowprotob.FlowMessage{{
		Type:           flowprotob.FlowMessage_NETFLOW_V5,
		Packets:        1,
		SamplerAddress: []byte{127, 0, 0, 1},
		SrcAddr:        []byte{8, 8, 8, 8},
		DstAddr:        []byte{192, 168, 1, 2},
		SrcPort:        53,
		DstPort:        31034,
		Proto:          0x11,
		SrcAS:          20,
	}})
	m := &dto.Metric{}
	assert.NoError(t, c.(*col).totalFlowsCounter.WithLabelValues("127.0.0.1").Write(m))
	assert.Equal(t, float64(1), m.Counter.GetValue())
}
