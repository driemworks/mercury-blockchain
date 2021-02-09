# FTP2P

FTP2P is a blockchain to facilitate sharing of data via IPFS.

### Installation
- navigate to main and run `go install ./cli/...`

### Usage
- List the version
```
ftp2p version
```

- Share CID with someone
```
ftp2p send --from="<name1>" --to="<name2>" --cid="Qmadfj83f3..."
```

- List the manifest
```
ftp2p manifest list
```

### Local Setup

### Development
TODO:
- Validate CID before adding to tx: https://github.com/ipfs/go-cid 
- build encryption/decryption functionality (need to integrate with go-ethereum first)
- determine best way to upload to IPFS
- avoid sending duplicate files
