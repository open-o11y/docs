# OpenTelemetry Go SDK Prometheus Remote Write Exporter

## Table of Contents

- [OpenTelemetry Go SDK Prometheus Remote Write Exporter](#opentelemetry-go-sdk-prometheus-remote-write-exporter)
  - [Table of Contents](#table-of-contents)
  - [Architecture Overview](#architecture-overview)
    - [Data Path](#data-path)
      - [OpenTelemetry SDK Data Path](#opentelemetry-sdk-data-path)
      - [Exporter Data Path](#exporter-data-path)
  - [Usage](#usage)
    - [1. Configure the Exporter](#1-configure-the-exporter)
    - [2. Setting up an Exporter](#2-setting-up-an-exporter)
    - [3. Set up backend](#3-set-up-backend)
  - [Repository Structure](#repository-structure)
  - [Testing](#testing)
  - [Future Enhancements](#future-enhancements)
  - [Pull Requests Filed and Merged](#pull-requests-filed-and-merged)
  - [Reference Documents](#reference-documents)
  - [Contributors](#contributors)

## Architecture Overview

> Note: Entities with an \* after their name are not actual classes, but instead logical groupings
> of functions within the Cortex package.

![Exporter UML Diagram](./images/exporter-uml.png)

### Data Path

#### OpenTelemetry SDK Data Path

![SDK Metrics Collection Pipeline](./images/sdk-data-path.png)

The diagram above outlines the SDK data path. This `Exporter` uses a `Push Controller`, so on a set
interval, Tick will be called starting the data collection. The `Accumulator` will collect all
metrics from the `Aggreators`. The collected metrics are saved in a CheckpointSet in the
`Processor`. The CheckpointSet is then sent to the `Export` when Export() is called.

#### Exporter Data Path

![SDK Metrics Collection Pipeline](./images/exporter-sequence.png)

The Exporter receives a CheckpointSet from the Push Controller, converts the CheckpointSet to
TimeSeries, and sends them in a snappy-compressed message via HTTP to Cortex.

## Usage

### 1. Configure the Exporter

The Exporter can be configured with a variety of settings as shown below. Defaults are
provided for most of the settings, but the endpoint must be set or else the Exporter will
not run. While not necessary, users may want to add extra histogram buckets or
distribution quantiles with the `HistogramBoundaries` and `Quantiles` settings.

For authentication, the exporter provides two options: basic authentication with a
username/password as well as bearer token authentication. Users can also configure TLS by
supplying certificates. There is also an option to provide a custom HTTP client in case
the user wants to add custom authentication or customize the HTTP Client settings.

```go
type Config struct {
	Endpoint            string            `mapstructure:"url"`
	RemoteTimeout       time.Duration     `mapstructure:"remote_timeout"`
	Name                string            `mapstructure:"name"`
	BasicAuth           map[string]string `mapstructure:"basic_auth"`
	BearerToken         string            `mapstructure:"bearer_token"`
	BearerTokenFile     string            `mapstructure:"bearer_token_file"`
	TLSConfig           map[string]string `mapstructure:"tls_config"`
	ProxyURL            string            `mapstructure:"proxy_url"`
	PushInterval        time.Duration     `mapstructure:"push_interval"`
	Quantiles           []float64         `mapstructure:"quantiles"`
	HistogramBoundaries []float64         `mapstructure:"histogram_boundaries"`
	Headers             map[string]string `mapstructure:"headers"`
	Client              *http.Client
}
```

```go
// Create Config struct using utils module.
config, err := utils.NewConfig("config.yml")
if err != nil {
    return err
}

// Setup the exporter.
pusher, err := cortex.InstallNewPipeline(config)
if err != nil {
    return err
}

// Add instruments and start collecting data.
```

### 2. Setting up an Exporter

Users can setup the Exporter with the `InstallNewPipeline` function. It requires the `Config` struct
created in the first step and returns a push Controller that will periodically collect and push
data. Users can also use an optional `WithPeriod(time.Duration)` as a parameter in
`InstallNewPipeline()` to set a custom push interval. The default interval is 10 seconds.

Example:

```go
pusher, err := cortex.InstallNewPipeline(config)
if err != nil {
    return err
}

// Make instruments and record data
```

### 3. Set up backend

Set up your desired backend, like Cortex, and start receiving data from the Exporter.

## Repository Structure

- `cortex/`
  - `auth.go`
  - `auth_test.go`
  - `cortex.go`
  - `cortex_test.go`
  - `testutil_test.go`
  - `config.go`
  - `config_test.go`
  - `config_data_test.go`
  - `sanitize.go`
  - `sanitize_test.go`
  - `go.mod`
  - `go.sum`
  - `README.md`
  - `utils/`
    - `config_utils.go`
    - `config_utils_test.go`
    - `config_utils_data_test.go`
    - `go.mod`
    - `go.sum`
  - `example/`
    - `main.go`
    - `go.mod`
    - `go.sum`
    - `config.yml`
    - `cortexConfig.yml`
    - `docker-compose.yml`
    - `README.md`
  - `pipeline/`
    - Will include info on E2E testing

## Testing

The exporter can be tested using the standard Go testing library. Here are different ways
to run the tests from the terminal:

```bash
# Run all tests in x_test.go files.
go test

# Run a specific test (X is the name of the test)
go test -run X 

# Run a specific test and store the cpu profile in cpu.prof
 go test -run X -cpuprofile cpu.prof

# Run a specific test and store the memory profile in mem.prof
 go test -run X -memprofile mem.prof
```

Users can check the cpu / memory profiles by using the `pprof` tool:

```bash
# Open pprof tool
go tool pprof cortex.test cpu.prof

# Check top 20 nodes that used the most cpu time
top 20
```

The exporter also provides two pipeline tests for validating the exporter. They can be
found on our team's public `opentelemetry-go-contrib` fork on the `pipeline` branch
[here](https://github.com/open-o11y/opentelemetry-go-contrib/tree/pipeline/exporters/metric/cortex/pipeline).
Instructions for running the two pipeline tests are located in the README.


## Future Enhancements

We have documented several future enhancements:

- Adding tests for histogram buckets and distribution quantiles
- Improving ValidCheckpointSet tests in `cortex_test.go`
  - Update `getValidCheckpointSet` to generate more records of different types
  - Update `wantValidCheckpointSet`
  - Update `wantLength`
- Add code coverage to project with a badge
- Increase configuration by allowing users to choose selectors
- Update tests to use new tests package from this [pull request](https://github.com/open-telemetry/opentelemetry-go/pull/1040)
- Refactor `CreateLabelSet` to use `KeyValue` structs from the `labels` package



## Pull Requests Filed and Merged

- [Cortex Exporter Project setup #202](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/202)
- [Cortex Exporter Setup Pipeline and Configuration #205](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/205)
- [Cortex Exporter Send Pipeline #210](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/210)
- [Add convertToTimeseries for Sum, LastValue, and MinMaxSumCount #211](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/211)
- [Add distribution and histogram #237](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/237)
- [Cortex example project #238](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/238)
- [Authentication Implementation and Timestamp fix #246](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/246)
- [Fix Panic Issue in MutualTLS Test #315](https://github.com/open-telemetry/opentelemetry-go-contrib/pull/315)
## Reference Documents

Designs for the Exporter can be found in our
[public documents repository](https://github.com/open-o11y/docs/blob/master/exporter/go-prometheus-remote-write/design-doc.md).

A simple usage example with explanation can be found on the
[OpenTelemetry Go SDK Contribution repository](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/master/exporters/metric/cortex/example)

## Contributors

- [Connor Lindsey](https://github.com/connorlindsey)

- [Eric Lee](https://github.com/ercl)
