Introduction

This document outlines a proposed design for the AWS X-Ray Propagator component in the OpenTelemetry Go SDK.

Summary

![image](./images/architecture.png)

The ability to correlate events across service boundaries is one of the principle concepts behind distributed tracing. To find these correlations, components in a distributed system need to be able to collect, store, and transfer metadata referred to as context. Propagators are configured inside Tracer objects in order to support transferring of context across process boundaries. A context will often have information identifying the current span and trace, and can contain arbitrary correlations as key-value pairs. Propagation is when context is bundled and transferred across services, often via HTTP headers. This is done by first injecting context into a request and then this is extracted by a receiving service which can then make additional requests, and inject context to be sent to other services and so on. Together, context and propagation represent the engine behind distributed tracing. 

Objective

The objective of the AWS X-Ray propagator is to provide HTTP header propagation for systems that are using AWS X-Ray HTTP header format (https://docs.aws.amazon.com/xray/latest/devguide/xray-concepts.html#xray-concepts-tracingheader). Without the proper HTTP headers, AWS X-Ray will not be able to pick up any traces and it’s metadata sent from the collector. The AWS X-Ray propagator translates the OpenTelemetry SpanContext into the equivalent AWS X-Ray header format, for use with the OpenTelemetry Go SDK. By default, OpenTelemetry uses the W3C Trace Context format (https://www.w3.org/TR/trace-context/)for propagating spans which is different than what AWS X-Ray takes in:

The following is an example of W3 trace header with root traceID.

traceparent: 5759e988bd862e3fe1be46a994272793 tracestate:optional

The following is an example of AWS X-Ray trace header with root trace ID and https://quip-amazon.com/H3ObAQoYvO8v/OpenTelemetry-Go-SDK-enhancements-for-AWS-X-Ray-Requirements-Document#ETI9CAnAh4Q:

X-Amzn-Trace-Id: Root=1-5759e988-bd862e3fe1be46a994272793;Sampled=1



Design Tenets

1. *Security* - Data will not be modified during the injection and extraction of the headers, it will only inject and extract from the headers of an HTTP request
2. *Test Driven Development* - We will follow TDD practices established by the team and ensure proper test coverage (at least 90%).
3. *Reliability* - The propagator should be reliable by gracefully handling errors such as invalid traceID, empty context, etc, and empty headers. As per the specifications (https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/context/api-propagators.md), if a value cannot be parsed from a carrier it will not throw an exception and not store a new value in Context, in order to preserve any previously existing valid value.
4. *Go Best Practices -* The propagator will conform to best practices for Go as described in Effective Go (https://golang.org/doc/effective_go.html).



Design Details

By default, OpenTelemetry uses the W3C Trace Context format (https://www.w3.org/TR/trace-context/) for propagating spans, and out-of-the-box supports B3 and Jaeger propagation. The TextMap interface allows other propagators to be implemented, so we implement a propagator that conforms with the X-Ray trace header format (https://docs.aws.amazon.com/xray/latest/devguide/xray-concepts.html#xray-concepts-tracingheader).

As described in specification (https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/context/api-propagators.md), a standard propagator should have the following methods

* inject
* extract
* fields

The propagator containing the mentioned methods should be held in a struct as shown below:

type AWSXRay struct {
    /**
    struct used to instantiate AWS X-Ray propagator  
    */
}

Fields

Fields refer to the predefined propagation fields. If the carrier is reused, the fields should be deleted before calling inject (https://github.com/open-telemetry/opentelemetry-specification/blob/b338f9f63dbf02ff8ebd100e8a847e7bf43e2682/specification/context/api-propagators.md#inject).
For example, if the carrier is a single-use or immutable request object, we don't need to clear fields as they couldn't have been set before. If it is a mutable, returnable object, successive calls should clear these fields first. This will return a list of fields that will be used by the TextMapPropagator.


func (awsXRay AWSXRay) Fields() []string {
  /**
  returns a list of fields that will be used by the TextMapPropagator.
  */
}

*Inject
*

The inject method injects the AWS X-Ray values into the header. The implementation should accept 2 parameters, the context format for propagating spans and textMapCarrier interface allowing our propagator to be implemented.

Inject(ctx context.Context, carrier otel.TextMapCarrier) {
   /**
   * Injects values from a given `Context` into a carrier
    * as AWS X-Ray headers.
   *
   * OpenTelemetry defines a common set of format values (TextMapPropagator),
   * and each has an expected `carrier` type.
   *
   * @param context the Context from which to inect values to transmit
   *     
   * @param carrier the carrier of propagation fields, such as http request
   *     headers.
   */
}

*Extract*

Extract is required in a propagator to extract the value from an incoming request. For example, the values from the headers of an HTTP request are extracted. Given a context and a carrier, extract(), extracts context values from a carrier and return a new context, created from the old context, with the extracted values. The Go SDK extract method should accept 2 parameters, the context and textMapCarrier interface.

Extract(ctx context.Context, carrier otel.TextMapCarrier) context.Context
/**
   * Given a `Context` and a carrier, extract context values from a
   * carrier and return a new context, created from the old context, with the
   * extracted values if the carrier contains AWS X-Ray headers. 
   *
   * @param context the Context from which to extract values to transmit over
   *     the wire.
   * @param carrier the carrier of propagation fields, such as http request
   *     headers.
   */



Test Strategy

We will follow TDD practices while completing this project. We’ll write unit tests before implementing production code. Tests will cover normal and abnormal inputs and test for edge cases. The standard Go testing library (https://golang.org/pkg/testing/) will be used for writing and running the unit tests. Go cmp (https://github.com/google/go-cmp) will be used to handle comparison of the headers.


Appendix

* https://quip-amazon.com/H3ObAQoYvO8v/OpenTelemetry-Go-SDK-enhancements-for-AWS-X-Ray-Requirements-Document
* OpenTelemetry (https://opentelemetry.io/)
* Go Test Package (https://golang.org/pkg/testing/)
* Propagator Specification (https://github.com/open-telemetry/opentelemetry-specification/blob/b338f9f63dbf02ff8ebd100e8a847e7bf43e2682/specification/context/api-propagators.md)
* AWS X-Ray Tracing Header (https://docs.aws.amazon.com/xray/latest/devguide/xray-concepts.html#xray-concepts-tracingheader)
* OpenTelemetry Specification (https://github.com/open-telemetry/opentelemetry-specification/blob/b338f9f63dbf02ff8ebd100e8a847e7bf43e2682/specification/overview.md#resources)

