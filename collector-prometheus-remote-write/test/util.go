package main

import (
	"strconv"
	"strings"
	"time"

	common "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	metrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
	otlp "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
)

type combination struct {
	ty   otlp.MetricDescriptor_Type
	temp otlp.MetricDescriptor_Temporality
}

var (
	testHeaders = map[string]string{"headerOne": "value1"}

	typeInt64           = "INT64"
	typeMonotonicInt64  = "MONOTONIC_INT64"
	typeMonotonicDouble = "MONOTONIC_DOUBLE"
	typeHistogram       = "HISTOGRAM"
	typeSummary         = "SUMMARY"

	label11 = "test_label11"
	value11 = "test_value11"
	label12 = "test_label12"
	value12 = "test_value12"
	label21 = "test_label21"
	value21 = "test_value21"
	label22 = "test_label22"
	value22 = "test_value22"
	label31 = "test_label31"
	value31 = "test_value31"
	label32 = "test_label32"
	value32 = "test_value32"
	dirty1  = "%"
	dirty2  = "?"

	intVal1   int64 = 1
	intVal2   int64 = 2
	floatVal1       = 1.0
	floatVal2       = 2.0

	lbs1      = getLabels(label11, value11, label12, value12)
	lbs2      = getLabels(label21, value21, label22, value22)
	lbs1Dirty = getLabels(label11+dirty1, value11, dirty2+label12, value12)

	lb1Sig = "-" + label11 + "-" + value11 + "-" + label12 + "-" + value12
	lb2Sig = "-" + label21 + "-" + value21 + "-" + label22 + "-" + value22
	ns1    = "test_ns"
	name1  = "valid_single_int_point"

	monotonicInt64Comb  = 0
	monotonicDoubleComb = 1
	intComb             = 4
	histogramComb       = 2
	summaryComb         = 3
	validCombinations   = []combination{
		{otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_HISTOGRAM, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_SUMMARY, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_INT64, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_DOUBLE, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_INT64, otlp.MetricDescriptor_INSTANTANEOUS},
		{otlp.MetricDescriptor_DOUBLE, otlp.MetricDescriptor_INSTANTANEOUS},
		{otlp.MetricDescriptor_INT64, otlp.MetricDescriptor_CUMULATIVE},
		{otlp.MetricDescriptor_DOUBLE, otlp.MetricDescriptor_CUMULATIVE},
	}
	invalidCombinations = []combination{
		{otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_HISTOGRAM, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_SUMMARY, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_HISTOGRAM, otlp.MetricDescriptor_DELTA},
		{otlp.MetricDescriptor_SUMMARY, otlp.MetricDescriptor_DELTA},
		{ty: otlp.MetricDescriptor_INVALID_TYPE},
		{temp: otlp.MetricDescriptor_INVALID_TEMPORALITY},
		{},
	}
)

// OTLP metrics
// labels must come in pairs
func getLabels(labels ...string) []*common.StringKeyValue {
	var set []*common.StringKeyValue
	for i := 0; i < len(labels); i += 2 {
		set = append(set, &common.StringKeyValue{
			Key:   labels[i],
			Value: labels[i+1],
		})
	}
	return set
}

func getDescriptor(name string, i int, comb []combination) *otlp.MetricDescriptor {
	return &otlp.MetricDescriptor{
		Name:        name,
		Description: "",
		Unit:        "",
		Type:        comb[i].ty,
		Temporality: comb[i].temp,
	}
}

func getIntDataPoint(labels []*common.StringKeyValue, value int64, ts uint64) *otlp.Int64DataPoint {
	return &otlp.Int64DataPoint{
		Labels:            labels,
		StartTimeUnixNano: 0,
		TimeUnixNano:      ts,
		Value:             value,
	}
}

func getHistogramDataPoint(labels []*common.StringKeyValue, ts uint64, sum float64, count uint64, bounds []float64, buckets []uint64) *otlp.HistogramDataPoint {
	bks := []*otlp.HistogramDataPoint_Bucket{}
	for _, c := range buckets {
		bks = append(bks, &otlp.HistogramDataPoint_Bucket{
			Count:    c,
			Exemplar: nil,
		})
	}
	return &otlp.HistogramDataPoint{
		Labels:            labels,
		StartTimeUnixNano: 0,
		TimeUnixNano:      ts,
		Count:             count,
		Sum:               sum,
		Buckets:           bks,
		ExplicitBounds:    bounds,
	}
}

func getSummaryDataPoint(labels []*common.StringKeyValue, ts uint64, sum float64, count uint64, pcts []float64, values []float64) *otlp.SummaryDataPoint {
	pcs := []*otlp.SummaryDataPoint_ValueAtPercentile{}
	for i, v := range values {
		pcs = append(pcs, &otlp.SummaryDataPoint_ValueAtPercentile{
			Percentile: pcts[i],
			Value:      v,
		})
	}
	return &otlp.SummaryDataPoint{
		Labels:            labels,
		StartTimeUnixNano: 0,
		TimeUnixNano:      ts,
		Count:             count,
		Sum:               sum,
		PercentileValues:  pcs,
	}
}
func buildScalarMetric(name string, labels []*common.StringKeyValue, val float64, kind int) *metrics.Metric {
	return &metrics.Metric{
		MetricDescriptor: getDescriptor(name, kind, validCombinations),
		Int64DataPoints: []*metrics.Int64DataPoint{
			getIntDataPoint(labels, int64(val), uint64(time.Now().UnixNano())),
		},
	}
}

func buildHistogramMetric(name string, labels []*common.StringKeyValue, val []uint64) *metrics.Metric {

	sum := float64(val[0])
	count := val[1]
	buckets := val[2:]

	return &metrics.Metric{
		MetricDescriptor: getDescriptor(name, histogramComb, validCombinations),
		HistogramDataPoints: []*metrics.HistogramDataPoint{
			getHistogramDataPoint(labels, uint64(time.Now().UnixNano()), sum, count, bounds, buckets),
		},
	}
}
func buildSummaryMetric(name string, labels []*common.StringKeyValue, val []float64) *metrics.Metric {
	sum := val[0]
	count := uint64(val[1])
	pcts := make([]float64, len(bounds), len(bounds))
	values := make([]float64, len(bounds), len(bounds))
	for i, bound := range bounds {
		pcts[i] = bound
		values[i] = val[2+i]
	}
	return &metrics.Metric{
		MetricDescriptor: getDescriptor(name, summaryComb, validCombinations),
		SummaryDataPoints: []*metrics.SummaryDataPoint{
			getSummaryDataPoint(labels, uint64(time.Now().UnixNano()), sum, count, pcts, values),
		},
	}
}

func parseNumber(str string) float64 {
	str = strings.Replace(str, "[", space, -1)
	str = strings.Replace(str, "]", space, -1)
	str = strings.Trim(str, space)
	num, _ := strconv.ParseFloat(str, 64)
	return num
}
func parseuUInt64Slice(str string) []uint64 {
	arr := strings.Split(strings.Trim(str, space), space)
	result := make([]uint64, len(arr), len(arr))
	for i, numStr := range arr {
		num, _ := strconv.Atoi(numStr)
		result[i] = uint64(num)
	}
	return result
}
func parseFloat64Slice(str string) []float64 {
	arr := strings.Split(strings.Trim(str, space), space)
	result := make([]float64, len(arr), len(arr))
	for i, numStr := range arr {
		result[i] = parseNumber(numStr)
	}
	return result
}
