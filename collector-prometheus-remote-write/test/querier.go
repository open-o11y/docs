package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

func getQueryAndStore(outputPath string) {
	// check if queryPath is valid
	url, err := url.ParseRequestURI(queryPath)
	if err != nil {
		log.Println("invalid Cortex endpoint")
		return
	}

	// create output file
	output, err := os.Create(outputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	// open input file to get metric names
	input, err := os.Open(inputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer input.Close()

	scanner := bufio.NewScanner(input)
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	// get query from each line of the input file
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), space)
		params := strings.Split(line, delimeter)

		// get metric name, type and labels
		name := strings.Trim(params[0], space)
		labelSet := strings.Split(strings.Trim(params[2], space), space)
		mType := params[1]

		// query and write metric to output
		b := &strings.Builder{}
		queryMetric(url, name, mType, labelSet, b)
		output.WriteString(b.String())
	}
}

func queryMetric(url *url.URL, name, mType string, labelSet []string, builder *strings.Builder) {

	switch mType {
	case gauge, counter:
		// get query result
		json, err := getJSON(url.String() + name)
		if err != nil {
			log.Println(err)
			return
		}

		// retrieve name and labels
		name, labels := parseMetric(gjson.Get(json, "data.result.0.metric"))
		writeQueryNameTypeLabels(name, mType, labels, builder)

		// retrieve metric value
		value := gjson.Get(json, "data.result.0.value.1")
		builder.WriteString(value.Str)
		builder.WriteString("\n")
	// need to query histogram_sum, histogram_count, and histogram_bucket,
	case histogram:
		// retrieve histogram_sum time series
		jsonSum, err := getJSON(url.String() + name + "_sum")
		if err != nil {
			log.Println(err)
			return
		}

		// retrieve name and labels of this metric
		_, labels := parseMetric(gjson.Get(jsonSum, "data.result.0.metric"))
		writeQueryNameTypeLabels(name, mType, labels, builder)

		// retrieve sum value
		value := gjson.Get(jsonSum, "data.result.0.value.1")
		builder.WriteString(value.Str)
		builder.WriteString(space)

		// retrieve histogram_count time series
		jsonCount, err := getJSON(url.String() + name + "_count")
		if err != nil {
			log.Println(err)
			return
		}
		// retrieve count value
		value = gjson.Get(jsonCount, "data.result.0.value.1")
		builder.WriteString(value.Str)
		builder.WriteString(space)

		// retrieve the buckets JSON
		jsonBuckets, err := getJSON(url.String() + name + "_bucket")
		if err != nil {
			log.Println(err)
			return
		}

		// iterate through the results object, which contains objects for each bucket
		results := gjson.Get(jsonBuckets, "data.result")
		results.ForEach(func(key, value gjson.Result) bool {
			bucketValue := gjson.Parse(value.String()).Get("value.1")
			metricBoundary := gjson.Parse(value.String()).Get("metric.le").String()
			if metricBoundary != "+Inf" {
				builder.WriteString(bucketValue.Str)
				builder.WriteString(space)
			}
			return true
		})
		builder.WriteString("\n")
	// need to query summary_sum, summary_count, and summary quantiles,
	case summary:
		// retrieve summary_sum time series
		jsonSum, err := getJSON(url.String() + name + "_sum")
		if err != nil {
			log.Println(err)
			return
		}

		// retrieve name and labels of this metric
		_, labels := parseMetric(gjson.Get(jsonSum, "data.result.0.metric"))
		writeQueryNameTypeLabels(name, mType, labels, builder)

		// retrieve sum value
		value := gjson.Get(jsonSum, "data.result.0.value.1")
		builder.WriteString(value.Str)
		builder.WriteString(space)

		// retrieve summary_count time series
		jsonCount, err := getJSON(url.String() + name + "_count")
		if err != nil {
			log.Println(err)
			return
		}
		// retrieve count value
		value = gjson.Get(jsonCount, "data.result.0.value.1")
		builder.WriteString(value.Str)
		builder.WriteString(space)

		// retrieve the quantiles JSON
		jsonBuckets, err := getJSON(url.String() + name)
		if err != nil {
			log.Println(err)
			return
		}

		// iterate through the results object, which contains objects for each quantile
		results := gjson.Get(jsonBuckets, "data.result")
		results.ForEach(func(key, value gjson.Result) bool {
			quantileValue := gjson.Parse(value.String()).Get("value.1")
			// need to add extra 0 to the end for numbers with less than 6 decimal place
			num, _ := strconv.ParseFloat(quantileValue.Str, 64)
			builder.WriteString(fmt.Sprintf("%f", num))
			builder.WriteString(space)
			return true
		})
		builder.WriteString("\n")
	}

}

func writeQueryNameTypeLabels(name, mType string, labels map[string]string, builder *strings.Builder) {
	// write name and type
	builder.WriteString(name)
	builder.WriteString(delimeter)
	builder.WriteString(mType)
	builder.WriteString(delimeter)
	// store keys and sort by keys so that output has the same order as input
	keys := make([]string, 0, len(labels))

	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		builder.WriteString(k)
		builder.WriteString(space)
		builder.WriteString(labels[k])
		builder.WriteString(space)
	}
	builder.WriteString(delimeter)
}

// getJSON makes a HTTP GET request to Cortex and returns a JSON as a string.
func getJSON(url string) (string, error) {

	res, err := client.Get(url)
	if err != nil {
		log.Printf("error querying this URL: %v\n", url)
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 status code: %v", res.StatusCode)
	}

	// Convert the response body into a JSON string.
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// parseMetric iterates through a JSON object representing a single metric and returns the
// name and the labels in it.
func parseMetric(metric gjson.Result) (string, map[string]string) {
	var name string
	labels := make(map[string]string)

	metric.ForEach(func(key, value gjson.Result) bool {
		// Everything other `__name__` is a label.
		if key.Str == "__name__" {
			name = value.Str
			return true
		}
		labels[key.Str] = value.Str
		return true
	})
	return name, labels
}
