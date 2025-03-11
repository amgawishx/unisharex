package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	corehost "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
)

const rendezvousString = "p2p-mdns-auto-discovery"

// Message struct represents the data to be sent
type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// metadata of node
type NodeMetadata struct {
	Name string `json:"name"`
	OS   string `json:"os"`
	IP   string `json:"content"`
}

func NewNodeMetadata() *NodeMetadata {
	hostname, _ := os.Hostname()
	ip := "Unknown"

	conn, err := net.Dial("udp", "192.168.1.1:80") // Target any local network address
	if err != nil {
		fmt.Println("Error:", err)
	}
	defer conn.Close()
	ip = conn.LocalAddr().(*net.UDPAddr).IP.String()

	return &NodeMetadata{
		Name: hostname,
		OS:   runtime.GOOS,
		IP:   ip,
	}
}


// discoveryNotifee implements mdns.Notifee for peer discovery
type discoveryNotifee struct {
	PeerChan chan peer.AddrInfo
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Println("Discovered new peer:", pi.ID)
	n.PeerChan <- pi
}

// Initializes the mDNS service for automatic peer discovery
func initMDNS(peerhost corehost.Host) chan peer.AddrInfo {
	n := &discoveryNotifee{PeerChan: make(chan peer.AddrInfo)}
	ser := mdns.NewMdnsService(peerhost, rendezvousString, n)
	if err := ser.Start(); err != nil {
		panic(err)
	}
	return n.PeerChan
}

// Creates a secure libp2p host with QUIC transport
func createSecureHost() (corehost.Host, error) {
	privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %v", err)
	}

	return libp2p.New(
		libp2p.Identity(privKey),
		libp2p.Transport(quic.NewTransport),
		libp2p.Security("sec", noise.New),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/udp/0/quic-v1"),
	)
}

// Receiver listens for incoming secure messages and auto-discovers peers
func receiver(ctx context.Context) {
	host, err := createSecureHost()
	if err != nil {
		log.Fatalf("Failed to create secure host: %v", err)
	}
	defer host.Close()

	host.SetStreamHandler("/json/1.0.0", func(s network.Stream) {
		defer s.Close()
		var msg Message
		data, err := io.ReadAll(s)
		if err != nil {
			log.Printf("Failed to read data: %v", err)
			return
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Failed to decode JSON: %v", err)
			return
		}
		fmt.Printf("✅ Secure Message Received: %+v\n", msg)
	})

	go initMDNS(host)
	fmt.Printf("🔐 Secure Receiver started. Peer ID: %s\n", host.ID())
	fmt.Println("Listening on:")
	for _, addr := range host.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, host.ID())
	}

	// Handle incoming peer discovery
	// for {
	// 	peer := <-peerChan
	// 	fmt.Println("Auto-discovered peer:", peer.ID)
	// 	if err := host.Connect(ctx, peer); err != nil {
	// 		fmt.Println("Connection failed:", err)
	// 		continue
	// 	}
	// 	fmt.Println("Connected to:", peer.ID)
	// }
	select {}
}

// Sender sends a JSON message securely to discovered peers
func sender(ctx context.Context) {
	host, err := createSecureHost()
	if err != nil {
		log.Fatalf("Failed to create secure host: %v", err)
	}
	defer host.Close()

	peerChan := initMDNS(host)
	fmt.Println("🔍 Waiting to discover peers...")

	peer := <-peerChan // Wait until we discover a peer
	fmt.Println("Discovered peer:", peer.ID)

	if err := host.Connect(ctx, peer); err != nil {
		log.Fatalf("Failed to connect securely: %v", err)
	}
	fmt.Println("🔐 Securely connected to discovered peer!")

	stream, err := host.NewStream(ctx, peer.ID, "/json/1.0.0")
	if err != nil {
		log.Fatalf("Failed to open secure stream: %v", err)
	}
	defer stream.Close()

	msg := Message{Type: "Greeting", Content: "Hello, securely over libp2p QUIC!"}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	_, err = stream.Write(data)
	if err != nil {
		log.Fatalf("Failed to send secure data: %v", err)
	}

	fmt.Println("✅ Secure Message Sent Successfully!")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <receiver|sender>")
		return
	}

	ctx := context.Background()

	if os.Args[1] == "receiver" {
		receiver(ctx)
	} else if os.Args[1] == "sender" {
		sender(ctx)
	} else {
		fmt.Println("Invalid argument. Use 'receiver' or 'sender'.")
	}
}