package main

//go:generate bpf2go -target amd64 tcpMonitor bpf/tcp_monitor.c -- -I/usr/include -I/usr/include/x86_64-linux-gnu

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

const (
	EventConnect    = 1
	EventRetransmit = 2
	EventClose      = 3
)

type TCPEvent struct {
	Saddr       uint32
	Daddr       uint32
	Sport       uint16
	Dport       uint16
	EventType   uint8
	Pad         [3]uint8
	TimestampNs uint64
}

// Service Registry - maps standard ports to protocol names
var defaultPortRegistry = map[uint16]string{
	5432:  "postgresql",
	6379:  "redis",
	3306:  "mysql",
	27017: "mongodb",
	9092:  "kafka",
	9200:  "elasticsearch",
	80:    "http",
	8080:  "http-alt",
	443:   "https",
	50051: "grpc",
}

// Noise Filter - K8s internal control plane ports ignored to keep logs clean
var controlPlaneNoise = map[uint16]bool{
	6443:  true, // kube-apiserver
	2379:  true, // etcd client
	2380:  true, // etcd peer
	2381:  true, // etcd metrics
	10250: true, // kubelet
	10257: true, // kube-controller-manager
	10259: true, // kube-scheduler
}

// Labeled metric counters: key = serviceName
var (
	connectionCounters sync.Map // map[string]*uint64
	retransmitCounters sync.Map // map[string]*uint64
	totalConnections   uint64
	totalRetransmits   uint64
)

func getCounter(m *sync.Map, key string) *uint64 {
	if val, ok := m.Load(key); ok {
		return val.(*uint64)
	}
	var newCounter uint64
	actual, _ := m.LoadOrStore(key, &newCounter)
	return actual.(*uint64)
}

func isNoise(sport, dport uint16) bool {
	return controlPlaneNoise[sport] || controlPlaneNoise[dport]
}

func resolveService(sport, dport uint16) string {
	if svc, ok := defaultPortRegistry[dport]; ok {
		return svc
	}
	if svc, ok := defaultPortRegistry[sport]; ok {
		return svc
	}
	// Fallback dynamic format: port-<dport>
	return fmt.Sprintf("port-%d", dport)
}

func intToIP(val uint32) net.IP {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], val)
	return net.IPv4(b[0], b[1], b[2], b[3])
}

func main() {
	log.Println("==> Starting SidKubers eBPF Network Observability Agent...")

	// 1. Allow the process to lock memory for eBPF maps
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatalf("Failed to remove memlock limit: %v", err)
	}

	// 2. Load pre-compiled eBPF programs and maps into kernel
	objs := tcpMonitorObjects{}
	if err := loadTcpMonitorObjects(&objs, nil); err != nil {
		log.Fatalf("Failed to load eBPF objects: %v", err)
	}
	defer objs.Close()

	// 3. Attach Tracepoints
	tpState, err := link.Tracepoint("sock", "inet_sock_set_state", objs.TraceInetSockSetState, nil)
	if err != nil {
		log.Fatalf("Failed to attach tracepoint inet_sock_set_state: %v", err)
	}
	defer tpState.Close()

	tpRetransmit, err := link.Tracepoint("tcp", "tcp_retransmit_skb", objs.TraceTcpRetransmit, nil)
	if err != nil {
		log.Fatalf("Failed to attach tracepoint tcp_retransmit_skb: %v", err)
	}
	defer tpRetransmit.Close()

	log.Println("==> eBPF Probes attached successfully to Linux kernel!")

	// 4. Open RingBuffer reader
	rd, err := ringbuf.NewReader(objs.Events)
	if err != nil {
		log.Fatalf("Failed to open ringbuf reader: %v", err)
	}
	defer rd.Close()

	// 5. Setup HTTP Prometheus endpoint on :9090
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "# HELP ebpf_tcp_connections_total Total TCP connections observed\n")
		fmt.Fprintf(w, "# TYPE ebpf_tcp_connections_total counter\n")
		fmt.Fprintf(w, "ebpf_tcp_connections_total %d\n\n", atomic.LoadUint64(&totalConnections))

		fmt.Fprintf(w, "# HELP ebpf_tcp_retransmissions_total Total TCP retransmitted packets observed\n")
		fmt.Fprintf(w, "# TYPE ebpf_tcp_retransmissions_total counter\n")
		fmt.Fprintf(w, "ebpf_tcp_retransmissions_total %d\n\n", atomic.LoadUint64(&totalRetransmits))

		fmt.Fprintf(w, "# HELP ebpf_service_connections_total Total connections per service\n")
		fmt.Fprintf(w, "# TYPE ebpf_service_connections_total counter\n")
		connectionCounters.Range(func(key, value any) bool {
			cnt := atomic.LoadUint64(value.(*uint64))
			fmt.Fprintf(w, "ebpf_service_connections_total{service=\"%s\"} %d\n", key.(string), cnt)
			return true
		})
		fmt.Fprintln(w)

		fmt.Fprintf(w, "# HELP ebpf_service_retransmissions_total Total retransmissions per service\n")
		fmt.Fprintf(w, "# TYPE ebpf_service_retransmissions_total counter\n")
		retransmitCounters.Range(func(key, value any) bool {
			cnt := atomic.LoadUint64(value.(*uint64))
			fmt.Fprintf(w, "ebpf_service_retransmissions_total{service=\"%s\"} %d\n", key.(string), cnt)
			return true
		})
		fmt.Fprintln(w)

		// Legacy aliases for backward compatibility
		fmt.Fprintf(w, "# HELP ebpf_postgres_connections_total Legacy alias for postgres connections\n")
		fmt.Fprintf(w, "ebpf_postgres_connections_total %d\n", atomic.LoadUint64(getCounter(&connectionCounters, "postgresql")))
		fmt.Fprintf(w, "ebpf_redis_connections_total %d\n", atomic.LoadUint64(getCounter(&connectionCounters, "redis")))
		fmt.Fprintf(w, "ebpf_mongodb_connections_total %d\n", atomic.LoadUint64(getCounter(&connectionCounters, "mongodb")))
		fmt.Fprintf(w, "ebpf_nginx_connections_total %d\n", atomic.LoadUint64(getCounter(&connectionCounters, "http"))+atomic.LoadUint64(getCounter(&connectionCounters, "http-alt")))
	})

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	go func() {
		log.Println("==> Metrics server listening on :9090/metrics")
		if err := http.ListenAndServe(":9090", nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// 6. Signal handling for graceful exit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	// 7. Event consumer loop
	go func() {
		var event TCPEvent
		for {
			record, err := rd.Read()
			if err != nil {
				if errors.Is(err, ringbuf.ErrClosed) {
					return
				}
				log.Printf("Ringbuf error: %v", err)
				continue
			}

			if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &event); err != nil {
				log.Printf("Failed to decode event: %v", err)
				continue
			}

			// Filter noisy K8s internal control plane traffic (kube-apiserver, etcd, kubelet)
			if isNoise(event.Sport, event.Dport) {
				continue
			}

			srcIP := intToIP(event.Saddr)
			dstIP := intToIP(event.Daddr)
			svc := resolveService(event.Sport, event.Dport)

			switch event.EventType {
			case EventConnect:
				atomic.AddUint64(&totalConnections, 1)
				atomic.AddUint64(getCounter(&connectionCounters, svc), 1)
				log.Printf("🔌 [TCP CONNECT] %s:%d -> %s:%d [%s] at %s",
					srcIP, event.Sport, dstIP, event.Dport, svc, time.Now().Format("15:04:05.000"))

			case EventRetransmit:
				atomic.AddUint64(&totalRetransmits, 1)
				atomic.AddUint64(getCounter(&retransmitCounters, svc), 1)
				log.Printf("⚠️  [TCP RETRANSMIT] %s:%d -> %s:%d [%s] packet dropped/retransmitted!",
					srcIP, event.Sport, dstIP, event.Dport, svc)

			case EventClose:
				// Connection teardown
			}
		}
	}()

	<-sig
	log.Println("==> Shutting down eBPF agent...")
}
