package grpcserver

import (
	"log/slog"
	"testing"
	"time"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

func newTestAgentService() *AgentService {
	return &AgentService{logger: slog.Default()}
}

func TestLogStream_SurvivesSubscriberLeave(t *testing.T) {
	s := newTestAgentService()
	const deployID = "deploy-survive-leave"

	ch1 := s.getOrCreateLogStream(deployID)
	logFn := s.buildLogFunc(deployID)

	// Simulate StreamDeployLogs exiting without Delete (reader dropped).
	// Writer must keep delivering into the same channel.
	logFn(pb.DeployStage_DEPLOY_STAGE_BUILD, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO, "still flowing")

	select {
	case entry := <-ch1:
		if entry.Message != "still flowing" {
			t.Fatalf("message = %q, want still flowing", entry.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("expected log entry after subscriber leave")
	}

	ch2 := s.getOrCreateLogStream(deployID)
	if ch1 != ch2 {
		t.Fatal("reattach must reuse the same channel owned by ExecuteDeploy")
	}

	s.closeLogStream(deployID)

	if _, ok := s.logStreams.Load(deployID); ok {
		t.Fatal("closeLogStream must remove the channel from the map")
	}

	// After close, further writes are no-ops (Load miss) — must not panic.
	logFn(pb.DeployStage_DEPLOY_STAGE_BUILD, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO, "after close")
}

func TestLogStream_CloseUnblocksReader(t *testing.T) {
	s := newTestAgentService()
	const deployID = "deploy-close-unblock"

	ch := s.getOrCreateLogStream(deployID)
	gotOK := make(chan bool, 1)
	go func() {
		_, ok := <-ch
		gotOK <- ok
	}()

	s.closeLogStream(deployID)

	select {
	case ok := <-gotOK:
		if ok {
			t.Fatal("expected closed channel (ok=false)")
		}
	case <-time.After(time.Second):
		t.Fatal("closeLogStream did not unblock reader")
	}
}

func TestLogStream_DeleteOnLeaveWouldDropLogs_RegressionGuard(t *testing.T) {
	// Documents the old bug: Delete on StreamDeployLogs exit made buildLogFunc
	// miss the channel. This test asserts current ownership keeps the map entry.
	s := newTestAgentService()
	const deployID = "deploy-no-delete"

	_ = s.getOrCreateLogStream(deployID)
	// Old code: defer s.logStreams.Delete(deployID) on stream exit.
	// We intentionally do NOT delete here.
	if _, ok := s.logStreams.Load(deployID); !ok {
		t.Fatal("channel must remain registered after reader leave")
	}

	logFn := s.buildLogFunc(deployID)
	logFn(pb.DeployStage_DEPLOY_STAGE_GIT_SYNC, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO, "git sync")

	val, ok := s.logStreams.Load(deployID)
	if !ok {
		t.Fatal("map entry missing — logs would be dropped")
	}
	ch := val.(chan *pb.DeployLogEntry)
	select {
	case entry := <-ch:
		if entry.Message != "git sync" {
			t.Fatalf("got %q", entry.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("log dropped after simulated subscriber leave")
	}

	s.closeLogStream(deployID)
}
