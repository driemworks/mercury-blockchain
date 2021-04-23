# Mercury Blockchain
Mercury Blockchain (or just Mercury) is a blockchain that acts as a decentralized, event-driven database.

## Motivation
The origin of this project stems from the question:
`How can you safely store data in a public IPFS gateway and how can you safely share the content with others?`

The resulting solution is Mercury, a blockchain that acts as a decentralized, event-driven database. Unlike a blockchain like bitcoin where transactions ultimately represent an exchange of some amount of bitcoin, transactions within Mercury represent a state mutation (as represented by a transaction type and an associated payload corresponding to a slice of the node state) published by a node. Transactions in this context only  require that a single node be available, and the mined blocks can be thought of as a global, append-only event log. Event sourcing allows  the *complete* state of the application to be rebuilt or recovered from any state/time, while the usage of a blockchain provides immutability to the stored events.

By configuring transaction types, associated schemas, and implementing appropriate behavior swfor handling state updates as a result of transactions with a given type being mined, Mercury can support nearly any use case. For example, see (TODO: ADD SOME EXAMPLE CONFIGS ONCE THIS IS BUILT OUT)

## Getting Started

### Pre requisites
- install `go`
- install `ipfs` (recommended)

### Installation 

- Install mercury from the source by cloning this repo and run `go install ./cli/...` from the root directory `mercury-blockchain/`.


- Install the latest version using:
```
go get github.com/driemworks/mercury-blockchain/cli/...
```
Note: if you're running linux you may need to set `export GO111MODULE=on`

## Usage
### CLI commands
`mercury [command] [options]`
- Available Commands:
  - `help`: Help about any command
  - `run`:  Run the mercury node
    -  options:
      - `--name`: (optional) The name of your node - Default: ?
      - `--datadir`: (optional) the directory where local data will be stored - Default: `.mercury`
      - `--ip`: (optional) the ip addreses of the mercury node - Default: `127.0.0.1`
      - `--port`: (optional) the port of the mercury node - Default: `8080`
      - `--miner`: (required) the public key to use (see: output of wallet command)
      - `--bootstrap-ip`: (optional) the ip address of the bootdstrap node - Default: `127.0.0.1`
      - `--bootstrap-port`: (optional) the port of the bootstrap node - Default: `8080`
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
  mercury run --datadir=./.mercury --name=Theo --miner=0x27084384033F90d96c3769e1b4fCE0E5ffff720B --port=8080 --bootstrap-ip=127.0.0.1 --bootstrap-port=8081
  ```

Note: This will only work if you have a publically exposed host.
To connect with the test network, use `--bootstrap-ip=ec2-34-207-242-13.compute-1.amazonaws.com` and `bootstrap-port=8080`
`mercury run --name=theo --datadir=.mercury/ --miner=0x990DB19D440124F3d5bA8867b3C35bC0D3c5Eda8 --ip=<your publicly exposed ip or dns address> --port=<your port> --bootstrap-ip=ec2-34-207-242-13.compute-1.amazonaws.com --bootstrap-port=8080`
There is a crude ui available to interact with your node. Run an ipfs node with `ipfs daemon` and navigate to `http://127.0.0.1:8080/ipfs/Qmc55mmfkrmTyhRRYsaU9d3sDUBbMPXrtExnrwVbuESEAY/build/`


Note: The ui is available via pinata, however, due to the crudeness of the UI it requires a local ipfs node to be running in order to be functional. https://gateway.pinata.cloud/ipfs/Qmc55mmfkrmTyhRRYsaU9d3sDUBbMPXrtExnrwVbuESEAY/build/

// ec2 public ip: 34.207.242.13

## API
Note: In order to use the API a node must be running.

### RPC
Mercury uses gRPC as a transport layer between nodes.

To interact with the RPC endpoints, I recommend using [grpcurl](https://github.com/fullstorydev/grpcurl).
Then you can run:
```
$ grpcurl 127.0.0.1:8081 proto.PublicNode/GetNodeStatus
{
  "address": "0xa7ED5257C26Ca5d8aF05FdE04919ce7d4a959147",
  "name": "tony",
  "hash": "0000000000000000000000000000000000000000000000000000000000000000"
}
```
Note: Pass the `-insecure` flag to grpcurl if you want to ignore any certificate issues.

To add a new pending transaction (publish a CID)
```
grpcurl -insecure 127.0.0.1:8081 proto.PublicNode/AddPendingPublishCIDTransaction <<EOM
{
	"cid": "a",
	"gateway": "b",
	"toAddress": "c",
	"name": "d"
}
EOM

```

### Development

The project is composed of the following packages:
#### cli
The `cli` package contains code and configs for the cli, as explained above.

#### common
Contains common structs and functions used across `cli`, `node`, `state`, and `wallet`

#### node
HTTP API

#### state
State management (used by the node). 
- block, transactions, etc

#### wallet
Create and manage your keystore

If you'd like to contribute send me an email at tonyrriemer@gmail.com or message me on discord: driemworks#1849

#### proto
To update server interface, run:
```
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/node.proto
```


### Issues
- don't sync with the bootstrap node's boostrap node (if it is itself)

### Testing
- example: $ go test ./node/ -test.v -test.run ^TestValidBlockHash$ 

## Acknowledgements
- This repository's basis is heavily influenced by this repo and the associated ebook https://github.com/web3coach/the-blockchain-bar