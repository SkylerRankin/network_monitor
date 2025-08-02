package constants

import (
	"time"

	"github.com/SkylerRankin/network_monitor/internal/types"
)

const (
	PingInterval      = 30 * time.Second
	SpeedtestInterval = 15 * time.Minute
	BytesToMbps       = (1.0 / 125000.0)
)

var PingConfigs = []types.PingConfig{
	{URL: "8.8.8.8", Name: "Google", Count: 3},
	{URL: "1.1.1.1", Name: "Cloudflare", Count: 3},
	{URL: "208.67.222.222", Name: "OpenDNS", Count: 3},
}
