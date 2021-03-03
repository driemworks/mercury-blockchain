# FTP2P
FTP2P is a blockchain to allow users (nodes) to securely share encrypted data with trusted peers without the need for a server. This is accomplished by through the use of asymmetric encryption and IPFS. 

## Getting Started
- The storage layer required by FTP2P must be an IPFS gateway, public or private. This allows nodes to choose where their data is stored (WIP). 
    - for more info on IPFS, see https://docs.ipfs.io/install/ipfs-desktop/ 

### Installation 
- run `go install ./cli/...`

## Usage
### CLI commands
`ftp2p [command] [options]`
- Available Commands:
  - `help`: Help about any command
  - `run`:  Run the ftp2p node
    -  options:
      - `--datadir`: (required) the directory where local data will be stored (ex: blockchain transactions)
      - `--ip`: the ip addreses of the ftp2p node
      - `--port`: the port of the ftp2p node
      - `--version`: Display the current CLI version
  - `wallet`: Access the node's wallet
    - `new-address` Generate a new address
        -  options:
            - `--datadir`: (required) the directory where local data will be stored (ex: blockchain transactions)

 example:
  ```
  # generate a new address
  ftp2p wallet new-address --datadir=./.ftp2p
  >  0x27084384033F90d96c3769e1b4fCE0E5ffff720B
  ftp2p run --datadir=./.ftp2p --name=Tony --miner=0x27084384033F90d96c3769e1b4fCE0E5ffff720B --port=8080
  ```

## API




### Node API -> could become rpc endpoints?
`POST /node/sync`
`POST /node/status`
`POST /node/peer`

### Local Setup

### Development
TODO:
- [-] Build encryption/decryption functionality (need to integrate with go-ethereum first) and expose via API
  - WIP: encryption/decryption is available in a limited way -> only for string data and you can only encrypt/decrypt for yourself
- [-] gRPC integration
- [-] consider separating miner/api
- [-] complete readme
- [-] Add tests

### Testing
- ex: $ go test ./node/ -test.v -test.run ^TestValidBlockHash$ 

## Acknowledgements
- This repository is heavily influenced by this repo and the associated ebook https://github.com/web3coach/the-blockchain-bar