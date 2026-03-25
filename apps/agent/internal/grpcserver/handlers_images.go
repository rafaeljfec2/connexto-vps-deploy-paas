package grpcserver

import (
	"context"
	"fmt"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

func (s *AgentService) ListImages(ctx context.Context, req *pb.ListImagesRequest) (*pb.ListImagesResponse, error) {
	images, err := s.docker.ListImages(ctx, req.All)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	pbImages := make([]*pb.ImageInfo, 0, len(images))
	for _, img := range images {
		pbImages = append(pbImages, &pb.ImageInfo{
			Id:         img.ID,
			Repository: img.Repository,
			Tag:        img.Tag,
			Size:       img.Size,
			Created:    img.Created,
			Dangling:   img.Dangling,
			Containers: int32(img.Containers),
		})
	}

	return &pb.ListImagesResponse{Images: pbImages}, nil
}

func (s *AgentService) RemoveImage(ctx context.Context, req *pb.RemoveImageRequest) (*pb.RemoveImageResponse, error) {
	if err := s.docker.RemoveImageByID(ctx, req.ImageId, req.Force); err != nil {
		return &pb.RemoveImageResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.RemoveImageResponse{Success: true, Message: "Image removed"}, nil
}

func (s *AgentService) PruneImages(ctx context.Context, _ *pb.PruneImagesRequest) (*pb.PruneImagesResponse, error) {
	result, err := s.docker.PruneImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prune images: %w", err)
	}
	return &pb.PruneImagesResponse{
		ImagesRemoved:       int32(result.ImagesDeleted),
		SpaceReclaimedBytes: result.SpaceReclaimed,
	}, nil
}

func (s *AgentService) PruneContainers(ctx context.Context, _ *pb.PruneContainersRequest) (*pb.PruneContainersResponse, error) {
	result, err := s.docker.PruneContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prune containers: %w", err)
	}
	return &pb.PruneContainersResponse{
		ContainersRemoved:   int32(result.ContainersDeleted),
		SpaceReclaimedBytes: result.SpaceReclaimed,
	}, nil
}

func (s *AgentService) PruneVolumes(ctx context.Context, _ *pb.PruneVolumesRequest) (*pb.PruneVolumesResponse, error) {
	result, err := s.docker.PruneVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prune volumes: %w", err)
	}
	return &pb.PruneVolumesResponse{
		VolumesRemoved:      int32(result.VolumesDeleted),
		SpaceReclaimedBytes: result.SpaceReclaimed,
	}, nil
}
