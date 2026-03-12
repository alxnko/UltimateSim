package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)

// Phase 12.1: Network Sockets & Stream Binding
// Server struct encapsulates both UDP and TCP endpoints to handle positional data (UDP)
// and SparseHookGraph exchanges (TCP). It establishes Server-Authoritative tick logic over all connected Clients.

type Server struct {
	tcpListener net.Listener
	udpConn     *net.UDPConn

	clients    map[string]net.Conn
	udpClients map[string]*net.UDPAddr
	mu         sync.RWMutex

	tcpPort string
	udpPort string

	// Added for testing
	boundTCPPort int
	boundUDPPort int

	ctx    context.Context
	cancel context.CancelFunc
}

func NewServer(tcpPort string, udpPort string) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		clients:    make(map[string]net.Conn),
		udpClients: make(map[string]*net.UDPAddr),
		tcpPort:    tcpPort,
		udpPort:    udpPort,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start opens both UDP and TCP connections.
func (s *Server) Start() error {
	// TCP Start
	tcpAddr := ":" + s.tcpPort
	listener, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("failed to start TCP server on %s: %v", tcpAddr, err)
	}
	s.tcpListener = listener
	s.boundTCPPort = listener.Addr().(*net.TCPAddr).Port

	// UDP Start
	udpAddrObj, err := net.ResolveUDPAddr("udp", ":"+s.udpPort)
	if err != nil {
		s.tcpListener.Close()
		return fmt.Errorf("failed to resolve UDP address %s: %v", s.udpPort, err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddrObj)
	if err != nil {
		s.tcpListener.Close()
		return fmt.Errorf("failed to start UDP server on %s: %v", s.udpPort, err)
	}
	s.udpConn = udpConn
	s.boundUDPPort = udpConn.LocalAddr().(*net.UDPAddr).Port

	// Accept connections asynchronously
	go s.acceptTCPConnections()
	go s.listenUDP()

	return nil
}

// Stop closes the connections and cancels the context.
func (s *Server) Stop() {
	s.cancel()

	if s.tcpListener != nil {
		s.tcpListener.Close()
	}
	if s.udpConn != nil {
		s.udpConn.Close()
	}

	s.mu.Lock()
	for _, conn := range s.clients {
		conn.Close()
	}
	// Clear the maps to release references
	s.clients = make(map[string]net.Conn)
	s.udpClients = make(map[string]*net.UDPAddr)
	s.mu.Unlock()
}

func (s *Server) acceptTCPConnections() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.tcpListener.Accept()
			if err != nil {
				// Only log if not shutting down
				select {
				case <-s.ctx.Done():
					return
				default:
					log.Printf("TCP Accept error: %v", err)
					continue
				}
			}

			s.mu.Lock()
			s.clients[conn.RemoteAddr().String()] = conn
			s.mu.Unlock()

			// Simple discard loop to keep connection alive and read bytes in this foundational step
			go s.handleTCPConnection(conn)
		}
	}
}

func (s *Server) handleTCPConnection(conn net.Conn) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, conn.RemoteAddr().String())
		s.mu.Unlock()
		conn.Close()
	}()

	buf := make([]byte, 1024)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			_, err := conn.Read(buf)
			if err != nil {
				return // Disconnect
			}
			// In Phase 12.2, we'll parse SparseHookGraph updates here.
		}
	}
}

func (s *Server) listenUDP() {
	buf := make([]byte, 1024)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// ReadFromUDP blocks until data is received or connection is closed
			n, addr, err := s.udpConn.ReadFromUDP(buf)
			if err != nil {
				// Only log if not shutting down
				select {
				case <-s.ctx.Done():
					return
				default:
					log.Printf("UDP Read error: %v", err)
					continue
				}
			}

			s.mu.Lock()
			s.udpClients[addr.String()] = addr
			s.mu.Unlock()

			// Process UDP payload here (In Phase 12.2 positional data parsing)
			_ = n // ignoring for now
		}
	}
}

// BroadcastTCP sends a payload to all connected TCP clients.
func (s *Server) BroadcastTCP(payload []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, conn := range s.clients {
		_, _ = conn.Write(payload)
	}
}

// BroadcastUDP sends a payload to all known UDP clients.
func (s *Server) BroadcastUDP(payload []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, addr := range s.udpClients {
		_, _ = s.udpConn.WriteToUDP(payload, addr)
	}
}

// GetTCPPort returns the bound TCP port.
func (s *Server) GetTCPPort() int {
	return s.boundTCPPort
}

// GetUDPPort returns the bound UDP port.
func (s *Server) GetUDPPort() int {
	return s.boundUDPPort
}
