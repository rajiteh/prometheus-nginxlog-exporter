/*
 * Copyright 2016 Martin Helmich <kontakt@martin-helmich.de>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/hpcloud/tail"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/satyrius/gonx"
)

var reqParamsRegex = regexp.MustCompile(`\?.*`)
var StaticLabels = []string{"method", "path", "status"}

// monitorApplication sets up the parsers and metrics for the log files
// that belong to a single application
func monitorApplication(cfg *ApplicationConfig) {
	cfg.OrderLabels()

	labelNames := append(StaticLabels, cfg.OrderedLabelNames...)

	metrics := newMetrics(cfg.Name, labelNames)

	parser := gonx.NewParser(cfg.Format)
	for _, file := range cfg.LogFiles {
		fmt.Printf("application/%s: monitoring log file %s\n", cfg.Name, file)
		go monitorFile(file, parser, metrics, &cfg.OrderedLabelValues, &cfg.Paths)
	}
}

// parseRequest parses an nginx $request value into method and path.
func parseRequest(request string) (string, string, error) {
	fields := strings.Split(request, " ")

	if len(fields) < 2 {
		return "", "", requestParseError(request)
	}

	path, err := url.PathUnescape(fields[1])
	if err != nil {
		return "", "", err
	}
	path = reqParamsRegex.ReplaceAllLiteralString(fields[1], "")

	return fields[0], path, nil
}

// parseUpstreamTime sums an nginx $upstream_response_time value into a single float.
func parseUpstreamTime(upstreamTime string) (float64, error) {
	var totalTime float64

	for _, timeString := range strings.Split(upstreamTime, ", ") {
		time, err := strconv.ParseFloat(timeString, 32)
		if err != nil {
			return 0, err
		}
		totalTime = totalTime + time
	}
	return totalTime, nil
}

// Tracks and collects metrics for a single log file.
func monitorFile(file string, parser *gonx.Parser, metrics *metrics, extraLabelValues *[]string,
	pathConfigs *[]PathConfig) {

	t, err := tail.TailFile(file, tail.Config{
		Follow: true,
		ReOpen: true,
	})
	if err != nil {
		panic(err)
	}

	labelValues := make([]string, len(*extraLabelValues)+len(StaticLabels))
	for i := range *extraLabelValues {
		labelValues[i+len(StaticLabels)] = (*extraLabelValues)[i]
	}

	for line := range t.Lines {
		entry, err := parser.ParseString(line.Text)
		if err != nil {
			fmt.Printf("failed to parse line in %s: %s\n", file, err)
			continue
		}

		labelValues[0] = "" // method
		labelValues[1] = "" // path
		labelValues[2] = "" // status

		if request, err := entry.Field("request"); err == nil {
			if method, path, err := parseRequest(request); err == nil {
				labelValues[0] = method
				for _, pathConfig := range *pathConfigs {
					path = pathConfig.CompiledPattern().ReplaceAllString(path, pathConfig.ReplaceWith)
				}
				labelValues[1] = path
			}
		}

		if status, err := entry.Field("status"); err == nil {
			labelValues[2] = status
		}

		if bytes, err := entry.FloatField("body_bytes_sent"); err == nil {
			metrics.bodyBytes.WithLabelValues(labelValues...).Observe(bytes)
		}

		if upstreamTime, err := entry.Field("upstream_response_time"); err == nil {
			if totalTime, err := parseUpstreamTime(upstreamTime); err == nil {
				metrics.upstreamSeconds.WithLabelValues(labelValues...).Observe(totalTime)
			}
		}

		if responseTime, err := entry.FloatField("request_time"); err == nil {
			metrics.requestSeconds.WithLabelValues(labelValues...).Observe(responseTime)
		}
	}
}

func main() {
	var opts StartupFlags
	var cfg = Config{
		Listen: ListenConfig{
			Port:    4040,
			Address: "0.0.0.0",
		},
	}

	flag.IntVar(&opts.ListenPort, "listen-port", 4040, "HTTP port to listen on")
	flag.StringVar(&opts.Format, "format", `$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" "$http_x_forwarded_for"`, "NGINX access log format")
	flag.StringVar(&opts.Application, "application", "default", "Name of application served by nginx")
	flag.StringVar(&opts.ConfigFile, "config-file", "", "Configuration file to read from")
	flag.Parse()

	opts.Filenames = flag.Args()

	if opts.ConfigFile != "" {
		fmt.Printf("loading configuration file %s\n", opts.ConfigFile)
		if err := LoadConfigFromFile(&cfg, opts.ConfigFile); err != nil {
			panic(err)
		}
	} else if err := LoadConfigFromFlags(&cfg, &opts); err != nil {
		panic(err)
	}

	for i := range cfg.Applications {
		monitorApplication(&cfg.Applications[i])
	}

	listenAddr := fmt.Sprintf("%s:%d", cfg.Listen.Address, cfg.Listen.Port)
	fmt.Printf("running HTTP server on address %s\n", listenAddr)

	http.Handle("/metrics", prometheus.Handler())
	http.ListenAndServe(listenAddr, nil)
}
