# Cortex Exporter Pipeline Test

This package contains utilities for testing the Cortex exporter. It has the following components:

- a [data generator](data.go) that randomly generates and writes metrics in the following format to a text file:
```  		 
 name, type, label1 labelvalue1 , value1 value2 value3 value4 value5
```
- a [OTLP sender](otlp.go) that reads from the text file, then builds and send metrics to the collector.

- a [querier](querier.go) that reads from the text file, query metrics in it, and writes the result to another
text file in the same format as the input file. 

- Input and output file path, metric type, number of metrics, labels, and value bounds of the 
 generated metrics are all defined [here](main.go).

## Running the Pipeline Test

To run the test, you need to first [setup a Cortex instance](https://cortexmetrics.io/docs/getting-started/getting-started-chunks-storage/)
and update the endpoint value in the sample [Collector configuration](otel-collector-config.yaml) and in [main.go](main.go).

Then, run the following command to start the test:

```$xslt
make testaps
```

This builds and runs the Collector, starts the data generator, the OTLP sender, and the querier. After the command finishes,
the content of the [input text file](data.txt) and the [output file](ans.txt) should be the same. Each querying requst 
is AWS sig V4 signed. 
