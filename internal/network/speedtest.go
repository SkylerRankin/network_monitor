package network

import (
	"context"

	"github.com/SkylerRankin/network_monitor/internal/constants"
	. "github.com/SkylerRankin/network_monitor/internal/types"
	"github.com/pkg/errors"
	"github.com/showwin/speedtest-go/speedtest"
)

func RunSpeedtest(ctx context.Context) (SpeedResult, error) {
	var speedtestClient = speedtest.New()
	serverList, err := speedtestClient.FetchServers()
	if err != nil {
		return SpeedResult{}, errors.Wrap(err, "failed to fetch servers")
	}

	targets, err := serverList.FindServer([]int{})
	if err != nil {
		return SpeedResult{}, errors.Wrap(err, "failed to find server")
	}

	if len(targets) == 0 {
		return SpeedResult{}, errors.New("no speed test servers reachable")
	}

	server := targets[0]
	err = server.DownloadTest()
	if err != nil {
		return SpeedResult{}, errors.Wrap(err, "failed to run download test")
	}

	err = server.UploadTest()
	if err != nil {
		return SpeedResult{}, errors.Wrap(err, "failed to run upload test")
	}

	return SpeedResult{
		Successful:  true,
		Description: server.String(),
		Download:    float64(server.DLSpeed) * constants.BytesToMbps,
		Upload:      float64(server.ULSpeed) * constants.BytesToMbps,
	}, nil
}
