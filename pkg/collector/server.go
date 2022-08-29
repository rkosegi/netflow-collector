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
	"fmt"
	"github.com/cloudflare/goflow/v3/pb"
	"github.com/cloudflare/goflow/v3/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rkosegi/ipfix-collector/pkg/public"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
)

type col struct {
	log                 *log.Logger
	cfgFile             string
	cfg                 config
	filters             []*flowMatcher
	enrichers           []public.Enricher
	metrics             []*metricEntry
	droppedFlowsCounter *prometheus.CounterVec
	totalFlowsCounter   *prometheus.CounterVec
}

func (c *col) Describe(descs chan<- *prometheus.Desc) {
	c.droppedFlowsCounter.Describe(descs)
	c.totalFlowsCounter.Describe(descs)
	for _, m := range c.metrics {
		m.counter.Describe(descs)
	}
}

func (c *col) Collect(ch chan<- prometheus.Metric) {
	c.droppedFlowsCounter.Collect(ch)
	c.totalFlowsCounter.Collect(ch)
	for _, m := range c.metrics {
		m.counter.Collect(ch)
	}
}

func (c *col) Publish(messages []*flowprotob.FlowMessage) {
	for _, msg := range messages {
		c.process(msg)
	}
}

func (c *col) process(msg *flowprotob.FlowMessage) {
	if msg.Type == flowprotob.FlowMessage_NETFLOW_V5 {
		flow := c.mapMsg(msg)
		c.processFlow(flow)
		c.totalFlowsCounter.WithLabelValues(flow.AsIp("sampler").String()).Inc()
	}
}

func (c *col) Run() error {
	err := c.start()
	if err != nil {
		return err
	}
	s := &utils.StateNFLegacy{
		Transport: c,
		Logger:    log.StandardLogger(),
	}
	host, port, err := net.SplitHostPort(c.cfg.NetflowEndpoint)
	if err != nil {
		return err
	}
	iport, err := strconv.Atoi(port)
	if err != nil {
		return err
	}
	c.log.Infof("Starting Netflow V5 listener @ %s:%d", host, iport)
	return s.FlowRoutine(4, host, iport, true)
}

func (c *col) start() error {
	c.log = log.New()
	c.log.Printf("Loading config from %s", c.cfgFile)
	data, err := ioutil.ReadFile(c.cfgFile)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, &c.cfg)
	if err != nil {
		return err
	}

	c.totalFlowsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: c.cfg.Pipeline.Metrics.Prefix,
		Subsystem: "server",
		Name:      "total_flows",
		Help:      "The total number of ingested flows.",
	}, []string{"sampler"})

	c.droppedFlowsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: c.cfg.Pipeline.Metrics.Prefix,
		Subsystem: "server",
		Name:      "dropped_flows",
		Help:      "The total number of dropped flows.",
	}, []string{"sampler"})

	c.filters = make([]*flowMatcher, 0)
	c.enrichers = make([]public.Enricher, 0)
	c.metrics = make([]*metricEntry, 0)
	if c.cfg.Pipeline.Filter != nil {
		for _, rule := range *c.cfg.Pipeline.Filter {
			m, err := getFilterMatcher(&rule)
			if err != nil {
				return err
			}
			c.filters = append(c.filters, m)
		}
	}
	if c.cfg.Pipeline.Enrich != nil {
		for _, name := range *c.cfg.Pipeline.Enrich {
			e := getEnricher(name)
			if e == nil {
				return fmt.Errorf("unknown enricher : %s", name)
			}

			if ext, ok := c.cfg.Extensions[name]; ok {
				e.Configure(ext)
			}
			err = e.Start()
			if err != nil {
				return err
			}
			c.enrichers = append(c.enrichers, e)
		}
	}
	if c.cfg.FlushInterval == 0 {
		c.cfg.FlushInterval = 180
	}
	for _, metric := range c.cfg.Pipeline.Metrics.Items {
		me := &metricEntry{}
		me.init(c.cfg.Pipeline.Metrics.Prefix, &metric, c.cfg.FlushInterval)
		c.metrics = append(c.metrics, me)
	}

	if c.cfg.TelemetryEndpoint != nil {
		prometheus.MustRegister(c)
		http.Handle("/metrics", promhttp.Handler())
		c.log.Infof("Starting metrics server @ %s", *c.cfg.TelemetryEndpoint)
		go func() {
			err := http.ListenAndServe(*c.cfg.TelemetryEndpoint, nil)
			if err != nil {
				c.log.Errorf("Fail to start metric server %v", err)
			}
		}()
	}
	return nil
}

func (c *col) processFlow(flow *public.Flow) {
	for _, m := range c.filters {
		if m.fn(flow) {
			c.droppedFlowsCounter.WithLabelValues(flow.AsIp("sampler").String()).Inc()
			return
		}
	}
	for _, en := range c.enrichers {
		en.Enrich(flow)
	}
	for _, m := range c.metrics {
		m.apply(flow)
	}
}

func (c *col) mapMsg(msg *flowprotob.FlowMessage) *public.Flow {
	f := &public.Flow{}
	f.AddAttr("source_ip", msg.SrcAddr)
	f.AddAttr("destination_ip", msg.DstAddr)
	if msg.SrcAS != 0 {
		f.AddAttr("source_as", msg.SrcAS)
	}
	if msg.DstAS != 0 {
		f.AddAttr("destination_as", msg.DstAS)
	}
	f.AddAttr("proto", msg.Proto)
	f.AddAttr("source_port", msg.SrcPort)
	f.AddAttr("destination_port", msg.DstPort)
	f.AddAttr("input_interface", msg.InIf)
	f.AddAttr("output_interface", msg.OutIf)
	f.AddAttr("next_hop", msg.NextHop)
	f.AddAttr("sampler", msg.SamplerAddress)
	f.AddAttr("bytes", msg.Bytes)
	f.AddAttr("packets", msg.Packets)
	return f
}

func New(cfgFile string) public.Collector {
	return &col{
		cfgFile: cfgFile,
	}
}
