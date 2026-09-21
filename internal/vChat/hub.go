package vChat

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512 KB (sufficient for SDP offers/ICE candidates)
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all web origins for WebRTC signaling
	},
}

type Client struct {
	ID     string
	Name   string
	RoomID string
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *Hub
}

type Hub struct {
	rooms      map[string]map[string]*Client
	broadcast  chan *SignalingMessage
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[string]*Client),
		broadcast:  make(chan *SignalingMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, exists := h.rooms[client.RoomID]; !exists {
				h.rooms[client.RoomID] = make(map[string]*Client)
			}
			h.rooms[client.RoomID][client.ID] = client

			// Collect existing peers in the room to inform the newcomer
			existingPeers := make([]map[string]string, 0)
			for peerID, peer := range h.rooms[client.RoomID] {
				if peerID != client.ID {
					existingPeers = append(existingPeers, map[string]string{
						"id":   peer.ID,
						"name": peer.Name,
					})
				}
			}
			h.mu.Unlock()

			log.Printf("[vChat Hub] Client %s (%s) joined room %s. Total in room: %d", client.ID, client.Name, client.RoomID, len(existingPeers)+1)

			// 1. Send existing peers list to the newly connected client
			roomInfoMsg := SignalingMessage{
				Type:     MsgTypeRoomInfo,
				RoomID:   client.RoomID,
				SenderID: "server",
				Payload: map[string]interface{}{
					"peers": existingPeers,
					"you": map[string]string{
						"id":   client.ID,
						"name": client.Name,
					},
				},
			}
			if msgBytes, err := json.Marshal(roomInfoMsg); err == nil {
				client.Send <- msgBytes
			}

			// 2. Notify other peers in the room about the new peer
			joinNotification := SignalingMessage{
				Type:       MsgTypeJoined,
				RoomID:     client.RoomID,
				SenderID:   client.ID,
				SenderName: client.Name,
			}
			h.broadcastToRoom(client.RoomID, &joinNotification, client.ID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, exists := h.rooms[client.RoomID]; exists {
				if _, ok := clients[client.ID]; ok {
					delete(clients, client.ID)
					close(client.Send)
					log.Printf("[vChat Hub] Client %s left room %s", client.ID, client.RoomID)

					if len(clients) == 0 {
						delete(h.rooms, client.RoomID)
					}
				}
			}
			h.mu.Unlock()

			// Broadcast peer-left to remaining peers in the room
			leaveNotification := SignalingMessage{
				Type:     MsgTypePeerLeft,
				RoomID:   client.RoomID,
				SenderID: client.ID,
			}
			h.broadcastToRoom(client.RoomID, &leaveNotification, client.ID)

		case message := <-h.broadcast:
			h.handleSignalingMessage(message)
		}
	}
}

func (h *Hub) handleSignalingMessage(msg *SignalingMessage) {
	h.mu.RLock()
	clients, exists := h.rooms[msg.RoomID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[vChat Hub] Error marshaling message: %v", err)
		return
	}

	// Direct 1-to-1 message (e.g. SDP offer/answer/ICE candidate destined for specific peer)
	if msg.TargetID != "" {
		h.mu.RLock()
		targetClient, ok := clients[msg.TargetID]
		h.mu.RUnlock()

		if ok {
			select {
			case targetClient.Send <- msgBytes:
			default:
				log.Printf("[vChat Hub] Client %s buffer full, dropping message", msg.TargetID)
			}
		}
		return
	}

	// Room Broadcast (excluding sender)
	h.broadcastToRoom(msg.RoomID, msg, msg.SenderID)
}

func (h *Hub) broadcastToRoom(roomID string, msg *SignalingMessage, excludeID string) {
	h.mu.RLock()
	clients, exists := h.rooms[roomID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for id, client := range clients {
		if id != excludeID {
			select {
			case client.Send <- msgBytes:
			default:
				log.Printf("[vChat Hub] Client %s send buffer full", id)
			}
		}
	}
}

func (h *Hub) GetParticipantCount(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, exists := h.rooms[roomID]; exists {
		return len(clients)
	}
	return 0
}

func (h *Hub) RegisterClientForTest(client *Client) {
	h.register <- client
}

func (h *Hub) UnregisterClientForTest(client *Client) {
	h.unregister <- client
}

func (h *Hub) BroadcastMessageForTest(msg *SignalingMessage) {
	h.broadcast <- msg
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[vChat Client %s] read error: %v", c.ID, err)
			}
			break
		}

		var sigMsg SignalingMessage
		if err := json.Unmarshal(message, &sigMsg); err != nil {
			log.Printf("[vChat Client %s] invalid signaling JSON: %v", c.ID, err)
			continue
		}

		sigMsg.SenderID = c.ID
		sigMsg.RoomID = c.RoomID
		c.Hub.broadcast <- &sigMsg
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Add queued chat/signaling messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
