# Implementation Details and Test for AWS ECS Plugin Resource Detector

![Data Path Diagram](../images/ResourceDetail.png)

## Implementation Details

`_getContainerId()`
This part is to implement the `DockerHelper` functionality in Java, only there are several different things in Typescript.

1. find the target file by given path and read it
    1. If file does not exist or cannot read the file, throw error. In Typescript, different from Java version, since we define this method inside the main class, so here we need to throw and catch the error inside the `_getContainerId()` method.
2. read the file line by line, when find a line has a length > 64, return the last the 64 character of string (No need in JavaScript)
    1. Here, since we are using fs.readFile method to read the content of file, we may not read and check line by line in the same time. Instead, we could use trim() and split() method to do similar things
3. If there is not satisfying line of string, return “”;

The implementation is shown below:

```
 private async _getContainerId(config: ResourceDetectionConfigWithLogger): Promise<string> {
    try {
      const rawData = await cache.readFileAsync(this.DEFAULT_CGROUP_PATH, 'utf8');
      const splitData = rawData.trim().split('\n');

      let res = '';
      splitData.forEach( str => {
        if (str.length > this.CONTAINER_ID_LENGTH) {
          res = str.substring(str.length - this.CONTAINER_ID_LENGTH);
          return;
        }
      });
      return res;
    } catch (e) {
      config.logger.warn('Cannot find cgroup file for AwsEcsDetector');
      return '';
    }
  } 
```

`detect()`

1. First, using the key parameters of environmental variables to identify whether the process is on ECS.
2. In the ECS resource detector, we basically want to assign 2 values: containerId and hostname. ContainerId is obtained by `_getContainerId()`, in this method, we are going to detect the hostname attribute:

In Java, they use:

```
String hostName = InetAddress.getLocalHost().getHostName();
```

In Typescipt, we can use:

```
hostName = os.hostname();
```

1. Finally, depending on different conditions of obtained hostname and containerId, we may return different resource instance.

The implementation is shown below:

```
  async detect(config: ResourceDetectionConfigWithLogger): Promise<Resource> {
    if (!process.env.ECS_CONTAINER_METADATA_URI_V4 ||
        !process.env.ECS_CONTAINER_METADATA_URI) {
      config.logger.debug('AwsEcsDetector failed: Process is not on ECS');
      return Resource.empty();
    }
    let containerId, hostName;

    try {
      hostName = os.hostname();
    } catch (e) {
      config.logger.warn(`AwsEcsDetector failed to read host name: ${e.message}`);
    }

    await this._getContainerId(config)
     .then(res => {
       containerId = res;
     });

    if (containerId && hostName) {
      return new Resource({
        [CONTAINER_RESOURCE.NAME]: hostName,
        [CONTAINER_RESOURCE.IMAGE_TAG]: containerId,
      });
    } else if (!hostName && containerId) {
      return new Resource({
        [CONTAINER_RESOURCE.IMAGE_TAG]: containerId,
      });
    } else if (hostName && !containerId) {
      return new Resource({
        [CONTAINER_RESOURCE.NAME]: hostName,
      });
    } else {
      return Resource.empty();
    }
  }
```

### Testing

Like the final testing plan in the Beanstalk design, we can take use of `sinon.sandbox` to simulate the return value of certain functions. Here, there are several things needs to be mentioned:

1. In order to set environmental variables, we use `process.env.ENVRIONMENT = ...` to setup, and note that cleaning environmental variable up in `beforeEach` sentence.
2. In order to mock the return value of certain functions, we use `sinon.sandbox.stub` to stub functions, note that we use` stub().returns()` to mock synchronous function and `stub.resolves()` to mock asynchronous function.
3. In order to check the correctness of returned resource instance, we use OTel built-in method like `assertContainerResource()` and `assertEmptyResource()`.
4. For instance, if we want to make sure our ECS resource detector successfully returned data, we have following implementation:

```
  it('should successfully return resource data', async () => {
    process.env.ECS_CONTAINER_METADATA_URI_V4 = 'ecs_metadata_v4_uri';
    hostStub = sandbox.stub(os, 'hostname').returns(hostNameData);
    readStub = sandbox.stub(cache, 'readFileAsync').resolves(correctCgroupData);

    const resource = await awsEcsDetector.detect({
      logger: new NoopLogger(),
    });

    sandbox.assert.calledOnce(hostStub);
    sandbox.assert.calledOnce(readStub);
    assert.ok(resource);
    assertContainerResource(resource, {
      name: 'abcd.test.testing.com',
      imageTag:
        'bcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklm',
    });
  });
```

Also note that before and after each test case, different from the testing in Beanstalk resource detector, we should not only clean the `sandbox`, but also take care of environmental variables. In this case, we should clean environmental variable each time before the test starts.

```
  beforeEach(() => {
    sandbox = sinon.createSandbox();
    process.env.ECS_CONTAINER_METADATA_URI_V4 = '';
    process.env.ECS_CONTAINER_METADATA_URI = '';
  });

  afterEach(() => {
    sandbox.restore();
  });
```