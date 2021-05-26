# Mercury Blockchain
Mercury is simple a blockchain built on top of go-libp2p. Currently, mercury only allows nodes to "publish" a topic encoded within a transaction, but the intention is achieve something similar to Holochain.

## TODOS
- Replace PoW with more efficient consensus algorithm.. proof of contribution?
- test coverage

## Getting Started
### Introduction
TODO

### Pre requisites
- install `go`
- install `ipfs` (recommended)

### Installation 
- Install mercury from the source by cloning this repo and run `go install ./cli/...` from the root directory `mercury-blockchain/`.


- Install the latest version using:
```
go get github.com/driemworks/mercury-blockchain/cli/...
```
Note: if you're running linux you may need to set `export GO111MODULE=on;go get github.com/driemworks/mercury-blockchain/cli/...`

## Usage
### CLI commands
`mercury [command] [options]`
- Available Commands:
  - `help`: Help about any command
  - `run`:  Run the mercury node
    -  options:
      - `--name`: (optional) The name of your node - Default: `""`
      - `--datadir`: (optional) the directory where local data will be stored - Default: `.mercury`
      - `--ip`: (optional) the ip addreses of the mercury node - Default: `127.0.01`
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
  mercury run --datadir=./.mercury --name=tony --port=8081 --miner=0x27084384033F90d96c3769e1b4fCE0E5ffff720B
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
The main functionality (to be extended...): Create a new pending transaction that, once mined, will allow us to send generic tx payloads across nodes.
`rpc AddTransaction(AddPendingPublishCIDTransactionRequest) returns (AddPendingPublishCIDTransactionResponse) {}`

Example
```
grpcurl -plaintext -d @ 127.0.0.1:9081 proto.NodeService/AddTransaction <<EOM
{
    "topic": "test"
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

### Testing
- example: $ go test ./node/ -test.v -test.run ^TestValidBlockHash$ 

## Contributing
If you'd like to contribute send me an email at tonyrriemer@gmail.com or message me on discord: driemworks#1849

## Acknowledgements
- This repository's basis is heavily influenced by this repo and the associated ebook https://github.com/web3coach/the-blockchain-bar