package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type ChatRoom struct {
	mu          sync.Mutex
	connections map[*websocket.Conn]bool
	roomId      string
}

type Server struct {
	mu    sync.Mutex
	rooms map[string]*ChatRoom
}

func NewServer() *Server {
	return &Server{
		rooms: make(map[string]*ChatRoom),
	}
}

func NewChatRoom(roomId string) *ChatRoom {
	return &ChatRoom{
		connections: make(map[*websocket.Conn]bool),
		roomId:      roomId,
	}
}

func (room *ChatRoom) handleWS(ws *websocket.Conn) {
	fmt.Println("New Incomming connection from client:", ws.RemoteAddr())
	room.mu.Lock()
	room.connections[ws] = true
	room.mu.Unlock()
	room.readLoop(ws)
}

func (room *ChatRoom) readLoop(ws *websocket.Conn) {
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
		room.broadcast(msg)
	}
	room.removeConnection(ws)
}

func (room *ChatRoom) broadcast(msg []byte) {
	room.mu.Lock()
	defer room.mu.Unlock()
	for ws := range room.connections {
		go func(ws *websocket.Conn) {
			if _, err := ws.Write(msg); err != nil {
				fmt.Println("Broadcast error:", err)
				room.removeConnection(ws)
			}
		}(ws)
	}
}

func (room *ChatRoom) removeConnection(ws *websocket.Conn) {
	room.mu.Lock()
	defer room.mu.Unlock()
	if _, ok := room.connections[ws]; ok {
		fmt.Println("Closing connection in chatroom:", room.roomId, "from client:", ws.RemoteAddr())
		ws.Close()
		delete(room.connections, ws)
	}
}

func (room *ChatRoom) roomSubscription(ws *websocket.Conn) {
	for {
		payload := fmt.Sprintf("subscription data -> %d\n", time.Now().UnixNano())
		if _, err := ws.Write([]byte(payload)); err != nil {
			fmt.Println("Subscription write error:", err)
			break
		}
		time.Sleep(time.Second * 2)
	}
}

func (s *Server) getOrCreateRoom(roomId string) *ChatRoom {
	s.mu.Lock()
	defer s.mu.Unlock()
	if room, exists := s.rooms[roomId]; exists {
		return room
	}
	// Generate new Id room
	newRoom := NewChatRoom(roomId)
	s.rooms[roomId] = newRoom
	return newRoom
}

func (s *Server) chatRoomHandler(ws *websocket.Conn) {
	roomId := ws.Request().URL.Query().Get("roomId")
	if roomId == "" {
		ws.Write([]byte("Room ID is required"))
		ws.Close()
		return
	}
	room := s.getOrCreateRoom(roomId)
	room.handleWS(ws)
	fmt.Println(s.rooms)
}

func (s *Server) handleSubscription(ws *websocket.Conn) {
	roomName := ws.Request().URL.Query().Get("roomId")
	room := s.getOrCreateRoom(roomName)
	room.roomSubscription(ws)
}

func main() {
	server := NewServer()
	http.Handle("/ws", websocket.Handler(server.chatRoomHandler))
	http.Handle("/subscription", websocket.Handler(server.handleSubscription))

	fmt.Println("WebSocket server is running on :8001...")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		fmt.Println("Server error:", err)
	}
}
