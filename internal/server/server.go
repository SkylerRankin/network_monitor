package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"text/template"
	"time"

	"github.com/SkylerRankin/network_monitor/internal/constants"
	"github.com/SkylerRankin/network_monitor/internal/database"
	"github.com/SkylerRankin/network_monitor/internal/types"
	websocket_client "github.com/SkylerRankin/network_monitor/internal/websocket"
)

const (
	port = ":8080"
)

type Server interface {
	Listen()
	Shutdown() error
}

type server struct {
	ctx             context.Context
	log             *slog.Logger
	assetsPath      string
	server          *http.Server
	database        database.Database
	websocketClient websocket_client.WebsocketClient
}

func NewServer(ctx context.Context, log *slog.Logger, assetsPath string, database database.Database, websocketClient websocket_client.WebsocketClient) Server {
	return &server{
		ctx:             ctx,
		log:             log,
		assetsPath:      assetsPath,
		database:        database,
		websocketClient: websocketClient,
	}
}

func (s *server) Listen() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(s.assetsPath))))
	http.HandleFunc("/", s.handleRoot)
	http.HandleFunc("/batch", s.handleBatch)
	http.HandleFunc("/ws", s.handleWebsocket)

	s.server = &http.Server{Addr: port}

	s.log.Info("http server listening", "port", port)
	err := s.server.ListenAndServe()
	if err != nil {
		s.log.Error("http server exited with error", "err", err)
	} else {
		s.log.Info("http server exited")
	}
}

func (s *server) Shutdown() error {
	return s.server.Shutdown(s.ctx)
}

func (s *server) handleRoot(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles(filepath.Join(s.assetsPath, "templates", "index.html"))
	if err != nil {
		// TODO: format/wrap error strings instead of using raw error message
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := types.IndexTemplateData{Commit: constants.Commit}
	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) handleBatch(w http.ResponseWriter, r *http.Request) {
	// Get 1 day worth of data from database
	startTime := time.Now().AddDate(0, 0, -1).UnixMilli()
	batch, err := s.database.GetNetworkInfoBatch(s.ctx, int(startTime))

	if err != nil {
		s.log.Error("failed to get batch from database", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(batch)
	if err != nil {
		s.log.Error("failed to marshal batch", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	// TODO: Set content length header
	w.Header().Set("Content-Length", "")
	_, err = w.Write(jsonData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	err := s.websocketClient.HandleConnection(w, r)
	if err != nil {
		s.log.Error("failed to handle websocket connection", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
