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

### API
`GET /mailbox`
#### Response:
    - **hash**: Your node's latest block's hash
    - **address**: Your node's address
    - **name:** Your (configured) name 
    - **mailbox** 
        - **inbox**: Transactions (CID/tokens) sent to this node by other nodes or yourself
        - **sent**: Transactions (CID/tokens) sent from this node to other nodes or yourself
        - **balance**: Your current balance
        - **pending_balance**: Your pending balance
Example:
```json
{
    "hash": "0100fd3640ff8c96de62cf96c559c8a9d5a6de5a2c1392c01efb9aa8d3256253",
    "address": "0x27084384033f90d96c3769e1b4fce0e5ffff720b",
    "name": "theo",
    "mailbox": {
        "sent": null,
        "inbox": [
            {
                "from": "0x5e79986470914df6cf60a232de6761bc862914c5",
                "cid": {
                    "cid": "QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQP",
                    "ipfs_gateway": "localhost:4001/ipfs/"
                },
                "hash": "699150c5d277d285a3563ed3b6d48ad8ba724c405486444101d3318869c1740a",
                "amount": 1
            }
        ],
        "balance": 1,
        "pending_balance": 1
    }
}
```
`POST /mailbox/send`
`POST /friends/add`
`POST /tokens`
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