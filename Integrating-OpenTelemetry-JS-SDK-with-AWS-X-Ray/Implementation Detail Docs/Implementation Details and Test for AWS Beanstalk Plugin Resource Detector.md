# Implementation Details and Test for AWS Beanstalk Plugin Resource Detector

![Data Path Diagram](../images/ResourceDetail.png)

## Implementation Details

Corresponding to design doc [Design Doc for AWS Beanstalk Plugin Resource Detector](https://quip-amazon.com/dxORAEy0UP4A)
For Beanstalk detector, the core functionality we will implement is to read the content from `environment.conf` file. Here we introduce several plans we tried and finally introduce the final plan.
Before start to detect, we need to make sure application is running on which Operating System, so we know the path of config file:

```
  readonly DEFAULT_BEANSTALK_CONF_PATH =
    '/var/elasticbeanstalk/xray/environment.conf';
  readonly WIN_OS_BEANSTALK_CONF_PATH =
    'C:\\Program Files\\Amazon\\XRay\\environment.conf';
  BEANSTALK_CONF_PATH: string;

  constructor() {
    if (process.platform === 'win32') {
      this.BEANSTALK_CONF_PATH = this.WIN_OS_BEANSTALK_CONF_PATH;
    } else {
      this.BEANSTALK_CONF_PATH = this.DEFAULT_BEANSTALK_CONF_PATH;
    }
  }
```

### File Reading method

1. `fs.createReadStream()`(Not adopted)

    1. As the sample code shown below,
        ```javascript
        import * as fs from 'fs';
        
        function testfunc(){
          var data = '';
        
          var readStream = fs.createReadStream('test.conf', 'utf8');
        
          readStream.on('data', function(chunk) {
              data += chunk;
          }).on('end', function() {
              console.log(data);
          });
        }
        ```
    2. Unfortunately, this method does not support reading file with .conf extension

2. Directly `import` (Not adopted)

    1. As the sample code shown below,

        ```
        import { jsonContent } from '/var/elasticbeanstalk/xray/environment.conf';
        ```
    2. Currently, Typescript usually support directly import things from .js, .ts, .json files. Whether this way is acceptable needs further discussion. 

1. `fs.ReadFile()` (Final plan)
    1. Our main function can be:
    
       ```
         async detect(config: ResourceDetectionConfigWithLogger): Promise<Resource> {
           fs.readFile(ENV_CONFIG_LOCATION, 'utf8', function(err, rawData) {
             if (err) {
               ...
             } else {
               var data = JSON.parse(rawData);
       
               var metadata = {
                 elastic_beanstalk: {
                   environment: data.environment_name,
                   version_label: data.version_label,
                   deployment_id: data.deployment_id
                 }
               };
               ...
             }
           });
         },
       ```

### Asynchronous Tackling

Except for file reading method, there are still 2 problems we need to tackle in the same time:

1. Since `detect()` method in interface is fixed to be an asynchronous method, we need to make sure we do operations following asynchronous standard.
2. Also, `fs.readFile()` is a desirable function to use, but it is default to be an asynchronous function, which means if we assign value to the callback function of `fs.readFile()`, variable has delay to really accept the value from `fs.readFile()`. As shown below,

    ```
    var value; // <- initialize value as "undefined" at timestamp 1
    fs.readFile('test1.conf', 'utf8', (err, data) => {
          if (err) {
            throw err;
          } else {
            value = data; // <- assign the data to value at timestamp 3
          }
    }
    console.log(value) // <- print the value, should be "undefined" at timestamp 2
    ```

Similar to file reading method, we also introduce the plans we tried and finally give the plan adopted.

1. Using `fs.readFileSync()`(Not adopted)
    1. Different from `fs.readFile()`, `fs.readFileSync()` acts like normal synchronous function and returns value after it really read value.
    2. However, it fails to tackle the problem 1, because this is a explicitly synchronous method which can never be suitable in a asynchronous method
2. Promisify `fs.readFile()` by hand (Not adopted)
    1. We can choose to promisify `fs.readFile()` by hand so we could make sure the file data assigned to variable and then do other operations. The implementation is shown below
    
        ```
        private async _getData(): Promise<string> {
            return new Promise((resolve, reject) => {
              fs.readFile(this.BEANSTALK_CONF_PATH, 'utf8', (err, data) => {
                if (err) {
                  reject(err);
                } else {
                  resolve(data);
                }
              });
            });
        ```
    3. It looks good and both 2 problems are solved, but pointed out by Daniel, JavaScript has built-in method to do promisify.
3. Using `util.promisify()` to promisify `fs.readFile()` (Final plan)
    1. The implementation is shown below:
    
        ```
        export const cache = { readFileAsync: util.promisify(fs.readFile) };
        
        class AwsBeanstalkDetector implements Detector {
          readonly BEANSTALK_CONF_PATH = '/var/elasticbeanstalk/xray/environment.conf';
        
          async detect(config: ResourceDetectionConfigWithLogger): Promise<Resource> {
            try {
              ...
        
              const rawData = await cache.readFileAsync(
                this.BEANSTALK_CONF_PATH,
                'utf8'
              );
              ...
            } catch (e) {
              ...
            }
          }
        }
        ```
    3. Note that the `util.promisify()` is module level operation, it should be completed out of `AwsBeanstalkDetector` class
    4. Also, since `sinon.sandbox.stub` method for testing demands an object to simulate, so we wrap our promisified method in cache.

## Test

### Plan A (Not adopted)

This test plan can be divided into 3 steps:

1. create a simulated folder and .conf file, here we may doing different test by assign different, using synchronous function to make sure there is no race condition during testing

   ```
     var content = '{\"noise\": \"noise\", \"deployment_id\":4,\"'
     + 'version_label\":\"2\",\"environment_name\":\"HttpSubscriber-env\"}';
     var filename = 'written.conf';
     var foldername = '/a';
   
     fs.mkdirSync(foldername, function(err){
       if (err) throw err;
       console.log('successfully created folder');
     })
     fs.writeFileSync(foldername + filename, content, function(err) {
       if (err) throw err;
       console.log('successfully created file');
     });
   ```
3. Test whether read the expected Resource instance.
3. delete created folder and file

   ```
    var filename = 'written.conf';
    var foldername = '/a';
     
    fs.unlinkSync(foldername + filename, function(err){
      if (err) throw err;
      console.log('successfully deleted file');
    });
    fs.rmdirSync(foldername, function(err){
      if (err) throw err;
      console.log('successfully deleted folder');
    })
   ```

**Cons**
Generally, these procedures make sense, but the implementation cannot be this way. In this implementation, to complete the test we need to modify file and folder which will definitely encounter writing rights problem.

### Plan B (Final plan)

Use `sinon.sandbox` to simulate the whole environment.
For instance we can use

```
readStub = sandbox.stub(fs, 'readFile').yields(null, data);
```

to simulate that when the `fs.readFile()` method is called, it will return `(err: null, data)`
And in our case, we need to simulate promisified method, so we do

```
    readStub = sandbox
      .stub(cache, 'readFileAsync')
      .resolves(JSON.stringify(data));
```

to simulate asynchronous method.
Note that each time before a test case started and after a test case finished, we need to ensure the status of sandbox:

```
  let sandbox: sinon.SinonSandbox;

  beforeEach(() => {
    sandbox = sinon.createSandbox();
  });

  afterEach(() => {
    sandbox.restore();
  });
```

Then for instance, if we would like to build a successful test case, we do:

```
  it('should successfully return resource data', async () => {
    fileStub = sandbox.stub(fs, 'access').yields(null);
    readStub = sandbox
      .stub(cache, 'readFileAsync')
      .resolves(JSON.stringify(data));
    sandbox.stub(JSON, 'parse').returns(data);

    const resource = await awsBeanstalkDetector.detect({
      logger: new NoopLogger(),
    });

    sandbox.assert.calledOnce(fileStub);
    sandbox.assert.calledOnce(readStub);
    assert.ok(resource);
    assertServiceResource(resource, {
      name: 'elastic_beanstalk',
      namespace: 'scorekeep',
      version: 'app-5a56-170119_190650-stage-170119_190650',
      instanceId: '32',
    });
  });
```

1. First stub methods need to be mocked
2. The take use of the function to be tested and get result
3. verify the result is as expected