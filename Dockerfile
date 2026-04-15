# Build stage
FROM golang:1.23-alpine AS builder

ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}" \
    -o /osm2ch ./cmd/osm2ch

# Runtime stage
FROM scratch

COPY --from=builder /osm2ch /osm2ch

WORKDIR /data

ENTRYPOINT ["/osm2ch"]
CMD ["--help"]
