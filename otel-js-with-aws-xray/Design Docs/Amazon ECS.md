# Design Doc for ECS Plugin Resource Detector

## Objective

Design and implement AWS ECS resource detector component in OpenTelemetry.

## Summary

![Data Path Diagram](../images/Instrumentation.png)

A [Resource](https://github.com/open-telemetry/opentelemetry-specification/blob/b338f9f63dbf02ff8ebd100e8a847e7bf43e2682/specification/overview.md#resources) is an immutable representation of the entity producing telemetry. For example, a process producing telemetry that is running in a container on Kubernetes has a Pod name, it is in a namespace and possibly is part of a Deployment which also has a name. All three of these attributes can be included in the `Resource`.
The primary purpose of resources as a first-class concept in the SDK is decoupling of discovery of resource information from exporters. This allows for independent development and easy customization for users that need to integrate with closed source environments. The SDK MUST allow for creation of `Resources` and for associating them with telemetry.
When used with distributed tracing, a resource can be associated with the [TracerProvider](https://github.com/open-telemetry/opentelemetry-specification/blob/b338f9f63dbf02ff8ebd100e8a847e7bf43e2682/specification/trace/sdk.md#tracer-sdk) when it is created. That association cannot be changed later. When associated with a `TracerProvider`, all `Span`s produced by any `Tracer` from the provider MUST be associated with this `Resource`.

## Goal

* Design the functionality and code structure to be implemented for AWS ECS resource detector.

## Design

`Resource` is used to define attributes of the application itself, for example the cloud environment it is running on. This corresponds with the plugins in the X-Ray SDKs. We implement `Resource`s that populate attributes we expect for AWS users. We can implement a `Resource` for each of our plugins and merge them together.
In opentelemetry-js repository, different from the content in [specification doc](https://github.com/open-telemetry/opentelemetry-specification/blob/b338f9f63dbf02ff8ebd100e8a847e7bf43e2682/specification/resource/sdk.md), Javascript repository does not explicitly provide `create()` and `merge()` method.

### create()

The interface MUST provide a way to create a new resource, from a collection of attributes. Examples include a factory method or a constructor for a resource object. A factory method is recommended to enable support for cached objects.
In JavaScript SDK, this part of functionality has been supported:

```
  static createTelemetrySDKResource(): Resource {
    return new Resource({
      [TELEMETRY_SDK_RESOURCE.LANGUAGE]: SDK_INFO.LANGUAGE,
      [TELEMETRY_SDK_RESOURCE.NAME]: SDK_INFO.NAME,
      [TELEMETRY_SDK_RESOURCE.VERSION]: SDK_INFO.VERSION,
    });
  }
```

### merge

In the specification document, it says the interface MUST provide a way for a primary resource and a secondary resource to be merged into a new resource.
The resulting resource MUST have all attributes that are on any of the two input resources. Conflicts (i.e. a key for which attributes exist on both the primary and secondary resource) MUST be handled as follows:

* If the value on the primary resource is an empty string, the result has the value of the secondary resource.
* Otherwise, the value of the primary resource is used.

This part of functionality is also supported by current JavaScript SDK:

```
  merge(other: Resource | null): Resource {
    if (!other || !Object.keys(other.labels).length) return this;

    // Labels from resource overwrite labels from other resource.
    const mergedLabels = Object.assign({}, other.labels, this.labels);
    return new Resource(mergedLabels);
  }
```

### Detector

**Java Implementation Analysis**
1. ***DockerHelper***
The key parameters for DockerHelper:

```
  private static final int CONTAINER_ID_LENGTH = 64;
  private static final String DEFAULT_CGROUP_PATH = "/proc/self/cgroup";
```

The core function is shown below:

```
  public String getContainerId() {
    try (BufferedReader br = new BufferedReader(new FileReader(cgroupPath))) {
      String line;
      while ((line = br.readLine()) != null) {
        if (line.length() > CONTAINER_ID_LENGTH) {
          return line.substring(line.length() - CONTAINER_ID_LENGTH);
        }
      }
    } catch (FileNotFoundException e) {
      ...
    } catch (IOException e) {
      ...
    }

    return "";
  }
```

As we can see, the core function of Dockerhelper is pretty similar to Beanstalk resource detector, we basically trying to read containerId from certain file. Here we can conclude this function into several steps:

1. find the target file by given path and read it
    1. If file does not exist or cannot read the file, throw error.
2. read the file line by line, when find a line has a length > 64, return the last the 64 character of string
3. If there is not satisfying line of string, return “”;

2. ***ECS Detector***
The key parameters:

```
  private static final String ECS_METADATA_KEY_V4 = "ECS_CONTAINER_METADATA_URI_V4";
  private static final String ECS_METADATA_KEY_V3 = "ECS_CONTAINER_METADATA_URI";
```

It first check whether we are currently on the ECS by using the parameters above:

```
  private boolean isOnEcs() {
    return (!Strings.isNullOrEmpty(sysEnv.get(ECS_METADATA_KEY_V3))
        || !Strings.isNullOrEmpty(sysEnv.get(ECS_METADATA_KEY_V4)));
  }
```

The ECS detector mainly to detect 2 attributes, one is containerId which can be detected by dockerhelper, the other is to get host name. Here is a java playground which can be helpful: https://compiler.javatpoint.com/opr/test.jsp?filename=JavaInetAddressGetLocalHostExample1
The core function 

```
    try {
      String hostName = InetAddress.getLocalHost().getHostName();
      attrBuilders.setAttribute(ResourceConstants.CONTAINER_NAME, hostName);
    } catch (UnknownHostException e) {
      ...
    }
```

After getting the value of containerId and hostname, it then builds a resource instance and return it.

### TypeScript Design

The general structure of code file should look like:

```javascript
class AwsEcsDetector implements Detector {
  readonly CONTAINER_ID_LENGTH = 64;
  readonly DEFAULT_CGROUP_PATH = "/proc/self/cgroup";
  readonly ECS_METADATA_KEY_V4 = "ECS_CONTAINER_METADATA_URI_V4";
  readonly ECS_METADATA_KEY_V3 = "ECS_CONTAINER_METADATA_URI";

  async detect(config: ResourceDetectionConfigWithLogger): Promise<Resource> {
    ...
  }
  /**
   * Read container ID from cgroup file
   * In ECS, even if we fail to find target file
   * or target file does not contain container ID
   * we do not throw an error but throw warning message
   * and then return null string
   *
   * The implementation logic is follow OTel-Java:
   * https://github.com/open-telemetry/opentelemetry-java/blob/master/sdk_extensions/aws_v1_support/src/main/java/io/opentelemetry/sdk/extensions/trace/aws/resource/EcsResource.java
   */
  private async _getContainerId(config: ResourceDetectionConfigWithLogger): Promise<string> {
    try {
      ...
    } catch (e) {
      ...
    }
  } 
}
```

