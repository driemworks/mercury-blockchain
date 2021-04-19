# FTP2P
FTP2P is a blockchain that acts as a decentralized, event-driven database.

## Motivation
The origin of this project stems from the question:
`How can you safely store data in a public IPFS gateway and how can you safely share the content with others?`

The resulting solution is FTP2P, a blockchain that acts as a decentralized, event-driven database. Unlike a blockchain like bitcoin where transactions ultimately represent an exchange of some amount of bitcoin, transactions within FTP2P represent a state mutation (as represented by a transaction type and an associated payload corresponding to a slice of the node state) published by a node. Transactions in this context only  require that a single node be available, and the mined blocks can be thought of as a global, append-only event log. Event sourcing allows  the *complete* state of the application to be rebuilt or recovered from any state/time, while the usage of a blockchain provides immutability to the stored events.

By configuring transaction types, associated schemas, and implementing appropriate behavior for handling state updates as a result of transactions with a given type being mined, FTP2P can support nearly any use case. For example, see (TODO: ADD SOME EXAMPLE CONFIGS ONCE THIS IS BUILT OUT)

## Getting Started

### Pre requisites
- install `go`
- install `ipfs` (recommended)

### Installation 

- Install ftp2p from the source by cloning this repo and run `go install ./cli/...` from the root directory `ftp2p/`.


- (TODO) Install f2p2p using:
```
go get github.com/driemworks/ftp2p/ftp2p
```

## Usage
### CLI commands
`ftp2p [command] [options]`
- Available Commands:
  - `help`: Help about any command
  - `run`:  Run the ftp2p node
    -  options:
      - `--name`: (optional) The name of your node - Default: ?
      - `--datadir`: (optional) the directory where local data will be stored - Default: `.ftp2p`
      - `--ip`: (optional) the ip addreses of the ftp2p node - Default: `127.0.0.1`
      - `--port`: (optional) the port of the ftp2p node - Default: `8080`
      - `--miner`: (required) the public key to use (see: output of wallet command)
      - `--bootstrap-ip`: (optional) the ip address of the bootstrap node - Default: `127.0.0.1`
      - `--bootstrap-port`: (optional) the port of the bootstrap node - Default: `8080`
  - `wallet`: Access the node's wallet
    - `new-address` Generate a new address
        -  options:
            - `--datadir`: (required) the directory where local data will be stored (ex: blockchain transactions)

 example:
  ```
  # generate a new address
  ftp2p wallet new-address --datadir=./.ftp2p
  >  0x27084384033F90d96c3769e1b4fCE0E5ffff720B
  # start a node using the new address as the miner
  ftp2p run --datadir=./.ftp2p --name=Theo --miner=0x27084384033F90d96c3769e1b4fCE0E5ffff720B --port=8080 --bootstrap-ip=127.0.0.1 --bootstrap-port=8081
  ```

## API
Note: In order to use the API a node must be running. 

See the [API documentation](https://github.com/driemworks/ftp2p/blob/master/docs/api/api.md)

### Node/Sync API
#### Pending Documentation
- `POST /node/sync`
- `POST /node/status`
- `POST /node/peer`


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


### Testing
- example: $ go test ./node/ -test.v -test.run ^TestValidBlockHash$ 

## Acknowledgements
- This repository's basis is heavily influenced by this repo and the associated ebook https://github.com/web3coach/the-blockchain-bar