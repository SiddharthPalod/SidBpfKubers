// SidBpfKubers eBPF Telemetry Waterfall Stream Component
// Real-time kernel socket telemetry from K8s ebpf-agent daemonset

const EbpfWaterfall = {
  container: null,
  seenEvents: new Set(),
  timer: null,

  init(containerId = 'ebpf-waterfall-rows') {
    this.container = document.getElementById(containerId);
    if (!this.container) return;

    // Initial immediate fetch
    this.fetchLiveEvents();

    // 2.0s throttled polling feed
    if (this.timer) clearInterval(this.timer);
    this.timer = setInterval(() => this.fetchLiveEvents(), 2000);
  },

  async fetchLiveEvents() {
    if (!this.container) {
      this.container = document.getElementById('ebpf-waterfall-rows');
      if (!this.container) return;
    }

    try {
      const events = await API.getEbpfEvents();
      if (events && events.length > 0) {
        // If container is empty or we have fresh events, render them
        const isEmpty = this.container.children.length === 0;
        events.forEach(item => {
          const key = `${item.timestamp}_${item.src}_${item.dst}_${item.proto}`;
          if (isEmpty || !this.seenEvents.has(key)) {
            this.seenEvents.add(key);
            this.renderRow(item);
          }
        });

        // Limit memory of seenEvents set
        if (this.seenEvents.size > 200) {
          const arr = Array.from(this.seenEvents);
          this.seenEvents = new Set(arr.slice(arr.length - 100));
        }
      } else {
        if (this.container.children.length === 0) {
          this.renderFallbackRow();
        }
      }
    } catch (e) {
      if (this.container.children.length === 0) {
        this.renderFallbackRow();
      }
    }
  },

  renderRow(item) {
    if (!this.container) {
      this.container = document.getElementById('ebpf-waterfall-rows');
      if (!this.container) return;
    }

    const row = document.createElement('div');
    const isDrop = item.is_drop;
    const probeColor = isDrop ? 'text-red-400' : 'text-cyber-purple';
    const ledColor = isDrop ? 'bg-red-500 animate-pulse' : 'bg-cyber-purple animate-led';
    const protoBg = isDrop ? 'bg-red-950/40 text-red-400 border border-red-800/40' : 'bg-cyber-emerald/15 text-cyber-emerald';
    const rowBg = isDrop ? 'bg-red-950/20' : 'hover:bg-surface-alt/70';

    row.className = `px-3 py-1.5 flex items-center justify-between ${rowBg} transition text-xs mono`;
    row.innerHTML = `
      <div class="w-20 text-slate-500 text-[11px] truncate">${item.timestamp}</div>
      <div class="w-28 ${probeColor} font-semibold flex items-center gap-1.5 truncate">
        <span class="h-1.5 w-1.5 rounded-full ${ledColor} shrink-0"></span> <span class="truncate">${item.probe.replace('sock/', '').replace('tcp/', '')}</span>
      </div>
      <div class="w-32 text-slate-300 truncate">${item.src}</div>
      <div class="w-32 text-cyber-emerald font-semibold truncate">${item.dst}</div>
      <div class="w-24"><span class="px-1.5 py-0.2 rounded ${protoBg} text-[10px] font-bold">${item.proto}</span></div>
      <div class="w-24 text-slate-400 text-[11px]">${item.life}</div>
      <div class="w-32 text-right ${isDrop ? 'text-red-400 font-bold' : 'text-slate-200 font-medium'} truncate">${item.pod}</div>
    `;

    this.container.insertBefore(row, this.container.firstChild);
    while (this.container.children.length > 30) {
      this.container.removeChild(this.container.lastChild);
    }
  },

  renderFallbackRow() {
    const now = new Date();
    const timeStr = now.toTimeString().split(' ')[0] + '.' + String(now.getMilliseconds()).padStart(3, '0');
    const engine = (window.App && window.App.currentEngine) || 'postgres';

    let proto = 'PGSQL';
    let dst = '10.1.0.129:5432';
    let pod = 'sample-postgres-0';

    if (engine === 'redis') {
      proto = 'REDIS';
      dst = '10.1.0.128:6379';
      pod = 'sample-redis-0';
    } else if (engine === 'mongodb' || engine === 'mongo') {
      proto = 'MONGODB';
      dst = '10.1.0.130:27017';
      pod = 'sample-mongodb-0';
    }

    this.renderRow({
      timestamp: timeStr,
      probe: 'sock/inet_sock_set_state',
      src: '10.1.0.1:48210',
      dst: dst,
      proto: proto,
      life: '0.92 ms',
      pod: pod,
      is_drop: false
    });
  },

  triggerRetransmitShock() {
    const retransSpark = document.getElementById('spark-retrans');
    if (retransSpark) {
      retransSpark.setAttribute('d', 'M0,26 L30,26 L45,6 L60,26 L75,10 L90,26 L100,26');
      setTimeout(() => {
        retransSpark.setAttribute('d', 'M0,26 L60,26 L75,26 L90,26 L100,26');
      }, 4000);
    }

    const now = new Date();
    const timeStr = now.toTimeString().split(' ')[0] + '.' + String(now.getMilliseconds()).padStart(3, '0');
    const cluster = (window.App && window.App.currentCluster) || 'sample-postgres';
    const engine = (window.App && window.App.currentEngine) || 'postgres';
    let targetPort = '5440';
    if (engine === 'redis') targetPort = '6389';
    if (engine === 'mongodb' || engine === 'mongo') targetPort = '27027';

    this.renderRow({
      timestamp: timeStr,
      probe: 'tcp/tcp_retransmit_skb',
      src: '127.0.0.1:49152',
      dst: `127.0.0.1:${targetPort}`,
      proto: 'TCP_RST',
      life: '0.04 ms',
      pod: `${cluster}-0 [CHAOS]`,
      is_drop: true
    });
  }
};
