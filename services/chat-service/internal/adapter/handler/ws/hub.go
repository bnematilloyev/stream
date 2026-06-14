package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]map[*Client]struct{})}
}

func (h *Hub) Join(streamID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[streamID] == nil {
		h.rooms[streamID] = make(map[*Client]struct{})
	}
	h.rooms[streamID][client] = struct{}{}
}

func (h *Hub) Leave(streamID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room := h.rooms[streamID]
	if room == nil {
		return
	}
	delete(room, client)
	if len(room) == 0 {
		delete(h.rooms, streamID)
	}
}

func (h *Hub) Broadcast(streamID string, payload []byte) {
	h.mu.RLock()
	room := h.rooms[streamID]
	clients := make([]*Client, 0, len(room))
	for c := range room {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		c.Send(payload)
	}
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
	once sync.Once
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{conn: conn, send: make(chan []byte, 64)}
}

func (c *Client) Send(payload []byte) {
	select {
	case c.send <- payload:
	default:
	}
}

func (c *Client) ReadPump(onMessage func([]byte)) {
	defer c.Close()
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		onMessage(data)
	}
}

func (c *Client) WritePump() {
	for payload := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			break
		}
	}
	c.Close()
}

func (c *Client) Close() {
	c.once.Do(func() {
		close(c.send)
		_ = c.conn.Close()
	})
}
