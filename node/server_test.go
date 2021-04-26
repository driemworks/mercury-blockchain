package node

// func Test_CanGetNodeStatus(t *testing.T) {
// 	pn := core.NewPeerNode("test", "127.0.0.1", 8081, true, state.NewAddress("0xa7ED5257C26Ca5d8aF05FdE04919ce7d4a959147"), true)
// 	// setup the node
// 	n := NewNode("test", ".test", "127.0.0.1", 8081, state.NewAddress("0xa7ED5257C26Ca5d8aF05FdE04919ce7d4a959147"), pn)
// 	manifest := make(map[common.Address]state.CurrentNodeState, 0)
// 	manifest[state.NewAddress("0xa7ED5257C26Ca5d8aF05FdE04919ce7d4a959147")] =
// 		state.CurrentNodeState{
// 			Balance: 0,
// 		}

// 	state := &state.State{
// 		Manifest: manifest,
// 	}
// 	n.state = state
// 	server := publicNodeServer{node: n}
// 	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
// 	res, err := server.GetNodeStatus(ctx, &pb.NodeInfoRequest{})
// 	if err != nil {
// 		t.Fatalf("%s", err)
// 	}
// 	assert.NotNil(t, res)
// }

// func Test_CanListKnownPeers(t *testing.T) {
// 	pn := core.NewPeerNode("test", "127.0.0.1", 8081, true, state.NewAddress("0xa7ED5257C26Ca5d8aF05FdE04919ce7d4a959147"), true)
// 	// setup the node
// 	n := NewNode("test", ".test", "127.0.0.1", 8081, state.NewAddress("0xa7ED5257C26Ca5d8aF05FdE04919ce7d4a959147"), pn)
// 	// manifest := make(map[common.Address]state.CurrentNodeState, 0)
// 	// manifest[state.NewAddress("0xa7ED5257C26Ca5d8aF05FdE04919ce7d4a959147")] =
// 	// 	state.CurrentNodeState{
// 	// 		Balance: 0,
// 	// 	}
// 	knownPeers := make(map[string]core.PeerNode, 0)
// 	knownPeers[""] = pn

// 	// state := &state.State{
// 	// 	Manifest: manifest,
// 	// }
// 	// n.state = state
// 	n.knownPeers = knownPeers
// 	server := publicNodeServer{node: n}
// 	// ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
// 	err := server.ListKnownPeers(&pb.ListKnownPeersRequest{})
// 	if err != nil {
// 		t.Fatalf("%s", err)
// 	}
// 	// assert.NotNil(t, res)
// }
