# Build from the repository root: the module replaces protocol/go with a
# sibling directory.
#   docker build -f node/Dockerfile -t ecoa-node .

FROM golang:1.26 AS build

WORKDIR /src

COPY protocol/go/go.mod protocol/go/go.sum protocol/go/
COPY node/go.mod node/go.sum node/
RUN cd node && go mod download

COPY protocol/go protocol/go
COPY node node

RUN cd node && CGO_ENABLED=0 go build -o /out/ecoa ./cmd/ecoa
RUN mkdir /out/data


FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/ecoa /ecoa
COPY --from=build --chown=nonroot:nonroot /out/data /data

# The defaults of LISTEN_ADDR and GRPC_ADDR are loopback, which nothing
# outside the container can reach.
ENV LISTEN_ADDR=/ip4/0.0.0.0/tcp/4001 \
    GRPC_ADDR=0.0.0.0:50051 \
    BOLT_DB_PATH=/data/bolt.db \
    PEER_KEY_PATH=/data/peer.key

VOLUME /data
EXPOSE 4001 50051

ENTRYPOINT ["/ecoa"]
