package main

import (
	"context"
	"fmt"
	"log"

	wg "github.com/aau-network-security/gwireguard/proto"
	"github.com/dgrijalva/jwt-go"

	"google.golang.org/grpc"
)

type Creds struct {
	Token    string
	Insecure bool
}

func (c Creds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"token": string(c.Token),
	}, nil
}

func (c Creds) RequireTransportSecurity() bool {
	return !c.Insecure
}

func main() {
	// change the endpoint address with your instance ip
	endpointAddress := "40.127.143.202"
	var conn *grpc.ClientConn
	// wg is AUTH_KEY from vpn/auth.go
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"wg": "deneme",
	})

	tokenString, err := token.SignedString([]byte("test"))
	if err != nil {
		fmt.Println("Error creating the token")
	}

	authCreds := Creds{Token: tokenString}
	dialOpts := []grpc.DialOption{}
	authCreds.Insecure = true
	dialOpts = append(dialOpts,
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(authCreds))

	conn, err = grpc.Dial(fmt.Sprintf("%s:5353", endpointAddress), dialOpts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	if err == nil {
		fmt.Printf("Client is connected successfully !")
	}
	defer conn.Close()

	client := wg.NewWireguardClient(conn)
	ctx := context.TODO()

	_, err = client.InitializeI(ctx, &wg.IReq{
		Address:    "10.2.11.1/24", // this should be randomized and should not collide with lab subnet like 124.5.6.0/24
		ListenPort: 51820,          // this should be randomized and should not collide with any used ports by host
		SaveConfig: true,
		Eth:        "eth0",
		IName:      "wg",
	})
	if err != nil {
		panic(err)
	}

	//log.Info().Msg("Getting server public key...")
	serverPubKey, err := client.GetPublicKey(ctx, &wg.PubKeyReq{PubKeyName: "wg", PrivKeyName: "wg"})
	if err != nil {
		panic(err)
	}

	resp, err := client.GenPublicKey(ctx, &wg.PubKeyReq{
		PubKeyName:  "client1",
		PrivKeyName: "client1",
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf(resp.Message)

	publicKey, err := client.GetPublicKey(ctx, &wg.PubKeyReq{PubKeyName: "client1"})
	if err != nil {
		panic(err)
	}
	clientPrivKey, err := client.GetPrivateKey(ctx, &wg.PrivKeyReq{PrivateKeyName: "client1"})
	if err != nil {
		panic(err)
	}

	peerIP := "10.2.11.3/32"

	_, err = client.AddPeer(ctx, &wg.AddPReq{
		Nic:        "wg",
		AllowedIPs: peerIP,
		PublicKey:  publicKey.Message,
	})
	if err != nil {
		panic(err)
	}

	// change allowed ips according to vlan ip that you would like to connect

	allowedIps := "10.134.130.1/24"
	clientConfig := fmt.Sprintf(
		`[Interface]
Address = %s
PrivateKey = %s
DNS = 1.1.1.1
MTU = 1500
[Peer]
PublicKey = %s
AllowedIps = %s
Endpoint =  %s
PersistentKeepalive = 25`, peerIP, clientPrivKey.Message, serverPubKey.Message, allowedIps, fmt.Sprintf("%s:51820", endpointAddress))

	fmt.Printf("Client Config \n %s ", clientConfig)
}
