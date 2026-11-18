-include .env
export

build:
	go build -o bin/indexer ./cmd/indexer

serve: build
	./bin/indexer serve

test:
	go test -v ./...
