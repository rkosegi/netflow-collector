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
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/rkosegi/ipfix-collector/pkg/collector"
)

const progName = "netflow_collector"

var (
	configFile = kingpin.Flag("config", "Path to the configuration file.").Default("config.yaml").String()
)

func main() {
	promlogConfig := &promslog.Config{
		Style: promslog.GoKitStyle,
	}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print(progName))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promslog.New(promlogConfig)
	logger.Info(fmt.Sprintf("Starting %s", progName), "version", version.Info(), "config", *configFile)
	collector.SetBaseLogger(logger)
	if cfg, err := collector.LoadConfig(*configFile); err != nil {
		panic(err)
	} else {
		panic(collector.New(cfg, logger).Run())
	}
}
