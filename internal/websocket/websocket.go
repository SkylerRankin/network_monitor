package websocket_client

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

const (
	bufferSize  = 10
	readTimeout = 10 * time.Second
)

type WebsocketClient interface {
	Listen(context.Context)
	HandleConnection(w http.ResponseWriter, r *http.Request) error
	Broadcast([]byte)
	Shutdown() error
}

var _ WebsocketClient = &websocketClient{}

type websocketClient struct {
	log         *slog.Logger
	upgrader    websocket.Upgrader
	connections map[*websocket.Conn]bool
	broadcast   chan []byte
	register    chan *websocket.Conn
	unregister  chan *websocket.Conn
}

func NewWebsocketClient(log *slog.Logger) WebsocketClient {
	return &websocketClient{
		log: log,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		connections: make(map[*websocket.Conn]bool),
		broadcast:   make(chan []byte, bufferSize),
		register:    make(chan *websocket.Conn, bufferSize),
		unregister:  make(chan *websocket.Conn, bufferSize),
	}
}

func (c websocketClient) Listen(ctx context.Context) {
	exit := false
	for !exit {
		select {
		case <-ctx.Done():
			exit = true
		case conn := <-c.register:
			if _, ok := c.connections[conn]; ok {
				c.log.Info("not registering already registered connection", "address", conn.RemoteAddr().String())
			} else {
				c.log.Info("registered connection", "address", conn.RemoteAddr().String())
				c.connections[conn] = true
			}
		case conn := <-c.unregister:
			if _, ok := c.connections[conn]; ok {
				c.log.Info("unregistered connection", "address", conn.RemoteAddr().String())
				delete(c.connections, conn)
			} else {
				c.log.Info("attempted to unregister unregistered connection", "address", conn.RemoteAddr().String())
			}
		case message := <-c.broadcast:
			brokenConnections := make([]*websocket.Conn, 0)
			for conn := range c.connections {
				conn.SetReadDeadline(time.Now().Add(readTimeout))
				err := conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					c.log.Info("broadcast message failed", "err", err)
					brokenConnections = append(brokenConnections, conn)
				}
			}

			// Unregister connections that are not accepting data
			for _, conn := range brokenConnections {
				c.unregister <- conn
			}
		}
	}
}

func (c websocketClient) HandleConnection(w http.ResponseWriter, r *http.Request) error {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return errors.Wrap(err, "failed to upgrade http connection to websocket")
	}

	c.register <- conn
	return nil
}

func (c websocketClient) Broadcast(bytes []byte) {
	c.broadcast <- bytes
}

func (c websocketClient) Shutdown() error {
	for conn := range c.connections {
		if conn != nil {
			conn.Close()
		}
	}

	return nil
}
