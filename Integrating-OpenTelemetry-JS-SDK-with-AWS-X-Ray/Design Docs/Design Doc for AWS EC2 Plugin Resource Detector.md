# Design Doc for AWS EC2 Plugin Resource Detector

## Objective

Design and implement AWS EC2 resource detector component in OpenTelemetry.

## Summary

![Data Path Diagram](../images/Instrumentation.png)A [Resource](https://github.com/open-telemetry/opentelemetry-specification/blob/b338f9f63dbf02ff8ebd100e8a847e7bf43e2682/specification/overview.md#resources) is an immutable representation of the entity producing telemetry. For example, a process producing telemetry that is running in a container on Kubernetes has a Pod name, it is in a namespace and possibly is part of a Deployment which also has a name. All three of these attributes can be included in the `Resource`.
The primary purpose of resources as a first-class concept in the SDK is decoupling of discovery of resource information from exporters. This allows for independent development and easy customization for users that need to integrate with closed source environments. The SDK MUST allow for creation of `Resources` and for associating them with telemetry.
When used with distributed tracing, a resource can be associated with the [TracerProvider](https://github.com/open-telemetry/opentelemetry-specification/blob/b338f9f63dbf02ff8ebd100e8a847e7bf43e2682/specification/trace/sdk.md#tracer-sdk) when it is created. That association cannot be changed later. When associated with a `TracerProvider`, all `Span`s produced by any `Tracer` from the provider MUST be associated with this `Resource`.

## Goal

* Design the functionality and code structure to be implemented for AWS EC2 resource detector.

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

The concept and standard of resource detector is not mentioned in specification file. It seems to be JavaScript SDK specific component. And here is the interface standard defined for resource detector:

```
/**
 * Interface for a Resource Detector. In order to detect resources in parallel
 * a detector returns a Promise containing a Resource.
 */
export interface Detector {
  detect(config: ResourceDetectionConfigWithLogger): Promise<Resource>;
}
```

According to AWS instance identity documentation: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
We have parameters for the endpoint and path we should build connection with:

```
  readonly HTTP_HEADER = 'http://';
  readonly AWS_IDMS_ENDPOINT = '169.254.169.254';
  readonly AWS_INSTANCE_TOKEN_DOCUMENT_PATH = '/latest/api/token';
  readonly AWS_INSTANCE_IDENTITY_DOCUMENT_PATH =
    '/latest/dynamic/instance-identity/document';
  readonly AWS_INSTANCE_HOST_DOCUMENT_PATH = '/latest/meta-data/hostname';
```

Currently, opentelemetry-js repository has provided implementation for IDMSv1 standard EC2 resource detector. It basically realize the functionality of below command:

```
``$ `curl http://169.254.169.254/latest/dynamic/instance-identity/document`
```

And expect to get following example information in response body:

```
`{
    "devpayProductCodes" : null,
    "marketplaceProductCodes" : [ "1abc2defghijklm3nopqrs4tu" ], 
    "availabilityZone" : "us-west-2b",
    "privateIp" : "10.158.112.84",
    "version" : "2017-09-30",
    "instanceId" : "i-1234567890abcdef0",
    "billingProducts" : null,
    "instanceType" : "t2.micro",
    "accountId" : "123456789012",
    "imageId" : "ami-5fb8c835",
    "pendingTime" : "2016-11-19T16:32:11Z",
    "architecture" : "x86_64",
    "kernelId" : null,
    "ramdiskId" : null,
    "region" : "us-west-2"
}`
```

However, since our AWS has released IDMSv2 standard to retrieve the plaintext instance identity document. We need to Connect to the instance and run one of the following commands depending on the Instance Metadata Service (IMDS) version used by the instance. 

```
``$ `TOKEN=`curl -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600"` \
&& curl -H "X-aws-ec2-metadata-token: $TOKEN" -v http://169.254.169.254/latest/dynamic/instance-identity/document`
```

In order to realize the connection and command functionality above, we need to

1. build a connection to 'http://169.254.169.254/latest/api/token' to get token.

2. using token and connect to 'http://169.254.169.254/latest/dynamic/instance-identity/document' to get identity.

3. using token and connect to ‘[http://169.254.169.254/latest/meta-data/hostname](http://169.254.169.254/latest/meta-data/hostname%E2%80%99)’ to get host info.

4. parse the response body of identity and host info

5. assign attributes to corresponding resource constant

6. General structure of this file is shown below

      ```
      class AwsEc2Detector implements Detector {
        readonly AWS_INSTANCE_TOKEN_DOCUMENT_URI =
          'http://169.254.169.254/latest/api/token';
        readonly AWS_INSTANCE_IDENTITY_DOCUMENT_URI =
          'http://169.254.169.254/latest/dynamic/instance-identity/document';
        readonly AWS_INSTANCE_HOST_DOCUMENT_URI =
          'http://169.254.169.254/latest/meta-data/hostname';
          
        async detect(config: ResourceDetectionConfigWithLogger): Promise<Resource> {
          try {
            const token = await this._fetchToken();
            const identity = await this._fetchIdentity(token);
            const hostname = await this._fetchHost(token);
            
            return new Resource({
              [CLOUD_RESOURCE.PROVIDER]: 'aws',
              [CLOUD_RESOURCE.ACCOUNT_ID]: accountId,
              [CLOUD_RESOURCE.REGION]: region,
              [CLOUD_RESOURCE.ZONE]: availabilityZone,
              [HOST_RESOURCE.ID]: instanceId,
              [HOST_RESOURCE.TYPE]: instanceType,
              [HOST_RESOURCE.NAME]: hostname,
              [HOST_RESOURCE.HOSTNAME]: hostname,
            });
          } catch (e) {
              ...
          }
        }
      
        private async _fetchToken(): Promise<string> {
          const options = {
            ...
          }
          return await this._fetchString(options);
        }
        
        private async _fetchIdentity(token: string): Promise<Object> {
          const options = {
            ...
          }
          const identityString = await this._fetchString(options);
          return JSON.parse(identityString);
        }
        
        private async _fetchHost(token: string): Promise<string> {
          const options = {
            ...
          }
          return await this._fetchString(options);
        }
        
        private async _fetchString(options: Object): Promise<string> {
          return new Promise((resolve, reject) => {
            ...
          });
        }
      }
      
      export const awsEc2Detector = new AwsEc2Detector();
      ```

And we design helper functions below to realize our target functionality:

`_fetchString(options): string
`
This function is used to fetch string from given url with given configuration. Since IDMS v2 demands more attributes to be sent, After careful investigate through https://nodejs.org/api/http.html#http_http_request_url_options_callback the official doc of `http` library of JavaScript, we chose to use `http.request` rather than original `http.get` method since it is more flexible and also intuitive.
Most of the code follows the standard of http official example.


`_fetchToken(): string
`
This function first construct the configuration and then use the configuration options to fetch token by using `_fetchString()`.

`_fetchIdentity(token): Object
`
This function first construct the configuration with the given token and then use the configuration options to fetch the identity by using `_fetchString()`. Finally use `JSON.parse` to parse the received string and return object.

`_fetchHost(token): string
`
This function first construct the configuration with the given token and then use the configuration options to fetch the hostname by using `_fetchString()`. Finally use `JSON.parse` to parse the received string and return object.