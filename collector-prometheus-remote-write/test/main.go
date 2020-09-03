package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
)

var (
	cortexEndpoint = "http://aps-workspaces-beta.us-west-2.amazonaws.com" // update this to query a different URL
	queryPath      = cortexEndpoint + "/workspaces/yang-yu-intern-test-ws/api/v1/query?query="
	inputPath      = "./test/data.txt" // data file path
	outputPath     = "./test/ans.txt"
	item           = 50           // total number of metrics / lines in output file
	metric         = "metricName" // base metricName. output file has only unique metricName with a number suffix
	gauge          = "gauge"
	counter        = "counter"
	histogram      = "histogram"
	summary        = "summary"
	types          = []string{ // types of metrics generatedq
		counter,
		gauge,
		histogram,
		summary,
	}
	// each metric will have from 1 to 4 sets of labels
	labels = []string{
		"label1 value1",
		"label2 value2",
		"label3 value3",
		"label4 value4",
	}
	delimeter  = ","                        // separate name, type, labels and metric value
	space      = " "                        // separate a set of label values or metric values
	valueBound = 5000                       // metric values are [0, valueBound)
	bounds     = []float64{0.01, 0.5, 0.99} // fixed quantile/buckets

	endpoint       = "localhost:55680"
	requestTimeout = 30 * time.Second // timeout for each gRPC request
	waitTime       = 1 * time.Second  // wait time between two sends

	bucketStr   = "bucket"
	quantileStr = "quantile"

	randomSuffix = strconv.Itoa(rand.Intn(5000)) // random suffix to avoid metric name collision between two tests
	client       http.Client

	awsService = "aps"
	awsRegion  = "us-west-2"
)
func init() {
	log.Println("initializing test pipeline...")

	rand.Seed(time.Now().UnixNano())
	randomSuffix = strconv.Itoa(rand.Intn(5000))

	// attach sig v4 signer for querier
	interceptor, err := NewAuth(awsService, awsRegion, http.DefaultTransport)
	if err != nil {
		log.Println(err)
		return
	}

	client = http.Client{
		Transport: interceptor,
		Timeout:   requestTimeout,
	}
	log.Println("finished.")
}
func main() {
	log.Println("waiting for the Collector to start...")

	// wait for collector to start
	time.Sleep(time.Second * 10)
	log.Println("generating metrics...")
	// Writes metrics in the following format to a text file:
	// 		name, type, label1 labelvalue1 , value1 value2 value3 value4 value5
	// gauge and counter has only one value
	generateData()
	log.Println("finished.")

	log.Println("sending metrics...")
	// send OTLP metrics to the Collector
	createAndSendLoad()
	log.Println("finished.")

	log.Println("querying metrics...")
	// retrieve and store metrics from Cortex
	getQueryAndStore(outputPath)
	log.Println("finished.")
}

// SigningRoundTripper is a Custom RoundTripper that performs AWS Sig V4
type SigningRoundTripper struct {
	transport http.RoundTripper
	signer    *v4.Signer
	cfg       *aws.Config
	service   string
}

// RoundTrip signs each outgoing request
func (si *SigningRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	// Sign the request
	_, err := si.signer.Sign(req, nil, si.service, *si.cfg.Region, time.Now())
	if err != nil {
		return nil, err
	}

	// Send the request to Cortex
	resp, err := si.transport.RoundTrip(req)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return resp, err
}

// NewAuth takes a map of strings as parameters and return a http.RoundTripper that perform Sig V4 signing on each
// request.
func NewAuth(service, region string, origTransport http.RoundTripper) (http.RoundTripper, error) {

	// Initialize session with default credential chain
	// https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
		aws.NewConfig().WithLogLevel(aws.LogDebugWithSigning),
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if _, err = sess.Config.Credentials.Get(); err != nil {
		log.Println(err)
		return nil, err
	}

	// Get Credentials, either from ./aws or from environmental variables
	creds := sess.Config.Credentials
	signer := v4.NewSigner(creds)
	signer.Debug = aws.LogDebugWithSigning
	signer.Logger = aws.NewDefaultLogger()
	rtp := SigningRoundTripper{
		transport: origTransport,
		signer:    signer,
		cfg:       sess.Config,
		service:   service,
	}
	// return a RoundTripper
	return &rtp, nil
}
