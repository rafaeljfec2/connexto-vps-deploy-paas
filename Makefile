.PHONY: proto proto-lint proto-go build build-agent bump-agent-version install-lint

PROTO_DIR := apps/proto
GEN_GO_DIR := apps/backend/gen/go
AGENT_VERSION := $(shell cat AGENT_VERSION | tr -d '\n')
GOLANGCI_LINT_VERSION ?= v2.12.2
GO_TOOLCHAIN ?= go1.25.5

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

# install-lint builds apps/backend/bin/golangci-lint locally with the project's
# Go toolchain. Pinned to v2.x because v1.x cannot load projects whose go.mod
# targets a Go toolchain newer than the one used to build the linter itself.
# The v2 module path lives under /v2/, hence the /v2/ in the go install target.
# Override GOLANGCI_LINT_VERSION or GO_TOOLCHAIN from the command line to test
# newer releases.
install-lint:
	@mkdir -p apps/backend/bin
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) with $(GO_TOOLCHAIN)..."
	GOTOOLCHAIN=$(GO_TOOLCHAIN) GOBIN=$(CURDIR)/apps/backend/bin go install \
		github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@echo ""
	@apps/backend/bin/golangci-lint --version
	@echo ""
	@echo "Installed at apps/backend/bin/golangci-lint"
