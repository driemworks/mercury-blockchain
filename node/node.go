package node

import (
	"context"
	"driemcoin/main/manifest"
	"fmt"
	"net/http"
)

// const httpPort = 8080

type PeerNode struct {
	IP          string `json:"ip"`
	Port        uint64 `json:"port"`
	IsBootstrap bool   `json:"is_bootstrap"`
	connected   bool
}

func (p PeerNode) TcpAddress() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}

type Node struct {
	datadir    string
	ip         string
	port       uint64
	state      *manifest.State
	knownPeers map[string]PeerNode
}

func NewNode(datadir string, ip string, port uint64, boostrap PeerNode) *Node {
	knownPeers := make(map[string]PeerNode)
	knownPeers[boostrap.TcpAddress()] = boostrap
	return &Node{
		datadir:    datadir,
		ip:         ip,
		port:       port,
		knownPeers: knownPeers,
	}
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, connected bool) PeerNode {
	return PeerNode{ip, port, isBootstrap, connected}
}

/**
* Start the node's HTTP client
 */
func (n *Node) Run() error {
	ctx := context.Background()
	state, err := manifest.NewStateFromDisk(n.datadir)
	if err != nil {
		return err
	}
	defer state.Close()
	n.state = state

	go n.sync(ctx)

	// list manifest
	http.HandleFunc("/manifest/list", func(w http.ResponseWriter, r *http.Request) {
		listManifestHandler(w, r, state)
	})
	// send CID to someone
	http.HandleFunc("/cid/add", func(w http.ResponseWriter, r *http.Request) {
		addCIDHandler(w, r, state)
	})
	// get the nodes' status
	http.HandleFunc("/node/status", func(w http.ResponseWriter, r *http.Request) {
		nodeStatusHandler(w, r, state)
	})
	/* sync endpoints */
	// peer sync
	http.HandleFunc("/node/sync", func(w http.ResponseWriter, r *http.Request) {
		syncHandler(w, r, n)
	})
	// block sync
	http.HandleFunc("/node/peer", func(w http.ResponseWriter, r *http.Request) {
		addPeerHandler(w, r, n)
	})

	return http.ListenAndServe(fmt.Sprintf(":%d", n.port), nil)
}

func (n *Node) AddPeer(peer PeerNode) {
	n.knownPeers[peer.TcpAddress()] = peer
}

func (n *Node) RemovePeer(peer PeerNode) {
	delete(n.knownPeers, peer.TcpAddress())
}

func (n *Node) IsKnownPeer(peer PeerNode) bool {
	if peer.IP == n.ip && peer.Port == n.port {
		return true
	}

	_, isKnownPeer := n.knownPeers[peer.TcpAddress()]

	return isKnownPeer
}
