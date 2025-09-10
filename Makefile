all: depend test

SHELL := /bin/bash
.DEFAULT_GOAL := all
DIR_PATH=$(shell dirname `pwd`)
BIN_DIR=$(shell echo $${GOPATH:-~/go} | awk -F':' '{ print $$1 "/bin"}')
BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
GOLANG_VERSION = 1.25.0
GINKGO=$(BIN_DIR)/ginkgo
DEST = .

GOFLAGS :=
GOIMPORTS=$(BIN_DIR)/goimports
GOLANG_LINTER=$(BIN_DIR)/golangci-lint

.PHONY: test lint format unit coverage depend

test: lint unit

$(GOIMPORTS):
	GOBIN=$(BIN_DIR) go install golang.org/x/tools/cmd/goimports@latest

LINTER_VERSION=v2.12.2
$(GOLANG_LINTER):
	GOBIN=$(BIN_DIR) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(LINTER_VERSION)

format: $(GOIMPORTS)
	@goimports -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

lint: $(GOLANG_LINTER)
	golangci-lint run

$(GINKGO):
	GOBIN=$(BIN_DIR) go install github.com/onsi/ginkgo/v2/ginkgo@v2.13.0

unit: $(GINKGO)
		ginkgo -r --keep-going --randomize-suites --randomize-all \
			cluster \
			conv \
			dbconn \
			gperror \
			gplog \
			iohelper \
			structmatcher \
			2>&1

coverage :
		@./show_coverage.sh

depend:
	go mod download

clean :
		# Test artifacts
		rm -rf /tmp/go-build*
		rm -rf /tmp/gexec_artifacts*
		rm -rf /tmp/ginkgo*
		# Code coverage files
		rm -rf /tmp/cover*
		rm -rf /tmp/unit*
