package network

import (
	"log/slog"
	"time"

	"github.com/SkylerRankin/network_monitor/internal/constants"
	. "github.com/SkylerRankin/network_monitor/internal/types"
	"github.com/pkg/errors"
	probing "github.com/prometheus-community/pro-bing"
)

var nextPingConfigIndex = 0

func RunPing(log *slog.Logger) (PingResult, error) {
	c := constants.PingConfigs[nextPingConfigIndex]
	nextPingConfigIndex += 1
	if nextPingConfigIndex == len(constants.PingConfigs) {
		nextPingConfigIndex = 0
	}

	startTime := time.Now().UnixMilli()

	pinger, err := probing.NewPinger(c.URL)
	if err != nil {
		return PingResult{}, errors.Wrap(err, "failed to create pinger")
	}

	pinger.SetPrivileged(true)
	pinger.Count = c.Count
	err = pinger.Run()
	if err != nil {
		return PingResult{}, errors.Wrap(err, "failed to run pinger")
	}

	stats := pinger.Statistics()
	return PingResult{
		Successful: stats.PacketLoss < 1,
		Host:       c.Name,
		HostName:   c.URL,
		Timestamp:  startTime,
		PacketLoss: stats.PacketLoss,
		RTTMS:      int(stats.AvgRtt.Milliseconds()),
	}, nil
}
