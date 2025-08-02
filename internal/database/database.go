package database

import (
	"context"
	"database/sql"
	"path/filepath"

	"github.com/SkylerRankin/network_monitor/internal/optional"
	"github.com/SkylerRankin/network_monitor/internal/types"
	"github.com/pkg/errors"
)

const (
	databaseFilename = "netmon.db"
)

type Database interface {
	InsertNetworkInfo(context.Context, *types.NetworkInfo) error
	GetNetworkInfoBatch(context.Context, int) (*types.NetworkInfoBatch, error)
}

var _ Database = &database{}

type database struct {
	db *sql.DB
}

func NewDatabase(ctx context.Context, path string) (Database, error) {
	db, err := sql.Open("sqlite", filepath.Join(path, databaseFilename))
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
	_, err = db.ExecContext(ctx, createText)
	if err != nil {
		return nil, err
	}

	return database{
		db: db,
	}, nil
}

func (d database) InsertNetworkInfo(ctx context.Context, info *types.NetworkInfo) error {
	pingSuccessful := 0
	if info.PingSuccessful {
		pingSuccessful = 1
	}

	// downloadSpeed := sql.NullFloat64{
	// 	Float64: info.DownloadSpeed.Else(0),
	// 	Valid:   info.DownloadSpeed.Has(),
	// }

	// uploadSpeed := sql.NullFloat64{
	// 	Float64: info.UploadSpeed.Else(0),
	// 	Valid:   info.DownloadSpeed.Has(),
	// }

	// TODO: why use a transaction, no point
	insertText :=
		`INSERT INTO network
		(timestamp, pingHost, pingHostName, pingSuccessful, packetLoss, rttMS, downloadSpeed, uploadSpeed) VALUES (?, ?, ?, ?, ?, ?, ?, ?);`

	// _, err := d.db.Exec(insertText, info.Timestamp, info.PingHost, info.PingHostName, pingSuccessful, info.PacketLoss, info.RTTMS, downloadSpeed, uploadSpeed)
	_, err := d.db.Exec(insertText, info.Timestamp, info.PingHost, info.PingHostName, pingSuccessful, info.PacketLoss, info.RTTMS, &info.DownloadSpeed, &info.UploadSpeed)

	// tx, err := d.db.BeginTx(ctx, nil)
	// if err != nil {
	// 	return errors.Wrap(err, "failed to begin transaction")
	// }

	// _, err = tx.Exec(`
	// 	INSERT INTO network
	// 	(timestamp, pingHost, pingHostName, pingSuccessful, packetLoss, rttMS, downloadSpeed, uploadSpeed) VALUES (?, ?, ?, ?, ?, ?, ?, ?);
	// `, info.Timestamp, info.PingHost, info.PingHostName, pingSuccessful, info.PacketLoss, info.RTTMS, downloadSpeed, uploadSpeed)
	if err != nil {
		return errors.Wrap(err, "failed to execute insert")
	}

	// err = tx.Commit()
	// if err != nil {
	// 	return errors.Wrap(err, "failed to commit transaction")
	// }

	return nil
}

func (d database) GetNetworkInfoBatch(ctx context.Context, startTime int) (*types.NetworkInfoBatch, error) {
	rows, err := d.db.QueryContext(ctx,
		`
			SELECT * FROM network
			WHERE timestamp > ?
			ORDER BY timestamp ASC
		`, startTime)

	if err != nil {
		return nil, errors.Wrap(err, "failed to query networks table")
	}
	defer rows.Close()

	batch := types.NetworkInfoBatch{
		Timestamps:     make([]int64, 0),
		PingValues:     make([]bool, 0),
		UploadValues:   make([]optional.Opt[float64], 0),
		DownloadValues: make([]optional.Opt[float64], 0),
	}

	for rows.Next() {
		var info types.NetworkInfo

		err := rows.Scan(&info.Timestamp, &info.PingHost, &info.PingHostName, &info.PingSuccessful, &info.PacketLoss, &info.RTTMS, &info.DownloadSpeed, &info.UploadSpeed)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row for network info values")
		}

		batch.Timestamps = append(batch.Timestamps, info.Timestamp)
		batch.PingValues = append(batch.PingValues, info.PingSuccessful)
		batch.DownloadValues = append(batch.DownloadValues, info.DownloadSpeed)
		batch.UploadValues = append(batch.UploadValues, info.UploadSpeed)
	}

	return &batch, nil
}
