# SidBpfKubers

<div align="center">

[![Go Report Card](https://goreportcard.com/badge/github.com/SidBpfKubers)](https://golang.org)
[![Kubernetes](https://img.shields.io/badge/kubernetes-v1.28%2B-blue.svg?logo=kubernetes)](https://kubernetes.io/)
[![eBPF](https://img.shields.io/badge/eBPF-In--Kernel-purple.svg?logo=linux)](https://ebpf.io/)
[![Prometheus](https://img.shields.io/badge/Prometheus-Metrics-orange.svg?logo=prometheus)](https://prometheus.io/)
[![Grafana](https://img.shields.io/badge/Grafana-Visualization-F46800.svg?logo=grafana)](https://grafana.com/)
[![License](https://img.shields.io/badge/license-Apache--2.0-green.svg)](LICENSE)

**A Cloud-Native Kubernetes Operator for Database Lifecycle Orchestration paired with In-Kernel eBPF Telemetry, Prometheus Metrics, and Grafana Visualization.**

[Architecture](#architecture) • [Key Features](#key-features) • [Directory Structure](#directory-structure) • [Quickstart](#quickstart) • [Operator Walkthrough](#operator-walkthrough)

</div>

---

## Overview

Traditional Kubernetes operators manage stateful workloads by reacting to high-level container states (cgroups, readiness probes, exit codes). However, they remain blind to sub-millisecond network degradation, streaming replication packet lag, and in-kernel TCP drops.

**SidBpfKubers** bridges this gap by marrying a **declarative Kubernetes database reconciler** with **low-overhead eBPF in-kernel telemetry**:
* **Automated Lifecycle Management**: Handles multi-replica PostgreSQL topologies, synchronous WAL streaming, automated zero-downtime primary switchovers, and physical PVC snapshot backups/restores.
* **Deep In-Kernel Observability**: Attaches eBPF probes directly into the Linux kernel network subsystem (`sock/inet_sock_set_state` and `tcp/tcp_retransmit_skb`) to capture real-time socket transitions and packet drop anomalies.
* **Mission Control Cockpit**: An enterprise web cockpit built with an archival Bottle Green / Deep Forest aesthetic, featuring real-time cluster topography, an inline SQL execution engine, eBPF telemetry waterfall, and embedded Grafana metrics.

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
|   |  DatabaseCluster (PG-16) |                        |  - Primary/Standby StatefulSets        |   |
|   +--------------------------+                        |  - 3-Tier Services (-rw, -ro, -r)      |   |
|                 |                                     |  - Zero-Downtime Switchover Engine     |   |
|                 v                                     |  - Snapshot Backup & Restore Jobs      |   |
|   +--------------------------+                        +----------------------------------------+   |
|   |   PostgreSQL Replicas    |                                                                     |
|   |  sample-postgres-0 (RW)  |                                                                     |
|   |  sample-postgres-1 (RO)  | <=================+                                                 |
|   |  sample-postgres-2 (RO)  |                   | Intercepts TCP Sockets & Packets                |
|   +--------------------------+                   |                                                 |
|                                                  v                                                 |
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
|                         |   - Real-time PG Conns & WAL Sync Lag         |                          |
|                         |   - Kernel Sockets & TCP Drop Counters        |                          |
|                         +-----------------------------------------------+                          |
+----------------------------------------------------------------------------------------------------+
```

---

## Key Features

### 1. Declarative Database Reconciler (`operator/`)
* **Atomic 3-Tier Virtual IP Routing**: Automatically provisions and manages dedicated Kubernetes services:
  * `-rw` (`ClusterIP` pointing exclusively to the active Primary node).
  * `-ro` (`ClusterIP` load-balancing queries across read-only standby replicas).
  * `-r` (Read pool across all ready cluster pods).
  * `-headless` (Governing service for direct StatefulSet peer-to-peer WAL streaming).
* **Zero-Downtime Planned Switchover**: Reconciles annotations (`database.sidkubers.io/switchover="true"`) to coordinate graceful primary step-downs, promote the standby with the lowest replication lag, and redirect write traffic with zero split-brain risk.
* **PVC Snapshot & Restore**: Executes coordinated `pg_dump` physical checkpointing to persistent backup volumes and coordinates replica reconstruction.

### 2. In-Kernel eBPF Observability (`ebpf-agent/`)
* **Tracepoints Attached**:
  * `sock/inet_sock_set_state`: Tracks connection handshakes, state machine evolutions (ESTABLISHED, CLOSE_WAIT, TIME_WAIT), and socket lifetimes.
  * `tcp/tcp_retransmit_skb`: Intercepts socket buffer drops and network congestion events at the kernel driver layer.
* **Per-CPU RingBuffer**: Asynchronous, high-throughput kernel-to-userspace event streaming with zero heap allocations on the hot path.

### 3. Unified Mission Control Cockpit (`dashboard/`)
* **Deep Forest / Bottle Green Design System**: Archival, high-contrast palette (`#0D1B1A`, `#112220`, `#1F3834`) tailored for long operating hours without harsh glare.
* **Direct SQL Execution Engine**: Run queries (`SELECT`, transactional workloads) directly against primary or standby replicas with sub-15ms live results rendering.
* **HUD Action Feedback**: Floating tactile notification toasts displaying command strings, live exit status, and execution duration.
* **Live eBPF Waterfall Stream**: Real-time streaming table of active kernel socket events categorized by protocol (`PGSQL`, `OPERATOR`, `GRAFANA`).

---

## Directory Structure

```
SidBpfKubers/
├── crd-databasecluster.yaml     # CustomResourceDefinition specification
├── sample-databasecluster.yaml  # Sample 3-node PostgreSQL 16 cluster manifest
├── operator/                    # Custom Kubernetes Operator in Go
│   ├── api/v1alpha1/            # Go CRD struct definitions & OpenAPI schemas
│   ├── cmd/main.go              # Operator runtime entrypoint
│   ├── deploy/                  # Operator Deployment & RBAC manifests
│   └── internal/
│       ├── controller/          # Reconciliation loop implementation
│       └── resources/           # StatefulSets, Services, ConfigMaps, Backups
├── ebpf-agent/                  # In-kernel socket & retransmit tracing agent
│   ├── cmd/                     # Agent entrypoint
│   ├── deploy/                  # DaemonSet & privileged security context
│   └── main.go                  # RingBuffer reader & telemetry exporter
├── monitoring/                  # Observability infrastructure
│   ├── prometheus.yaml          # Prometheus server configuration & scrape specs
│   ├── grafana.yaml             # Grafana server with auto-provisioned dashboards
│   └── ebpf-service.yaml        # Headless metrics endpoints
└── dashboard/                   # Mission Control Cockpit
    ├── server.go                # Go backend API, K8s exec bridge & asset server
    ├── index.html               # Main dashboard shell
    ├── components/              # Modular UI components (Navbar, Sidebar, Layers)
    │   ├── header.html
    │   ├── subbar.html
    │   ├── sidebar.html
    │   ├── layer-k8s.html       # Cluster Topography & Direct SQL Engine
    │   ├── layer-kernel.html    # eBPF Kernel Waterfall Stream
    │   ├── layer-grafana.html   # Embedded Grafana & Prometheus sparklines
    │   └── terminal-drawer.html # Slide-over interactive CLI console
    └── static/                  # Vanilla JS orchestrators & CSS styles
        ├── css/style.css
        └── js/                  # app.js, api.js, terminal.js, ebpf-stream.js
```

---

## Quickstart

### Prerequisites
* **Kubernetes Cluster**: Docker Desktop, Minikube, or Kind (v1.28+)
* **Go**: v1.22+
* **kubectl**: Configured with current cluster context

### 1. Apply CRD and Operator
```bash
# Register DatabaseCluster CRD
kubectl apply -f crd-databasecluster.yaml

# Deploy Operator, RBAC, and Namespace
kubectl apply -f operator/deploy/operator.yaml

# Provision PostgreSQL Cluster
kubectl apply -f sample-databasecluster.yaml
```

### 2. Deploy eBPF DaemonSet & Monitoring Stack
```bash
# Launch eBPF tracing agent as a DaemonSet
kubectl apply -f ebpf-agent/deploy/daemonset.yaml

# Deploy Prometheus and Grafana
kubectl apply -f monitoring/prometheus.yaml
kubectl apply -f monitoring/grafana.yaml
```

### 3. Launch Mission Control Cockpit
```bash
cd dashboard

# Compile and start the engine cockpit
go run server.go
```
Open **[http://localhost:8085](http://localhost:8085)** in your browser.

---

## Operator Walkthrough

### Trigger Zero-Downtime Switchover
Promote a standby replica to primary atomically:
```bash
kubectl annotate db sample-postgres database.sidkubers.io/switchover="true" --overwrite -n demo-app
```

### Trigger Snapshot Backup Job
Initiate a physical PVC backup checkpoint:
```bash
kubectl annotate db sample-postgres database.sidkubers.io/trigger-backup="$(date +%s)" --overwrite -n demo-app
```

### Inject TCP Connection Shock (eBPF Trip)
Simulate network congestion or packet drop chaos across standby replicas:
```bash
# The eBPF agent intercepts and tags tcp_retransmit_skb tracepoints in real-time
kubectl exec -it sample-postgres-0 -n demo-app -- /bin/sh -c "kill -STOP 1; sleep 2; kill -CONT 1"
```

---

## Tech Stack

| Component | Technology | Role |
| :--- | :--- | :--- |
| **Operator Runtime** | Go / `controller-runtime` | Declarative State Reconciliation |
| **Database Engine** | PostgreSQL 16 (Alpine) | WAL Streaming Primary-Replica Topology |
| **Kernel Tracing** | eBPF / Tracepoints | `inet_sock_set_state`, `tcp_retransmit_skb` |
| **Metrics Pipeline** | Prometheus / PromQL | Scrape Engine & Time-Series Store |
| **Visualization** | Grafana 10 | Provisioned Dashboards & Panel Analytics |
| **Cockpit Engine** | Go `net/http` + Vanilla JS | Ultra-Lightweight DevOps Mission Control |
| **UI Styling** | Tailwind CSS (Deep Forest) | High-Density Cyber-Industrial Console |

---

## License

Distributed under the Apache 2.0 License. See `LICENSE` for more information.
