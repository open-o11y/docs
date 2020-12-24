# Introduction

This document outlines the implementation for the Amazon EKS Plugin Detector component in the OpenTelemetry JS SDK.

## What is AWS EKS? 
Amazon Elastic Kubernetes Service [EKS](https://docs.aws.amazon.com/eks/latest/userguide/what-is-eks.html) is a managed Kubernetes service that makes it easy to run Kubernetes on AWS without needing to install, operate, and maintain your own Kubernetes control plane.

## Summary

[Image: image]Above Diagram taken from here (https://github.com/open-o11y/docs/blob/master/Integrating-OpenTelemetry-JS-SDK-with-AWS-X-Ray/Design%20Docs/Design%20Doc%20for%20AWS%20Beanstalk%20Plugin%20Resource%20Detector.md):

As defined by OpenTelemetry specifications, a Resource (https://github.com/open-telemetry/opentelemetry-specification/blob/b338f9f63dbf02ff8ebd100e8a847e7bf43e2682/specification/overview.md#resources) is an immutable representation of the entity producing telemetry. For example, a process producing telemetry that is running in a container on Kubernetes has a Pod name, it is in a namespace and possibly is part of a Deployment which also has a name. All three of these attributes can be included in the Resource.

The primary purpose of resources as a first-class concept in the SDK is to decouple discovery of resource information from exporters. This allows for independent development and easy customization for users that need to integrate with closed source environments. The SDK must allow for creation of Resources and for associating them with telemetry.

## Design Tenets

1. Security- The detector will properly handle authentication by including credentials and signing HTTP requests sent to AWSAuth (https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html).
2. Reliability- The resource detector should properly handle errors such as when the environment is not running on a Kubernetes process, EKS process, or failed HTTP calls. As per the specifications, failure to detect a resource should not throw an error.
3. Test Driven Development- TDD practices established by the team will be closely followed and achieve significant test coverage(greater than 90%).

## Design Details

Resource is used to define attributes of the application itself, for example the cloud environment it is running on. This corresponds with the plugins in the X-Ray SDKs. We implement Resources that populate attributes we expect for AWS users. We can implement a Resource for each of our plugins and merge them together.
In the opentelemetry-js repository, the JavaScript repository provides create() and merge() method as described in the OpenTelemetry _specification_ (https://github.com/open-telemetry/opentelemetry-specification/blob/b338f9f63dbf02ff8ebd100e8a847e7bf43e2682/specification/resource/sdk.md).

### Create()

The interface must provide a way to create a new resource, from a collection of attributes. Examples include a factory method or a constructor for a resource object. A factory method is recommended to enable support for cached objects.
In the JavaScript SDK, the functionality to create a resource is supported:

```
  static createTelemetrySDKResource(): Resource {
    return new Resource({
      [TELEMETRY_SDK_RESOURCE.LANGUAGE]: SDK_INFO.LANGUAGE,
      [TELEMETRY_SDK_RESOURCE.NAME]: SDK_INFO.NAME,
      [TELEMETRY_SDK_RESOURCE.VERSION]: SDK_INFO.VERSION,
    });
  }
```

### Merge()

In the specification document, it says the interface must provide a way for different resources to be merged into a new resource.
The resulting resource must have all attributes that are on any of the two input resources. Conflicts (i.e. a key for which attributes exist on both the primary and secondary resource) must be handled as follows:

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

A resource detector is not a concept that is defined in the OpenTelemetry specification file. The following code snippet is the interface for a standard resource detector; it may be a JavaScript SDK specific interface. For the EKS detector (https://docs.aws.amazon.com/xray/latest/devguide/xray-sdk-java-configuration.html#xray-sdk-java-configuration-plugins), the expected attributes AWS users need to use to include the container ID, cluster name, pod ID, and the Amazon CloudWatch Logs Group.

```
/**
 * Interface for a Resource Detector. In order to detect resources in parallel
 * a detector returns a Promise containing a Resource.
 */
export interface Detector {
  detect(config: ResourceDetectionConfigWithLogger): Promise<Resource>;
}
```

According to the official Kubernetes [documentation](https://kubernetes.io/docs/tasks/access-application-cluster/access-cluster/), the following parameters are the recommended way to locate the api server within a pod, credentials for a service account, and verify serving credentials.

```
readonly KBS_SVC_URL = "https://kubernetes.default.svc";
readonly K8S_TOKEN_PATH = "/var/run/secrets/kubernetes.io/serviceaccount/token";
readonly KBS_CERT_PATH = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt";
```

The following parameters can be used to manage users or IAM roles for a cluster and grab cluster info from Amazon CloudWatch.

```
readonly AUTH_CONFIGMAP_PATH ="/api/v1/namespaces/kube-system/configmaps/aws-auth";
readonly CW_CONFIGMAP_PATH = "/api/v1/namespaces/amazon-cloudwatch/configmaps/cluster-info";
```

1. The following code can be used to detect whether the object is a Kubernetes instance by checking that a key and a key path exists.
     ```
         private async _isK8s(): Promise<Boolean> {
            ...
            return keyExists() && keysPathExists();
        }
    ```
2. The Kubernetes credential header can be obtained from ["https://kubernetes.default.svc](https://kubernetes.default.svc/)“  and will be used to fetch the AWSauth config map.
```
     private async getK8sCredHeader(): Promise<String> {
            File file = new File(K8S_TOKEN_PATH); 
            try {
              // Check whether kubernetes client token is loadable
              return header;
            } catch (IOException e) {
              // Unable to load kubernetes client token
            }
            return "";
        }
```
3. The _isEks method checks whether it is a Kubernetes instance. If it is, the method connects to "/api/v1/namespaces/kube-system/configmaps/aws-auth" to check whether cluster is an EKS cluster.
```
     private async _isEks(): Promise<Boolean> {
            if (!this._isK8s()) {
                ...
            }
            const options {
                ...
            }
            return await !!this._fetchString(options)
        }
```
4. Get cluster name from response body of Amazon CloudWatch through “/api/v1/namespaces/amazon-cloudwatch/configmaps/cluster-info”.
```
        private async _getClusterName(): Promise<String> {
            const options = {
                ...
            }
            String json = await  this._fetchString(options);
            try {
                ObjectMapper mapper = new ObjectMapper();
                return mapper.readTree(json).at("/data/cluster.name").asText();
            } catch () {
                ...
            }
            return ""
        }
```
5.  Return expected attributes such as container Id and cluster name
```
        protected async _getAttributes(): Promise<Attributes> {
            if(!this._isEks()) {
                ...
            }
            ...
            return new Resource({
              [K8S_RESOURCE.CLUSTER_NAME]: clusterName || '',
              [CONTAINER_RESOURCE.ID]: containerId || '',
            });
        }
```

Within this class, we will also design the follow helper functions/classes:

_fetchString(options): string

This function is used to fetch string from given URL with given configuration. We will have a very similar code implementation to the standard Node.js HTTP library (https://nodejs.org/api/http.html#http_http_request_url_options_callback) example for the HTTP.request method.

DockerHelper Class:
The docker helper class is used to fetch the docker containerId from the local cgroup file. The structure of the docker helper class will be similar to below:
```
 class DockerHelper {
    readonly CONTAINER_ID_LENGTH = 64;
    readonly String DEFAULT_CGROUP_PATH = "/proc/self/cgroup";
    readonly String cgroupPath;
  
    public async getContainerId(): Promise<String> {
      try {
      FileReader(DEFAULT_CGROUP_PATH);
      //Read through each line until we find
      //Use a file reader to check whether we can find the container Id
      } catch (FileNotFoundException e) {
        ...
        // Throw error if cgroup file does not exist
      } catch (IOException e) {
        ...
        // Unable to read docker container id
      }
  
      return "";
    }
  }
```

## Goal

Design the code structure and functionality required to implement the AWS EKS plugin resource Detector

### Testing Strategy

We can use sinon.sandbox to simulate the whole environment and stub HTTP methods as it has been used in all other resource detectors. The following steps will be followed to use our test strategy:

1. In the beforeEach() function we will be creating a new sandbox.
```
   let sandbox: sinon.SinonSandbox;
    
      beforeEach(() => {
        sandbox = sinon.createSandbox();
      });
```
2. In the afterEach() function we will restore the sandbox to its previous environment before the test ran.
```
    afterEach(() => {
        sandbox.restore();
      });
```
3. We will have to check whether resource instance returned is correct, we can use the OpenTelemetry built in methods such as assertContainerResource() and assertEmptyResource(). 
4. We will need to stub and mock HTTP calls and all other methods to achieve > 90% code coverage.
5. Verify that the results are expected
6. In order to build a successful test case, we can use the OpenTelemetry built in methods such as assertK8SResource() and assertContainerResource()

