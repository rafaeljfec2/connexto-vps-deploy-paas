package cleanup

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/paasdeploy/shared/pkg/docker"
)

const defaultCleanupInterval = 24 * time.Hour

type Scheduler struct {
	docker   *docker.Client
	logger   *slog.Logger
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewScheduler(dockerClient *docker.Client, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		docker:   dockerClient,
		logger:   logger.With("component", "cleanup_scheduler"),
		interval: defaultCleanupInterval,
		stopCh:   make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("Starting cleanup scheduler", "interval", s.interval)

	s.wg.Add(1)
	go s.run(ctx)
}

func (s *Scheduler) Stop() {
	s.logger.Info("Stopping cleanup scheduler")
	close(s.stopCh)
	s.wg.Wait()
	s.logger.Info("Cleanup scheduler stopped")
}

func (s *Scheduler) RunOnce(ctx context.Context) {
	s.performCleanup(ctx)
}

func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.performCleanup(ctx)
		}
	}
}

func (s *Scheduler) performCleanup(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	s.logger.Info("Starting Docker cleanup")

	containerResult, err := s.docker.PruneContainers(ctx)
	if err != nil {
		s.logger.Error("Failed to prune containers", "error", err)
	} else {
		s.logger.Info("Pruned containers",
			"containersRemoved", containerResult.ContainersDeleted,
			"spaceReclaimed", containerResult.SpaceReclaimed,
		)
	}

	imageResult, err := s.docker.PruneImages(ctx)
	if err != nil {
		s.logger.Error("Failed to prune images", "error", err)
	} else {
		s.logger.Info("Pruned images",
			"imagesRemoved", imageResult.ImagesDeleted,
			"spaceReclaimed", imageResult.SpaceReclaimed,
		)
	}

	s.logger.Info("Docker cleanup completed")
}
