### API Documentation


<details>
    <summary>Get Node Details</summary>
    <div style="border:solid 1px black;padding:10px;">
        <span style="font-weight:bold">URL</span>: <span style="">/info</span>
        <br>
        <span style="font-weight:bold">Description</span>:  <span style="">Provides node address, name, and balance</span>
        <br>
        <span style="font-weight:bold">Method</span>:  <span style="">GET</span>
        <div style="border:solid 1px black;">
            <div style="padding:10px">
            <details>
                <summary>Request Parameters</summary>
                       -    
            </details>
            <details>
                <summary>Response</summary>
                <span style="font-weight:bold">address</span>: The node address <br>
                &emsp;<span style="font-weight:bold">name</span>: The name of the node<br>
                &emsp;<span style="font-weight:bold">balance</span>: The node's balance (FTC) <br>
                <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                    *: should add size of inbox, sent
                    <pre style="padding: 10px;">
{
    "address": "0x5e79986470914df6cf60a232de6761bc862914c5",
    "name" :"tony",
    "balance": 43332.556
}
                        </pre>
                    </div>
                </details>
            </div>
        </div>
    </div>
</details>

<details>
    <summary>Get Inbox</summary>
    <div style="border:solid 1px black;padding:10px;">
        <span style="font-weight:bold">URL</span>: <span style="">/inbox</span>
        <br>
        <span style="font-weight:bold">Description</span>:  <span style="">The inbox endpoint allows the owner of the node to view transactions for which it is the direct recipient</span>
        <br>
        <span style="font-weight:bold">Method</span>:  <span style="">GET</span>
        <div style="border:solid 1px black;">
            <div style="padding:10px">
            <details>
                    <summary>Request Parameters</summary>
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;padding:5px;">
                            <span style="font-weight:bold;">limit</span>: <span>The maximum number of items to retrieve. Default = none</span>
                            <span style="font-weight:bold;">from</span>: <span>The address to retrieve transactiosn from. Default = all</span>
                            <span style="font-weight:bold;">Example</span>: localhost:8080/inbox?from=0x5e79986470914df6Cf60a232dE6761Bc862914c5&limit=10
                        </div>         
                </details>
                <details>
                    <summary>Response</summary>
                    <span style="font-weight:bold">inbox</span>: A collection of transactions for which this node has been the recipient. e.g. A CID or FTC sent by another node. <br>
                    &emsp;<span style="font-weight:bold">from</span>: The node who authored the transactions <br>
                    &emsp;<span style="font-weight:bold">cid</span>: The object representing some data as a CID and where to find it in IPFS <br>
                    &emsp;&emsp;<span style="font-weight:bold">cid</span>: The CID <br>
                    &emsp;&emsp;<span style="font-weight:bold">ipfs_gateway</span>: The gateway that the data was uploaded to <br>
                    &emsp;<span style="font-weight:bold">hash</span>: The transaction's hash <br>
                    &emsp;<span style="font-weight:bold">amount</span>: The amount of FTC sent in the transaction <br>
                    <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                        <pre style="padding: 10px;">
{
    "inbox": [
        {
            "from": "0x5e79986470914df6cf60a232de6761bc862914c5",
            "cid": {
                "cid": "QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQP",
                "ipfs_gateway": "localhost:4001/ipfs/"
            },
            "hash": "699150c5d277d285a356............",
            "amount": 0
        }, 
        {
            "from": "0x5e79986470914df6cf60a232de6761bc862914c5",
            "cid": {},
            "hash": "699150c5d277d28.............",
            "amount": 10.017
        }]
    }
}
                        </pre>
                    </div>
                </details>
            </div>
        </div>
    </div>
</details>

<details>
    <summary>Get sent items</summary>
    <div style="border:solid 1px black;padding:10px;">
        <span style="font-weight:bold">URL</span>: <span style="">/sent</span>
        <br>
        <span style="font-weight:bold">Description</span>:  <span style="">The 'sent' endpoint allows the owner of the node to view transactions for which it is the author</span>
        <br>
        <span style="font-weight:bold">Method</span>:  <span style="">GET</span>
        <div style="border:solid 1px black;">
            <div style="padding:10px">
                <details>
                    <summary>Request Parameters</summary>
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;padding:5px;">
                            <span style="font-weight:bold;">limit</span>: <span>The maximum number of items to retrieve. Default = none</span>
                            <span style="font-weight:bold;">from</span>: <span>The address to retrieve transactiosn from. Default = all</span>
                            <span style="font-weight:bold;">Example</span>: localhost:8080/sent?to=0x5e79986470914df6Cf60a232dE6761Bc862914c5&limit=10
                        </div>         
                </details>
                <details>
                    <summary>Response</summary>
                    <span style="font-weight:bold">inbox</span>: A collection of transactions for which this node has been the recipient. e.g. A CID or FTC sent by another node. <br>
                    &emsp;<span style="font-weight:bold">from</span>: The node who authored the transactions <br>
                    &emsp;<span style="font-weight:bold">cid</span>: The object representing some data as a CID and where to find it in IPFS <br>
                    &emsp;&emsp;<span style="font-weight:bold">cid</span>: The CID <br>
                    &emsp;&emsp;<span style="font-weight:bold">ipfs_gateway</span>: The gateway that the data was uploaded to <br>
                    &emsp;<span style="font-weight:bold">hash</span>: The transaction's hash <br>
                    &emsp;<span style="font-weight:bold">amount</span>: The amount of FTC sent in the transaction <br>
                    <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                        <pre style="padding: 10px;">
{
    "inbox": [
        {
            "to": "0x5e79986470914df6cf60a232de6761bc862914c5",
            "cid": {
                "cid": "QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQP",
                "ipfs_gateway": "localhost:4001/ipfs/"
            },
            "hash": "699150c5d277d285a356............",
            "amount": 0
        }, 
        {
            "to": "0x5e79986470914df6cf60a232de6761bc862914c5",
            "cid": { },
            "hash": "699150c5d277d28.............",
            "amount": 10.017
        }]
    }
}
                        </pre>
                    </div>
                </details>
            </div>
        </div>
    </div>
</details>

<details>
    <summary>Send FTC</summary>
    <div style="border:solid 1px black;padding:10px;">
        <span style="font-weight:bold">URL</span>: <span style="">/send/tokens</span>
        <br>
        <span style="font-weight:bold">Description</span>:  <span style="">The '/send/tokens' endpoint allows a node to send tokens to another node</span>
        <br>
        <span style="font-weight:bold">Method</span>:  <span style="">POST</span>
        <div style="border:solid 1px black;">
            <div style="padding:10px">
                <details>
                    <summary>Request</summary>
                        <span style="font-weight:bold">to</span>: The address of the node that the amount will be sent to.
                        <span style="font-weight:bold">amount</span>: The amount in FTC to send to the 'to' address
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                            <pre style="padding: 10px;">
{
    "to": "0x...",
    "amount": 17.332
}
                            </pre>
                        </div>
                </details>
                <details>
                    <summary>Response</summary>
                        <span style="font-weight:bold">has</span>: The hash of the pending transaction
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                            <pre style="padding: 10px;">
{
    "hash": "ad99es..."
}
                            </pre>
                        </div> 
                </details>
            </div>
        </div>
    </div>
</details>

<details>
    <summary>Publish Content</summary>
    <div style="border:solid 1px black;padding:10px;">
        <span style="font-weight:bold">URL</span>: <span style="">/publish</span>
        <br>
        <span style="font-weight:bold">Description</span>:  <span style="">The '/publish' endpoint allows a node to publish content so that it is visible to the network, with visibility determined by the 'to' request parameter.</span>
        <br>
        <span style="font-weight:bold">Method</span>:  <span style="">POST</span>
        <div style="border:solid 1px black;">
            <div style="padding:10px">
                <details>
                    <summary>Request</summary>
                        <span style="font-weight:bold">to</span>: The address of the node that the amount will be sent to.
                        <span style="font-weight:bold">amount</span>: The amount in FTC to send to the 'to' address
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                            <pre style="padding: 10px;">
{
    "to": "0x...",
    "cid": "Qm...",
    "ipfs_gateway": "example.ipfs.io"
}
                            </pre>
                        </div>
                </details>
                <details>
                    <summary>Response</summary>
                        <span style="font-weight:bold">hash</span>: The hash of the pending transaction
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                            <pre style="padding: 10px;">
{
    "hash": "ad99es..."
}
                            </pre>
                        </div> 
                </details>
            </div>
        </div>
    </div>
</details>

<details>
    <summary>Get Known Peers</summary>
    <div style="border:solid 1px black;padding:10px;">
        <span style="font-weight:bold">URL</span>: <span style="">/peers/known</span>
        <br>
        <span style="font-weight:bold">Description</span>:  <span style="">Retrieve a list of all known (online and synced) peers</span>
        <br>
        <span style="font-weight:bold">Method</span>:  <span style="">GET</span>
        <div style="border:solid 1px black;">
            <div style="padding:10px">
                <details>
                    <summary>Request Parameters</summary>
                        <span style="font-weight:bold">limit</span>: the maximum number of peers to retrieve
                </details>
                <details>
                    <summary>Response</summary>
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                            <span style="font-weight:bold">known_peers</span>: The collection of trusted peers
                            &nbsp;<span style="font-weight:bold">address</span>: The peer node's address
                            &nbsp;<span style="font-weight:bold">name</span>: The peer node's name
                            &nbsp;<span style="font-weight:bold">ip</span>: The peer node's ip address
                            &nbsp;<span style="font-weight:bold">port</span>: The peer node's port
                            <span>Example: </span>
                            <pre style="padding: 10px;">
{
  "known_peers": [
      {
          "address": "0x0...",
          "name": "theo",
          "ip": "192.168.1.201",
          "port": "5002"
      },
      {
          "address": "0x1...",
          "name": "athena",
          "ip": "192.168.1.202",
          "port": "5002"
      }
  ]  
}
                            </pre>
                        </div> 
                </details>
            </div>
        </div>
    </div>
</details>

<details>
    <summary>Get Trusted Peers</summary>
    <div style="border:solid 1px black;padding:10px;">
        <span style="font-weight:bold">URL</span>: <span style="">/peers/trusted</span>
        <br>
        <span style="font-weight:bold">Description</span>:  <span style="">Retrieve a list of all trusted peers</span>
        <br>
        <span style="font-weight:bold">Method</span>:  <span style="">GET</span>
        <div style="border:solid 1px black;">
            <div style="padding:10px">
                <details>
                    <summary>Request Parameters</summary>
                        <span style="font-weight:bold">limit</span>: the maximum number of peers to retrieve
                </details>
                <details>
                    <summary>Response</summary>
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                            <span style="font-weight:bold">trusted_peers</span>: The collection of trusted peers
                            &nbsp;<span style="font-weight:bold">address</span>: The peer node's address
                            &nbsp;<span style="font-weight:bold">name</span>: The peer node's name
                            &nbsp;<span style="font-weight:bold">ip</span>: The peer node's ip address
                            &nbsp;<span style="font-weight:bold">port</span>: The peer node's port
                            <span>Example: </span>
                            <pre style="padding: 10px;">
{
  "trusted_peers": [
      {
          "address": "0x0...",
          "name": "theo",
          "ip": "192.168.1.201",
          "port": "5002"
      },
      {
          "address": "0x1...",
          "name": "athena",
          "ip": "192.168.1.202",
          "port": "5002"
      }
  ]  
}
                            </pre>
                        </div> 
                </details>
            </div>
        </div>
    </div>
</details>

<details>
    <summary>Follow</summary>
    <div style="border:solid 1px black;padding:10px;">
        <span style="font-weight:bold">URL</span>: <span style="">/follow</span>
        <br>
        <span style="font-weight:bold">Description</span>:  <span style="">"Follow" another node. This a llows a node to append known peers to their trusted peers list. A trusted peer is info of a particular peer node that can be easily retrieved/accessed even if the peer is offline</span>
        <br>
        <span style="font-weight:bold">Method</span>:  <span style="">POST</span>
        <div style="border:solid 1px black;">
            <div style="padding:10px">
                <details>
                    <summary>Request</summary>
                        <span style="font-weight:bold">tcp_address</span>: The tcp address of the node to add to trusted peers.
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                            <pre style="padding: 10px;">
{
    "tcp_address": "192.168.1.201:4001"
}
                            </pre>
                        </div>
                </details>
                <details>
                    <summary>Response</summary>
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                            <span>200 OK</span>
                        </div> 
                </details>
            </div>
        </div>
    </div>
</details>

<details>
    <summary>Encrypt data</summary>
    <span style="font-weight:bold;color:blue;">Note</span>: There are some issues with this design.. primarily that you have to pass a password in plaintext.. need to fix this somehow. There are also issues with the data only allowing string content.
    <div style="border:solid 1px black;padding:10px;">
        <span style="font-weight:bold">URL</span>: <span style="">/encrypt</span>
        <br>
        <span style="font-weight:bold">Description</span>:  <span style="">Assymetrically encrypt data that can only be decrypted by the 'to' address</span>
        <br>
        <span style="font-weight:bold">Method</span>:  <span style="">POST</span>
        <div style="border:solid 1px black;">
            <div style="padding:10px">
                <details>
                    <summary>Request</summary>
                        <span style="font-weight:bold">from_pwd</span>: Your password
                        <span style="font-weight:bold">to</span>: The public key (i.e. address) who can decrypt the encrypted data
                        <span style="font-weight:bold">data</span>: The data to encrypt
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                            <pre style="padding: 10px;">
{
    "from_pwd": "test",
    "to": "0x5e79986470914df6cf60a232de6761bc862914c5",
    "data": "Hello there"
}
                            </pre>
                        </div>
                </details>
                <details>
                    <summary>Response</summary>
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                            <span>200 OK</span>
                        </div> 
                </details>
            </div>
        </div>
    </div>
</details>

<details>
    <summary>Decrypt data</summary>
    <span style="font-weight:bold;color:blue;">Note</span>: There are some issues with this design.. primarily that you have to pass a password in plaintext.. need to fix this somehow. There are also issues with the data only allowing string content.
    <div style="border:solid 1px black;padding:10px;">
        <span style="font-weight:bold">URL</span>: <span style="">/encrypt</span>
        <br>
        <span style="font-weight:bold">Description</span>:  <span style="">Decrypt assymetrically encrypted data</span>
        <br>
        <span style="font-weight:bold">Method</span>:  <span style="">POST</span>
        <div style="border:solid 1px black;">
            <div style="padding:10px">
                <details>
                    <summary>Request</summary>
                        <span style="font-weight:bold">encrypted_data</span>: The structure representing encrypted data along with method of encryption.
                        &emsp;<span style="font-weight:bold">version</span>: The encryption version used (only 'x25519-xsalsa20-poly1305' currently supported)
                        &emsp;<span style="font-weight:bold">nonce</span>: The nonce
                        &emsp;<span style="font-weight:bold">public_key</span>: The public key associated with the private key of the address that encrypted this data.
                        &emsp;<span style="font-weight:bold">cipher_text</span>: The encrypted data
                        <span style="font-weight:bold">from_pwd</span>: Your password
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                            <pre style="padding: 10px;">
{
    "encrypted_data": {
        "version": "x25519-xsalsa20-poly1305",
        "nonce": "6gpqHx8uQp7iGyRIYISOpUYrGa0CdEku",
        "public_key": "pWK0XMJJs5tbXOz9Zo7z+HDPJ1iDgG6KyzhtfYd4Eg4=",
        "cipher_text": "ms5HHlzn3i/Srah2Gh+iuPKblbBvmelrjFMV"
    },
    "from_pwd": "test"
}
                            </pre>
                        </div>
                </details>
                <details>
                    <summary>Response</summary>
                        <div style="background:lightGray;font-family:helvetica;border:solid 1px black;margin:10px;">
                            <span>200 OK</span>
                        </div> 
                </details>
            </div>
        </div>
    </div>
</details>