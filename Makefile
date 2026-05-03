BINARY := lynxeye

.PHONY: build test tidy fmt run docker-build

build:
	go build -o $(BINARY) ./cmd/lynxeye

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	gofmt -w $(shell find . -name '*.go')

run:
	go run ./cmd/lynxeye run --config config.example.yaml --once

docker-build:
	docker build -t lynxeye:latest .
