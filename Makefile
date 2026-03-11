.PHONY: build test clean install

BINARY=bin/claudette
MODULE=claudette

build:
	CGO_ENABLED=1 go build -o $(BINARY) ./cmd/claudette

test:
	CGO_ENABLED=1 go test ./internal/... ./cmd/...

clean:
	rm -rf bin/ .claudette/

install:
	CGO_ENABLED=1 go install ./cmd/claudette

tidy:
	go mod tidy

fmt:
	gofmt -w .

vet:
	go vet ./...
