package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"strings"
	"time"

	service "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/metrics/v1"
	metrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
	"google.golang.org/grpc"
)

type sender struct {
	client service.MetricsServiceClient
}

func createAndSendLoad() {

	// connect to the Collector
	clientConn, err := grpc.Dial(endpoint, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	client := service.NewMetricsServiceClient(clientConn)
	s := &sender{
		client,
	}
	// read from file and send metrics
	s.createAndSendMetricsFromFile()
}

// createAndSendMetricsFromFile reads a text file, parse each line to build the corresponding otlp metric, then send the
// metric to the Collector
func (s *sender) createAndSendMetricsFromFile() {
	file, err := os.Open(inputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	// parse each line and build metric
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), space)
		params := strings.Split(line, delimeter)

		// get metric name and labels
		name := strings.Trim(params[0], space)
		labelSet := getLabels(strings.Split(strings.Trim(params[2], space), space)...)

		mType := params[1]
		values := params[3]
		var m *metrics.Metric
		// build metrics
		switch mType {
		case gauge, counter:
			m = buildScalarMetric(name, labelSet, parseNumber(values), intComb)
		case histogram:
			m = buildHistogramMetric(name, labelSet, parseuUInt64Slice(values))
		case summary:
			m = buildSummaryMetric(name, labelSet, parseFloat64Slice(values))
		default:
			log.Println("Invalid metric type")
			continue
		}
		log.Printf("%+v\n", m)
		s.sendMetric(m)
	}
}

// sendMetric sends m to an endpoint using gRPC protocol. After the request is send, it waits for a while before
// returning. The timeout for a request is 30 secondes by default
func (s *sender) sendMetric(m *metrics.Metric) {
	// build gRPC request
	request := service.ExportMetricsServiceRequest{
		ResourceMetrics: []*metrics.ResourceMetrics{
			{
				InstrumentationLibraryMetrics: []*metrics.InstrumentationLibraryMetrics{
					{
						Metrics: []*metrics.Metric{
							m,
						},
					},
				},
			},
		},
	}
	// specifc
	ctx, _ := context.WithTimeout(context.Background(), requestTimeout)
	_, err := s.client.Export(ctx, &request)
	time.Sleep(waitTime)
	if err != nil {
		log.Fatal(err)
	}
}
