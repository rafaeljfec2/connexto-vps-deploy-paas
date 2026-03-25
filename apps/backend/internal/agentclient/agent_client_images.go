package agentclient

import (
	"context"
	"fmt"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

func (c *AgentClient) ListImages(ctx context.Context, host string, port int, all bool) ([]*pb.ImageInfo, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.ListImages(ctx, &pb.ListImagesRequest{All: all})
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}
	return resp.Images, nil
}

func (c *AgentClient) RemoveImage(ctx context.Context, host string, port int, imageID string, force bool) error {
	cl, err := c.client(host, port)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.RemoveImage(ctx, &pb.RemoveImageRequest{ImageId: imageID, Force: force})
	if err != nil {
		return fmt.Errorf("remove image: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("remove image failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) PruneImages(ctx context.Context, host string, port int) (*pb.PruneImagesResponse, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return cl.PruneImages(ctx, &pb.PruneImagesRequest{})
}
