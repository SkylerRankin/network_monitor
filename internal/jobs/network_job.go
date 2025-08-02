package jobs

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/SkylerRankin/network_monitor/internal/database"
	"github.com/SkylerRankin/network_monitor/internal/network"
	"github.com/SkylerRankin/network_monitor/internal/optional"
	"github.com/SkylerRankin/network_monitor/internal/types"
	websocket_client "github.com/SkylerRankin/network_monitor/internal/websocket"
	"github.com/pkg/errors"
)

const (
	// Time between each call to the network info job.
	NetworkJobInterval = 5 * time.Second
	// Ratio of all network info jobs to network info jobs that also do the speed test.
	speedInterval = 5
)

type networkInfoJob struct {
	ctx                  context.Context
	log                  *slog.Logger
	speedInterval        int
	currentSpeedInterval int
	intervalMutex        sync.Mutex
	database             database.Database
	websocket            websocket_client.WebsocketClient
}

func NewNetworkInfoJob(ctx context.Context, log *slog.Logger, database database.Database, websocket websocket_client.WebsocketClient) (SchedulerJob, error) {
	return &networkInfoJob{
		ctx:                  ctx,
		log:                  log,
		speedInterval:        speedInterval,
		currentSpeedInterval: speedInterval,
		database:             database,
		websocket:            websocket,
	}, nil
}

func (j *networkInfoJob) Run() error {
	// Multiple running jobs may attempt to update the speed interval. Lock to
	// prevent repeated intervals.
	j.intervalMutex.Lock()
	runSpeedTest := j.currentSpeedInterval == 0
	if j.currentSpeedInterval == 0 {
		j.currentSpeedInterval = j.speedInterval
	} else {
		j.currentSpeedInterval -= 1
	}
	j.intervalMutex.Unlock()

	ping, err := network.RunPing(j.log)
	if err != nil {
		return errors.Wrap(err, "failed to run network ping")
	}

	networkInfo := types.NetworkInfo{
		PingSuccessful:       ping.Successful,
		PingHost:             ping.Host,
		PingHostName:         ping.HostName,
		Timestamp:            time.Now().UnixMilli(),
		PacketLoss:           float32(ping.PacketLoss),
		RTTMS:                ping.RTTMS,
		SpeedTestDescription: optional.Empty[string](),
		DownloadSpeed:        optional.Empty[float64](),
		UploadSpeed:          optional.Empty[float64](),
	}

	if runSpeedTest {
		speedInfo, err := network.RunSpeedtest(j.ctx)
		if err != nil {
			return errors.Wrap(err, "failed to run speed test")
		}

		networkInfo.SpeedTestDescription = optional.New(speedInfo.Description)
		networkInfo.DownloadSpeed = optional.New(speedInfo.Download)
		networkInfo.UploadSpeed = optional.New(speedInfo.Upload)
	}

	err = j.database.InsertNetworkInfo(j.ctx, &networkInfo)
	if err != nil {
		return errors.Wrap(err, "failed to insert network info")
	}

	batch := types.NetworkInfoBatch{
		Timestamps:     []int64{networkInfo.Timestamp},
		PingValues:     []bool{networkInfo.PingSuccessful},
		UploadValues:   []optional.Opt[float64]{networkInfo.UploadSpeed},
		DownloadValues: []optional.Opt[float64]{networkInfo.DownloadSpeed},
	}

	batchJson, err := json.Marshal(batch)
	if err != nil {
		return errors.Wrap(err, "failed to marshal network ")
	}
	j.websocket.Broadcast(batchJson)

	return nil
}
