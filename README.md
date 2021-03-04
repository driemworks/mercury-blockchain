# FTP2P
FTP2P is a p2p file transfer tool.

## Getting Started

### Pre requisites
- FTP2P accomplishes decentralized file transfer by leveraging the `CID` of data stored in IPFS. 
Though not technically used in this project, for more info on IPFS, see https://docs.ipfs.io/install/ipfs-desktop/ 

### Installation 
- navigate to the root directory `ftp2p/` and run `go install ./cli/...` to install the go modules

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
  ftp2p run --datadir=./.ftp2p --name=Tony --miner=0x27084384033F90d96c3769e1b4fCE0E5ffff720B --port=8080
  ```

## API
See the [API documentation](https://github.com/driemworks/ftp2p/blob/master/docs/api/api.md)

### Node/Sync API
`POST /node/sync`
`POST /node/status`
`POST /node/peer`


### Development
TODO:
- [ ] Build encryption/decryption functionality (need to integrate with go-ethereum first) and expose via API
  - WIP: encryption/decryption is available in a limited way -> only for string data and you can only encrypt/decrypt for yourself
- [ ] gRPC migration
- [ ] research admin/moderation capabilities
- [ ] consider separating miner/api
- [ ] complete readme
- [ ] Add tests

### Testing
- example: $ go test ./node/ -test.v -test.run ^TestValidBlockHash$ 

## Acknowledgements
- This repository is heavily influenced by this repo and the associated ebook https://github.com/web3coach/the-blockchain-bar