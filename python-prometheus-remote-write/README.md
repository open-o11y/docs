# OpenTelemetry Python SDK Prometheus Remote Write Exporter
This package contains an exporter to send [OTLP](https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/protocol/otlp.md)
metrics from the Python SDK directly to a Prometheus Remote Write integrated
backend (such as Cortex or Thanos) without having to run an instance of the
Prometheus server. The image below shows the two Prometheus exporters in the OpenTelemetry Python SDK.


Pipeline 1 illustrates the setup required for a Prometheus "pull" exporter.


Pipeline 2 illustrates the setup required for the Prometheus Remote Write exporter.

![Prometheus SDK pipelines](https://user-images.githubusercontent.com/20804975/100285430-e320fd80-2f3e-11eb-8217-a562c559153c.png)


The Prometheus Remote Write Exporter is a "push" based exporter and only works with the OpenTelemetry [push controller](https://github.com/open-telemetry/opentelemetry-python/blob/master/opentelemetry-sdk/src/opentelemetry/sdk/metrics/export/controller.py).
The controller periodically collects data and passes it to the exporter. This
exporter then converts the data into [`timeseries`](https://prometheus.io/docs/concepts/data_model/) and sends it to the Remote Write integrated backend through HTTP
POST requests. The metrics collection datapath is shown below:

![controller_datapath_final](https://user-images.githubusercontent.com/20804975/100486582-79d1f380-30d2-11eb-8d17-d3e58e5c34e9.png)

See the `example` folder for a demo usage of this exporter

# Table of Contents
   * [Summary](#opentelemetry-python-sdk-prometheus-remote-write-exporter)
   * [Table of Contents](#table-of-contents)
      * [Installation](#installation)
      * [Quickstart](#quickstart)
      * [Configuring the Exporter](#configuring-the-exporter)
      * [Securing the Exporter](#securing-the-exporter)
         * [Authentication](#authentication)
         * [TLS](#tls)
      * [Supported Aggregators](#supported-aggregators)
      * [Error Handling](#error-handling)
      * [Retry Logic](#retry-logic)
      * [Contributing](#contributing)
         * [Design Doc](#design-doc)

## Installation

* To install from the latest PyPi release,
  run `pip install opentelemetry-exporter-prometheus-remote-write`
* To install from the local repository, run
  `pip install -e exporter/opentelemetry-exporter-prometheus-remote-write/` in
  the project root

## Quickstart

```python
from opentelemetry import metrics
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.exporter.prometheus_remote_write import (
    PrometheusRemoteWriteMetricsExporter
)

# Sets the global MeterProvider instance
metrics.set_meter_provider(MeterProvider())

# The Meter is responsible for creating and recording metrics. Each meter has a unique name, which we set as the module's name here.
meter = metrics.get_meter(__name__)

exporter = PrometheusRemoteWriteMetricsExporter(endpoint="endpoint_here") # add other params as needed

metrics.get_meter_provider().start_pipeline(meter, exporter, 5)
```

## Configuring the Exporter

The exporter can be configured through parameters passed to the constructor.
Here are all the options:

* `endpoint`: url where data will be sent **(Required)**
* `basic_auth`: username and password for authentication **(Optional)**
* `headers`: additional headers for remote write request as determined by the remote write backend's API **(Optional)**
* `timeout`: timeout for requests to the remote write endpoint in seconds **(Optional)**
* `proxies`: dict mapping request proxy protocols to proxy urls **(Optional)**
* `tls_config`: configuration for remote write TLS settings **(Optional)**

Example with all the configuration options:

```python
exporter = PrometheusRemoteWriteMetricsExporter(
    endpoint="http://localhost:9009/api/prom/push",
    timeout=30,
    basic_auth={
        "username": "user",
        "password": "pass123",
    },
    headers={
        "X-Scope-Org-ID": "5",
        "Authorization": "Bearer mytoken123",
    },
    proxies={
        "http": "http://10.10.1.10:3000",
        "https": "http://10.10.1.10:1080",
    },
    tls_config={
        "cert_file": "path/to/file",
        "key_file": "path/to/file",
        "ca_file": "path_to_file",
		"insecure_skip_verify": true, # for developing purposes
    }
)

```
## Securing the Exporter

### Authentication

The exporter provides two forms of authentication which are shown below. Users
can add their own custom authentication by setting the appropriate values in the `headers` dictionary

1. Basic Authentication: 
Basic authentication sets a HTTP Authorization header containing a base64 encoded username/password pair. See [RFC 7617](https://tools.ietf.org/html/rfc7617) for more information.

```python
exporter = PrometheusRemoteWriteMetricsExporter(
    basic_auth={"username": "base64user",  "password": "base64pass"}
)
```
2. Bearer Token Authentication: 
This custom configuration can be achieved by passing in a custom `header` to
the constructor. See [RFC 6750](https://tools.ietf.org/html/rfc6750) for more information.


```python
header = {
    "Authorization": "Bearer mytoken123"
}
```

### TLS
Users can add TLS to the exporter's HTTP Client by providing certificate and key files in the `tls_config` parameter.

## Supported Aggregators

* Sum
* MinMaxSumCount
* Histogram
* LastValue
* ValueObserver

## Error Handling
In general, errors are raised by the calling function. The exception is for failed requests where any error status code is logged as a warning instead.

This is because the exporter does not implement any retry logic as it sends cumulative metrics data. This means that data will be preserved even if some exports fail.

For example, consider a situation where a user increments a Counter instrument 5 times and an export happens between each increment. If the exports happen like so:
```
SUCCESS FAIL FAIL SUCCESS SUCCESS
1       2    3    4       5
```
Then the recieved data will be:
```
1 4 5
```
The end result is the same since the aggregations are cumulative.

## Contributing

This exporter's datapath is as follows:

![Exporter datapath](https://user-images.githubusercontent.com/20804975/100285717-604c7280-2f3f-11eb-9b73-bdf70afce9dd.png)
*Entites with `*` after their name are not actual classes but rather logical
groupings of functions within the exporter.*

If you would like to learn more about the exporter's structure and design decisions please view the design document below.

### Design Doc

[Design Document](https://github.com/open-o11y/docs/tree/master/python-prometheus-remote-write)

This document is stored elsewhere as it contains large images which will significantly increase the size of this repo.
