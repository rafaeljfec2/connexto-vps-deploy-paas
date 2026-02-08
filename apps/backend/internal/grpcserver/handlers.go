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

	var previousVersion string
	if srv, findErr := s.serverRepo.FindByID(serverID); findErr == nil && srv.AgentVersion != nil {
		previousVersion = *srv.AgentVersion
	}

	if err := s.serverRepo.UpdateHeartbeat(serverID, req.AgentVersion); err != nil {
		s.logger.Error("failed to update heartbeat", "serverId", serverID, "error", err)
	}
	s.hub.Update(serverID)

	newVersion := req.GetAgentVersion()
	if s.agentUpdateNotifier != nil && newVersion != "" && newVersion != previousVersion {
		s.agentUpdateNotifier.NotifyUpdateCompleted(serverID, newVersion)
	}

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

	var previousVersion string
	if srv, findErr := s.serverRepo.FindByID(serverID); findErr == nil && srv.AgentVersion != nil {
		previousVersion = *srv.AgentVersion
	}

	if hbErr := s.serverRepo.UpdateHeartbeat(serverID, req.GetAgentVersion()); hbErr != nil {
		s.logger.Error("failed to update heartbeat", "serverId", serverID, "error", hbErr)
	}
	s.hub.Update(serverID)

	commands := s.cmdQueue.GetAndClear(serverID)

	s.emitAgentUpdateEvents(serverID, previousVersion, req.GetAgentVersion(), commands)

	return &pb.HeartbeatResponse{
		Acknowledged: true,
		Commands:     commands,
	}, nil
}

func (s *Server) emitAgentUpdateEvents(serverID, previousVersion, newVersion string, commands []*pb.AgentCommand) {
	if s.agentUpdateNotifier == nil {
		if len(commands) > 0 {
			s.logger.Warn("agentUpdateNotifier is nil, cannot emit agent update SSE events",
				"serverId", serverID, "commandCount", len(commands))
		}
		return
	}

	for _, cmd := range commands {
		if cmd.GetType() == pb.AgentCommandType_AGENT_COMMAND_UPDATE_AGENT {
			s.logger.Info("agent update command delivered via heartbeat", "serverId", serverID)
			s.agentUpdateNotifier.NotifyUpdateDelivered(serverID)
			break
		}
	}

	if newVersion != "" && newVersion != previousVersion {
		s.logger.Info("agent version changed", "serverId", serverID,
			"previousVersion", previousVersion, "newVersion", newVersion)
		s.agentUpdateNotifier.NotifyUpdateCompleted(serverID, newVersion)
	}
}
