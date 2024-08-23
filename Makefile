GIT_VERSION ?= $(shell git describe --abbrev=4 --dirty --always --tags)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD)
COMMIT_TIME ?= $(shell git log -1 --format="%at" | xargs -I{} date -d @{} +%Y/%m/%d_%H:%M:%S)
VERSION=$(GIT_VERSION)
BIN_DIR = $(GOPATH)/bin
TMPDIR ?= $(shell dirname $$(mktemp -u))

COVER_FILE ?= $(TMPDIR)/$(PACKAGE)-coverage.out
APP_NAME = "viral-backend"
COVER_FILE ?= $(TMPDIR)/$(PACKAGE)-coverage.out

all: image run test 

.PHONY: image
image:
	docker build \
		--build-arg COMMIT_HASH=$(COMMIT_HASH) \
		--build-arg COMMIT_TIME=$(COMMIT_TIME) \
		--build-arg VERSION=$(GIT_VERSION) \
		--build-arg GOPRIVATE=$(GONOSUMDB) \
		-t faustvx/todo_server:v1 -f ./Dockerfile .


build:
 	docker build --tag faustvx/todo_server:v1 .

test:
	docker run go clean -testcache
	docker run --rm faustvx/todo_server:v1 go test -v ./...

run:
	docker run -p 7540:7540 faustvx/todo_server:v1
