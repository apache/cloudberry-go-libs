all: depend test

SHELL := /bin/bash
.DEFAULT_GOAL := all
DIR_PATH=$(shell dirname `pwd`)
BIN_DIR=$(shell echo $${GOPATH:-~/go} | awk -F':' '{ print $$1 "/bin"}')
BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
GOLANG_VERSION = 1.19.6
GINKGO=$(GOPATH)/bin/ginkgo
DEST = .

GOFLAGS :=

.PHONY: test lint goimports golangci-lint gofmt unit coverage depend set-dev set-prod

test: lint unit

lint:
	$(MAKE) goimports gofmt golangci-lint

goimports:
	docker run --rm -i -v "${PWD}":/data -w /data unibeautify/goimports -w -l /data

golangci-lint:
	docker run --rm -v ${PWD}:/data -w /data golangci/golangci-lint:v1.64.8 golangci-lint run -v

gofmt:
	docker run --rm -v ${PWD}:/data cytopia/gofmt --ci .

$(GINKGO):
	go install github.com/onsi/ginkgo/v2/ginkgo@v2.13.0

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


