package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/SkylerRankin/network_monitor/internal/constants"
	"github.com/SkylerRankin/network_monitor/internal/database"
	"github.com/SkylerRankin/network_monitor/internal/jobs"
	"github.com/SkylerRankin/network_monitor/internal/server"
	websocket_client "github.com/SkylerRankin/network_monitor/internal/websocket"
	_ "modernc.org/sqlite"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	if len(os.Args) != 2 {
		log.Error("incorrect arguments, expected 2", "args", len(os.Args))
		return
	}

	assetsPath, err := filepath.Abs(os.Args[1])
	if err != nil {
		log.Error("failed to get assets absolute path", "path", os.Args[1], "err", err)
		return
	}

	if _, err := os.Stat(assetsPath); errors.Is(err, os.ErrNotExist) {
		log.Error("assets path does not exist", "path", os.Args[1], "err", err)
		return
	}

	database, err := database.NewDatabase(ctx, assetsPath)
	if err != nil {
		log.Error("failed to create database", "path", assetsPath)
		return
	}

	websocketClient := websocket_client.NewWebsocketClient(log)

	networkInfoJob, err := jobs.NewNetworkInfoJob(ctx, log, database, websocketClient)
	if err != nil {
		log.Error("failed to create network info job", "err", err)
		return
	}

	scheduler, err := jobs.NewScheduler(ctx, log, networkInfoJob)
	if err != nil {
		log.Error("failed to create job scheduler", "err", err)
		return
	}

	server := server.NewServer(ctx, log, assetsPath, database, websocketClient)

	log.Info("starting network monitor", "assets_path", assetsPath, "commit", constants.Commit)

	go websocketClient.Listen(ctx)
	go server.Listen()
	scheduler.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	log.Info("received signal", "signal", sig)

	websocketClient.Shutdown()
	scheduler.Shutdown()
	server.Shutdown()

	log.Info("exiting network monitor")
}
