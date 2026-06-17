package websocket

import (
	"encoding/json"
	"sync"
)

type broadcast struct {
	groupID string
	payload []byte
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[string]map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan broadcast
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan broadcast, 128),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.groupID] == nil {
				h.clients[client.groupID] = make(map[*Client]struct{})
			}
			h.clients[client.groupID][client] = struct{}{}
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if group := h.clients[client.groupID]; group != nil {
				if _, ok := group[client]; ok {
					delete(group, client)
					close(client.send)
				}
				if len(group) == 0 {
					delete(h.clients, client.groupID)
				}
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients[message.groupID] {
				select {
				case client.send <- message.payload:
				default:
					go func(c *Client) { h.unregister <- c }(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Broadcast(groupID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.broadcast <- broadcast{groupID: groupID, payload: payload}
}
