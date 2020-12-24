# ElasticSearch Exporter Design

## Introduction 

This document outlines the design of an Elasticsearch log exporter for the C++ Logging Library, which is one of the goals from GitHub issue ([#337](https://github.com/open-telemetry/opentelemetry-cpp/issues/337)). This exporter converts logs into a JSON object, which is then set as the body of an HTTP request, and this request is sent to Elasticsearch by a specified URL. The design of the Elasticsearch exporter should agree with three design tenants:

1. **Reliability** - Communicating with a server through HTTP means that there are many potential errors, such as packet loss or request timeout. The exporter must be able to handle these errors in a way that the data has the highest chance of reaching Elasticsearch, as well as not blocking indefinitely. 
2. **Extensibility -** The exporter has similarities between all the other C++ log exporters. It must implement from the SDK’s exporter interface to keep this similarity, which makes connecting log exporters to the SDK much simpler.
3. **Flexibility -** The logs being sent to Elasticsearch have some variation in terms of the attributes, resources, and whether a given Log Data Model field is included. The exporter must be able to send the logs, regardless of the data stored inside of it.

## Design Choices

Several high level design choices were decided in before the design of the Elasticsearch exporter could be formalized:
**(1) Exporter Options**

When an exporter is created, it requires certain parameters to be passed to it that specify its endpoint, the username/password, and any other required information specific to it. The [SDK environmental variables](https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/sdk-environment-variables.md#language-specific-environment-variables) spec defines environmental variables for each existing exporter, which can be expanded to include Elasticsearch by adding the following environmental variables to the table within the spec:

|Name	|Description	|Default	|
|---	|---	|---	|
|OTEL_EXPORTER_ELASTICSEARCH_HOST	|Hostname for the Elasticsearch exporter	|
|OTEL_EXPORTER_ELASTICSEARCH_PORT	|Port for the Elasticsearch exporter	|9200	|
|OTEL_EXPORTER_ELASTICSEARCH_ENDPOINT	|HTTP endpoint for Elasticsearch	|"http://localhost:9200/logs/_doc?pretty"	|

In the C++ SDK however, environmental variables are not being used. Instead, options classes are created that contains the environmental variables as fields, and are set to the default value if not specified. This is what is done inside the OTLP exporter, as seen by the [OtlpExporterOptions class](https://github.com/open-telemetry/opentelemetry-cpp/blob/master/exporters/otlp/include/opentelemetry/exporters/otlp/otlp_exporter.h#L14). For the sake of time management and consistency, an ElasticSearchExporterOptions class will be used instead.

**(2) Writing to Elasticsearch API**

Elasticsearch provides many APIs that make it easy to interact with the data stored inside. There are APIs for [searching the data,](https://www.elastic.co/guide/en/elasticsearch/reference/current/search-search.html) but of relevance to the Elasticsearch exporter are the [APIs for](https://www.elastic.co/guide/en/elasticsearch/reference/current/docs-index_.html)[storing data](https://www.elastic.co/guide/en/elasticsearch/reference/current/docs-index_.html). The generic endpoint used to store data inside Elasticsearch is `POST host:port/<target>/<type>`, where `<target>` is the index to write the data to, and `<type>` is either _doc or _bulk. 

Using the _doc endpoint allows writing a single JSON object to Elasticsearch, whereas the _bulk endpoint allows writing any number of JSON objects within the same request. For the Elasticsearch exporter, it will typically be exporting batches of logs, so it is more efficient to write a request that contains many log’s information instead of just one. Therefore, the _bulk endpoint will be used. An example request to Elasticsearch looks like:

```
curl -X POST 'http://localhost:9200/logs/_bulk' -H 'Content-Type: application/json' --data-binary '
{"index" : {}}
{"name": "Log1","timestamp": 14}
{"index":{}}
{"name": "Log2","timestamp": 21}
'
```

The `{“index” : {}}` tells Elasticsearch that the JSON object following it does not specify the index, and to use the index specified from the endpoint. In the example above, the index was `logs`.

**(3) JSON library**

The format of data that Elasticsearch expects is JSON. The exporter could either provide a custom JSON log formatter, or instead use an existing JSON library. The [Nlohmann JSON library](https://github.com/nlohmann/json) for C++ is very lightweight, easy to use, has no restrictive licence attached to it, and has an established reputation. Therefore, a custom JSON converter will not be made, and the Nlohmann library will be used instead.


## Design Overview

The following UML diagram shows all the components present in the design of the Elasticsearch exporter. Each of the components in red will be described in more detail below. (the components in green are interfaces from the SDK and grey being third party libraries)

[Image: ES_Exporter-6.png]



### ESLogExporterOptions Struct

The Elasticsearch log exporter needs to have several values specified when it is being configured. Typically this would be done through a config file or though environment variables, but the OTel C++ project doesn’t have any examples of using this. Instead, these parameters are passed through an ESLogExporterOptions struct, which has all the fields required for the exporter to get initialized. This is similarly done within the [OTLP exporter](https://github.com/open-telemetry/opentelemetry-cpp/blob/master/exporters/otlp/include/opentelemetry/exporters/otlp/otlp_exporter.h#L14), where the options class contains `std::string endpoint = "localhost:55680";`, 

For the Elasticsearch exporter, the ESLogExporterOptions struct will contain the following fields:

```
struct ESLogExporterOptions
    string host = "localhost"
    int port = 9200
    string index = "logs"
    int response_timeout = 30
    bool console_debug = false
```



### ESLogExporter Class

The ESLogExporter class is derived from the LogExporter interface defined in the SDK. This class is responsible for the following pipeline: (1) receiving batches of Elasticsearch Recordables, (2) constructing an HTTP request with JSON from those recordables as the payload, (3) sending that HTTP request to Elasticsearch given the host:port/index from the ESLogExporterOptions class, and (4) blocking until the response message is received, and parsing the response body to determine whether the request was successful or not. Both of these classes have the following pseudocode:

```
class ESLogExporter : LogExporter
    private options_ : ESLogExporterOptions
    private is_shutdown_ : bool
    
    public Recordable MakeRecordable():
        return ESRecordable
        
    public ExportResult Export(list(ESRecordable) records):
        if is_shutdown_ is true:
            return failure
            
        // Convert the records from JSON into string
        string body = ""
        for all record in records:
           body += record.dump() 
        
        // Create a connection to Elasticsearch
        session = session_manager.CreateSession(options.host, options.port)
        request = session.CreateRequest()
        
        // Populate the request and send it
        request.SetType(POST)
        request.AddHeader("type", "JSON")
        request.AddBody(body)
        ResponseHandler response
        request.AddResponseHandler(response)
        bool result = session.SendRequest()
        response.WaitForResponse()
        
        // Return fail if the result of the send is failure
        if result is NOT true:
            return failure
            
        // Get the response body and make sure it doesn't contain failure='1'
        string body = response.GetResponseBody()
        if body contains "failure='1'":
            return failure
        return success
        
    public bool Shutdown():
        is_shutdown_ = true
```



### ResponseHandler Class

This class is used to receive and store information about the status of the HTTP request to Elasticsearch. When the response is received, it is sent to the OnResponse(string) method and if any error occurs, it is sent to the OnEvent(string) method. The pseudocode for this class is as follows:

```
class ResponseHandler : HTTP:EventHandler:
    string body
    bool response_received = false
    
    public OnResponse(string responsemessage):
        body = responsemessage
        response_received = true
        
    public OnEvent(HTTP_Event event)
        if event is bad
            response_received = true
    
    public WaitForResponse:
        while not response_received:
            do nothing
        return
```

### ESRecordable Class

The exporters are responsible for creating a concrete implementation for the recordable class, which gets passed to the SDK and populated with the log data. If the exporter doesn’t specify, by default it will return a `LogRecord` recordable which is a concrete definition defined in the SDK that stores each of the Log fields in its own datatype. For some exporters that have a specific format they are expecting, a custom implementation of the recordable is required. For Elasticsearch, the logs are written as a JSON object via HTTP, so a custom recordable implementation is required that converts logs into JSON. The `ESRecordable` class has many Set methods that stores the data inside a JSON object, and one `GetJSON()` method that returns the JSON blob. It has the following pseudocode:

```
class ESRecordable : SDK::Recordable:
    private json log_
    
    public SetName(string name):
        log_["name"] = name
        
    public SetBody(string body):
        log_["body"] = body
        
    // The other fields of the Log Data Model
    public Set...()
    
    public GetJSON():
        return log_
```

### Nlohmann::JSON Class

A header only .hpp file containing the implementation of a JSON class, which the `ESRecordable` class uses. It is populated in a similar manner as an `std::map`, where doing `my_json[“key”] = value` adds a `{“key”, value}` object into the `my_json` object. The data from within the JSON object can be retrieved by calling `my_json.dump()`. 


## **Testing Strategy**

To ensure the Elasticsearch exporter is working as expected, three different types of tests must be made:

1. **Unit tests:** The Elasticsearch exporter has several different components, and for each one a unit test must exist that focusses on testing the edge cases of it. The unit tests will cover four components of the Elasticsearch exporter:
    1. JSON object creation: No matter the information stored in the recordable, it must be able to be stuffed into a JSON object. The unit test should try writing logs with no information, logs with all the fields present, and logs that have different values stored in the attributes/resource. 
    2. Return Logic for the HTTP request: One of the main sources of error that a user would experience with this exporter is incorrectly setting the endpoint of Elasticsearch, or if the Elasticsearch instance becomes unresponsive. To ensure the user’s program doesn't block indefinitely, the Elasticsearch exporter must detect any errors that come up during the `Export()`, and return a status code of failure.
    3. Shutdown method: After the exporter’s Shutdown method is called, all future calls to `Export()` must immediately return failure.
    4. Mock Server tests: Using the [HTTP Server code](https://github.com/open-telemetry/opentelemetry-cpp/tree/master/ext/include/opentelemetry/ext/http/server), a mock Elasticsearch instance can be created that returns valid and invalid response messages depending on the log send to it. The server should be programmed to reject some logs, for example if the name is “Bad log”, and the unit test should check that the `Export()` method returns failure for this log. The server will accept other logs, and the unit test should check that success is returned.
2. **Integration tests:** Once the unit tests are passing, a complete pipeline test can be written and put into the `examples/` directory in the cpp repo. This will showcase the functionality of the Elasticsearch exporter, and also give developers wanting to use the exporter an example of how to do it properly.
3. **Benchmark tests:** Additionally, if there is time, we may do some testing to determine the performance of the Elasticsearch instance given different parameters. This is useful since the exporter is what interacts with outside components, so having a stable performance is important.

## References

[What is ElasticSearch](https://medium.com/better-programming/quick-start-elasticsearch-with-python-7756ea45d815)
[ElasticSearch Bulk API](https://www.elastic.co/guide/en/elasticsearch/reference/current/docs-bulk.html)
[Nlohmann JSON Library](https://www.elastic.co/guide/en/elasticsearch/reference/current/docs-bulk.html)
