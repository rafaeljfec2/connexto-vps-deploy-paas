package grpcserver

import (
	"context"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	serverID, err := extractServerIDFromCert(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.serverRepo.UpdateHeartbeat(serverID, req.AgentVersion); err != nil {
		s.logger.Error("failed to update heartbeat", "serverId", serverID, "error", err)
	}
	s.hub.Update(serverID)

	resp := &pb.RegisterResponse{
		Accepted: true,
		Message:  "registered",
		Config: &pb.AgentConfig{
			HeartbeatIntervalSeconds: 30,
			LogBufferSize:            1000,
			MaxConcurrentDeploys:     2,
			EnableMetrics:            true,
			EnableAutoUpdate:         false,
		},
	}

	return resp, nil
}

func (s *Server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	serverID, err := extractServerIDFromCert(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.serverRepo.UpdateHeartbeat(serverID, req.GetAgentVersion()); err != nil {
		s.logger.Error("failed to update heartbeat", "serverId", serverID, "error", err)
	}
	s.hub.Update(serverID)

	commands := s.cmdQueue.GetAndClear(serverID)
	return &pb.HeartbeatResponse{
		Acknowledged: true,
		Commands:     commands,
	}, nil
}
