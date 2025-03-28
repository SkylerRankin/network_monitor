package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	pool *ClientPool
	conn *websocket.Conn
}

type ClientPool struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func makePool() *ClientPool {
	return &ClientPool{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (c *ClientPool) start(ctx context.Context, wg *sync.WaitGroup) {
	for {
		select {
		case client := <-c.register:
			log.Printf("Client registered %s\n", client.conn.RemoteAddr().String())
			c.clients[client] = true
		case client := <-c.unregister:
			log.Printf("Client unregistered %s\n", client.conn.RemoteAddr().String())
			delete(c.clients, client)
		case message := <-c.broadcast:
			var clientsToDelete []*Client
			for client := range c.clients {
				// Give client 10 seconds to read message from web socket, otherwise assume tab was closed.
				client.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
				err := client.conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					clientsToDelete = append(clientsToDelete, client)
				}
			}

			for _, client := range clientsToDelete {
				log.Printf("Connection lost to client %s, unregistering\n", client.conn.RemoteAddr().String())
				delete(c.clients, client)
			}

		case <-ctx.Done():
			log.Println("Client pool stopping")
			wg.Done()
			return
		}
	}
}

func (c *ClientPool) sendNetworkInfo(info *NetworkInfo) {
	result := SingleResponse{
		Timestamp:     info.Timestamp,
		PingValue:     info.PingSuccessful,
		UploadValue:   info.UploadSpeed,
		DownloadValue: info.DownloadSpeed,
	}
	bytes, _ := json.Marshal(result)
	c.broadcast <- bytes
}
