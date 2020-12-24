# Logging Elasticsearch Exporter README

## Table of Contents

* Architecture Overview (add links in markdown)
* Usage
* Repository Structure
* Outstanding Tasks
* Pull Requests
* Reference Documents
* Contributors

## Architecture Overview

[Image: ES Exporter.png]

* **`ElasticsearchExporter`** class:
    * Inherits from the SDK’s LogExporter interface, and contains the implementation of the `Export()`, `Shutdown()`, and `MakeRecordable()` methods. 
* **`ElasticsearchExporterOptions`** class:
    * Contains five fields that are used to set the configuration options for the exporter. Once the fields are set, this class is sent to the main `ESLogExporter` class as an argument in the constructor. The five fields are shown below:

        1. **Host** - The host of the Elasticsearch instance to connect to
        2. **Port** - The port of the Elasticsearch instance to connect to
        3. **Index** - The index of the Elasticsearch instance to write the logs into
        4. **Timeout** - How long to wait for a response from Elasticsearch for
        5. **Console_debug** - Whether to print the status of the Exporter to console

* **`ElasticsearchRecordable`** class:
    * An implementation of the SDK’s Recordable interface which uses JSON to store the log data. This format is required since it is what [Elasticsearch’s index API](https://www.elastic.co/guide/en/elasticsearch/reference/current/docs-bulk.html) expects in a body of a  request. To convert logs into a JSON representation, the [Nlohmann JSON library](https://github.com/nlohmann/json) was used, since it has an open-source licence and is well established and tested.
* **`ResponseHandler`** class:
    * Receives the status of an HTTP request, using the [OTel C++ HTTP client](https://github.com/open-telemetry/opentelemetry-cpp/tree/master/ext/include/opentelemetry/ext/http/client) implementation. The `OnEvent(event)` method is called when some change occurs, for example if the request timed out or the connection was lost. The `OnResponse(response)` method is called when the response is received from Elasticsearch. Additionally, the main `Export()` method needs to wait until the response is received from Elasticsearch, so this class offers a `WaitForResponse()` method that blocks until this is true.

## Usage Example

1. Run an Elasticsearch instance
2. (*optional*) Run Kibana instance for data visualization
3. Create a basic C++ application with the Elasticsearch Exporter included
4. Build application with Bazel and execute it
5. View data on Kibana

An example is shown in the [logs-demo-example](https://github.com/open-o11y/opentelemetry-cpp/tree/logs-demo-example/examples/elasticsearch) team branch, within the  `/examples` folder.

## Respository Structure

**Interface**

* `LogExporter` interface is defined in `sdk/include/opentelemetry/sdk/logs`.
* `Recordable` that is exported is defined in `sdk/include/opentelemetry/sdk/logs`.


**Elasticsearch Exporter Implementation**

* All header files are located in the folder `exporters/elasticsearch/include/opentelemetry/exporters/elasticsearch`.
* All implementation files can be found in `exporters/elasticsearch/src`.
* The unit tests are located in `exporters/elasticsearch/test`

## Outstanding Tasks

* Convert to use Synchronous HTTP methods instead of Asynchronous. This will simplify the Export() logic, because it will automatically block the Export() method until the response is received, thereby removing the need for the ResponseHandler class. The progress of the Synchronous HTTP methods PR can [be found here](https://github.com/open-telemetry/opentelemetry-cpp/pull/448).
* Add additional unit tests using a Mock Server, similar to [what is done here](https://github.com/open-telemetry/opentelemetry-cpp/blob/master/ext/test/http/curl_http_test.cc#L52). Currently, the unit tests for the ES exporter only test the Shutdown logic and the invalid host:port logic. No tests exist that check what happens when an HTTP request is written to a valid connection to a Server. These tests should check for:
    * A Valid HTTP request is written to the server, but it exceeds the max timeout specified from the ElasticsearchExporterOptions class.
    * A Valid HTTP request is written to the server, but the server responds with a response message that is interpreted as a failure.
    * A Valid HTTP request is written to the server, and the server responds with a response message that is interpreted as a success.
* Add Performance and Benchmark tests. 
* Migrate the Exporter to the [Cpp-contrib repo](https://github.com/open-telemetry/opentelemetry-cpp-contrib).

## Pull Requests

[Elasticsearch Exporter](https://github.com/open-o11y/opentelemetry-cpp/pull/18)

## Reference Documents

* The Elasticsearch Exporter design doc can be found in the o11y team `/docs` repository 

## Contributors

* Mark Seufert
* Karen Xu

