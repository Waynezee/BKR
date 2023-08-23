# BKR

## Getting Started


### Environment Setup
* latest verion of Golang (1.19 will be ok)

* python3+
  ```bash
  # boto3 offers AWS APIs which we can use to access the service of AWS in a shell 
  pip3 install boto3==1.16.0
  ```

* protobuf, the implementation of BKR uses protobuf to serialize messages, refer [this](https://github.com/protocolbuffers/protobuf) to get a pre-built binary. (libprotoc 3.14.0 will be ok)


 
### Build
```bash
# Download dependencies.
cd [working_directory]/src
go get -v -t -d ./...

# Build Node.
cd [working_directory]/src/acs/server/cmd
go build -o main main.go
```

### Generate BLS keys
```bash
# Download dependencies.
cd [working_directory]/src/crypto
go get -v -t -d ./...

# Generate bls keys.
# Default n=4 (f = 1, n = 3f + 1), t=2 (t = f + 1)
cd [working_directory]/src/crypto/cmd/bls
go run main.go -n 4 -t 2
```

### Client
```bash
# Compile protobuf.
cd [working_directory]/src/client/proto
make

# Build Client.
cd [working_directory]/src/client
go build -o client main.go
```

## Testing

**First of all, you are supposed to change `key` in every shell script to the path of your own ssh key to access your remote machines. If use AWS, please refer [this](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html#cli-configure-quickstart-creds) to set up your AWS credentials. If not use AWS, please use generateLocal.py instead of aws.py and generate.py to create json files**

**In a remote machine, we will deploy one client and one consensus node**

### Run ACS on a cluster of remote machines (We use AWS EC2s)
1. Create a cluster of EC2 machines.

2. (if use AWS) Fetch machine information from AWS.
    ```bash
    cd [working_directory]/script/server
    python aws.py
    ```

3. (if use AWS) Generate confige file(`node.json`) for every node.
    ```bash
    python generate.py
    ```
4. (if not use AWS) add your machine ips to the variable `ipset` in generateLocal.py, and then
    ```bash
    python generateLocal.py
    ```
> Before proceeding to the next step, first prepare all executable files and bls keys through Getting Started

5. Deliver nodes. 
    ```bash
    chmod +x *.sh

    # Compress BLS keys.
    ./tarKeys.sh
    
    # Deliver to every node.
    # n is the number of nodes in the cluster.
    ./deliverNode.sh n
    ```

5. Run nodes.
   ```bash
   ./beginNode.sh n
   ```
6. (After fetching client logs but before the next experiment) Stop nodes.
   ```bash
   # stop node process
   ./stopNode.sh n
   # clear node log files
   ./rmLog.sh n
   ```

### Run clients to send requests to BKR nodes.
1. Deliver client. (Open another terminal)
   ```bash
   cd [working_directory]/script/client
   chmod +x *.sh
   
   # n is the number of nodes in the cluster
   ./deliverClient.sh n
   ```

2. Run client and wait for a period of time. (change the batch to get different latency-throughput)
   ```bash
   ./beginClient.sh n <size of payload (size of a request)> <size of batch (number of requests)>  <running time>
   ```

3. Copy result from client node.
   ```bash
   # execute once
   mkdir log
   # create dirs to store logs, execute once
   ./createDir.sh n 
   # fetch logs from remote machines
   ./copyResult.sh n <name of log file>
   ```

4. Calculate throughput and latency.
   ```bash
   # example: python cal.py 4 100 [working_directory]/srcipt/client/log test 30
   python cal.py <number of clients> <batchsize> <path of log directory> <name of log file> <test time>
   ```

## Brief introduction of scripts

### script/server

* aws.py: get machine information from AWS.
  > You may need to change line 5-7 and 23 to your own config.

* generate.py：generate configuration for every node.

* tarKeys.sh: compress BLS keys.

* deliverNode.sh: deliver node to remote machines.
  * `./deliverNode.sh <the number of remote machines>`

* beginNode.sh: run node on remote machines.
  * `./beginNode.sh <the number of remote machines>`

* stopNode.sh: stop node on remote machines.
  * `./stopNode.sh <the number of remote machines>`

* Simulate the crash of specific nodes.
  * You can refer to `crash33.sh`, and change `id` to your specific situation.
  * `./crash33.sh`

* Simulate the crash of the last few nodes.
  * eg. simulate the crash of the last 33 nodes of 100 nodes.
    ```bash
    ./crash.sh 100 33
    ```

* rmLog.sh: remove log file on remote machines.
  * `./rmLog.sh <the number of remote machines>`

### script/client
* deliverClient.sh: deliver client to remote machines.
  * `./deliverClient.sh <the number of remote machines>`

* Fetch log files from remote machines
    ```bash
    mkdir log
    ./createDir.sh <the number of remote machines>
    ./copyResult.sh <the number of remote machines> <name of log files>
    ```

