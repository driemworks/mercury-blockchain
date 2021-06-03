# Mercury Blockchain
Mercury is simple a blockchain built on top of go-libp2p. Current, Mercury is in phase 0: in the current incarnation Mercury is a blockchain-based p2p chat app, where different chat rooms are encoded as transactions within blocks. The intention is to accomplish decentralized state management via blockchain (reminiscent of Holochain).

## TODOS
- Replace PoW with more efficient consensus algorithm.. proof of contribution?
- test coverage

## Getting Started
### Introduction
For the time being, assume that each topic requires at least one node at all times. 
TODO

### Pre requisites
- install `go`

### Installation 
- Install mercury from the source by cloning this repo and run `go install ./cli/...` from the root directory `mercury-blockchain/`.


- Install the latest version using:
```
go get github.com/driemworks/mercury-blockchain/cli/...
```
Note: if you're running linux you may need to set `export GO111MODULE=on;go get github.com/driemworks/mercury-blockchain/cli/...`

## Usage
This section explains how to get started using mercury. 

### CLI commands
`mercury [command] [options]`
- Available Commands:
  - `help`: Help about any command
  - `run`:  Run the mercury node
    -  options:
      - `--name`: (optional) The name of your node - Default: `""`
      - `--datadir`: (optional) the directory where local data will be stored - Default: `.mercury`
      - `--host`: (optional) the ip addreses of the mercury node - Default: `127.0.0.1`
      - `--port`: (optional) the port of the mercury node - Default: `8080`
      - `--miner`: (required) the public key to use (see: output of wallet command)
      - `--bootstrap`: (optional) Multihash of the peer you want to use as a bootstrap. This will be in the form `/ip4/<peer-ip>/tcp/<peer-port>/p2p/<peer node hash>` - Defaut: `""`
  - `wallet`: Access the node's wallet
    - `new-address` Generate a new address
        -  options:
            - `--datadir`: (required) the directory where local data will be stored (ex: blockchain transactions)

 example:
  ```
  mkdir .mercury
  # generate a new address
  mercury wallet new-address --datadir=./.mercury
  >  0x27084384033F90d96c3769e1b4fCE0E5ffff720B
  # start a node using the new address as the miner
  mercury run --datadir=./.mercury --name=tony --host=192.168.1.XYZ --port=8081 --miner=0x27084384033F90d96c3769e1b4fCE0E5ffff720B
  ```



### UI  (Does not work currently - keeping this only for informational purpose)
There is a crude ui available to interact with your node. Run an ipfs node with `ipfs daemon` and navigate to `http://127.0.0.1:8080/ipfs/Qmc55mmfkrmTyhRRYsaU9d3sDUBbMPXrtExnrwVbuESEAY/build/`


Note: The ui is available via pinata, however, due to the crudeness of the UI it requires a local ipfs node to be running in order to be functional. https://gateway.pinata.cloud/ipfs/Qmc55mmfkrmTyhRRYsaU9d3sDUBbMPXrtExnrwVbuESEAY/build/

### Connect to test network (does not work currently - keeping just in case)
To connect with the test network, use `--bootstrap-ip=ec2-34-207-242-13.compute-1.amazonaws.com` and `bootstrap-port=8080`
`mercury run --name=theo --datadir=.mercury/ --miner=0x990DB19D440124F3d5bA8867b3C35bC0D3c5Eda8 --port=<your port> --bootstrap-ip=ec2-34-207-242-13.compute-1.amazonaws.com --bootstrap-port=8080`

### RPC
Mercury uses gRPC to let you communicate directly with a node.
Authentication and Security is pending.

#### GetNodeStatus
Query the node for a status report
`rpc GetNodeStatus(NodeInfoRequest) returns (NodeInfoResponse) {}`

example with grpcurl:
```
grpcurl -plaintext 127.0.0.1:9081 proto.NodeService/GetNodeStatus
> {
>   "address": "0xEA3d0650a05d8F94DFFEd9514594BE2532Bec001",
>   "balance": 8,
>   "channels": [
>     "test",
>     "test"
>   ]
> }

```
#### AddTransaction
The main functionality (to be extended...): Create a new pending transaction that, once mined, will allow us to send generic tx payloads across nodes. Security has not been considered whatsoever with the current implementation.

In the current implementation this is synonymous with defining a new topic.

`rpc AddTransaction(AddPendingPublishCIDTransactionRequest) returns (AddPendingPublishCIDTransactionResponse) {}`

Example 
```
grpcurl -plaintext -d @ 127.0.0.1:9082 proto.NodeService/AddTransaction <<EOM
{
    "label": "hello",
    "password": "test"
}
EOM
```

### ListBlocks
TODO: this needs to be updated so we can actually stream blocks instead of just list them
List blocks from some given block hash, fromBlock. If empty, it is assumed to be from the genesis block.

In the current implementation this is synonymous with listing all defined topics and topic creators.
`rpc ListBlocks(ListBlocksRequest) returns (stream BlockResponse) {}`

Example 
```
grpcurl -plaintext -d @ 127.0.0.1:9082 proto.NodeService/ListBlocks <<EOM
{
    "fromBlock": ""
}
EOM
```

### Subscribe
Allows a node to subscribe to a topic, identified by the transaction hash within which it was defined.
`rpc Subscribe(JoinChannelRequest) returns (stream ChannelData) {}`

Example:
```
grpcurl -plaintext -d @ 0.0.0.0:8080 proto.NodeService/Subscribe <<EOM
{
    "txHash": "197e33d7b4b7c987c3739689978a4a88745e3ef095b3df7878774d10b09b7e7c"
}
EOM
```

### Publish
Publish to a pubsub topic which is defined within a transaction.
`rpc Publish(PublishRequest) returns (PublishResponse) {}`

```
grpcurl -plaintext -d @ 127.0.0.1:9082 proto.NodeService/Publish <<EOM
{
  "txHash": "cac49c8e6a9a93560c444a9b568565a24fbffb7420d53ca56c3961870b90cff6",
	"message": "hello world"
}
EOM
```

## Development

The project is composed of the following packages:
#### cli
The `cli` package contains code and configs for the cli, as explained above.

#### core
Contains common structs and functions used across `cli`, `node`, `state`, and `wallet`

#### node
HTTP API

#### state
State management (used by the node). 
- block, transactions, etc

#### wallet
Create and manage your keystore

#### proto
To update server interface, run:
```
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/node.proto
```


### Issues
- test coverage is nil

### Testing
- example: $ go test ./node/ -test.v -test.run ^TestValidBlockHash$ 

## Contributing
If you'd like to contribute send me an email at tonyrriemer@gmail.com or message me on discord: driemworks#1849
