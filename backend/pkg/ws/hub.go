package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second
)

type Client struct {
	Conn       *websocket.Conn
	Send       chan []byte
	BotSpaceID string
	hub        *Hub
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]struct{}
	log   logrus.Ext1FieldLogger
}

func NewHub(log logrus.Ext1FieldLogger) *Hub {
	return &Hub{
		rooms: make(map[string]map[*Client]struct{}),
		log:   log,
	}
}

func (h *Hub) NewClient(conn *websocket.Conn, botSpaceID string) *Client {
	return &Client{
		Conn:       conn,
		Send:       make(chan []byte, 256),
		BotSpaceID: botSpaceID,
		hub:        h,
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[client.BotSpaceID] == nil {
		h.rooms[client.BotSpaceID] = make(map[*Client]struct{})
	}
	h.rooms[client.BotSpaceID][client] = struct{}{}
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.rooms[client.BotSpaceID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			close(client.Send)
			if len(clients) == 0 {
				delete(h.rooms, client.BotSpaceID)
			}
		}
	}
}

func (h *Hub) Broadcast(botSpaceID string, data []byte) {
	h.mu.RLock()
	roomClients, ok := h.rooms[botSpaceID]
	if !ok || len(roomClients) == 0 {
		h.mu.RUnlock()
		return
	}
	clients := make([]*Client, 0, len(roomClients))
	for client := range roomClients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.Send <- data:
		default:
			go h.Unregister(client)
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer c.Conn.Close()

	for {
		select {
		case msg, ok := <-c.Send:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		if _, _, err := c.Conn.ReadMessage(); err != nil {
			break
		}
	}
}
