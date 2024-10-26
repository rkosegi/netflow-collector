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

package main

import (
	"fmt"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/rkosegi/ipfix-collector/pkg/collector"
)

const progName = "netflow_collector"

var (
	logger     = log.NewNopLogger()
	configFile = kingpin.Flag("config", "Path to the configuration file.").Default("config.yaml").String()
)

func main() {
	kingpin.Version(version.Print(progName))
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger = promlog.New(promlogConfig)
	collector.SetBaseLogger(logger)

	level.Info(logger).Log("msg", fmt.Sprintf("Starting %s", progName),
		"version", version.BuildContext(),
		"config", *configFile)

	if cfg, err := collector.LoadConfig(*configFile); err != nil {
		panic(err)
	} else {
		panic(collector.New(cfg, logger).Run())
	}
}
