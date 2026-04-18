.PHONY: all build clean test run-apiserver dev-infra dev-infra-down docker-build

BINARY_DIR := bin
GO_FLAGS := -ldflags="-s -w" -trimpath

all: build

build: apiserver

apiserver:
	CGO_ENABLED=0 go build $(GO_FLAGS) -o $(BINARY_DIR)/kube-apiserver ./cmd/apiserver

clean:
	rm -rf $(BINARY_DIR)

test:
	go test -v -race ./...

run-apiserver:
	go run ./cmd/apiserver

deps:
	go mod tidy
	go mod download

lint:
	golangci-lint run ./...

docker-build:
	docker build -t kube-apiserver:latest -f docker/Dockerfile.apiserver .

dev-infra:
	docker compose up -d mongodb

dev-infra-down:
	docker compose down
