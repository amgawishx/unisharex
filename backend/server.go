package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	corehost "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/multiformats/go-multiaddr"
)

// Message struct represents the data to be sent
type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Creates a secure libp2p host with QUIC transport
func createSecureHost() (corehost.Host, error) {
	// Generate a new cryptographic key pair
	privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %v", err)
	}

	// Create a new libp2p host with QUIC transport and Noise security
	return libp2p.New(
		libp2p.Identity(privKey),                     // Set secure peer identity
		libp2p.Transport(quic.NewTransport),         // Use QUIC transport
		libp2p.Security("sec", noise.New),                  // Use Noise security for encryption
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/udp/0/quic-v1"), // Listen on any available port
	)
}

// Receiver listens for incoming secure messages
func receiver(ctx context.Context) {
	host, err := createSecureHost()
	if err != nil {
		log.Fatalf("Failed to create secure host: %v", err)
	}
	defer host.Close()

	host.SetStreamHandler("/json/1.0.0", func(s network.Stream) {
		defer s.Close()

		// Read JSON data
		var msg Message
		data, err := io.ReadAll(s)
		if err != nil {
			log.Printf("Failed to read data: %v", err)
			return
		}

		// Decode JSON
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Failed to decode JSON: %v", err)
			return
		}

		fmt.Printf("✅ Secure Message Received: %+v\n", msg)
	})

	fmt.Printf("🔐 Secure Receiver started. Peer ID: %s\n", host.ID())
	fmt.Println("Listening on:")
	for _, addr := range host.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, host.ID())
	}

	// Keep the receiver running
	select {}
}

// Sender sends a JSON message securely
func sender(ctx context.Context, targetAddr string) {
	host, err := createSecureHost()
	if err != nil {
		log.Fatalf("Failed to create secure host: %v", err)
	}
	defer host.Close()

	// Parse receiver's multiaddress
	addr, err := multiaddr.NewMultiaddr(targetAddr)
	if err != nil {
		log.Fatalf("Invalid target address: %v", err)
	}

	// Extract peer info
	peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		log.Fatalf("Failed to parse peer info: %v", err)
	}

	// Connect to receiver securely
	if err := host.Connect(ctx, *peerInfo); err != nil {
		log.Fatalf("Failed to connect securely: %v", err)
	}
	fmt.Println("🔐 Securely connected to receiver!")

	// Open a secure QUIC stream
	stream, err := host.NewStream(ctx, peerInfo.ID, "/json/1.0.0")
	if err != nil {
		log.Fatalf("Failed to open secure stream: %v", err)
	}
	defer stream.Close()

	// Create a message
	msg := Message{
		Type:    "Greeting",
		Content: "Hello, securely over libp2p QUIC!",
	}

	// Serialize to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Send JSON data securely
	_, err = stream.Write(data)
	if err != nil {
		log.Fatalf("Failed to send secure data: %v", err)
	}

	fmt.Println("✅ Secure Message Sent Successfully!")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <receiver|sender> [receiver-multiaddr]")
		return
	}

	ctx := context.Background()

	if os.Args[1] == "receiver" {
		receiver(ctx)
	} else if os.Args[1] == "sender" {
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go sender <receiver-multiaddr>")
			return
		}
		targetAddr := os.Args[2]
		time.Sleep(2 * time.Second) // Wait for the receiver to start
		sender(ctx, targetAddr)
	} else {
		fmt.Println("Invalid argument. Use 'receiver' or 'sender'.")
	}
}
