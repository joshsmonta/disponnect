package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type Server struct {
	mu          sync.Mutex
	connections map[*websocket.Conn]bool
}

func NewServer() *Server {
	return &Server{
		connections: make(map[*websocket.Conn]bool),
	}
}

func (s *Server) handleWS(ws *websocket.Conn) {
	fmt.Println("New Incomming connection from client:", ws.RemoteAddr())
	s.mu.Lock()
	s.connections[ws] = true
	s.mu.Unlock()
	s.readLoop(ws)
}

func (s *Server) handleSubscription(ws *websocket.Conn) {
	for {
		payload := fmt.Sprintf("subscription data -> %d\n", time.Now().UnixNano())
		ws.Write([]byte(payload))
		time.Sleep(time.Second * 2)
	}
}

func (s *Server) readLoop(ws *websocket.Conn) {
	buffer := make([]byte, 1024)
	for {
		n, err := ws.Read(buffer)
		if err != nil {
			// break if the other connection is already dead
			if err == io.EOF {
				break
			}
			fmt.Println("Connection Read Error:", err)
			continue
		}
		msg := buffer[:n]
		s.broadcast(msg)
	}
}

func (s *Server) broadcast(msg []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ws := range s.connections {
		go func(ws *websocket.Conn) {
			if _, err := ws.Write(msg); err != nil {
				fmt.Println("Broadcast error:", err)
				s.removeConnection(ws)
			}
		}(ws)
	}
}

func (s *Server) removeConnection(ws *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.connections[ws]; ok {
		fmt.Println("Closing connection:", ws.RemoteAddr())
		ws.Close()
		delete(s.connections, ws)
	}
}

func main() {
	server := NewServer()
	http.Handle("/ws", websocket.Handler(server.handleWS))
	http.Handle("/subscription", websocket.Handler(server.handleSubscription))
	fmt.Println("WebSocket server is running on :8001...")
	if err := http.ListenAndServe(":8001", nil); err != nil {
		fmt.Println("Server error:", err)
	}
}
