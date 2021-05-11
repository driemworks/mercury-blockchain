package node

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	noise "github.com/libp2p/go-libp2p-noise"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
)

func makeHost(port int, insecure bool) (host.Host, error) {
	r := rand.Reader
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
		libp2p.Security(noise.ID, noise.New),
	}
	if insecure {
		opts = append(opts, libp2p.NoSecurity)
	}
	host, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", host.ID().Pretty()))
	addr := host.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	fmt.Printf("I am %s\n", fullAddr)
	fmt.Println(host.ID().Pretty())
	return host, nil
}

/*
	Manually add a peer to the DHT
	If doRelay = true, then open a new stream and announce yourself to the new peer
*/
func addPeers(ctx context.Context, n Node, kad *dht.IpfsDHT, peersArg string, doRelay bool) {
	if len(peersArg) == 0 {
		return
	}

	peerStrs := strings.Split(peersArg, ",")
	for i := 0; i < len(peerStrs); i++ {
		peerID, peerAddr := MakePeer(peerStrs[i])
		n.host.Peerstore().AddAddr(peerID, peerAddr, peerstore.PermanentAddrTTL)
		_, err := kad.RoutingTable().TryAddPeer(peerID, false, false)
		if err != nil {
			log.Fatalln(err)
		}
		// if the peer is already in the table, do not call it again
		if doRelay {
			peerinfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
			if err != nil {
				log.Fatalln(err)
			}
			err = n.host.Connect(ctx, *peerinfo)
			if err != nil {
				log.Fatalln(err)
			}
			s, err := n.host.NewStream(ctx, peerID, DiscoveryServiceTag)
			if err != nil {
				log.Fatalln(err)
			}
			js, err := json.Marshal(n.pendingTXs)
			if err != nil {
				log.Fatalln(err)
			}
			s.Write([]byte(js))
		}
	}
}

func (n *Node) runLibp2pNode(ctx context.Context, port int, peer string, name string) error {
	host, err := makeHost(port, false)
	n.host = host
	if err != nil {
		return err
	}
	// 1) Start a DHT
	kademliaDHT, err := dht.New(ctx, host)
	if err != nil {
		log.Fatal(err)
	}
	defer kademliaDHT.Close()
	// add bootstrap nodes if provided
	addPeers(ctx, *n, kademliaDHT, peer, true)
	log.Printf("Listening on %v (Protocols: %v)", host.Addrs(), host.Mux().Protocols())
	// handle new nodes
	// strict bootstrap nodes
	if peer == "" {
		host.SetStreamHandler(DiscoveryServiceTag, func(s network.Stream) {
			// TODO this is pretty bad...
			addPeers(ctx, *n, kademliaDHT, s.Conn().RemoteMultiaddr().String()+"/p2p/"+s.Conn().RemotePeer().String(), false)
		})
	}
	// else {
	// 	host.SetStreamHandler(DiscoveryServiceTag, func(s network.Stream) {
	// 		// sync incoming bulk pending txs from bootstrap node
	// 		reader := bufio.NewReader(s)
	// 		data, err := reader.ReadBytes(0)
	// 		if err != nil {
	// 			log.Fatalln(err)
	// 		}
	// 		var txs []state.SignedTx
	// 		err = json.Unmarshal(data, &txs)
	// 		if err != nil {
	// 			log.Fatalln(err)
	// 		}
	// 		fmt.Printl
	// 		for _, tx := range txs {
	// 			n.AddPendingTX(tx)
	// 		}
	// 	})
	// }
	// create a pubsub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Fatalln(err)
	}
	// will need a channel for syncing:
	// 1) pending txs
	// 2) new blocks
	cr, err := JoinChannel(ctx, ps, host.ID(), name, "sync")
	for {
		select {
		case m := <-cr.PendingTransactions:
			n.AddPendingTX(*m)
		case tx := <-n.newPendingTXs:
			cr.Publish(&tx)
		}
	}
}
