package types

import "github.com/SkylerRankin/network_monitor/internal/optional"

type NetworkInfo struct {
	PingSuccessful       bool
	PingHost             string
	PingHostName         string
	Timestamp            int64
	PacketLoss           float32
	RTTMS                int
	SpeedTestDescription optional.Opt[string]
	DownloadSpeed        optional.Opt[float64]
	UploadSpeed          optional.Opt[float64]
}

type SpeedResult struct {
	Successful       bool
	Description      string
	Download, Upload float64
}

type PingResult struct {
	Successful bool
	Host       string
	HostName   string
	Timestamp  int64
	PacketLoss float64
	RTTMS      int
}

type NetworkInfoBatch struct {
	Timestamps     []int64                 `json:"timestamps"`
	PingValues     []bool                  `json:"ping"`
	UploadValues   []optional.Opt[float64] `json:"upload"`
	DownloadValues []optional.Opt[float64] `json:"download"`
}

type IndexTemplateData struct {
	Commit string
}

type PingConfig struct {
	URL   string
	Name  string
	Count int
}
