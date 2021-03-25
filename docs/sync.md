# Sync
Syncing occurs every `X` seconds as defined by the node: `TODO`: Make this configurable... Currently it's hardcoded.

Below is a detailed explanation of how the sync algorithm works. This executes every `syncIntervalSeconds`.
## 0. Remove offline peers
Loop over all currently known peers. Query each peer node's status endpoint `peer_host:peer_port/node/status`. If the peers if unreachable, remove it from the known peers array and move to the next peer, otherwise continue. 

`note`:Peer sync is accomplished via a bootstrap node.

## 1. Add new peers
In order to find new peers, a node queries each peer in its "known peers" array for new peers by invoking the `node/peer` endpoint. If any new peers are discovered they are added as a known peer. If any peer node is unreachable, it is removed from known peers.

## 2. Block Sync

## 3. Pending Tx Sync