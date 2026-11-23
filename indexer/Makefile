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

# regenerates each subject's sqlc/ from its queries.sql; needs sqlc v1.31.1
# (go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1)
sqlc:
	sqlc generate

# regenerates internal/protobuf from ../protobuf/node_api.proto; needs protoc,
# and protoc-gen-go v1.36.12 and protoc-gen-go-grpc v1.6.2 from `go install`
proto:
	protoc -I ../protobuf \
		--go_out=internal/protobuf --go_opt=paths=source_relative \
		--go_opt=Mnode_api.proto=github.com/eduardokairalla1/EcoaNetwork/indexer/internal/protobuf \
		--go-grpc_out=internal/protobuf --go-grpc_opt=paths=source_relative \
		--go-grpc_opt=Mnode_api.proto=github.com/eduardokairalla1/EcoaNetwork/indexer/internal/protobuf \
		../protobuf/node_api.proto
