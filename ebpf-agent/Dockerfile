FROM golang:1.24-bookworm AS builder

WORKDIR /workspace

# Copy dependencies
COPY go.mod ./
RUN go mod download

# Copy source code (including pre-generated tcpmonitor_x86_bpfel.go and .o)
COPY . .

# Build static Linux ELF binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /ebpf-agent .

# Runtime container
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /
COPY --from=builder /ebpf-agent .

EXPOSE 9090

ENTRYPOINT ["/ebpf-agent"]
