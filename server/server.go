package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	probing "github.com/prometheus-community/pro-bing"
	"github.com/showwin/speedtest-go/speedtest"
	_ "modernc.org/sqlite"
)

type NetworkInfo struct {
	PingSuccessful       bool
	PingHost             string
	PingHostName         string
	Timestamp            int64
	PacketLoss           float32
	RTTMS                int
	SpeedTestDescription string
	DownloadSpeed        float32
	UploadSpeed          float32
}

type SpeedResult struct {
	Successful       bool
	Description      string
	Download, Upload float32
}

type BatchResponse struct {
	Timestamps     []int64   `json:"timestamps"`
	PingValues     []bool    `json:"ping"`
	UploadValues   []float32 `json:"upload"`
	DownloadValues []float32 `json:"download"`
}

type SingleResponse struct {
	Timestamp     int64   `json:"timestamp"`
	PingValue     bool    `json:"ping"`
	UploadValue   float32 `json:"upload"`
	DownloadValue float32 `json:"download"`
}

type IndexTemplateData struct {
	Commit string
}

const pingInterval = 30 * time.Second
const speedtestInterval = 15 * time.Minute

var PingConfig = []struct {
	URL   string
	Name  string
	Count int
}{
	{"8.8.8.8", "Google", 1},
	{"1.1.1.1", "Cloudflare", 1},
	{"208.67.222.222", "OpenDNS", 1},
}

func runPeriodicPing(db *sql.DB, ctx context.Context, wg *sync.WaitGroup, speedChannel chan SpeedResult, clientPool *ClientPool) {
	log.Println("Ping routine starting")
	first_iteration := true

	configIndex := 0

	for {
		timer_wait := pingInterval
		if first_iteration {
			first_iteration = false
			timer_wait = 0
		}
		timer := time.After(timer_wait)
		select {
		case <-timer:
			c := PingConfig[configIndex]
			configIndex = (configIndex + 1) % len(PingConfig)

			startTime := time.Now().UnixMilli()
			pinger, err := probing.NewPinger(c.URL)
			if err != nil {
				log.Println(err)
			}
			pinger.SetPrivileged(true)
			pinger.Count = c.Count
			err = pinger.Run()
			if err != nil {
				log.Println(err)
			}
			stats := pinger.Statistics()

			result := NetworkInfo{
				PingSuccessful:       true,
				PingHost:             c.Name,
				PingHostName:         c.URL,
				Timestamp:            startTime,
				PacketLoss:           float32(stats.PacketLoss),
				RTTMS:                int(stats.AvgRtt.Milliseconds()),
				SpeedTestDescription: "",
				DownloadSpeed:        -1,
				UploadSpeed:          -1,
			}

			select {
			case speedResult := <-speedChannel:
				if speedResult.Successful {
					result.SpeedTestDescription = speedResult.Description
					result.DownloadSpeed = speedResult.Download
					result.UploadSpeed = speedResult.Upload
				} else {
					log.Println("Speed result not successful")
				}
			default:
			}

			clientPool.sendNetworkInfo(&result)
			saveNetworkInfoToDatabase(db, &result)
		case <-ctx.Done():
			log.Println("Ping routine stopping")
			wg.Done()
			return
		}
	}
}

func runPeriodicSpeedtest(ctx context.Context, wg *sync.WaitGroup, speedChannel chan SpeedResult) {
	log.Println("Speedtest routine starting")

	const bytesToMbps = (1.0 / 125000.0)

	var speedtestClient = speedtest.New()
	serverList, _ := speedtestClient.FetchServers()
	targets, _ := serverList.FindServer([]int{})
	server := targets[0]
	log.Printf("Speedtest using server %s\n", server)
	first_iteration := true

	for {
		timer_wait := speedtestInterval
		if first_iteration {
			first_iteration = false
			timer_wait = 0
		}

		timer := time.After(timer_wait)
		select {
		case <-timer:
			server.DownloadTest()
			server.UploadTest()
			result := SpeedResult{
				Successful:  true,
				Description: server.String(),
				Download:    float32(server.DLSpeed) * bytesToMbps,
				Upload:      float32(server.ULSpeed) * bytesToMbps,
			}
			speedChannel <- result
			server.Context.Reset()
		case <-ctx.Done():
			log.Println("Speedtest routine stopping")
			wg.Done()
			return
		}
	}
}

func initDatabase(assetsPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", filepath.Join(assetsPath, "ping_graph.db"))
	if err != nil {
		return nil, err
	}

	createText :=
		`CREATE TABLE IF NOT EXISTS network (
			timestamp INTEGER PRIMARY KEY,
			pingHost TEXT NOT NULL,
			pingHostName TEXT NOT NULL,
			pingSuccessful INTEGER NOT NULL,
			packetLoss REAL,
			rttMS INTEGER,
			downloadSpeed REAL,
			uploadSpeed REAL
		)`

	_, err = db.ExecContext(context.Background(), createText)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func saveNetworkInfoToDatabase(db *sql.DB, info *NetworkInfo) {
	pingSuccessful := 0
	if info.PingSuccessful {
		pingSuccessful = 1
	}

	downloadSpeed := sql.NullFloat64{
		Float64: float64(info.DownloadSpeed),
		Valid:   info.DownloadSpeed >= 0,
	}

	uploadSpeed := sql.NullFloat64{
		Float64: float64(info.UploadSpeed),
		Valid:   info.DownloadSpeed >= 0,
	}

	tx, _ := db.Begin()
	tx.Exec(`
		INSERT INTO network
		(timestamp, pingHost, pingHostName, pingSuccessful, packetLoss, rttMS, downloadSpeed, uploadSpeed) VALUES (?, ?, ?, ?, ?, ?, ?, ?);
	`, info.Timestamp, info.PingHost, info.PingHostName, pingSuccessful, info.PacketLoss, info.RTTMS, downloadSpeed, uploadSpeed)
	tx.Commit()
}

func main() {
	if len(os.Args) != 2 {
		log.Println("Usage:")
		log.Println("\tping_graph /path/to/static")
		return
	}

	assetsPath, _ := filepath.Abs(os.Args[1])
	if _, err := os.Stat(assetsPath); errors.Is(err, os.ErrNotExist) {
		log.Printf("Provided static assets path doesn't exist: %s\n", assetsPath)
	}

	log.Printf("ping_graph (%s) (%s)\n", assetsPath, Commit)

	port := ":8080"
	server := &http.Server{Addr: port}

	db, err := initDatabase(assetsPath)
	if err != nil {
		log.Printf("Failed to initialize database: %s\n", err)
		return
	}
	defer db.Close()

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(3)
	routineContext, cancelRoutine := context.WithCancel(context.Background())
	clientPool := makePool()
	speedChannel := make(chan SpeedResult, 5)

	go runPeriodicPing(db, routineContext, &waitGroup, speedChannel, clientPool)
	go runPeriodicSpeedtest(routineContext, &waitGroup, speedChannel)
	go clientPool.start(routineContext, &waitGroup)

	// /static
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(assetsPath, "static")))))

	// /
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t, err := template.ParseFiles(filepath.Join(assetsPath, "templates", "index.html"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := IndexTemplateData{Commit: Commit}
		err = t.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// /batch
	http.HandleFunc("/batch", func(w http.ResponseWriter, r *http.Request) {
		const msPerDay = 1 * 24 * 60 * 60 * 1000
		startTime := time.Now().UnixMilli() - msPerDay

		rows, err := db.Query(`
			SELECT * FROM network
			WHERE timestamp > ?
			ORDER BY timestamp ASC
		`, startTime)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var result BatchResponse
		for rows.Next() {
			var info NetworkInfo
			rows.Scan(&info.Timestamp, &info.PingHost, &info.PingHostName, &info.PingSuccessful, &info.PacketLoss, &info.RTTMS, &info.DownloadSpeed, &info.UploadSpeed)
			result.Timestamps = append(result.Timestamps, info.Timestamp)
			result.PingValues = append(result.PingValues, info.PingSuccessful)
			result.DownloadValues = append(result.DownloadValues, info.DownloadSpeed)
			result.UploadValues = append(result.UploadValues, info.UploadSpeed)
		}

		jsonData, _ := json.Marshal(result)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	})

	// /ws
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		client := &Client{
			pool: clientPool,
			conn: conn,
		}
		clientPool.register <- client
	})

	go func() {
		log.Printf("Server listening on port %s\n", server.Addr)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Failure during server listening: %v\n", err)
		}
		log.Printf("Server closed to connections\n")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	log.Printf("Received signal %s\n", sig)

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("Server shutdown error: %v\n", err)
	}

	cancelRoutine()
	waitGroup.Wait()

	log.Printf("App exiting normally\n")
}
