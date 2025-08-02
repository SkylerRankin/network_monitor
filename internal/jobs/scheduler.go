package jobs

import (
	"context"
	"log/slog"

	"github.com/go-co-op/gocron/v2"
	"github.com/pkg/errors"
)

type Scheduler interface {
	Start()
	Shutdown() error
}

type SchedulerJob interface {
	Run() error
}

var _ Scheduler = &scheduler{}

type scheduler struct {
	gocronScheduler gocron.Scheduler
}

func NewScheduler(ctx context.Context, log *slog.Logger, networkJob SchedulerJob) (Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gocron scheduler")
	}

	_, err = s.NewJob(gocron.DurationJob(NetworkJobInterval), gocron.NewTask(func() {
		if err := networkJob.Run(); err != nil {
			log.Info("failed to run network job", "err", err)
		}
	}))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create job")
	}

	return &scheduler{
		gocronScheduler: s,
	}, nil
}

func (s scheduler) Start() {
	s.gocronScheduler.Start()
}

func (s scheduler) Shutdown() error {
	if err := s.gocronScheduler.Shutdown(); err != nil {
		return errors.Wrap(err, "failed to shutdown gocron scheduler")
	}
	return nil
}
