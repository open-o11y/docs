# Metrics API/SDK Implementation Details

This document will represent the implementation details we plan on using when implementing the OpenTelemetry Metrics API & SDK in C++. **This is not a final, correct implementation. It is simply a snapshot of our current plan and is subject to change.**

### Class names (and naming convention)

**OpenTelemetry uses the [Google naming convention](https://google.github.io/styleguide/cppguide.html#Naming).**

* MeterProvider
* Meter
* Metric (Base Class) or Instrument (for parallel naming)
* BoundInstrument (Base Class)
* Counter, SumObserver, etc. (Inherit Instrument)
* BoundCounter, BoundSumObserver, etc (Inherit BoundInstrument)
    * Bound instruments simply add the bound prefix
* Accumulator/Meter SDK
* Processor
* Aggregator
    * Counter, MinMaxSumCount, Sketch, Histogram, gauge, and Exact
* Exporter (Base class)
    * OStreamExporter, PrometheusExporter, etc.
    * Name of the service as a prefix
* Controller (Base class)
    * PushController, PullController, etc.
    * Type of controller as a prefix

### Parameters

* MeterProvider
    * Two parameters in `get_meter` function: A string containing the library name and, optionally, a string containing the library version. If no version is supplied, it will default to the empty string.
* Meter
    * Each synchronous metric instrument constructor has four parameters: A string representing the name of the metric instrument, a string containing a description of the instrument, a string describing the unit of metric values to be captured by this instrument (https://unitsofmeasure.org/ucum.html), and a boolean representing whether or not the instrument is recording. Asynchronous instruments have one more parameter, the function callback to observe at each collection interval.
    * The `record_batch` function has two parameters: The first is a map of strings to strings containing the labels of the metric instruments the user would like to apply. The second is a sequence of pairs containing metric instruments, and the value to be recorded to that instrument.
* Metric
    * `Update`, `Add`, and `Record` functions requires the value to send for aggregation as well as a label set in the case of an unbound instrument.  
* Aggregators
    * `Merge` requires another aggregator of the same type.
    * `Update`, `Add`, and `Record` functions take either a value, or a value and labels to apply to the instrument.
* Controller
    * Depending on the controller type, specify the Meter, Exporter, Processor, and Interval. Essentially identifying which meter the controller governs, how data is exported, and how often to collect information from that Meter.
    * Class: PushController
* Accumulator/Meter SDK
    * Constructor parameters are the name and version of the meter.
    * Collect() takes no parameters.
    * Rest of the function are the same as the `Meter` class in the API.
* Processor
    * One parameter, State: The processor, if defined to be stateful, merges `records` with the same name and labels together over time to create cumulative metrics.
* Exporter
    * Similar to the accumulator, choosing the correct exporter is important but exporters are highly customized to their output service
    * Also needs to receive a collection of check-pointed records to be exported.

### Behavior

* MeterProvider
    * The class’s main function is to provide a `get_meter(name)` method which returns a meter for each module in an application.  Other than this, the class is basically a placeholder which maintains a connection between all the application modules. 
* Meter
    * The `Meter` class provides a facility to construct the various metric instruments as well as a way to record a batch of values to those instruments. Additionally, each instantiation of the `Meter` class will possess a container holding each metric instrument that was created from this `Meter`.
* Metric
    * Metric instruments capture raw measurements of designated quantities in instrumented applications.  All measurements captured by the Metrics API are associated with the instrument which collected that measurement. 
* Aggregators
    * Is an interface which implements a concrete strategy for aggregating metric updates. Several Aggregator implementations are provided by the SDK. Aggregators may be lock-free or use locking, depending on their structure and semantics. Aggregators implement an Update method, to receive a single metric event. Aggregators implement a Checkpoint method, called in collection context, to save a checkpoint of the current state. Aggregators implement a Merge method, also called in collection context, that combines state from two aggregators into one.
* Controller
    * The Controller orchestrates the export pipeline. The “Push” controller will establish a periodic timer to regularly collect and export metrics. A “Pull” controller will await a pull request before initiating metric collection. The overall job is the call the collect method, read the checkpoint, then invoke the exporter.
* Accumulator/Meter SDK
    * Collects on all instruments created from this meter at the request of the controller. Sends these collected records back to the controller.
* Processor
    * The Processor is an interface which sits between the SDK and an exporter. The Processor supports submitting checkpoint aggregators and produces a complete checkpoint set for the exporter with a possible dimensional reduction of the label set, and merging of cumulative metrics.
* Exporter
    * The Exporter is the final stage of an export pipeline. It is called with a checkpoint set capable of enumerating all of the updated metrics. The exporter then sends that checkpoint set to a specified backend service such as Prometheus.

### Error Handling Policy

One of OpenTelemetry’s design tenets is that the library will not hinder the operation of any instrumented applications.  As such, errors in the API cannot actually throw exceptions (also due to ABI stability being required).  Instead, any errors should be logged and tracked in a separate channel: STDERR or a dedicated export lane perhaps.

We will implement the necessary error handling in our implementation of the Metrics API and SDK. User-facing portions of the library will provide more robust, complete error handling. Adding more error handling will always be good, but we will first focus on getting the necessary error handling to start off.

* MeterProvider
    * MissingModuleName — if the instrumenting module name is not given in calls to get_meter()
* Meter
    * DuplicateRegistration — if a user attempts to register two metric instruments with the same name
    * DuplicateCallback — if a user specific more than one callback for an asynchronous instrument
    * InvalidName — if the metric name is empty or doesn’t conform to the spec naming convention
* Metric
    * No errors for this class since the Meter class serves as the manager for all Metrics and is the exit point for relevant data
* Controller, Exporter, Aggregator
    * There should be no errors as these all simple process metric event data with little user interference.  They should be robust and address all data types given to the
* processor
    * Depending on how far we get, the Processor will have to check for invalid filtering schemes

Due to the ABI stability requirement imposed on the project we are not able to throw exceptions in the API. Because of this, errors in the API will be handled gracefully and information surrounding the error will be logged. 

However, the SDK is not required to abide by the same ABI policy, meaning we can throw exceptions in the SDK. When an unrecoverable error is encountered the SDK will throw an exception. However, small errors that can be recovered from will log error information and return rather than throw an exception. This is to satisfy the specification’s requirement to be non-intrusive or disruptive to the instrumented library.

### Test cases

We intend to have 2 broad categories of tests. First, we will create a suite of cases for each segment to ensure that they work individually. For example, ensuring that a Counter aggregator’s maintained value is updated correctly when the function `Add` is called. Next, we need integration tests which assess the pipeline as a whole. Data should move correctly from the time it is triggered to the final exporting to whichever service we provide. A complete integration test-bed would include benchmark testing to analyze the performance of our design.

### Dependencies

* For ABI stability, there can be only be minimal STL use. Most data structures/algorithms we require must be implemented in the “nostd” package bundled with the library.  
    * Must be conscious of API function prototypes.

### File Structure

Mimics the structure of the Tracer API/SDK.

* `api/include/opentelemetry/metrics/ `
    * `Instrument.h`  — Base instrument classes
    * `Sync_instruments.h` — All synchronous instrument definitions
    * `Async_Instruments.h` — All asynchronous instrument definitions
    * `Noop.h` — Noop implementations for all API classes
    * `Provider.h`
    * `MeterProvider.h`
    * `Meter.h`
    * `Record.h`
* `sdk/include/opentelemetry/metrics/aggregator`
    * `aggregator.h` — Base aggregator class
    * `counter_aggregator.h`
    * `min_max_sum_counter_aggregator.h`
    * `histogram_aggregator.h`
    * `exact_aggregator.h`
    * `sketch_aggregator.h`
    * `gauge_aggregator.h`
* `sdk/include/opentelemetry/metrics/` — implementation of template classes must be in .h files
    * `Instrument.h`  
    * `Sync_instruments.h` 
    * `Async_Instruments.h` 
    * `Provider.h`
    * `MeterProvider.h`
    * `Meter.h`
    * `Record.h`
* `sdk/src/metrics`
    * `meter.cc`
    * `meter_provider.cc`
    * `ungrouped_processor.cc`
    * `controller.cc`
* `sdk/test/metrics`
    * `counter_aggregator_test.cc`
    * `min_max_sum_counter_aggregator_test.cc`
    * `histogram_aggregator_test.cc`
    * `exact_aggregator_test.cc`
    * `sketch_aggregator_test.cc`
    * `gauge_aggregator_test.cc`
    * `metric_instrument_test.cc`
    * `meter_test.cc`
