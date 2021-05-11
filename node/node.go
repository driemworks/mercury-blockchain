package node

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/driemworks/mercury-blockchain/core"
	pb "github.com/driemworks/mercury-blockchain/proto"
	"github.com/driemworks/mercury-blockchain/state"
	"github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	multiaddr "github.com/multiformats/go-multiaddr"
	"github.com/raphamorim/go-rainbow"
)

// var logger = log.Logger{Prefix: "rendezvous"}

func handleStream(stream network.Stream) {

}

func makeRoutedHost(listenPort int, randseed int64, bootstrapPeers []peer.AddrInfo, globalFlag string) (host.Host, error) {

	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
	}

	ctx := context.Background()

	basicHost, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// Construct a datastore (needed by the DHT). This is just a simple, in-memory thread-safe datastore.
	dstore := dsync.MutexWrap(ds.NewMapDatastore())

	// Make the DHT
	dht := dht.NewDHT(ctx, basicHost, dstore)

	// Make the routed host
	routedHost := rhost.Wrap(basicHost, dht)

	// connect to the chosen ipfs nodes
	err = bootstrapConnect(ctx, routedHost, bootstrapPeers)
	if err != nil {
		return nil, err
	}

	// Bootstrap the host
	err = dht.Bootstrap(ctx)
	if err != nil {
		return nil, err
	}

	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", routedHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	// addr := routedHost.Addrs()[0]
	addrs := routedHost.Addrs()
	log.Println("I can be reached at:")
	for _, addr := range addrs {
		log.Println(addr.Encapsulate(hostAddr))
	}

	log.Printf("Now run \"./routed-echo -l %d -d %s%s\" on a different terminal\n", listenPort+1, routedHost.ID().Pretty(), globalFlag)

	return routedHost, nil
}

func (n *Node) RunLibP2P() {
	ctx := context.Background()
	r := mrand.New(mrand.NewSource(int64(n.port)))
	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", n.ip, n.port))
	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Host created: %s", rainbow.Green(host.ID().Pretty()))
	fmt.Println()
	fmt.Println(host.Addrs())

	host.SetStreamHandler("/mercury/1.0.0", handleStream)
	kademliaDHT, err := dht.New(ctx, host)
	if err != nil {
		panic(err)
	}
	// bootstrap the dht
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}
	// connect to bootstrap nodes
	var wg sync.WaitGroup
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinfo); err != nil {
				fmt.Errorf("failed to establish connection with bootstrap", err)
			} else {
				fmt.Printf("Connection established with bootstrap node: %x", *peerinfo)
			}
			fmt.Println()
		}()
	}
	wg.Wait()
	// make yourself discoverable
	routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
	discovery.Advertise(ctx, routingDiscovery, "mercury")
	fmt.Println("Searching for peers...")
	peerChan, err := routingDiscovery.FindPeers(ctx, "mercury")
	for peer := range peerChan {
		if peer.ID == host.ID() {
			continue
		}
		fmt.Println()
		fmt.Printf("Found peer: %x", peer)
		fmt.Println()
		stream, err := host.NewStream(ctx, peer.ID, "/mercury/1.0.0")

		if err != nil {
			fmt.Errorf("Connection failed:", err)
			continue
		}
		_, err = stream.Write(n.info.Address.Bytes())
		if err != nil {
			fmt.Errorf("Failed to write to the stream", err)
			continue
		}
		out, err := ioutil.ReadAll(stream)
		fmt.Println(out)
		// } else {
		// sync pending transactions (using the stream)
		// rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
		// whenever there's a new pending transaction, push it to the stream
		// go writeData(n, rw)
		// go readData(rw)
		// }
	}
	select {}
}

const miningIntervalSeconds = 45
const syncIntervalSeconds = 2

// DiscoveryInterval is how often we re-publish our mDNS records.
const DiscoveryInterval = time.Minute

// DiscoveryServiceTag is used in our mDNS advertisements to discover other chat peers.
const DiscoveryServiceTag = "mercury-service-tag"

type Node struct {
	datadir         string
	ip              string
	port            uint64
	state           *state.State
	info            core.PeerNode
	trustedPeers    map[string]core.PeerNode
	pendingTXs      map[string]state.SignedTx
	archivedTXs     map[string]state.SignedTx
	newSyncedBlocks chan state.Block
	newPendingTXs   chan state.SignedTx
	isMining        bool
	name            string
	tls             bool
	host            host.Host
}

func NewNode(name string, datadir string, ip string, port uint64, tls bool) *Node {
	return &Node{
		name:            name,
		datadir:         datadir,
		ip:              ip,
		port:            port,
		pendingTXs:      make(map[string]state.SignedTx),
		archivedTXs:     make(map[string]state.SignedTx),
		newSyncedBlocks: make(chan state.Block),
		newPendingTXs:   make(chan state.SignedTx, 10000),
		isMining:        false,
		tls:             tls,
	}
}

/**
Run an RPC server to serve the implementation of the NodeServer
*/
func (n *Node) runRPCServer(tls bool, certFile string, keyFile string) error {
	// start the server
	var opts []grpc.ServerOption
	if tls {
		tlsCredentials, err := loadTLSCredentials()
		if err != nil {
			return err
		}
		opts = []grpc.ServerOption{grpc.Creds(tlsCredentials)}
	}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterPublicNodeServer(grpcServer, newNodeServer(n))
	reflection.Register(grpcServer) // must register reflection api in order to invoke rpc externally
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", n.ip, n.port+1000))
	if err != nil {
		fmt.Printf("Could not listen on %s:%d", n.ip, n.port+1000)
	}
	fmt.Println(fmt.Sprintf("Listening on: %s:%d", n.ip, n.port+1000))
	err = grpcServer.Serve(lis)
	if err != nil {
		return err
	}
	return nil
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	serverCert, err := tls.LoadX509KeyPair("resources/cert/server-cert.pem", "resources/cert/server-key.pem")
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.NoClientCert,
	}
	return credentials.NewTLS(config), nil
}

func (n *Node) Run(ctx context.Context, port int, peer string, name string) error {
	// load the state
	state, err := state.NewStateFromDisk(n.datadir)
	if err != nil {
		return err
	}
	defer state.Close()
	n.state = state
	go n.runRPCServer(n.tls, "", "")
	// go n.sync(ctx)
	// convert peer string to multiaddr and add to array
	err = n.runLibp2pNode(ctx, port, peer, name)
	if err != nil {
		return err
	}
	return nil
}

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
func addPeers(ctx context.Context, h host.Host, kad *dht.IpfsDHT, peersArg string, doRelay bool) {
	if len(peersArg) == 0 {
		return
	}

	peerStrs := strings.Split(peersArg, ",")
	for i := 0; i < len(peerStrs); i++ {
		peerID, peerAddr := MakePeer(peerStrs[i])
		h.Peerstore().AddAddr(peerID, peerAddr, peerstore.PermanentAddrTTL)
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
			err = h.Connect(ctx, *peerinfo)
			if err != nil {
				log.Fatalln(err)
			}
			s, err := h.NewStream(ctx, peerID, DiscoveryServiceTag)
			if err != nil {
				log.Fatalln(err)
			}
			s.Write([]byte(""))
		}
	}
}

func (n *Node) runLibp2pNode(ctx context.Context, port int, peer string, name string) error {
	host, err := makeHost(port, true)
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
	addPeers(ctx, host, kademliaDHT, peer, true)
	log.Printf("Listening on %v (Protocols: %v)", host.Addrs(), host.Mux().Protocols())
	// handle new nodes
	// strict bootstrap nodes
	if peer == "" {
		host.SetStreamHandler(DiscoveryServiceTag, func(s network.Stream) {
			// TODO this is super gross.. there has to be a better way to handle this
			fmt.Println("Adding new peer: ", s.Conn().RemoteMultiaddr().String()+"/p2p/"+s.Conn().RemotePeer().String())
			// extract peer id from the stream
			addPeers(ctx, host, kademliaDHT, s.Conn().RemoteMultiaddr().String()+"/p2p/"+s.Conn().RemotePeer().String(), false)
		})
	}
	// create a pubsub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Fatalln(err)
	}
	cr, err := JoinChatRoom(ctx, ps, host.ID(), name, "test")
	for {
		select {
		case m := <-cr.PendingTransactions:
			n.AddPendingTX(*m)
		}
	}
}

func (n *Node) mine(ctx context.Context) error {
	var miningCtx context.Context
	var stopCurrentMining context.CancelFunc

	ticker := time.NewTicker(time.Second * miningIntervalSeconds)

	for {
		select {
		case <-ticker.C:
			go func() {
				if len(n.pendingTXs) > 0 && !n.isMining {
					n.isMining = true

					miningCtx, stopCurrentMining = context.WithCancel(ctx)
					err := n.minePendingTXs(miningCtx)
					if err != nil {
						fmt.Printf(rainbow.Red("ERROR: %s\n"), err)
					}

					n.isMining = false
				}
			}()

		case block, _ := <-n.newSyncedBlocks:
			if n.isMining {
				blockHash, _ := block.Hash()
				fmt.Printf("\nPeer mined next Block '%s' faster :(\n", rainbow.Yellow(blockHash.Hex()))

				n.removeMinedPendingTXs(block)
				stopCurrentMining()
			}

		case <-ctx.Done():
			ticker.Stop()
			return nil
		}
	}
}

func (n *Node) minePendingTXs(ctx context.Context) error {
	blockToMine := NewPendingBlock(
		n.state.LatestBlockHash(),
		n.state.NextBlockNumber(),
		n.info.Address,
		n.getPendingTXsAsArray(),
	)
	minedBlock, err := Mine(ctx, blockToMine)
	if err != nil {
		return err
	}

	n.removeMinedPendingTXs(minedBlock)
	_, _, err = n.state.AddBlock(minedBlock)
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) removeMinedPendingTXs(block state.Block) {
	if len(block.TXs) > 0 && len(n.pendingTXs) > 0 {
		fmt.Println("Updating in-memory Pending TXs Pool:")
	}

	for _, tx := range block.TXs {
		txHash, _ := tx.Hash()
		if _, exists := n.pendingTXs[txHash.Hex()]; exists {
			fmt.Printf("\tArchiving mined TX: %s\n", rainbow.Yellow(txHash.Hex()))
			n.archivedTXs[txHash.Hex()] = tx
			delete(n.pendingTXs, txHash.Hex())
		}
	}
}

// func (n *Node) AddPeer(peer core.PeerNode) {
// 	n.knownPeers[peer.TcpAddress()] = peer
// }

// func (n *Node) RemovePeer(peer core.PeerNode) {
// 	delete(n.knownPeers, peer.TcpAddress())
// }

// func (n *Node) IsKnownPeer(peer core.PeerNode) bool {
// 	if peer.IP == n.info.IP && peer.Port == n.info.Port {
// 		return true
// 	}

// 	_, isKnownPeer := n.knownPeers[peer.TcpAddress()]

// 	return isKnownPeer
// }

/**
Add a pending transaction to the node's pending transactions array
*/
func (n *Node) AddPendingTX(tx state.SignedTx) error {
	txHash, err := tx.Hash()
	if err != nil {
		return err
	}

	_, isAlreadyPending := n.pendingTXs[txHash.Hex()]
	_, isArchived := n.archivedTXs[txHash.Hex()]

	if !isAlreadyPending && !isArchived {
		txJSON, err := json.Marshal(tx)
		if err != nil {
			return err
		}
		prettyTxJSON, err := core.PrettyPrintJSON(txJSON)
		if err != nil {
			return err
		}

		fmt.Printf("Adding pending transactions: \n%s\n", &prettyTxJSON)

		n.pendingTXs[txHash.Hex()] = tx
		n.newPendingTXs <- tx
		tmpFrom := n.state.Manifest[tx.From]
		if tmpFrom.Sent == nil {
			tmpFrom.Inbox = make([]state.InboxItem, 0)
			tmpFrom.Sent = make([]state.SentItem, 0)
			// uncomment the below in order to automatically reward new addresses
			// tmpFrom.Balance += manifest.BlockReward
			// tmpFrom.PendingBalance += tmpFrom.Balance
		}
		// TODO - the cost of the transaction is one coin for now, but should this always be the case?
		//         could file size factor into the cost? -> maybe when I get to the concept of gas?
		if tx.Amount > 0 {
			tmpFrom.PendingBalance -= tx.Amount
		}
		n.state.Manifest[tx.From] = tmpFrom
		// increase the account to nonce value => allows us to support mining blocks with multiple transactions
		n.state.PendingAccount2Nonce[tx.From]++
		tmpTo := n.state.Manifest[tx.To]
		// needed?
		if tmpTo.Inbox == nil {
			tmpTo.Sent = make([]state.SentItem, 0)
			tmpTo.Inbox = make([]state.InboxItem, 0)
			// uncomment the below in order to automatically reward new addresses
			// tmpTo.Balance += manifest.BlockReward
			// tmpTo.PendingBalance += tmpTo.Balance
		}
		tmpTo.PendingBalance += tx.Amount
		n.state.Manifest[tx.To] = tmpTo
	}
	return nil
}

func (n *Node) getPendingTXsAsArray() []state.SignedTx {
	txs := make([]state.SignedTx, len(n.pendingTXs))
	i := 0
	for _, tx := range n.pendingTXs {
		txs[i] = tx
		i++
	}
	return txs
}
