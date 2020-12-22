## Components

### AWS X-Ray Propagator
[Link to Implementation](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/master/propagators/aws/xray)

The AWS X-Ray propagator provides HTTP header propagation for systems that are using [AWS X-Ray HTTP header format](https://docs.aws.amazon.com/xray/latest/devguide/xray-concepts.html#xray-concepts-tracingheader). Without the proper HTTP headers, AWS X-Ray will not be able to pick up any traces and it’s metadata sent from the collector. The AWS X-Ray propagator translates the OpenTelemetry SpanContext into the equivalent AWS X-Ray header format, for use with the OpenTelemetry Go SDK. By default, OpenTelemetry uses the [W3C Trace Context format](https://www.w3.org/TR/trace-context/) for propagating spans which is different than what AWS X-Ray takes in:

The following is an example of W3 trace header with root traceID.

```
traceparent: 5759e988bd862e3fe1be46a994272793 tracestate:optional
```

The following is an example of AWS X-Ray trace header with root trace ID and sampling decision.

```
X-Amzn-Trace-Id: Root=1-5759e988-bd862e3fe1be46a994272793;Sampled=1
```

As described in [specification](https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/context/api-propagators.md), a standard propagator has the following functions:

* inject()
   - The inject method injects the AWS X-Ray values into the header. The implementation should accept 2 parameters, the context format for propagating spans and textMapCarrier interface allowing our propagator to be implemented.
* extract()
    - Extract is required in a propagator to extract the value from an incoming request. For example, the values from the headers of an HTTP request are extracted. Given a context and a carrier, extract(), extracts context values from a carrier and return a new context, created from the old context, with the extracted values. The Go SDK extract method should accept 2 parameters, the context and textMapCarrier interface.
* fields()
    - Fields refer to the predefined propagation fields. If the carrier is reused, the fields should be deleted before calling [inject](https://github.com/open-telemetry/opentelemetry-specification/blob/b338f9f63dbf02ff8ebd100e8a847e7bf43e2682/specification/context/api-propagators.md#inject).
For example, if the carrier is a single-use or immutable request object, we don't need to clear fields as they couldn't have been set before. If it is a mutable, returnable object, successive calls should clear these fields first. This will return a list of fields that will be used by the TextMapPropagator.

### Amazon Elastic Container Service Resource Detector
[Link to Implementation](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/master/detectors/aws/ecs)

The objective of an ECS resource detector is to provide information about the container of a process running on an ECS environment. The ECS resource detector will first detect whether an application instrumented with OpenTelemetry Go SDK is running on ECS or not.

If the ECS resource detector successfully detects that a process is running on an ECS environment, it will populate the resource with metadata about the container the process is in. This will include the containerId(the docker ID of the container) and hostName(name of the container).

The ECS resource detector will return either an empty Resource or a Resource which is filled with metadata depending on if the application instrumented is running on ECS or not.

The resource detector contains the following functions:
* Detect()
  - This method is responsible for returning the resource with it's hostName and 
   containerId. In the event that the application is not running on ECS it will
   return an empty resource.
* getContainerID()
    - This method is responsible for returning the docker ID of the container found
    in its CGroup file. 
* getHostName()
    - This method will return the host name of the container the process is in.

## Usage


### AWS X-Ray Propagator
Import the package:
```
"go.opentelemetry.io/contrib/propagators/aws/xray"
"go.opentelemetry.io/otel"
```
Set OpenTelemetry to use the AWS X-Ray Propagator:

```
otel.SetTextMapPropagator(xray.Propagator{})
```

### ECS Resource Detector
Import the package:
```
"context"
"go.opentelemetry.io/contrib/detectors/ecs"
```
Usage:
```
// Instantiate resource
detectorUtils := new(ecsDetectorUtils)
ecsResourceDetector := ResourceDetector{detectorUtils}
resource, err := ecsResourceDetector.Detect(context.Background())

//Associate resource with traceProvider
tracerProvider := sdktrace.NewTracerProvider(
	sdktrace.WithResource(resource),
)
```
## Testing
### Tests
The components can be tested using the standard Go testing library. Here's how you can use the different commands in terminal to run the tests:
```
# Run all tests in X_test.go files (X to the file name)
go test

# Run a specific test (X refers to the specific test)
go test -run X
```

The AWS Distribution of OpenTelemetry for Go also provides a pipeline/integration test for validating the data. This can be found on the [aws-otel-go repository](https://github.com/aws-observability/aws-otel-go/blob/master/.github/workflows/main.yml). The pipeline will automatically build the components and deploy a docker image of it against the [test framework](https://github.com/aws-observability/aws-otel-test-framework) to validate data

### Steps on Running Integration Tests locally:
The integration tests consists of three parts:
* [AWS Distro for OpenTelemetry Collector](https://github.com/aws-observability/aws-otel-collector)
* [AWS OpenTelmetry Test Framework](https://github.com/aws-observability/aws-otel-test-framework)
* Go Sample Integration app

#### Step 1 - Configure AWS Credentials
You will need to configure your AWS Credential profile yet, please follow these [instructions](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-quickstart.html) for setting up your AWS credentials.
#### Step 2 - Install and Start the OpenTelemetry Collector
The first step is to install and start an instance of the AWS OpenTelemetry Collector. The purpose of the collector here is to export trace data to AWS X-Ray.
```
git clone https://github.com/aws-observability/aws-otel-collector.git ; \
    cd aws-otel-collector; \
    docker run --rm -p 55680:55680 -p 55679:55679 -p 8889:8888 \
      -e AWS_REGION=us-west-2 \
      -v "${PWD}/examples/config-test.yaml":/otel-local-config.yaml \
      --name awscollector public.ecr.aws/aws-observability/aws-otel-collector:latest \
      --config otel-local-config.yaml; \
```

#### Step 3 - Start Go Sample Integration App
The second step is to start a sample HTTP server written in Go. The purpose of the app is to generate traces and send them to AWS X-Ray so that we can validate the data.
```
git clone https://github.com/aws-observability/aws-otel-go.git ; \
    cd sample apps; \
    docker build --tag "sample-app" --file sampleapp/Dockerfile .

docker run -e LISTEN_ADDRESS=0.0.0.0:8080 \
    -e OTEL_EXPORTER_OTLP_ENDPOINT=172.17.0.1:55680 \
    -e OTEL_RESOURCE_ATTRIBUTES="aws-otel-integ-test" \
    -p 8080:8080 sample-app
```

#### Step 4 - Run Integration Tests
The last step is to clone the test framework and run the integration tests.

```
git clone https://github.com/aws-observability/aws-otel-test-framework.git

cd aws-otel-test-framework &&
    ./gradlew :validator:run --args='-c go-otel-trace-validation.yml --endpoint http://127.0.0.1:8080 --metric-namespace aws-otel-integ-test -t "sample-app"
```

## Design Docs
The design docs for the listed components can be found [here](design.md).

## Contributors
- [Kelvin Lo](https://github.com/kkelvinlo)
- [Wilbert Guo](https://github.com/wilguo)