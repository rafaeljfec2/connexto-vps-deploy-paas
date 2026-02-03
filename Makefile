.PHONY: proto proto-lint proto-go build build-agent

PROTO_DIR := apps/proto
GEN_GO_DIR := apps/backend/gen/go

proto: proto-lint proto-go

proto-lint:
	buf lint $(PROTO_DIR)

proto-go:
	cd $(PROTO_DIR) && buf generate

build:
	cd apps/backend && go build -o ../../dist/backend ./cmd/api

build-agent:
	cd apps/agent && go build -o ../../dist/agent ./cmd/agent
