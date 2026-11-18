-include .env
export

build:
	go build -o bin/ecoa ./cmd/ecoa

run: build
	./bin/ecoa

test:
	go test -v ./...

# regenerates internal/protobuf from ../protobuf; needs protoc, and
# protoc-gen-go v1.36.12 and protoc-gen-go-grpc v1.6.2 from `go install`
proto:
	protoc -I ../protobuf \
		--go_out=internal/protobuf --go_opt=paths=source_relative \
		--go-grpc_out=internal/protobuf --go-grpc_opt=paths=source_relative \
		../protobuf/node_api.proto ../protobuf/sync.proto
