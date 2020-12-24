# Design Docs for AWS X-Ray Propagator

## Objective

Design and implement AWS X-Ray specific `Propagator` component in OpenTelemetry.

## Summary

![Data Path Diagram](../images/Instrumentation.png)OpenTelemetry aims to be a common API for applications to have instrumentation such as tracing. Because X-Ray is supported by the OpenTelemetry collector, a process that bridges the OpenTelemetry format with various backends, in theory any user of an OpenTelemetry language SDK can have their app traced and show up on the X-Ray console. In practice, because of some unique aspects of the X-Ray backend such as the format of IDs and propagation headers, there is language-specific work that must be done in an SDK to properly support X-Ray. We will implement these features for all the supported OpenTelemetry languages and aim to provide an out of the box experience that is as easy to get started with for a user to use OpenTelemetry with X-Ray. This document is framed with the work for supporting JavaScript - most concepts will translate well to other languages but some will be language specific. 
For `Propagator`, Cross-cutting concerns send their state to the next process using `Propagator`s, which are defined as objects used to read and write context data to and from messages exchanged by the applications. Each concern creates a set of `Propagator`s for every supported `Format`. Propagators leverage the `Context` to inject and extract data for each cross-cutting concern, such as traces and correlation context.

## Goals & Non-Goals

### Goals

* Design the functionality and code structure to be implemented for AWS X-Ray propagator.

### Non-Goals

* Since OpenTelemetry group no longer holds vendor-specific propagators. Also need to do repository setup.

## Design

By default, OpenTelemetry uses the W3C Trace Context format for propagating spans, and out-of-the-box supports B3 and Jaeger propagation. The `HttpTextFormat` interface allows other propagators to be implemented, so we implement a propagator that conforms with the X-Ray trace header format. 
As illustrated in specification(link in appendix), a standard propagator should contain 3 interfaces:

### Fields

The propagation fields defined. If carrier is reused, we should delete the fields here before calling [inject](https://github.com/open-telemetry/opentelemetry-specification/blob/b338f9f63dbf02ff8ebd100e8a847e7bf43e2682/specification/context/api-propagators.md#inject).
For example, if the carrier is a single-use or immutable request object, we don't need to clear fields as they couldn't have been set before. If it is a mutable, retryable object, successive calls should clear these fields first.
**In Javascript standard, we do not need to provide a Field interface**

### Inject
Injects the value downstream.
It should accept 3 params:

```
inject(context: Context, carrier: unknown, setter: SetterFunction): void;
```

**Context**
context to be extracted from or to be injected into carrier. Encoding is expected to conform to the HTTP Header Field semantics. Values are often encoded as RPC/HTTP request headers.
**Carrier**
The The carrier of propagated data on both the client (injector) and server (extractor) side is usually an object such as http headers. Propagation is usually implemented via library-specific request interceptors, where the client-side injects values and the server-side extracts them.
It is the carrier of propagation fields, such as http request headers. In our case, it may be AWS [Xray header](https://docs.aws.amazon.com/xray/latest/devguide/xray-concepts.html#xray-concepts-tracingheader): 
**Setter**
Setter is an important argument in `Inject` that sets value into given field. Setter allows a `HTTPTextFormat` to set propagated fields into a carrier and MUST be stateless and allowed to be saved as a constant to avoid runtime allocations.
The existing api provide [setter implementation](https://github.com/open-telemetry/opentelemetry-js/blob/master/packages/opentelemetry-api/src/context/propagation/setter.ts):

```
export type SetterFunction<Carrier = any> = (
  carrier: Carrier,
  key: string,
  value: unknown
) => void;

export function defaultSetter(carrier: any, key: string, value: unknown) {
  carrier[key] = value;
}
```

### Extract
Extracts the value from an incoming request. For example, from the headers of an HTTP request. Given a Context and a carrier, extract context values from a carrier and return a new context, created from the old context, with the extracted values. The interface looks like

```
extract(context: Context, carrier: unknown, getter: GetterFunction): Context;
```

context and carrier are simliar to what in Inject method
**Getter**

```
export type GetterFunction<Carrier = any> = (
  carrier: Carrier,
  key: string
) => unknown;

export function defaultGetter(carrier: any, key: string): unknown {
  return carrier[key];
}
```

Note that in JS context item is mandatory (different from Java)

### Code Structure

Generally, our propagator should follow the structure defined in propagation API (link in appendix):

```
/**
 * Injects `Context` into and extracts it from carriers that travel
 * in-band across process boundaries. Encoding is expected to conform to the
 * HTTP Header Field semantics. Values are often encoded as RPC/HTTP request
 * headers.
 *
 * The carrier of propagated data on both the client (injector) and server
 * (extractor) side is usually an object such as http headers. Propagation is
 * usually implemented via library-specific request interceptors, where the
 * client-side injects values and the server-side extracts them.
 */
export interface HttpTextPropagator {
  /**
   * Injects values from a given `Context` into a carrier.
   *
   * OpenTelemetry defines a common set of format values (HttpTextPropagator),
   * and each has an expected `carrier` type.
   *
   * @param context the Context from which to extract values to transmit over
   *     the wire.
   * @param carrier the carrier of propagation fields, such as http request
   *     headers.
   * @param setter a function which accepts a carrier, key, and value, which
   *     sets the key on the carrier to the value.
   */
  inject(context: Context, carrier: unknown, setter: SetterFunction): void;
  /**
   * Given a `Context` and a carrier, extract context values from a
   * carrier and return a new context, created from the old context, with the
   * extracted values.
   *
   * @param context the Context from which to extract values to transmit over
   *     the wire.
   * @param carrier the carrier of propagation fields, such as http request
   *     headers.
   * @param getter a function which accepts a carrier and a key, and returns
   *     the value from the carrier identified by the key.
   */
  extract(context: Context, carrier: unknown, getter: GetterFunction): Context;
}
```

## Appendix

The specification for propagators can be found: https://github.com/open-telemetry/opentelemetry-specification/blob/b338f9f63dbf02ff8ebd100e8a847e7bf43e2682/specification/context/api-propagators.md
Javascript propagation API: https://github.com/open-telemetry/opentelemetry-js/blob/master/packages/opentelemetry-api/src/context/propagation/HttpTextPropagator.ts
The standard of format can be found: https://docs.aws.amazon.com/xray/latest/devguide/xray-concepts.html#xray-concepts-tracingheader
Java Implementation: https://github.com/open-telemetry/opentelemetry-java/blob/a59748a904e0cb1e2e4f8df50dd346a82f26ff1e/extensions/trace_propagators/src/main/java/io/opentelemetry/extensions/trace/propagation/AwsXRayPropagator.java#L51
The API to be used can be found:  https://github.com/open-telemetry/opentelemetry-js/tree/master/packages/opentelemetry-api/src/context/propagation