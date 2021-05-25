package node

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/driemworks/mercury-blockchain/state"
	"github.com/fatih/structs"
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

const (
	DiscoveryServiceTag            = "mercury-service-tag"
	DiscoveryServiceTag_PendingTxs = "pending_txs"
)

/*
	Build a libp2p host
*/
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
	If doRelay = true then opena connection with the peer
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
		// if the peer is already in the DHT, do not call it again
		if doRelay {
			peerinfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
			if err != nil {
				log.Fatalln(err)
			}
			err = n.host.Connect(ctx, *peerinfo)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func (n *Node) runLibp2pNode(ctx context.Context, port int, bootstrapPeer string, name string) error {
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
	if bootstrapPeer == "" {
		// TODO leaving as localhost for now. Should this be configurable?
		bootstrapPeer = fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", "127.0.0.1", port, host.ID().Pretty())
	} else {
		addPeers(ctx, *n, kademliaDHT, bootstrapPeer, true)
	}

	log.Printf("Listening on %v (Protocols: %v)", host.Addrs(), host.Mux().Protocols())
	host.SetStreamHandler(pubsub.RemoteTracerProtoID, func(s network.Stream) {
		// called when peer connects only?? verify this...
		s, err := host.NewStream(ctx, s.Conn().RemotePeer(), "PENDING_TX_SYNC")
		if err != nil {
			log.Fatalln(err)
		}
		pendingJson, err := json.Marshal(n.pendingTXs)
		if err != nil {
			log.Fatalln(err)
		}
		pendingJson = append(pendingJson, '\n')
		s.Write(pendingJson)
	})
	host.SetStreamHandler("PENDING_TX_SYNC", func(s network.Stream) {
		buf := bufio.NewReader(s)
		bytes, err := buf.ReadBytes('\n')
		if err != nil {
			log.Fatalln(err)
		}
		var txs map[string]state.SignedTx
		err = json.Unmarshal(bytes, &txs)
		if err != nil {
			log.Fatalln(err)
		}
		n.pendingTXs = txs
	})
	peerinfo, err := peer.AddrInfoFromP2pAddr(multiaddr.StringCast(bootstrapPeer))
	tracer, err := pubsub.NewRemoteTracer(ctx, host, *peerinfo)
	if err != nil {
		panic(err)
	}
	// create a pubsub service using the GossipSub router
	var ps *pubsub.PubSub
	ps, err = pubsub.NewGossipSub(ctx, host, pubsub.WithEventTracer(tracer))
	if err != nil {
		log.Fatalln(err)
	}
	pending_tx_channel, err := InitChannel(ctx, PENDING_TX_TOPIC, 128, ps, host.ID())
	new_block_cr, err := JoinNewBlockExchange(ctx, ps, host.ID())
	for {
		select {
		// youre reading a tx from the stream
		case data := <-pending_tx_channel.Data:
			txMap := data["Tx"]
			bytes, err := json.Marshal(txMap)
			if err != nil {
				return err
			}
			var tx state.Tx
			err = json.Unmarshal(bytes, &tx)
			if err != nil {
				return err
			}
			sig_string := fmt.Sprintf("%v", data["Sig"])
			sig, _ := base64.RawStdEncoding.DecodeString(sig_string)
			signedTx := state.NewSignedTx(tx, sig)
			n.AddPendingTX(signedTx)
			//add to dht... how to clear?
			kademliaDHT.PutValue(ctx, "PENDING_TXS", bytes)
		// youre writing txs to the stream
		case tx := <-n.newPendingTXs:
			txMap := structs.Map(tx)
			pending_tx_channel.Publish(txMap)
		case b := <-new_block_cr.NewBlocks:
			s, _, err := n.state.AddBlock(*b)
			if err != nil {
				if s != nil {
					n.state = s
				}
				return err
			}
			n.newSyncedBlocks <- *b
		case block := <-n.newMinedBlocks:
			new_block_cr.Publish(&block)
		}
	}
}

type TransactionTransmissionWrapper struct {
	SignedTx state.SignedTx `json:"signed_transaction"`
}
