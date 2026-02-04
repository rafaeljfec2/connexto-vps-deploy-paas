package grpcserver

import (
	"sync"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

type AgentCommandQueue struct {
	mu   sync.Mutex
	cmds map[string][]*pb.AgentCommand
}

func NewAgentCommandQueue() *AgentCommandQueue {
	return &AgentCommandQueue{cmds: make(map[string][]*pb.AgentCommand)}
}

func (q *AgentCommandQueue) Enqueue(serverID string, cmd *pb.AgentCommand) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.cmds[serverID] = append(q.cmds[serverID], cmd)
}

func (q *AgentCommandQueue) GetAndClear(serverID string) []*pb.AgentCommand {
	q.mu.Lock()
	defer q.mu.Unlock()
	pending := q.cmds[serverID]
	delete(q.cmds, serverID)
	return pending
}
