# Build from the repository root: the module replaces protocol/go with a
# sibling directory.
#   docker build -f indexer/Dockerfile -t ecoa-indexer .

FROM golang:1.26 AS build

WORKDIR /src

# go.sum appears with the first dependency; the wildcard lets it be absent
COPY protocol/go/go.mod protocol/go/go.sum protocol/go/
COPY indexer/go.mod indexer/go.sum* indexer/
RUN cd indexer && go mod download

COPY protocol/go protocol/go
COPY indexer indexer

RUN cd indexer && CGO_ENABLED=0 go build -o /out/indexer ./cmd/indexer


FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/indexer /indexer

# The default of HTTP_ADDR is loopback, which nothing outside the container
# can reach.
ENV HTTP_ADDR=0.0.0.0:8081

EXPOSE 8081

# The subcommand is the argument: `docker run ecoa-indexer project`.
ENTRYPOINT ["/indexer"]
CMD ["serve"]
