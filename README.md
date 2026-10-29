# SidBpfKubers

<div align="center">

[![Go](https://goreportcard.com/badge/github.com/SidBpfKubers)](https://golang.org)
[![Kubernetes](https://img.shields.io/badge/kubernetes-v1.28%2B-blue.svg?logo=kubernetes)](https://kubernetes.io/)
[![eBPF](https://img.shields.io/badge/eBPF-In--Kernel-purple.svg?logo=linux)](https://ebpf.io/)
[![Prometheus](https://img.shields.io/badge/Prometheus-Metrics-orange.svg?logo=prometheus)](https://prometheus.io/)
[![Grafana](https://img.shields.io/badge/Grafana-Visualization-F46800.svg?logo=grafana)](https://grafana.com/)
[![License](https://img.shields.io/badge/license-Apache--2.0-green.svg)](LICENSE)

**An Enterprise Multi-Engine Cloud-Native Database Operator for Kubernetes Paired with Low-Overhead In-Kernel eBPF Telemetry, Prometheus Metrics, and an Archival Mission Control Cockpit.**

[Overview](#overview) • [Architecture](#architecture) • [Supported Database Engines](#supported-database-engines) • [Key Features](#key-features) • [Directory Structure](#directory-structure) • [Get Started](#get-started) • [Operator Walkthrough](#operator-walkthrough) • [Tech Stack](#tech-stack)

</div>

---

## Overview

Traditional Kubernetes database operators manage stateful workloads reactively through high-level container states (cgroups, readiness probes, exit codes). However, they remain blind to sub-millisecond network degradation, streaming replication packet lag, and in-kernel TCP drops.

**SidBpfKubers** bridges this operational gap by combining a **declarative multi-engine Kubernetes database controller** with **deep in-kernel eBPF network telemetry**:

* **5-Engine Enterprise Distributed Suite**: Declarative orchestration across Relational SQL (**PostgreSQL 16**), In-Memory Caching (**Redis 7**), Document NoSQL (**MongoDB 7.0**), Consensus Quorum (**etcd 3.5**), and Wide-Column Peer-to-Peer Rings (**Apache Cassandra 4.1**).
* **Deep In-Kernel Observability**: Attaches eBPF probes directly into Linux kernel network tracepoints (`sock/inet_sock_set_state` and `tcp/tcp_retransmit_skb`) to capture real-time socket handshakes, lifecycle states, and packet drop anomalies.
* **Atomic 3-Tier Service Routing**: Automatically allocates virtual IPs (`-rw` write primary, `-ro` read-only pool, `-r` all-replica load-balanced, `-headless` peer discovery) with native engine port mapping.
* **Mission Control Cockpit**: An archival Deep Forest / Obsidian visual cockpit featuring real-time cluster topography, glitch-free optimistic skeleton loaders, direct query test benches, and live embedded Grafana & Prometheus sparklines.

---

## Architecture

```
                                      +------------------------------------+
                                      |     SidBpfKubers Web Cockpit       |
                                      |   (:8085 / Deep Forest Theme)      |
                                      +-----------------+------------------+
                                                        |
                                          HTTP / Direct | CLI Drawer
                                                        v
+----------------------------------------------------------------------------------------------------+
|  Kubernetes Cluster (demo-app & sidkubers-system)                                                  |
|                                                                                                    |
|   +--------------------------+       Reconciles       +----------------------------------------+   |
|   |   Custom Resource (CRD)  | <====================> |        database-operator (Go)          |   |
|   |  DatabaseCluster Specs   |                        |  - Multi-Engine State Reconcilers      |   |
|   |  (PG, Redis, Mongo,      |                        |  - 3-Tier Services (-rw, -ro, -r)      |   |
|   |   etcd, Cassandra)       |                        |  - Zero-Downtime Switchover Engine     |   |
|   +--------------------------+                        |  - Snapshot Backup & Restore Jobs      |   |
|                 |                                     +----------------------------------------+   |
|                 v                                                                                  |
|   +--------------------------------------------------------------------------------------------+   |
|   |   StatefulSet Distributed Pod Replicas                                                     |   |
|   |   - sample-postgres-0..N (WAL Streaming Primary/Standby)                                   |   |
|   |   - sample-redis-0..N    (Master/Replica Replication)                                      |   |
|   |   - sample-mongodb-0..N  (Replica Set rs0 WiredTiger Quorum)                               |   |
|   |   - sample-etcd-0..N     (Raft Peer Consensus Quorum)                                      |   |
|   |   - sample-cassandra-0..N(Gossip Peer-to-Peer DHT Ring)                                     |   |
|   +--------------------------------------------------------------------------------------------+   |
|                 |                                                                                  |
|                 +====================+ Intercepts TCP Sockets & Packets                            |
|                                      v                                                             |
|                         +-----------------------------------------------+                          |
|                         |         ebpf-agent (DaemonSet)                |                          |
|                         |   - tracepoint/sock/inet_sock_set_state       |                          |
|                         |   - tracepoint/tcp/tcp_retransmit_skb         |                          |
|                         |   - 64 KB Per-CPU Kernel RingBuffer           |                          |
|                         +-----------------------+-----------------------+                          |
|                                                 |                                                  |
|                                                 v Metrics & Logs                                   |
|                         +-----------------------+-----------------------+                          |
|                         |     Prometheus (:9090) & Grafana (:3000)      |                          |
|                         |   - Real-time PG, Redis, Mongo, Cass, etcd    |                          |
|                         |   - Kernel Sockets & TCP Drop Counters        |                          |
|                         +-----------------------------------------------+                          |
+----------------------------------------------------------------------------------------------------+
```

---

## Supported Database Engines

SidBpfKubers leverages the **Strategy Pattern** (`operator/internal/engines/`) to orchestrate distinct stateful topologies:

| Engine | Version | Default Topology | Protocol Port(s) | Role Model | Native Verification Command |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **PostgreSQL** | `16-alpine` | WAL Streaming Replication | `5432` | `👑 PRIMARY` / `📦 STANDBY` | `SELECT version(), pg_is_in_recovery();` |
| **Redis** | `7.2-alpine` | Asynchronous Master / Replica | `6379` | `👑 MASTER` / `📦 REPLICA` | `redis-cli role` / `INFO replication` |
| **MongoDB** | `7.0` | Replica Set (`rs0`) Quorum | `27017` | `👑 PRIMARY` / `🍃 SECONDARY` | `rs.status().members` |
| **etcd** | `3.5` | Raft Consensus Distributed Quorum | `2379`, `2380` | `👑 LEADER` / `🏛️ FOLLOWER` | `etcdctl endpoint status` |
| **Cassandra**| `4.1` | Peer-to-Peer Gossip & DHT Ring | `9042`, `7000`, `7199` | `⚡ SEED PEER` / `🧬 RING PEER` | `nodetool status` / `system.local` |

---

## Key Features

### 1. Storage Strategy Pattern Reconciler (`operator/internal/engines/`)
* **Unified Interface**: Decoupled engine abstractions handling bootstrapping, configuration generation (`pg_hba.conf`, `mongod.conf`, `cassandra.yaml`), and replica peering.
* **Atomic 3-Tier Virtual IP Routing**:
  * `-rw` (`ClusterIP` routing exclusively to the active Primary / Master / Leader / Seed node).
  * `-ro` (`ClusterIP` load-balancing queries across read-only standby replicas).
  * `-r` (Balanced pool across all healthy cluster replicas).
  * `-headless` (Governing headless service for direct StatefulSet peer-to-peer communication).
* **Automated & Planned Failover**: Annotate `database.sidkubers.io/switchover="true"` to coordinate zero-split-brain leadership transition.
* **Snapshot Backup & Restore**: Initiates physical PVC volume checkpoints and coordinates state reconstruction.

### 2. In-Kernel eBPF Telemetry (`ebpf-agent/`)
* **Kernel Tracepoints Hooked**:
  * `sock/inet_sock_set_state`: Real-time tracking of connection handshakes and lifecycle states (ESTABLISHED, CLOSE_WAIT, TIME_WAIT).
  * `tcp/tcp_retransmit_skb`: Interception of kernel socket buffer retransmissions and drops.
* **Per-CPU RingBuffer**: Asynchronous, zero-heap-allocation event streaming directly from kernel space to userspace Prometheus exporters.

### 3. Mission Control Cockpit (`dashboard/`)
* **Multi-Cluster Switcher**: One-click cockpit switcher across all 5 database engines with honest replica indicators.
* **Glitch-Free Skeleton Loading**: Instant pulsing skeleton placeholders during cluster switches with zero misleading text or leftover port badges.
* **Direct Database Console**: Engine-tailored diagnostic actions (`Direct SQL Test`, `Redis Role Test`, `Mongo RS Test`, `etcd Endpoint Test`, and `nodetool Ring Status`).
* **Embedded Grafana Canvas Block**: Provisioned 6-panel real-time protocol monitor (PostgreSQL, Redis, MongoDB, Cassandra, etcd, HTTP/Nginx) perfectly synchronized with the top Prometheus Live Telemetry Grid.

---

## Directory Structure

```
SidBpfKubers/
├── crd-databasecluster.yaml     # CustomResourceDefinition specification
├── examples/                    # Ready-to-apply database cluster manifests
│   ├── sample-postgres.yaml     # PostgreSQL 16 streaming replication cluster
│   ├── sample-redis.yaml        # Redis 7 Master/Replica cluster
│   ├── sample-mongodb.yaml      # MongoDB 7.0 Replica Set (rs0) cluster
│   ├── sample-etcd.yaml         # etcd 3.5 Raft consensus quorum
│   └── sample-cassandra.yaml    # Apache Cassandra 4.1 peer-to-peer ring
├── operator/                    # Custom Kubernetes Operator in Go
│   ├── api/v1alpha1/            # Go CRD struct definitions & OpenAPI schemas
│   ├── cmd/main.go              # Operator runtime entrypoint
│   ├── deploy/                  # Operator Deployment & RBAC manifests
│   └── internal/
│       ├── controller/          # Reconciliation loop implementation
│       ├── engines/             # Strategy Pattern (Postgres, Redis, Mongo, etcd, Cassandra)
│       └── resources/           # Generalized StatefulSets, Services, ConfigMaps
├── ebpf-agent/                  # In-kernel socket & retransmit tracing agent
│   ├── cmd/                     # Agent entrypoint
│   ├── deploy/                  # DaemonSet & privileged security context
│   └── main.go                  # RingBuffer reader & Prometheus metrics exporter
├── monitoring/                  # Observability infrastructure
│   ├── prometheus.yaml          # Prometheus server configuration & scrape specs
│   ├── grafana.yaml             # Grafana server with auto-provisioned dashboards
│   └── ebpf-service.yaml        # Headless metrics endpoints
└── dashboard/                   # Mission Control Cockpit
    ├── server.go                # Go backend API, K8s exec bridge & port-forward manager
    ├── index.html               # Main dashboard shell
    ├── components/              # Modular UI components
    │   ├── header.html          # Global status & telemetry headers
    │   ├── subbar.html          # Active cluster strip & latency display
    │   ├── sidebar.html         # Spec breakdown & topology tree
    │   ├── layer-k8s.html       # Cluster Topography & Atomic Routing cards
    │   ├── layer-kernel.html    # eBPF Kernel Waterfall Stream
    │   ├── layer-grafana.html   # Embedded Grafana Canvas & Prometheus grid
    │   └── terminal-drawer.html # Interactive slide-over console drawer
    └── static/                  # Vanilla JS orchestrators & CSS styles
        ├── css/style.css
        └── js/                  # app.js, api.js, terminal.js, ebpf-stream.js
```

---

## Get Started

### Prerequisites

* **Local Kubernetes Cluster**: Docker Desktop with Kubernetes enabled (recommended: 4+ CPUs, 6+ GB RAM) or Minikube/Kind.
* **Go**: v1.22+ (to run the cockpit server or compile operator binaries).
* **kubectl**: Configured to connect to your local Kubernetes cluster.

---

### Deployment Steps

#### Step 1: Register CRD and Deploy the Database Operator
```bash
# 1. Register CustomResourceDefinition
kubectl apply -f crd-databasecluster.yaml

# 2. Deploy Operator, RBAC policies, and namespace
kubectl apply -f operator/deploy/operator.yaml
```

#### Step 2: Provision Database Clusters (from `examples/`)
Apply any or all of the production-grade sample manifests:

```bash
# Provision PostgreSQL 16 WAL Cluster
kubectl apply -f examples/sample-postgres.yaml

# Provision Redis 7 Master/Replica Cluster
kubectl apply -f examples/sample-redis.yaml

# Provision MongoDB 7.0 Replica Set
kubectl apply -f examples/sample-mongodb.yaml

# Provision etcd 3.5 Raft Quorum
kubectl apply -f examples/sample-etcd.yaml

# Provision Apache Cassandra 4.1 Gossip Ring
kubectl apply -f examples/sample-cassandra.yaml
```

Verify that the operator and database pods are running:
```bash
kubectl get databaseclusters -n demo-app
kubectl get pods -n demo-app -o wide
```

#### Step 3: Deploy In-Kernel eBPF Agent & Monitoring Stack
```bash
# Launch eBPF tracing DaemonSet
kubectl apply -f ebpf-agent/deploy/daemonset.yaml

# Deploy Prometheus and Grafana
kubectl apply -f monitoring/prometheus.yaml
kubectl apply -f monitoring/grafana.yaml
```

#### Step 4: Launch Mission Control Cockpit
The cockpit server auto-discovers your clusters, manages background port-forwards to Grafana (`:3000`) and Prometheus (`:9091`), and serves the interactive UI:

```bash
cd dashboard
go run server.go
```

Open your browser to:
👉 **[http://localhost:8085](http://localhost:8085)**

---

## Operator Walkthrough

### 1. Zero-Downtime Planned Failover
Gracefully step down the primary and promote a standby replica without split-brain:
```bash
kubectl annotate db sample-postgres database.sidkubers.io/switchover="true" --overwrite -n demo-app
```

### 2. Physical PVC Backup Checkpoint
Initiate a physical volume checkpoint and snapshot job:
```bash
kubectl annotate db sample-postgres database.sidkubers.io/trigger-backup="$(date +%s)" --overwrite -n demo-app
```

### 3. Cassandra Gossip Ring Verification
Query the live distributed hash ring topology:
```bash
kubectl exec -i sample-cassandra-0 -c cassandra -n demo-app -- nodetool -h 127.0.0.1 -p 7199 status
```

### 4. Inject Network Chaos & TCP Drops
Simulate network congestion or TCP retransmission shocks across replicas:
```bash
kubectl exec -it sample-postgres-0 -n demo-app -- /bin/sh -c "kill -STOP 1; sleep 2; kill -CONT 1"
```
Observe the real-time drop spike in the **eBPF Retransmits** sparkline and the **TCP Retransmissions / Drops** panel in Grafana.

---

## Tech Stack

| Component | Technology | Role |
| :--- | :--- | :--- |
| **Operator Runtime** | Go / `controller-runtime` | Declarative State Reconciliation |
| **Database Engines** | Postgres 16, Redis 7, Mongo 7, etcd 3.5, Cassandra 4.1 | Multi-Topology Distributed Fleet |
| **Kernel Tracing** | eBPF / Tracepoints / RingBuffer | `inet_sock_set_state`, `tcp_retransmit_skb` |
| **Metrics Pipeline** | Prometheus / PromQL | Scrape Engine & Time-Series Store |
| **Visualization** | Grafana 10 | Auto-Provisioned Dashboards & Panel Analytics |
| **Cockpit Engine** | Go `net/http` + Vanilla JS | Ultra-Lightweight DevOps Mission Control |
| **UI Styling** | Tailwind CSS (Deep Forest / Bottle Green) | High-Density Cyber-Industrial Console |

---

## License

Distributed under the Apache 2.0 License. See `LICENSE` for more information.
