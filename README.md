# FTP2P

File Transfer: Peer-to-Peer is a blockchain to facilitate sharing of data via IPFS.

### Installation
- navigate to main and run `go install ./cli/...`

## Usage
### CLI commands
`ftp2p [command] [options]`
- Available Commands:
  - help: Help about any command
  - run:  Run the ftp2p node
    -  options:
      - --datadir (required): the directory where local data will be stored (ex: blockchain transactions)
      - --ip: the ip addreses of the ftp2p node
      - -- port: the port of the ftp2p node
  - version: Display the current CLI version

  ex:
  ```
  ftp2p wallet new-address --datadir=./.ftp2p
  >  0x...
  ftp2p run --datadir=./.ftp2p --alias=tony --miner=0x... --port=8080
  ```

### API

### Local Setup

### Development
TODO:
- Validate CID before adding to tx: https://github.com/ipfs/go-cid 
- build encryption/decryption functionality (need to integrate with go-ethereum first)
- determine best way to upload to IPFS
- avoid sending duplicate files

### Testing
- ex: $ go test ./node/ -test.v -test.run ^TestValidBlockHash$

### Just some thoughts...
- by default there is duplicate transaction (i.e. duplicate file upload) protection!
- PoW is too slow... consider replacing


## Acknowledgements
- This repository is heavily influenced by this repo and the associated ebook https://github.com/web3coach/the-blockchain-bar