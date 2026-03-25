package grpcserver

import (
	"context"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *AgentService) ExecuteDeploy(ctx context.Context, req *pb.DeployRequest) (*pb.DeployResponse, error) {
	s.getOrCreateLogStream(req.DeploymentId)
	logFn := s.buildLogFunc(req.DeploymentId)
	resp := s.deployExecutor.Execute(ctx, req, logFn)
	s.closeLogStream(req.DeploymentId)
	return resp, nil
}

func (s *AgentService) StreamDeployLogs(sub *pb.DeployLogSubscription, stream pb.AgentService_StreamDeployLogsServer) error {
	ch := s.getOrCreateLogStream(sub.DeploymentId)
	defer s.logStreams.Delete(sub.DeploymentId)

	s.logger.Info("Deploy log stream opened", "deploymentId", sub.DeploymentId)

	for {
		select {
		case entry, ok := <-ch:
			if !ok {
				s.logger.Info("Deploy log stream closed", "deploymentId", sub.DeploymentId)
				return nil
			}
			if err := stream.Send(entry); err != nil {
				s.logger.Warn("Failed to send deploy log entry", "deploymentId", sub.DeploymentId, "error", err)
				return err
			}
		case <-stream.Context().Done():
			s.logger.Info("Deploy log stream context cancelled", "deploymentId", sub.DeploymentId)
			return nil
		}
	}
}

func (s *AgentService) getOrCreateLogStream(deploymentID string) chan *pb.DeployLogEntry {
	ch := make(chan *pb.DeployLogEntry, logStreamBuffer)
	actual, _ := s.logStreams.LoadOrStore(deploymentID, ch)
	return actual.(chan *pb.DeployLogEntry)
}

func (s *AgentService) buildLogFunc(deploymentID string) func(pb.DeployStage, pb.DeployLogLevel, string) {
	return func(stage pb.DeployStage, level pb.DeployLogLevel, message string) {
		val, ok := s.logStreams.Load(deploymentID)
		if !ok {
			return
		}
		ch := val.(chan *pb.DeployLogEntry)
		entry := &pb.DeployLogEntry{
			DeploymentId: deploymentID,
			Timestamp:    timestamppb.Now(),
			Level:        level,
			Stage:        stage,
			Message:      message,
		}
		select {
		case ch <- entry:
		default:
			s.logger.Debug("Deploy log channel full, dropping entry", "deploymentId", deploymentID)
		}
	}
}

func (s *AgentService) closeLogStream(deploymentID string) {
	val, loaded := s.logStreams.LoadAndDelete(deploymentID)
	if !loaded {
		return
	}
	close(val.(chan *pb.DeployLogEntry))
}
