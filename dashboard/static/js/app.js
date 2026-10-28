// SidKubers Dashboard Core Controller & UI Orchestrator

const App = {
  activeLayer: 'k8s',

  async init() {
    // 1. Asynchronously load all isolated HTML components first
    try {
      await ComponentLoader.loadAll();
    } catch (e) {
      console.error('Failed to load components:', e);
    }

    // 2. Initialize subcomponents and clock
    try {
      this.initClock();
    } catch (e) {
      console.error('Failed to init clock:', e);
    }

    try {
      this.setLayer('k8s');
    } catch (e) {
      console.error('Failed to set layer:', e);
    }

    try {
      EbpfWaterfall.init('ebpf-waterfall-rows');
    } catch (e) {
      console.error('Failed to init EbpfWaterfall:', e);
    }

    // 3. Trigger initial status poll
    try {
      await this.triggerSync();
      setInterval(() => this.triggerSync(), 2000);
    } catch (e) {
      console.error('Failed to start status poll:', e);
    }
  },

  initClock() {
    const update = () => {
      const clockEl = document.getElementById('clock');
      if (clockEl) {
        const now = new Date();
        clockEl.textContent = now.toISOString().substring(11, 19) + ' UTC';
      }
    };
    update();
    setInterval(update, 1000);
  },

  setLayer(layer) {
    this.activeLayer = layer;
    const k8sView = document.getElementById('component-layer-k8s');
    const ebpfView = document.getElementById('component-layer-kernel');
    const grafanaView = document.getElementById('component-layer-grafana');

    const btnK8s = document.getElementById('toggle-layer-k8s');
    const btnEbpf = document.getElementById('toggle-layer-ebpf');
    const btnGrafana = document.getElementById('toggle-layer-grafana');

    const animateView = (view) => {
      if (!view) return;
      view.classList.remove('animate-layer-in');
      void view.offsetWidth; // trigger reflow for smooth animation
      view.classList.add('animate-layer-in');
    };

    if (layer === 'k8s') {
      if (k8sView) { k8sView.classList.remove('hidden'); animateView(k8sView); }
      if (ebpfView) ebpfView.classList.add('hidden');
      if (grafanaView) grafanaView.classList.add('hidden');

      if (btnK8s) btnK8s.className = 'px-2.5 py-1 rounded text-[11px] font-bold bg-surface text-white border-sharp shadow-sm';
      if (btnEbpf) btnEbpf.className = 'px-2.5 py-1 rounded text-[11px] font-medium text-slate-400 hover:text-white transition flex items-center gap-1';
      if (btnGrafana) btnGrafana.className = 'px-2.5 py-1 rounded text-[11px] font-medium text-slate-400 hover:text-white transition';
    } else if (layer === 'ebpf') {
      if (k8sView) k8sView.classList.add('hidden');
      if (ebpfView) { ebpfView.classList.remove('hidden'); animateView(ebpfView); }
      if (grafanaView) grafanaView.classList.add('hidden');

      if (btnEbpf) btnEbpf.className = 'px-2.5 py-1 rounded text-[11px] font-bold bg-cyber-purple/20 text-cyber-purple border border-cyber-purple/40 shadow-sm flex items-center gap-1';
      if (btnK8s) btnK8s.className = 'px-2.5 py-1 rounded text-[11px] font-medium text-slate-400 hover:text-white transition';
      if (btnGrafana) btnGrafana.className = 'px-2.5 py-1 rounded text-[11px] font-medium text-slate-400 hover:text-white transition';

      try {
        EbpfWaterfall.fetchLiveEvents();
      } catch (e) {}
    } else if (layer === 'grafana') {
      if (k8sView) k8sView.classList.add('hidden');
      if (ebpfView) ebpfView.classList.add('hidden');
      if (grafanaView) { grafanaView.classList.remove('hidden'); animateView(grafanaView); }

      if (btnGrafana) btnGrafana.className = 'px-2.5 py-1 rounded text-[11px] font-bold bg-surface text-white border-sharp shadow-sm';
      if (btnK8s) btnK8s.className = 'px-2.5 py-1 rounded text-[11px] font-medium text-slate-400 hover:text-white transition';
      if (btnEbpf) btnEbpf.className = 'px-2.5 py-1 rounded text-[11px] font-medium text-slate-400 hover:text-white transition flex items-center gap-1';
    }
  },

  showToast(command, output, durationMs, success) {
    const container = document.getElementById('hud-toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    const borderClass = success ? 'border-cyber-mint/50 shadow-[0_4px_24px_rgba(0,245,160,0.18)]' : 'border-red-500/50 shadow-[0_4px_24px_rgba(239,68,68,0.18)]';
    const badgeBg = success ? 'bg-cyber-mint/15 text-cyber-mint border border-cyber-mint/30' : 'bg-red-500/15 text-red-400 border border-red-500/30';
    const statusText = success ? `✔ SUCCESS · ${durationMs}ms` : `✖ ALERT · ${durationMs}ms`;

    toast.className = `border-sharp bg-[#112220]/95 backdrop-blur-md rounded p-3 text-xs mono space-y-1.5 animate-toast pointer-events-auto transition-all ${borderClass}`;
    
    let snippet = (output || '').trim().split('\n').slice(0, 2).join('\n');
    if (!snippet) snippet = success ? 'Execution finished with return code 0.' : 'Command error.';

    toast.innerHTML = `
      <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-1.5 truncate">
          <span class="h-1.5 w-1.5 rounded-full ${success ? 'bg-cyber-mint animate-led' : 'bg-red-500'}"></span>
          <span class="text-[10px] font-bold uppercase tracking-wider px-1.5 py-0.2 rounded ${badgeBg}">${statusText}</span>
        </div>
        <div class="flex items-center gap-2">
          <button onclick="toggleTerminalDrawer()" class="text-[10px] text-slate-400 hover:text-white px-1.5 py-0.5 rounded bg-surface border-sharp transition">Open CLI</button>
          <button onclick="this.closest('.animate-toast').remove()" class="text-slate-400 hover:text-white text-xs px-1">✕</button>
        </div>
      </div>
      <div class="text-[11px] text-cyber-emerald truncate font-mono">$ ${command}</div>
      <div class="text-[10px] text-slate-300 bg-obsidian/80 p-1.5 rounded border border-surface-border truncate font-mono">${snippet}</div>
    `;

    container.appendChild(toast);

    while (container.children.length > 3) {
      container.removeChild(container.firstChild);
    }

    setTimeout(() => {
      if (toast.parentNode) {
        toast.style.opacity = '0';
        toast.style.transform = 'translateY(10px)';
        setTimeout(() => toast.remove(), 250);
      }
    }, 4500);
  },

  displaySQLResults(targetName, command, output, durationMs, success) {
    const panel = document.getElementById('quick-sql-results-panel');
    const targetLabel = document.getElementById('quick-sql-target-label');
    const durationEl = document.getElementById('quick-sql-duration');
    const timeEl = document.getElementById('quick-sql-timestamp');
    const outPre = document.getElementById('quick-sql-output');
    const led = document.getElementById('quick-sql-status-led');

    if (!panel || !outPre) return;

    panel.classList.remove('hidden');
    panel.classList.remove('action-flash-emerald', 'action-flash-blue', 'action-flash-success');
    void panel.offsetWidth;
    panel.classList.add(success ? 'action-flash-success' : 'action-flash-emerald');

    if (targetLabel) targetLabel.textContent = targetName;
    if (durationEl) durationEl.textContent = `${durationMs}ms`;
    if (timeEl) {
      const d = new Date();
      timeEl.textContent = d.toTimeString().split(' ')[0];
    }
    if (led) {
      led.className = `h-2 w-2 rounded-full ${success ? 'bg-cyber-mint animate-led' : 'bg-red-500'}`;
    }

    outPre.textContent = output ? output.trim() : (success ? '✔ (Query executed with 0 rows returned)' : '✖ Query execution failed');
  },

  clearSQLResults() {
    const panel = document.getElementById('quick-sql-results-panel');
    if (panel) panel.classList.add('hidden');
  },

  async runAction(action, params = {}) {
    const spinner = document.getElementById('sync-spinner');
    if (spinner) spinner.classList.add('animate-spin');

    try {
      const d = await API.runAction(action, params);
      Terminal.appendLog(d.command, d.output, d.duration_ms, d.success);

      // Instant HUD status feedback toast directly on screen
      this.showToast(d.command, d.output, d.duration_ms, d.success);

      // Direct inline SQL output updates
      if (action === 'query_primary') {
        this.displaySQLResults('sample-postgres-rw (👑 Primary RW)', d.command, d.output, d.duration_ms, d.success);
      } else if (action === 'query_replica') {
        this.displaySQLResults('sample-postgres-ro (📦 Standby RO)', d.command, d.output, d.duration_ms, d.success);
      } else if (action === 'test_pod') {
        const podName = params.pod || 'Pod Test';
        this.displaySQLResults(`${podName} (Direct SQL)`, d.command, d.output, d.duration_ms, d.success);
      }

      this.triggerSync();
      return d;
    } catch (err) {
      Terminal.appendLog(action, err.message, 0, false);
      this.showToast(action, err.message, 0, false);
    } finally {
      if (spinner) spinner.classList.remove('animate-spin');
    }
  },

  async triggerSync() {
    try {
      const data = await API.getStatus();
      this.renderStatus(data);
    } catch (e) {
      // Offline fallback state
    }
  },

  renderStatus(data) {
    if (!data) return;

    // 1. Header Telemetry Badges
    const opStatus = document.getElementById('header-operator-status');
    if (opStatus && data.operator_status) {
      opStatus.innerHTML = `<span class="h-1.5 w-1.5 rounded-full bg-cyber-mint"></span> ${data.operator_status}`;
    }

    const crdCount = document.getElementById('header-crds-count');
    if (crdCount) {
      crdCount.textContent = data.crd_count || 1;
    }

    const ebpfProbes = document.getElementById('header-ebpf-probes');
    if (ebpfProbes) {
      ebpfProbes.innerHTML = `<span class="h-1.5 w-1.5 rounded-full bg-cyber-purple animate-led"></span> ${data.ebpf_agent_status || '1/1'}`;
    }

    const headerRetrans = document.getElementById('header-retransmits');
    if (headerRetrans && data.metrics) {
      headerRetrans.textContent = data.metrics.tcp_retransmits || 0;
    }

    // 2. Sub-Bar Strip
    const subState = document.getElementById('sub-cluster-state');
    if (subState) {
      subState.innerHTML = `<span class="h-1.5 w-1.5 rounded-full bg-cyber-mint"></span> ${data.cluster_condition || 'SYNCED'}`;
    }

    const subPrimary = document.getElementById('sub-primary-node');
    if (subPrimary && data.primary_pod) {
      subPrimary.textContent = data.primary_pod;
    }

    const subSockets = document.getElementById('sub-socket-count');
    if (subSockets && data.metrics) {
      subSockets.textContent = data.metrics.tcp_connections ? data.metrics.tcp_connections.toLocaleString() : '7,415';
    }

    const subLatency = document.getElementById('sub-cluster-latency');
    if (subLatency && data.metrics) {
      const lat = ((data.metrics.pg_connections % 20) * 0.04 + 0.38).toFixed(2);
      subLatency.textContent = `${lat} ms`;
    }

    // 3. Sidebar CRD Tree & Probes
    const treePgReplicas = document.getElementById('tree-pg-replicas');
    if (treePgReplicas) {
      treePgReplicas.textContent = `${data.pods ? data.pods.length : 3} / ${data.replicas || 3} Ready`;
    }

    const sideOpStatus = document.getElementById('sidebar-operator-status');
    if (sideOpStatus && data.operator_status) {
      sideOpStatus.textContent = `${data.operator_status} HA`;
    }

    const treeEbpf = document.getElementById('tree-ebpf-probes');
    if (treeEbpf) {
      treeEbpf.textContent = `${data.ebpf_agent_status || '1 / 1'} Running`;
    }

    // 4. K8s Service Routing IPs
    if (data.services && data.services.length > 0) {
      data.services.forEach(svc => {
        if (svc.role === 'rw') {
          const el = document.getElementById('svc-rw-ip');
          if (el) el.textContent = `${svc.cluster_ip}:${svc.port}`;
        } else if (svc.role === 'ro') {
          const el = document.getElementById('svc-ro-ip');
          if (el) el.textContent = `${svc.cluster_ip}:${svc.port}`;
        } else if (svc.role === 'r') {
          const el = document.getElementById('svc-r-ip');
          if (el) el.textContent = `${svc.cluster_ip}:${svc.port}`;
        } else if (svc.role === 'headless') {
          const el = document.getElementById('svc-headless-ip');
          if (el) el.textContent = `${svc.cluster_ip}:${svc.port}`;
        }
      });
    }

    // 5. Render Pods & Standby Lists
    if (data.pods && data.pods.length > 0) {
      const treeList = document.getElementById('tree-pod-list');
      const podsContainer = document.getElementById('pods-container');

      let treeHtml = '';
      let cardsHtml = '';
      let standbyPods = [];

      data.pods.forEach(pod => {
        const isPrimary = pod.role === 'primary';
        const badgeClass = isPrimary ? 'bg-cyber-emerald/15 text-cyber-emerald border border-cyber-emerald/30' : 'bg-slate-800 text-slate-300 border border-slate-700';
        const badgeText = isPrimary ? '👑 PRIMARY (RW)' : '📦 STANDBY (RO)';
        const roleDesc = isPrimary ? 'Master (WAL Read/Write)' : 'Streaming WAL Standby';
        if (!isPrimary) standbyPods.push(pod.name);

        treeHtml += `
          <div class="p-1.5 rounded bg-surface border-sharp flex items-center justify-between cursor-pointer hover:border-cyber-emerald/40 transition" onclick="App.testPod('${pod.name}')">
            <div class="flex items-center gap-1.5 truncate">
              <span class="h-1.5 w-1.5 rounded-full ${pod.ready ? 'bg-cyber-mint animate-led' : 'bg-cyber-amber'}"></span>
              <span class="text-white font-medium truncate">${pod.name}</span>
            </div>
            <span class="text-[9px] px-1 rounded ${badgeClass} uppercase font-bold">${isPrimary ? 'PRIMARY' : 'STANDBY'}</span>
          </div>
        `;

        cardsHtml += `
          <div class="border-sharp bg-surface rounded p-3.5 space-y-3 relative hover:border-cyber-emerald/40 transition">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <span class="h-2 w-2 rounded-full ${pod.ready ? 'bg-cyber-mint animate-led' : 'bg-cyber-amber'}"></span>
                <span class="mono font-bold text-white text-xs">${pod.name}</span>
              </div>
              <span class="mono text-[10px] font-bold px-2 py-0.5 rounded ${badgeClass} uppercase">
                ${badgeText}
              </span>
            </div>

            <div class="grid grid-cols-2 gap-2 text-[11px] mono bg-obsidian/70 p-2 rounded border-sharp">
              <div><span class="text-slate-500 text-[10px]">Status:</span> <span class="text-cyber-mint font-semibold">${pod.status}</span></div>
              <div><span class="text-slate-500 text-[10px]">Ready:</span> <span class="text-slate-200">${pod.ready ? '1 / 1' : '0 / 1'}</span></div>
              <div><span class="text-slate-500 text-[10px]">Pod IP:</span> <span class="text-cyber-emerald font-semibold">${pod.ip || '10.1.0.x'}</span></div>
              <div><span class="text-slate-500 text-[10px]">Restarts:</span> <span class="text-slate-200">${pod.restarts}</span></div>
              <div><span class="text-slate-500 text-[10px]">Node:</span> <span class="text-slate-300 truncate">${pod.node || 'k8s-node'}</span></div>
              <div><span class="text-slate-500 text-[10px]">Role:</span> <span class="${isPrimary ? 'text-cyber-emerald font-semibold' : 'text-slate-300'}">${roleDesc}</span></div>
            </div>

            <div class="flex items-center gap-2 pt-1 text-xs mono">
              <button onclick="App.testPod('${pod.name}')" class="flex-1 py-1.5 bg-surface-alt hover:bg-surface-hover border-sharp text-slate-200 hover:text-white rounded font-medium transition flex items-center justify-center gap-1">
                <span>🔍 Direct SQL Test</span>
              </button>
              <button onclick="App.killPod('${pod.name}')" class="py-1.5 px-3 bg-red-950/20 hover:bg-red-900/40 border border-red-800/40 text-red-400 rounded font-medium transition">
                💥 Kill Pod
              </button>
            </div>
          </div>
        `;
      });

      if (treeList) treeList.innerHTML = treeHtml;
      if (podsContainer) podsContainer.innerHTML = cardsHtml;

      const svcRw = document.getElementById('svc-rw-target');
      if (svcRw && data.primary_pod) {
        svcRw.textContent = `${data.primary_pod} (👑 Primary)`;
      }

      const svcRo = document.getElementById('svc-ro-target');
      if (svcRo) {
        svcRo.textContent = `${standbyPods.join(', ') || 'Standby'} (📦 Standby)`;
      }

      const svcR = document.getElementById('svc-r-target');
      if (svcR) {
        svcR.textContent = `All ${data.pods.length} Replicas (Load Balanced)`;
      }
    }

    // 6. Prometheus Live Telemetry Grid
    if (data.metrics) {
      const mPg = document.getElementById('metric-pg-conns');
      if (mPg) mPg.textContent = data.metrics.pg_connections ? data.metrics.pg_connections.toLocaleString() : '2,520';

      const mSock = document.getElementById('metric-total-sockets');
      if (mSock) mSock.textContent = data.metrics.tcp_connections ? data.metrics.tcp_connections.toLocaleString() : '9,245';

      const mRetrans = document.getElementById('metric-retransmits-count');
      if (mRetrans) mRetrans.textContent = data.metrics.tcp_retransmits ? data.metrics.tcp_retransmits.toLocaleString() : '715';
    }

    // 7. Grafana Indicator
    const grafanaDot = document.getElementById('grafana-indicator');
    if (grafanaDot && data.grafana_live) {
      grafanaDot.className = 'flex items-center gap-1 text-[10px] mono px-2 py-0.2 rounded border border-cyber-mint/30 bg-cyber-mint/10 text-cyber-mint font-bold';
    }
  },

  // High-Level Operator Triggers
  switchover() {
    Terminal.appendLog('switchover', 'Triggering planned zero-downtime primary failover...', 0, true);
    return this.runAction('switchover');
  },

  backup() {
    return this.runAction('trigger_backup');
  },

  restore() {
    return this.runAction('trigger_restore');
  },

  networkChaos() {
    EbpfWaterfall.triggerRetransmitShock();
    return this.runAction('simulate_packet_drop');
  },

  killPod(podName) {
    return this.runAction('kill_primary', { pod: podName });
  },

  testPod(podName) {
    return this.runAction('test_pod', { pod: podName });
  },

  executeRawCommand() {
    const input = document.getElementById('terminal-input');
    const cmd = input ? input.value.trim() : '';
    if (!cmd) return;
    input.value = '';
    return this.runAction('raw_cmd', { command: cmd });
  },

  executeQuickSQL(target) {
    const sql = document.getElementById('quick-sql-input').value.trim();
    const action = target === 'primary' ? 'query_primary' : 'query_replica';
    return this.runAction(action, { sql });
  },

  burstQueries() {
    return this.runAction('burst_traffic');
  },

  reloadGrafana() {
    const frame = document.getElementById('grafana-frame');
    if (frame) frame.src = frame.src;
  },

  handleGrafanaError() {
    const fallback = document.getElementById('grafana-fallback');
    if (fallback) fallback.classList.remove('hidden');
  }
};

// Global Exposure for HTML onclick bindings
window.toggleLayer = (layer) => App.setLayer(layer);
window.triggerSync = () => App.triggerSync();
window.toggleTerminalDrawer = () => Terminal.toggle();
window.clearTerminalLogs = () => Terminal.clear();
window.executeRawCommand = () => App.executeRawCommand();
window.triggerSwitchover = () => App.switchover();
window.triggerBackup = () => App.backup();
window.triggerRestore = () => App.restore();
window.triggerNetworkChaos = () => App.networkChaos();
window.executeQuickSQL = (target) => App.executeQuickSQL(target);
window.runBurstQueries = () => App.burstQueries();
window.testPodDirect = (pod) => App.testPod(pod);
window.killPod = (pod) => App.killPod(pod);
window.selectInstance = (pod) => App.testPod(pod);
window.reloadGrafana = () => App.reloadGrafana();
window.handleGrafanaFrameError = () => App.handleGrafanaError();

// Initialize on DOM load
document.addEventListener('DOMContentLoaded', () => App.init());
