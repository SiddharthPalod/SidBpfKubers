# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /workspace

# Copy go modules
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

# Build static binary with BuildKit caching and incremental compilation
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o manager cmd/main.go

# Runtime stage
FROM alpine:3.20

WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
