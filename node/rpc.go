package node

import (
	"crypto/tls"
	"fmt"
	"net"

	pb "github.com/driemworks/mercury-blockchain/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

/**
Run an RPC server to serve the implementation of the NodeServer
*/
func (n *Node) runRPCServer(certFile string, keyFile string, host string, port uint64) error {
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
	pb.RegisterNodeServiceServer(grpcServer, newNodeServer(n))
	reflection.Register(grpcServer) // register reflection api in order to invoke rpc externally
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	log.Infoln(fmt.Sprintf("RPC server listening on: %s:%d", host, port))
	err = grpcServer.Serve(lis)
	if err != nil {
		return err
	}
	return nil
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	serverCert, err := tls.LoadX509KeyPair("resources/cert/server-cert.pem", "resources/cert/server-key.pem")
	if err != nil {
		return nil, err
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.NoClientCert,
	}
	return credentials.NewTLS(config), nil
}
