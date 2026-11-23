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
