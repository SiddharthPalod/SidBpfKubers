//go:build ignore

#include <linux/bpf.h>
#include <linux/types.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

char LICENSE[] SEC("license") = "Dual MIT/GPL";

#ifndef BPF_MAP_TYPE_RINGBUF
#define BPF_MAP_TYPE_RINGBUF 27
#endif

#define BPF_TCP_ESTABLISHED 1
#define BPF_TCP_SYN_SENT    2
#define BPF_TCP_CLOSE       7

#define EVENT_CONNECT    1
#define EVENT_RETRANSMIT 2
#define EVENT_CLOSE      3

// TCP Event structure emitted to user-space Go
struct tcp_event {
    __u32 saddr;
    __u32 daddr;
    __u16 sport;
    __u16 dport;
    __u8  event_type;
    __u8  pad[3];
    __u64 timestamp_ns;
};
const struct tcp_event *unused_event __attribute__((unused));

// RingBuffer map shared with Go
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 24); // 16 MB buffer
} events SEC(".maps");

// Tracepoint: inet_sock_set_state
struct trace_event_raw_inet_sock_set_state {
    __u64 unused;
    const void *skaddr;
    int oldstate;
    int newstate;
    __u16 sport;
    __u16 dport;
    __u16 family;
    __u8 protocol;
    __u8 saddr[4];
    __u8 daddr[4];
};

SEC("tracepoint/sock/inet_sock_set_state")
int trace_inet_sock_set_state(struct trace_event_raw_inet_sock_set_state *ctx) {
    if (ctx->family != 2) {
        return 0;
    }

    __u8 event_type = 0;
    if (ctx->oldstate == BPF_TCP_SYN_SENT && ctx->newstate == BPF_TCP_ESTABLISHED) {
        event_type = EVENT_CONNECT;
    } else if (ctx->newstate == BPF_TCP_CLOSE) {
        event_type = EVENT_CLOSE;
    }

    if (event_type == 0) {
        return 0;
    }

    struct tcp_event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
    if (!e) {
        return 0;
    }

    e->saddr = *(__u32 *)ctx->saddr;
    e->daddr = *(__u32 *)ctx->daddr;
    e->sport = ctx->sport;
    e->dport = ctx->dport;
    e->event_type = event_type;
    e->timestamp_ns = bpf_ktime_get_ns();

    bpf_ringbuf_submit(e, 0);
    return 0;
}

// Tracepoint: tcp_retransmit_skb (Packet drops & retransmissions)
struct trace_event_raw_tcp_retransmit {
    __u64 unused;
    const void *skaddr;
    const void *skbaddr;
    int state;
    __u16 sport;
    __u16 dport;
    __u16 family;
    __u8 saddr[4];
    __u8 daddr[4];
};

SEC("tracepoint/tcp/tcp_retransmit_skb")
int trace_tcp_retransmit(struct trace_event_raw_tcp_retransmit *ctx) {
    if (ctx->family != 2) {
        return 0;
    }

    struct tcp_event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
    if (!e) {
        return 0;
    }

    e->saddr = *(__u32 *)ctx->saddr;
    e->daddr = *(__u32 *)ctx->daddr;
    e->sport = ctx->sport;
    e->dport = ctx->dport;
    e->event_type = EVENT_RETRANSMIT;
    e->timestamp_ns = bpf_ktime_get_ns();

    bpf_ringbuf_submit(e, 0);
    return 0;
}
