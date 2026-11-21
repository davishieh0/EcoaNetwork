-include .env
export

build:
	go build -o bin/indexer ./cmd/indexer

migrate: build
	./bin/indexer migrate

serve: build
	./bin/indexer serve

test:
	go test -v ./...
