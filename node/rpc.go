package node

import (
	"crypto/tls"
	"fmt"
	"net"

	pb "github.com/driemworks/mercury-blockchain/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

/**
Run an RPC server to serve the implementation of the NodeServer
*/
func (n *Node) runRPCServer(certFile string, keyFile string) error {
	// start the server
	var opts []grpc.ServerOption
	if n.tls {
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
