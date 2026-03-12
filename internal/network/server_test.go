package network_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/ALXNKO/UltimateSim/internal/network"
)

// Phase 12.1: Network Sockets & Stream Binding Tests
// Verifies that both UDP and TCP endpoints can be bound and gracefully stopped.

func TestServer_StartAndStop(t *testing.T) {
	// Start server on free ports
	server := network.NewServer("0", "0")
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Wait briefly to ensure background goroutines start
	time.Sleep(50 * time.Millisecond)

	server.Stop()
}

func TestServer_Connections(t *testing.T) {
	server := network.NewServer("0", "0")
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Wait briefly for server to bind
	time.Sleep(50 * time.Millisecond)

	// Test TCP
	tcpAddr := fmt.Sprintf("127.0.0.1:%d", server.GetTCPPort())
	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		t.Fatalf("Failed to connect to TCP server: %v", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte("TCP Test Payload"))
	if err != nil {
		t.Fatalf("Failed to write to TCP server: %v", err)
	}

	// Test UDP
	udpAddr := fmt.Sprintf("127.0.0.1:%d", server.GetUDPPort())
	udpConn, err := net.Dial("udp", udpAddr)
	if err != nil {
		t.Fatalf("Failed to connect to UDP server: %v", err)
	}
	defer udpConn.Close()

	_, err = udpConn.Write([]byte("UDP Test Payload"))
	if err != nil {
		t.Fatalf("Failed to write to UDP server: %v", err)
	}

	// Allow server to process messages
	time.Sleep(50 * time.Millisecond)
}
