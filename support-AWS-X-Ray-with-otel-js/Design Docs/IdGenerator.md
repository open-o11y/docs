# Design Docs for AWS X-Ray IdGenerator

## Objective

Design and implement AWS X-Ray specific `IdGenerator` component in OpenTelemetry.

## Summary

![Data Path Diagram](../images/Instrumentation.png)OpenTelemetry aims to be a common API for applications to have instrumentation such as tracing. Because X-Ray is supported by the OpenTelemetry collector, a process that bridges the OpenTelemetry format with various backends, in theory any user of an OpenTelemetry language SDK can have their app traced and show up on the X-Ray console. In practice, because of some unique aspects of the X-Ray backend such as the format of IDs and propagation headers, there is language-specific work that must be done in an SDK to properly support X-Ray. We will implement these features for all the supported OpenTelemetry languages and aim to provide an out of the box experience that is as easy to get started with for a user to use OpenTelemetry with X-Ray. This document is framed with the work for supporting JavaScript - most concepts will translate well to other languages but some will be language specific. 
As for `IdGenerator`, it is a configurable part of `TracerProvider`. By setting different `IdGenerator`, `TracerProvider` can generate `TraceId` and `SpanId` following different rules.

## Goals & Non-Goals

### Goals

* Design the functionality and code structure to be implemented for both web and node version of AWS X-Ray.

### Non-Goals

* Describe the current implementation in Detail, pointing out the defects and the plan to improve it.
* Since OpenTelemetry group no longer holds vendor-specific IdGenerator. Here to describe and design the file structure and repository setup.

## Design

### 1. Current Implementation Analysis and Improvement

Currently, the implementation of IdGenerator has 2 crucial defects:

1. The functionality of IdGenerator is provided by 2 nude functions. On the one hand, providing 2 functions like: `randomTraceId()` and `randomSpanId()`,  can be kind of ugly and definitely not a good coding style. On the other hand, by providing 2 functions can be hard for `TracerProvider` to config since different functions do not share a common interface. As shown below:

    ```
    export function randomTraceId(): string {...}
    export function randomSpanId(): string {...}
    ```
2. The other defect is, current implementation of `TracerProvider` is fixed to use `randomTraceId()` and `randomSpanId()`, it is totally unacceptable because user can never adjust IdGenerator with fixed functionality. As shown below:

    ```
        const spanId = randomSpanId();
        let traceId;
        let traceState;
        if (!parentContext || !isValid(parentContext)) {
          // New root span.
          traceId = randomTraceId();
        } else {
          // New child span.
          ...
    ```

In order to fix these 2 defects, we have 2 corresponding design:

1. Migrate the original function-based IdGenerator to class-based IdGenerator, to complete this we need to:
    1. Define a standard interface for IdGenerator, as shown below,
    
        ```
        /** IdGenerator provides an interface for generating Trace Id and Span Id */
        export interface IdGenerator {
          /** Returns a trace ID composed of 32 lowercase hex characters. */
          generateTraceId(): string;
          /** Returns a span ID composed of 16 lowercase hex characters. */
          generateSpanId(): string;
        }
        ```
    3. Also, re-implement current IdGenerator functionality by using the interface above. The details of this step will be illustrated in correponding Implementation Details and Testing doc.
2. For `TracerProvider`, rather than directly using the ID generation functionality, we could add another IdGenerator attribute in `TracerConfig`, so user could adjust IdGenerator by modify configuration file. As shown below:

     ```
     /**
      * TracerConfig provides an interface for configuring a Basic Tracer.
      */
     export interface TracerConfig {
       ...
       /**
        * Generator of trace and span IDs
        * The default idGenerator generates random ids
        */
       idGenerator?: IdGenerator;
     }
     ```

### 2. AWS X-Ray IdGenerator

By default, OpenTelemetry uses purely random trace IDs, which differs from X-Ray where the first 4 bytes of the trace ID must be set to the start time of the trace. OpenTelemetry provides an extension point, `IdsGenerator` to allow us to use a custom generator that conforms to the X-Ray requirement.
As for span ID, in X-Ray, it is also randomly generated 8 bytes number.
Also, note that there is a language specific feature: since JavaScript is a language can support both front-end(web version) and back-end(node version), IdGenerator also needs to support both web and node version.
The core functions can be shown below:
**Node Version**

```
AWSXrayTraceId(): string {
  var nowSec = Math.floor(Date.now() / 1000).toString(16);
  return nowSec + crypto.randomBytes(12).toString('hex');
}
```

**Web Version**
Since in web version, JavaScript may not support `crypto` library, we need to first define a way to generate random number:

```
type WindowWithMsCrypto = Window & {
  msCrypto?: Crypto;
};
const cryptoLib = window.crypto || (window as WindowWithMsCrypto).msCrypto;
```

Then take use of the newly defined `cryproLib` to generate random bytes:

```
  AWSXrayTraceId(): string {
    const date = new Date();
    const nowSec = Math.floor(date.getTime() / 1000).toString(16);
    cryptoLib.getRandomValues(randomBytesArray);
    return (
      nowSec +
      this.toHex(randomBytesArray.slice(0, TRACE_ID_BYTES - TIME_BYTES))
    );
  }
```