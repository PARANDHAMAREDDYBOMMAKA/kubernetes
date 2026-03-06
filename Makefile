.PHONY: all build clean test run-apiserver run-controller run-agent run-scheduler

BINARY_DIR := bin
GO_FLAGS := -ldflags="-s -w" -trimpath

all: build

build: apiserver controller agent scheduler kubectl dashboard

apiserver:
	CGO_ENABLED=0 go build $(GO_FLAGS) -o $(BINARY_DIR)/kube-apiserver ./cmd/apiserver

controller:
	CGO_ENABLED=0 go build $(GO_FLAGS) -o $(BINARY_DIR)/kube-controller ./cmd/controller

agent:
	CGO_ENABLED=0 go build $(GO_FLAGS) -o $(BINARY_DIR)/kube-agent ./cmd/agent

scheduler:
	CGO_ENABLED=0 go build $(GO_FLAGS) -o $(BINARY_DIR)/kube-scheduler ./cmd/scheduler

kubectl:
	CGO_ENABLED=0 go build $(GO_FLAGS) -o $(BINARY_DIR)/kube ./cmd/kubectl

dashboard:
	CGO_ENABLED=0 go build $(GO_FLAGS) -o $(BINARY_DIR)/kube-dashboard ./cmd/dashboard

clean:
	rm -rf $(BINARY_DIR)

test:
	go test -v -race ./...

run-apiserver:
	go run ./cmd/apiserver -memory

run-controller:
	go run ./cmd/controller -memory

run-agent:
	go run ./cmd/agent -memory -mock-runtime

run-scheduler:
	go run ./cmd/scheduler -memory

deps:
	go mod tidy
	go mod download

lint:
	golangci-lint run ./...

docker-build:
	docker build -t kube-apiserver:latest -f docker/Dockerfile.apiserver .
	docker build -t kube-controller:latest -f docker/Dockerfile.controller .
	docker build -t kube-agent:latest -f docker/Dockerfile.agent .
	docker build -t kube-scheduler:latest -f docker/Dockerfile.scheduler .

docker-push:
	docker push kube-apiserver:latest
	docker push kube-controller:latest
	docker push kube-agent:latest
	docker push kube-scheduler:latest
