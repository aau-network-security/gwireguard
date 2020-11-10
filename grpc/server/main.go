package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/aau-network-security/gwireguard/config"

	"google.golang.org/grpc/reflection"

	proto "github.com/aau-network-security/gwireguard/proto"
	wg "github.com/aau-network-security/gwireguard/vpn"
)

var (
	configPath = os.Getenv("CONFIG_PATH")
)

func main() {
	if configPath == "" {
		panic("Set CONFIG_PATH environment variable correctly ! ")
	}
	configuration, err := config.NewConfig(configPath)
	if err != nil {
		panic("Configuration initialization error: " + err.Error())
	}
	port := strconv.FormatUint(uint64(configuration.ServiceConfig.Domain.Port), 10)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	wgServer, err := wg.InitServer(configuration)
	if err != nil {
		return
	}
	opts, err := wg.SecureConn(configuration)
	if err != nil {
		log.Fatalf("failed to retrieve secure options %s", err.Error())
	}

	gRPCEndpoint := wgServer.AddAuth(opts...)

	reflection.Register(gRPCEndpoint)
	proto.RegisterWireguardServer(gRPCEndpoint, wgServer)

	fmt.Printf("wireguard gRPC server is running at port %s...\n", port)
	if err := gRPCEndpoint.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
