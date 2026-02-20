.PHONY: proto proto-lint proto-go build build-agent bump-agent-version

PROTO_DIR := apps/proto
GEN_GO_DIR := apps/backend/gen/go
AGENT_VERSION := $(shell cat AGENT_VERSION | tr -d '\n')

proto: proto-lint proto-go

proto-lint:
	buf lint $(PROTO_DIR)

proto-go:
	cd $(PROTO_DIR) && buf generate

build:
	cd apps/backend && go build \
		-ldflags="-X github.com/paasdeploy/backend/internal/handler.LatestAgentVersion=$(AGENT_VERSION)" \
		-o ../../dist/backend ./cmd/api

build-agent:
	cd apps/agent && go build \
		-ldflags="-X github.com/paasdeploy/agent/internal/agent.Version=$(AGENT_VERSION)" \
		-o ../../dist/agent ./cmd/agent

bump-agent-version:
ifndef v
	$(error Usage: make bump-agent-version v=0.7.0)
endif
	@echo "$(v)" > AGENT_VERSION
	@echo "Agent version bumped to $(v)"
