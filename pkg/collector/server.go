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
	"errors"
	"fmt"
	"log/slog"

	flowpb "github.com/netsampler/goflow2/v2/pb"
	"github.com/netsampler/goflow2/v2/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rkosegi/ipfix-collector/pkg/public"

	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type col struct {
	logger              *slog.Logger
	ready               sync.WaitGroup
	cfg                 *public.Config
	filters             []FlowMatcher
	enrichers           []public.Enricher
	metrics             []*metricEntry
	droppedFlowsCounter *prometheus.CounterVec
	totalFlowsCounter   *prometheus.CounterVec
	scrapingSum         *prometheus.SummaryVec
	ap                  *utils.AutoFlowPipe
	recv                *utils.UDPReceiver
}

func (c *col) Close() error {
	if c.ap != nil {
		c.ap.Close()
	}
	if c.recv != nil {
		return c.recv.Stop()
	}
	return nil
}

func (c *col) Describe(descs chan<- *prometheus.Desc) {
	c.droppedFlowsCounter.Describe(descs)
	c.totalFlowsCounter.Describe(descs)
	c.scrapingSum.Describe(descs)
	for _, m := range c.metrics {
		m.Describe(descs)
	}
}

func (c *col) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	defer func() {
		c.scrapingSum.WithLabelValues().Observe(float64(time.Now().UnixMicro() - start.UnixMicro()))
		c.scrapingSum.Collect(ch)
	}()

	c.droppedFlowsCounter.Collect(ch)
	c.totalFlowsCounter.Collect(ch)
	for _, m := range c.metrics {
		m.Collect(ch)
	}
}

func (c *col) Publish(messages []*flowpb.FlowMessage) {
	for _, msg := range messages {
		c.Consume(msg)
	}
}

func (c *col) Consume(msg *flowpb.FlowMessage) {
	if msg.Type == flowpb.FlowMessage_NETFLOW_V5 {
		flow := c.mapMsg(msg)
		c.processFlow(flow)
		c.totalFlowsCounter.WithLabelValues(flow.AsIp("sampler").String()).Inc()
	}
}

func (c *col) waitUntilReady() {
	c.ready.Wait()
}

func (c *col) Run() (err error) {
	err = c.start()
	if err != nil {
		return err
	}
	if c.recv, err = utils.NewUDPReceiver(&utils.UDPReceiverConfig{
		Workers:   2,
		Sockets:   1,
		Blocking:  false,
		QueueSize: 100,
	}); err != nil {
		return err
	}
	c.ap = utils.NewFlowPipe(&utils.PipeConfig{
		Producer: &producerMetricAdapter{consumer: c},
	})

	host, port, err := net.SplitHostPort(c.cfg.NetflowEndpoint)
	if err != nil {
		return err
	}
	iport, err := strconv.Atoi(port)
	if err != nil {
		return err
	}
	c.logger.Info("starting Netflow V5 listener", "host", host, "port", iport)

	defer func() {
		_ = c.Close()
	}()

	err = c.recv.Start(host, iport, c.ap.DecodeFlow)
	if err != nil {
		return err
	}
	<-make(chan struct{})
	return nil
}

func (c *col) startEnrichers() (err error) {
	if c.cfg.Pipeline.Enrich != nil {
		c.logger.Debug("starting enrichers", "enrichers", len(*c.cfg.Pipeline.Enrich))
		for _, name := range *c.cfg.Pipeline.Enrich {
			c.logger.Info("starting enricher", "name", name)
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
	return nil
}

func (c *col) startFilters() error {
	if c.cfg.Pipeline.Filter != nil {
		c.logger.Info("starting filters", "rules", len(*c.cfg.Pipeline.Filter))
		for _, rule := range *c.cfg.Pipeline.Filter {
			m, err := getFilterMatcher(rule)
			if err != nil {
				return err
			}
			c.filters = append(c.filters, *m)
		}
	}
	return nil
}

func (c *col) start() (err error) {
	defer c.ready.Done()

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
	c.scrapingSum = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: c.cfg.Pipeline.Metrics.Prefix,
		Subsystem: "server",
		Name:      "scrape",
		Help:      "The summary of time spent by scraping in microseconds",
	}, []string{})
	if err = c.startFilters(); err != nil {
		return err
	}
	if err = c.startEnrichers(); err != nil {
		return err
	}
	if c.cfg.FlushInterval == 0 {
		c.cfg.FlushInterval = 180
	}
	c.logger.Info("creating metric items", "count", len(c.cfg.Pipeline.Metrics.Items))
	for _, metric := range c.cfg.Pipeline.Metrics.Items {
		me := &metricEntry{}
		me.init(c.cfg.Pipeline.Metrics.Prefix, &metric, c.cfg.FlushInterval)
		c.metrics = append(c.metrics, me)
	}

	if c.cfg.TelemetryEndpoint != nil {
		prometheus.MustRegister(c)
		prometheus.MustRegister(collectors.NewBuildInfoCollector())
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		})

		c.logger.Info("starting metrics server", "address", *c.cfg.TelemetryEndpoint)
		go func() {
			if err = http.ListenAndServe(*c.cfg.TelemetryEndpoint, nil); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					panic(err)
				} else {
					c.logger.Info("metrics server closed")
				}
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

func (c *col) mapMsg(msg *flowpb.FlowMessage) *public.Flow {
	f := &public.Flow{}
	f.AddAttr("source_ip", msg.SrcAddr)
	f.AddAttr("destination_ip", msg.DstAddr)
	if msg.SrcAs != 0 {
		f.AddAttr("source_as", msg.SrcAs)
	}
	if msg.DstAs != 0 {
		f.AddAttr("destination_as", msg.DstAs)
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

func New(cfg *public.Config, logger *slog.Logger) public.Collector {
	c := &col{
		logger:    logger,
		cfg:       cfg,
		filters:   []FlowMatcher{},
		enrichers: []public.Enricher{},
		metrics:   []*metricEntry{},
	}
	c.ready.Add(1)
	return c
}
