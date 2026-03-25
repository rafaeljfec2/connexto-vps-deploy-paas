package docker

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const pruneReclaimedPrefix = "Total reclaimed space"

type PruneResult struct {
	ImagesDeleted     int
	ContainersDeleted int
	VolumesDeleted    int
	SpaceReclaimed    int64
}

func parsePruneOutput(stdout string, countPrefix string) (count int, spaceReclaimed int64) {
	for _, line := range strings.Split(stdout, "\n") {
		if strings.Contains(line, pruneReclaimedPrefix) {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				spaceReclaimed = ParseImageSize(strings.TrimSpace(parts[1]))
			}
			continue
		}
		if countPrefix != "" && strings.HasPrefix(line, countPrefix) {
			count++
		}
	}
	return count, spaceReclaimed
}

func (d *Client) PruneUnusedImages(ctx context.Context) error {
	d.logger.Info("Pruning unused Docker images")

	_, err := d.executor.RunQuietWithTimeout(ctx, 5*time.Minute, "docker", "image", "prune", "-f")
	if err != nil {
		d.logger.Debug("Failed to prune images", "error", err)
		return nil
	}

	return nil
}

func (d *Client) PruneImages(ctx context.Context) (*PruneResult, error) {
	d.logger.Info("Pruning all unused Docker images")

	result, err := d.executor.RunWithTimeout(ctx, 5*time.Minute, "docker", "image", "prune", "-a", "-f")
	if err != nil {
		return nil, fmt.Errorf("failed to prune images: %w", err)
	}

	count, space := parsePruneOutput(result.Stdout, "Untagged:")
	return &PruneResult{
		ImagesDeleted:  count,
		SpaceReclaimed: space,
	}, nil
}

func (d *Client) PruneContainers(ctx context.Context) (*PruneResult, error) {
	d.logger.Info("Pruning stopped Docker containers")

	result, err := d.executor.RunWithTimeout(ctx, 5*time.Minute, "docker", "container", "prune", "-f")
	if err != nil {
		return nil, fmt.Errorf("failed to prune containers: %w", err)
	}

	_, space := parsePruneOutput(result.Stdout, "")
	pruneResult := &PruneResult{SpaceReclaimed: space}
	for _, line := range strings.Split(result.Stdout, "\n") {
		if isHexContainerID(strings.TrimSpace(line)) {
			pruneResult.ContainersDeleted++
		}
	}
	return pruneResult, nil
}

func (d *Client) PruneVolumes(ctx context.Context) (*PruneResult, error) {
	d.logger.Info("Pruning anonymous Docker volumes")

	result, err := d.executor.RunWithTimeout(ctx, 5*time.Minute, "docker", "volume", "prune", "-f")
	if err != nil {
		return nil, fmt.Errorf("failed to prune volumes: %w", err)
	}

	_, space := parsePruneOutput(result.Stdout, "")
	pruneResult := &PruneResult{SpaceReclaimed: space}
	for _, line := range strings.Split(result.Stdout, "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 && !strings.Contains(trimmed, pruneReclaimedPrefix) && !strings.HasPrefix(trimmed, "Deleted") {
			pruneResult.VolumesDeleted++
		}
	}
	return pruneResult, nil
}

func isHexContainerID(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
