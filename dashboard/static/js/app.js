// SidKubers Dashboard Core Controller & UI Orchestrator

const App = {
  activeLayer: 'k8s',
  currentCluster: 'sample-postgres',
  currentEngine: 'postgres',

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

  toggleClusterDropdown(e) {
    if (e) e.stopPropagation();
    const menu = document.getElementById('cluster-dropdown-menu');
    const chevron = document.getElementById('dropdown-chevron');
    if (!menu) return;
    const isHidden = menu.classList.contains('hidden');
    if (isHidden) {
      menu.classList.remove('hidden');
      if (chevron) chevron.classList.add('rotate-180');
    } else {
      menu.classList.add('hidden');
      if (chevron) chevron.classList.remove('rotate-180');
    }
  },

  closeClusterDropdown() {
    const menu = document.getElementById('cluster-dropdown-menu');
    const chevron = document.getElementById('dropdown-chevron');
    if (menu) menu.classList.add('hidden');
    if (chevron) chevron.classList.remove('rotate-180');
  },

  updateDropdownTriggerUI(clusterName, engine) {
    const iconEl = document.getElementById('active-cluster-icon');
    const labelEl = document.getElementById('active-cluster-label');
    const pillEl = document.getElementById('active-cluster-engine-pill');

    let icon = '🐘';
    let pillText = 'POSTGRES';
    let pillClass = 'text-[9px] px-1.5 py-0.2 rounded bg-surface border-sharp text-cyber-emerald uppercase font-bold';

    if (engine === 'redis') {
      icon = '⚡';
      pillText = 'REDIS';
      pillClass = 'text-[9px] px-1.5 py-0.2 rounded bg-cyber-mint/15 border border-cyber-mint/30 text-cyber-mint uppercase font-bold';
    } else if (engine === 'mongodb' || engine === 'mongo') {
      icon = '🍃';
      pillText = 'MONGODB';
      pillClass = 'text-[9px] px-1.5 py-0.2 rounded bg-emerald-500/15 border border-emerald-500/30 text-emerald-400 uppercase font-bold';
    } else if (engine === 'cassandra') {
      icon = '🌀';
      pillText = 'CASSANDRA';
      pillClass = 'text-[9px] px-1.5 py-0.2 rounded bg-cyan-500/15 border border-cyan-500/30 text-cyan-400 uppercase font-bold';
    } else if (engine === 'etcd') {
      icon = '🏛️';
      pillText = 'ETCD';
      pillClass = 'text-[9px] px-1.5 py-0.2 rounded bg-purple-500/15 border border-purple-500/30 text-purple-400 uppercase font-bold';
    }

    if (iconEl) iconEl.textContent = icon;
    if (labelEl) labelEl.textContent = clusterName;
    if (pillEl) {
      pillEl.textContent = pillText;
      pillEl.className = pillClass;
    }

    // Highlight selected item in dropdown menu
    document.querySelectorAll('.cluster-option').forEach(el => {
      if (el.dataset.cluster === clusterName) {
        el.classList.add('bg-surface-hover', 'border-l-2', 'border-cyber-mint');
      } else {
        el.classList.remove('bg-surface-hover', 'border-l-2', 'border-cyber-mint');
      }
    });
  },

  async selectCluster(clusterName) {
    if (!clusterName) return;
    this.closeClusterDropdown();
    this.currentCluster = clusterName;
    if (clusterName.includes('redis')) {
      this.currentEngine = 'redis';
    } else if (clusterName.includes('mongo')) {
      this.currentEngine = 'mongodb';
    } else if (clusterName.includes('cassandra')) {
      this.currentEngine = 'cassandra';
    } else if (clusterName.includes('etcd')) {
      this.currentEngine = 'etcd';
    } else {
      this.currentEngine = 'postgres';
    }

    // 1. Instant Optimistic UI Updates & Skeleton Loading (Smooth Transition)
    this.applyOptimisticUI(clusterName, this.currentEngine);

    // 2. Update Console query placeholder and labels to match engine immediately
    const sqlInput = document.getElementById('quick-sql-input');
    const labelPrimary = document.getElementById('label-query-primary');
    const labelReplica = document.getElementById('label-query-replica');
    const targetDesc = document.getElementById('quick-query-target-desc');
    const engineBadge = document.getElementById('quick-query-engine-badge');

    if (this.currentEngine === 'redis') {
      if (sqlInput) sqlInput.value = 'INFO replication';
      if (labelPrimary) labelPrimary.textContent = '⚡ Query Master (RW)';
      if (labelReplica) labelReplica.textContent = '🔍 Query Replica (RO)';
      if (targetDesc) targetDesc.textContent = 'sample-redis-rw / -ro (Port 6379)';
      if (engineBadge) engineBadge.textContent = 'Redis 7 Engine';
    } else if (this.currentEngine === 'mongodb') {
      if (sqlInput) sqlInput.value = 'rs.status()';
      if (labelPrimary) labelPrimary.textContent = '⚡ Query Primary (RW)';
      if (labelReplica) labelReplica.textContent = '🔍 Query Secondary (RO)';
      if (targetDesc) targetDesc.textContent = 'sample-mongodb-rw / -ro (Port 27017)';
      if (engineBadge) engineBadge.textContent = 'MongoDB 7.0 Engine';
    } else if (this.currentEngine === 'cassandra') {
      if (sqlInput) sqlInput.value = 'nodetool status';
      if (labelPrimary) labelPrimary.textContent = '⚡ Query Coordinator (RW)';
      if (labelReplica) labelReplica.textContent = '🔍 Query Ring Peer (RO)';
      if (targetDesc) targetDesc.textContent = 'sample-cassandra-client (Port 9042)';
      if (engineBadge) engineBadge.textContent = 'Cassandra 4.1 Ring';
    } else if (this.currentEngine === 'etcd') {
      if (sqlInput) sqlInput.value = 'etcdctl endpoint status -w table';
      if (labelPrimary) labelPrimary.textContent = '⚡ Query Leader (RW)';
      if (labelReplica) labelReplica.textContent = '🔍 Query Follower (RO)';
      if (targetDesc) targetDesc.textContent = 'sample-etcd-client (Port 2379)';
      if (engineBadge) engineBadge.textContent = 'etcd 3.5 Raft Quorum';
    } else {
      if (sqlInput) sqlInput.value = 'SELECT current_database(), inet_server_addr(), pg_is_in_recovery(), now();';
      if (labelPrimary) labelPrimary.textContent = '⚡ Query RW (Primary)';
      if (labelReplica) labelReplica.textContent = '🔍 Query RO (Replica)';
      if (targetDesc) targetDesc.textContent = 'sample-postgres-rw / -ro (Port 5432)';
      if (engineBadge) engineBadge.textContent = 'PostgreSQL 16 Engine';
    }

    try {
      await this.runAction('select_cluster', { cluster: clusterName });
    } catch (e) {
      console.error('Failed to select cluster:', e);
    }
  },

  applyOptimisticUI(clusterName, engine) {
    // 1. Dropdown Trigger Button
    this.updateDropdownTriggerUI(clusterName, engine);

    // 2. Sidebar CRD & Engine
    const sideCrName = document.getElementById('sidebar-cr-name');
    if (sideCrName) sideCrName.textContent = clusterName;

    const sideBadge = document.getElementById('sidebar-engine-badge');
    if (sideBadge) {
      if (engine === 'redis') {
        sideBadge.textContent = 'REDIS-7';
        sideBadge.className = 'text-[9px] mono px-1.5 py-0.2 rounded bg-cyber-mint/10 text-cyber-mint border border-cyber-mint/30 font-semibold';
      } else if (engine === 'mongodb' || engine === 'mongo') {
        sideBadge.textContent = 'MONGO-7.0';
        sideBadge.className = 'text-[9px] mono px-1.5 py-0.2 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/30 font-semibold';
      } else if (engine === 'cassandra') {
        sideBadge.textContent = 'CASSANDRA-4.1';
        sideBadge.className = 'text-[9px] mono px-1.5 py-0.2 rounded bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 font-semibold';
      } else if (engine === 'etcd') {
        sideBadge.textContent = 'ETCD-3.5';
        sideBadge.className = 'text-[9px] mono px-1.5 py-0.2 rounded bg-purple-500/10 text-purple-400 border border-purple-500/30 font-semibold';
      } else {
        sideBadge.textContent = 'PG-16';
        sideBadge.className = 'text-[9px] mono px-1.5 py-0.2 rounded bg-cyber-emerald/10 text-cyber-emerald border border-cyber-emerald/30 font-semibold';
      }
    }

    const sideRepMode = document.getElementById('sidebar-replication-mode');
    if (sideRepMode) {
      if (engine === 'redis') {
        sideRepMode.textContent = 'Master/Replica (Sync)';
      } else if (engine === 'mongodb' || engine === 'mongo') {
        sideRepMode.textContent = 'Replica Set (rs0 Quorum)';
      } else if (engine === 'cassandra') {
        sideRepMode.textContent = 'Peer-to-Peer Ring (Gossip & DHT)';
      } else if (engine === 'etcd') {
        sideRepMode.textContent = 'Raft Consensus Quorum';
      } else {
        sideRepMode.textContent = 'Sync (Zero Lag)';
      }
    }

    // 3. Service Routing Endpoints Skeleton Placeholders (Zero fake text or labels)
    const svcRwName = document.getElementById('svc-rw-name');
    if (svcRwName) svcRwName.innerHTML = `<span class="inline-block h-3 w-32 bg-slate-800/80 rounded animate-pulse align-middle"></span>`;
    const svcRwTarget = document.getElementById('svc-rw-target');
    if (svcRwTarget) svcRwTarget.innerHTML = `<span class="inline-block h-2.5 w-24 bg-slate-800/80 rounded animate-pulse align-middle"></span>`;
    const svcRwIp = document.getElementById('svc-rw-ip');
    if (svcRwIp) svcRwIp.innerHTML = `<span class="inline-block h-2.5 w-20 bg-slate-800/80 rounded animate-pulse align-middle"></span>`;

    const svcRoName = document.getElementById('svc-ro-name');
    if (svcRoName) svcRoName.innerHTML = `<span class="inline-block h-3 w-32 bg-slate-800/80 rounded animate-pulse align-middle"></span>`;
    const svcRoTarget = document.getElementById('svc-ro-target');
    if (svcRoTarget) svcRoTarget.innerHTML = `<span class="inline-block h-2.5 w-28 bg-slate-800/80 rounded animate-pulse align-middle"></span>`;
    const svcRoIp = document.getElementById('svc-ro-ip');
    if (svcRoIp) svcRoIp.innerHTML = `<span class="inline-block h-2.5 w-20 bg-slate-800/80 rounded animate-pulse align-middle"></span>`;

    const svcRName = document.getElementById('svc-r-name');
    if (svcRName) svcRName.innerHTML = `<span class="inline-block h-3 w-28 bg-slate-800/80 rounded animate-pulse align-middle"></span>`;
    const svcRTarget = document.getElementById('svc-r-target');
    if (svcRTarget) svcRTarget.innerHTML = `<span class="inline-block h-2.5 w-24 bg-slate-800/80 rounded animate-pulse align-middle"></span>`;
    const svcRIp = document.getElementById('svc-r-ip');
    if (svcRIp) svcRIp.innerHTML = `<span class="inline-block h-2.5 w-20 bg-slate-800/80 rounded animate-pulse align-middle"></span>`;

    const svcHeadlessName = document.getElementById('svc-headless-name');
    if (svcHeadlessName) svcHeadlessName.innerHTML = `<span class="inline-block h-3 w-36 bg-slate-800/80 rounded animate-pulse align-middle"></span>`;
    const svcHeadlessIp = document.getElementById('svc-headless-ip');
    if (svcHeadlessIp) svcHeadlessIp.innerHTML = `<span class="inline-block h-2.5 w-16 bg-slate-800/80 rounded animate-pulse align-middle"></span>`;

    const treePgReplicas = document.getElementById('tree-pg-replicas');
    if (treePgReplicas) treePgReplicas.innerHTML = `<span class="inline-block h-3 w-16 bg-slate-800/80 rounded animate-pulse align-middle"></span>`;

    // 4. Honest Skeleton Loaders for Pods & Tree List (No fake numbers/statuses)
    this.renderPodSkeletons();
  },

  renderPodSkeletons() {
    const podsContainer = document.getElementById('pods-container');
    const treeList = document.getElementById('tree-pod-list');

    if (treeList) {
      treeList.innerHTML = [0, 1, 2].map(() => `
        <div class="p-2 rounded bg-surface/40 border border-surface-border/40 flex items-center justify-between animate-pulse">
          <div class="flex items-center gap-2">
            <span class="h-2 w-2 rounded-full bg-slate-700"></span>
            <span class="h-3 w-24 bg-slate-800/80 rounded"></span>
          </div>
          <span class="h-3 w-10 bg-slate-800/80 rounded"></span>
        </div>
      `).join('');
    }

    if (podsContainer) {
      podsContainer.innerHTML = [0, 1, 2].map(() => `
        <div class="border-sharp bg-surface/50 rounded p-3.5 space-y-3 relative border-surface-border/40 animate-pulse">
          <!-- Card Header Skeleton -->
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <span class="h-2 w-2 rounded-full bg-slate-700"></span>
              <span class="h-3.5 w-28 bg-slate-800/80 rounded"></span>
            </div>
            <span class="h-4 w-16 bg-slate-800/70 rounded border border-surface-border/30"></span>
          </div>

          <!-- Metadata Grid Skeleton -->
          <div class="grid grid-cols-2 gap-2.5 bg-obsidian/60 p-2.5 rounded border border-surface-border/30">
            <div class="flex items-center gap-2">
              <span class="text-slate-600 text-[10px] mono">Status:</span>
              <span class="h-3 w-14 bg-slate-800/80 rounded"></span>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-slate-600 text-[10px] mono">Ready:</span>
              <span class="h-3 w-10 bg-slate-800/80 rounded"></span>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-slate-600 text-[10px] mono">Pod IP:</span>
              <span class="h-3 w-16 bg-slate-800/80 rounded"></span>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-slate-600 text-[10px] mono">Restarts:</span>
              <span class="h-3 w-8 bg-slate-800/80 rounded"></span>
            </div>
            <div class="col-span-2 flex items-center gap-2 pt-0.5">
              <span class="text-slate-600 text-[10px] mono">Role:</span>
              <span class="h-3 flex-1 bg-slate-800/80 rounded"></span>
            </div>
          </div>

          <!-- Action Buttons Skeleton -->
          <div class="flex items-center gap-2 pt-1">
            <div class="h-7 flex-1 bg-slate-800/70 rounded border border-surface-border/30"></div>
            <div class="h-7 w-16 bg-slate-800/50 rounded border border-surface-border/30"></div>
          </div>
        </div>
      `).join('');
    }
  },

  async runAction(action, params = {}) {
    const spinner = document.getElementById('sync-spinner');
    if (spinner) spinner.classList.add('animate-spin');

    if (!params.cluster) {
      params.cluster = this.currentCluster;
    }

    try {
      const d = await API.runAction(action, params);
      Terminal.appendLog(d.command, d.output, d.duration_ms, d.success);

      // Instant HUD status feedback toast directly on screen (muted for clean cluster switching)
      if (action !== 'select_cluster') {
        this.showToast(d.command, d.output, d.duration_ms, d.success);
      }

      // Direct inline command/SQL output updates
      if (action === 'query_primary') {
        const title = this.currentEngine === 'redis' ? `${this.currentCluster}-rw (👑 Master RW)` : (this.currentEngine === 'mongodb' ? `${this.currentCluster}-rw (👑 Primary RW)` : (this.currentEngine === 'etcd' ? `${this.currentCluster}-rw (👑 Raft Leader RW)` : (this.currentEngine === 'cassandra' ? `${this.currentCluster}-rw (⚡ Seed Coordinator RW)` : `${this.currentCluster}-rw (👑 Primary RW)`)));
        this.displaySQLResults(title, d.command, d.output, d.duration_ms, d.success);
      } else if (action === 'query_replica') {
        const title = this.currentEngine === 'redis' ? `${this.currentCluster}-ro (📦 Standby RO)` : (this.currentEngine === 'mongodb' ? `${this.currentCluster}-ro (📦 Secondary RO)` : (this.currentEngine === 'etcd' ? `${this.currentCluster}-ro (🏛️ Raft Follower RO)` : (this.currentEngine === 'cassandra' ? `${this.currentCluster}-ro (🧬 Ring Node Status)` : `${this.currentCluster}-ro (📦 Standby RO)`)));
        this.displaySQLResults(title, d.command, d.output, d.duration_ms, d.success);
      } else if (action === 'test_pod') {
        const podName = params.pod || 'Pod Test';
        const testType = this.currentEngine === 'redis' ? 'Direct Redis Role' : (this.currentEngine === 'mongodb' ? 'Direct Mongo RS State' : (this.currentEngine === 'etcd' ? 'Direct etcd Endpoint' : (this.currentEngine === 'cassandra' ? 'Direct nodetool Status' : 'Direct SQL')));
        this.displaySQLResults(`${podName} (${testType})`, d.command, d.output, d.duration_ms, d.success);
      }

      this.triggerSync();
      return d;
    } catch (err) {
      Terminal.appendLog(action, err.message, 0, false);
      if (action !== 'select_cluster') {
        this.showToast(action, err.message, 0, false);
      }
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

    // Prevent in-flight poll responses for old cluster from overriding newly selected cluster
    if (this.currentCluster && data.active_cluster && data.active_cluster !== this.currentCluster) {
      return;
    }

    if (data.active_cluster) {
      this.currentCluster = data.active_cluster;
    }
    if (data.engine) {
      this.currentEngine = data.engine;
    }

    // 0. Update Dropdown Trigger & Options
    this.updateDropdownTriggerUI(data.cluster_name || this.currentCluster, data.engine || this.currentEngine);

    if (data.clusters) {
      const pgDetail = data.clusters['sample-postgres'];
      const redisDetail = data.clusters['sample-redis'];
      const mongoDetail = data.clusters['sample-mongodb'];
      const cassDetail = data.clusters['sample-cassandra'];
      const etcdDetail = data.clusters['sample-etcd'];

      const bPg = document.getElementById('badge-dropdown-postgres');
      if (bPg && pgDetail) {
        bPg.textContent = `${pgDetail.ready_replicas || (pgDetail.pods ? pgDetail.pods.length : 3)}/${pgDetail.replicas || 3}`;
      }
      const bRedis = document.getElementById('badge-dropdown-redis');
      if (bRedis && redisDetail) {
        bRedis.textContent = `${redisDetail.ready_replicas || (redisDetail.pods ? redisDetail.pods.length : 3)}/${redisDetail.replicas || 3}`;
      }
      const bMongo = document.getElementById('badge-dropdown-mongodb');
      if (bMongo && mongoDetail) {
        bMongo.textContent = `${mongoDetail.ready_replicas || (mongoDetail.pods ? mongoDetail.pods.length : 3)}/${mongoDetail.replicas || 3}`;
      }
      const bCass = document.getElementById('badge-dropdown-cassandra');
      if (bCass && cassDetail) {
        bCass.textContent = `${cassDetail.ready_replicas || (cassDetail.pods ? cassDetail.pods.length : 3)}/${cassDetail.replicas || 3}`;
      }
      const bEtcd = document.getElementById('badge-dropdown-etcd');
      if (bEtcd && etcdDetail) {
        bEtcd.textContent = `${etcdDetail.ready_replicas || (etcdDetail.pods ? etcdDetail.pods.length : 3)}/${etcdDetail.replicas || 3}`;
      }

      // Update trigger button active badge
      const activeDetail = data.clusters[this.currentCluster];
      const activeBadge = document.getElementById('active-cluster-badge');
      if (activeBadge && activeDetail) {
        activeBadge.textContent = `${activeDetail.ready_replicas || (activeDetail.pods ? activeDetail.pods.length : 3)}/${activeDetail.replicas || 3}`;
      }
    }

    const clusterBadge = document.getElementById('k8s-cluster-badge');
    if (clusterBadge) {
      clusterBadge.textContent = `CR: ${data.cluster_name || this.currentCluster} (${(data.engine || this.currentEngine).toUpperCase()})`;
    }

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
    const subCluster = document.getElementById('sub-active-cluster');
    if (subCluster) {
      subCluster.textContent = `${data.cluster_name || this.currentCluster}`;
    }

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

    // 3. Sidebar CRD Specification
    const sideCrName = document.getElementById('sidebar-cr-name');
    if (sideCrName) {
      sideCrName.textContent = data.cluster_name || this.currentCluster;
    }

    const sideBadge = document.getElementById('sidebar-engine-badge');
    if (sideBadge) {
      if (this.currentEngine === 'redis') {
        sideBadge.textContent = 'REDIS-7';
        sideBadge.className = 'text-[9px] mono px-1.5 py-0.2 rounded bg-cyber-mint/10 text-cyber-mint border border-cyber-mint/30 font-semibold';
      } else if (this.currentEngine === 'mongodb' || this.currentEngine === 'mongo') {
        sideBadge.textContent = 'MONGO-7.0';
        sideBadge.className = 'text-[9px] mono px-1.5 py-0.2 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/30 font-semibold';
      } else if (this.currentEngine === 'etcd') {
        sideBadge.textContent = 'ETCD-3.5';
        sideBadge.className = 'text-[9px] mono px-1.5 py-0.2 rounded bg-purple-500/10 text-purple-400 border border-purple-500/30 font-semibold';
      } else if (this.currentEngine === 'cassandra') {
        sideBadge.textContent = 'CASSANDRA-4.1';
        sideBadge.className = 'text-[9px] mono px-1.5 py-0.2 rounded bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 font-semibold';
      } else {
        sideBadge.textContent = 'PG-16';
        sideBadge.className = 'text-[9px] mono px-1.5 py-0.2 rounded bg-cyber-emerald/10 text-cyber-emerald border border-cyber-emerald/30 font-semibold';
      }
    }

    const sideRepMode = document.getElementById('sidebar-replication-mode');
    if (sideRepMode) {
      if (this.currentEngine === 'redis') {
        sideRepMode.textContent = 'Master/Replica (Sync)';
      } else if (this.currentEngine === 'mongodb' || this.currentEngine === 'mongo') {
        sideRepMode.textContent = 'Replica Set (rs0 Quorum)';
      } else if (this.currentEngine === 'etcd') {
        sideRepMode.textContent = 'Raft Consensus Quorum';
      } else if (this.currentEngine === 'cassandra') {
        sideRepMode.textContent = 'Peer-to-Peer Ring (Gossip & DHT)';
      } else {
        sideRepMode.textContent = 'Sync (Zero Lag)';
      }
    }

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

    // 4. K8s Service Routing Endpoints
    if (data.services && data.services.length > 0) {
      data.services.forEach(svc => {
        if (svc.role === 'rw') {
          const elName = document.getElementById('svc-rw-name');
          if (elName) elName.innerHTML = `<span class="h-1.5 w-1.5 rounded-full bg-cyber-emerald"></span> ${svc.name}`;
          const el = document.getElementById('svc-rw-ip');
          if (el) el.textContent = `${svc.cluster_ip}:${svc.port}`;
        } else if (svc.role === 'ro') {
          const elName = document.getElementById('svc-ro-name');
          if (elName) elName.innerHTML = `<span class="h-1.5 w-1.5 rounded-full bg-cyber-mint"></span> ${svc.name}`;
          const el = document.getElementById('svc-ro-ip');
          if (el) el.textContent = `${svc.cluster_ip}:${svc.port}`;
        } else if (svc.role === 'r') {
          const elName = document.getElementById('svc-r-name');
          if (elName) elName.innerHTML = `<span class="h-1.5 w-1.5 rounded-full bg-slate-400"></span> ${svc.name}`;
          const el = document.getElementById('svc-r-ip');
          if (el) el.textContent = `${svc.cluster_ip}:${svc.port}`;
        } else if (svc.role === 'headless') {
          const elName = document.getElementById('svc-headless-name');
          if (elName) elName.innerHTML = `<span class="h-1.5 w-1.5 rounded-full bg-slate-500"></span> ${svc.name}`;
          const el = document.getElementById('svc-headless-ip');
          if (el) el.textContent = `${svc.cluster_ip}:${svc.port}`;
        }
      });
    }

    // 5. Render Pods & Standby Bento Cards
    if (data.pods && data.pods.length > 0) {
      const treeList = document.getElementById('tree-pod-list');
      const podsContainer = document.getElementById('pods-container');

      let treeHtml = '';
      let cardsHtml = '';
      let standbyPods = [];

      data.pods.forEach(pod => {
        const isPrimary = pod.role === 'primary';
        const isRedis = this.currentEngine === 'redis';
        const isMongo = this.currentEngine === 'mongodb' || this.currentEngine === 'mongo';
        const isEtcd = this.currentEngine === 'etcd';
        const isCassandra = this.currentEngine === 'cassandra';

        let badgeClass = isPrimary ? 'bg-cyber-emerald/15 text-cyber-emerald border border-cyber-emerald/30' : 'bg-slate-800 text-slate-300 border border-slate-700';
        let badgeText = isPrimary ? '👑 PRIMARY (RW)' : '📦 STANDBY (RO)';
        let roleDesc = isPrimary ? 'Master (WAL Read/Write)' : 'Streaming WAL Standby';
        let testBtnLabel = '🔍 Direct SQL Test';

        if (isRedis) {
          badgeText = isPrimary ? '👑 MASTER (RW)' : '📦 REPLICA (RO)';
          roleDesc = isPrimary ? 'Redis Master (In-Memory RW)' : 'Replica (replicaof Master)';
          testBtnLabel = '🔍 Redis Role Test';
        } else if (isMongo) {
          badgeText = isPrimary ? '👑 PRIMARY (RW)' : '🍃 SECONDARY (RO)';
          roleDesc = isPrimary ? 'Replica Set PRIMARY (WiredTiger RW)' : 'Replica Set SECONDARY (Oplog Sync)';
          testBtnLabel = '🔍 Mongo RS Test';
        } else if (isEtcd) {
          badgeText = isPrimary ? '👑 LEADER (RW)' : '🏛️ FOLLOWER (RO)';
          roleDesc = isPrimary ? 'Raft Leader (Consensus Quorum RW)' : 'Raft Follower (Log Replicated)';
          testBtnLabel = '🔍 etcd Endpoint Test';
        } else if (isCassandra) {
          badgeClass = 'bg-cyan-500/15 text-cyan-400 border border-cyan-500/30';
          badgeText = isPrimary ? '⚡ SEED PEER (RW)' : '🧬 RING PEER (RW)';
          roleDesc = isPrimary ? 'Seed Node (Gossip Coordinator RW)' : 'Gossip Peer (Token Ring Coordinator RW)';
          testBtnLabel = '🔍 nodetool Ring Status';
        }

        if (!isPrimary) standbyPods.push(pod.name);

        treeHtml += `
          <div class="p-1.5 rounded bg-surface border-sharp flex items-center justify-between cursor-pointer hover:border-cyber-emerald/40 transition" onclick="App.testPod('${pod.name}')">
            <div class="flex items-center gap-1.5 truncate">
              <span class="h-1.5 w-1.5 rounded-full ${pod.ready ? 'bg-cyber-mint animate-led' : 'bg-cyber-amber'}"></span>
              <span class="text-white font-medium truncate">${pod.name}</span>
            </div>
            <span class="text-[9px] px-1 rounded ${badgeClass} uppercase font-bold">${isCassandra ? (isPrimary ? 'SEED' : 'PEER') : (isPrimary ? (isEtcd ? 'LEADER' : 'PRIMARY') : (isEtcd ? 'FOLLOWER' : 'STANDBY'))}</span>
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
                <span>${testBtnLabel}</span>
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
        let rwLabel = '👑 Primary';
        if (this.currentEngine === 'redis') rwLabel = '👑 Master';
        else if (this.currentEngine === 'mongodb' || this.currentEngine === 'mongo') rwLabel = '👑 Primary';
        else if (this.currentEngine === 'etcd') rwLabel = '⚡ Leader';
        else if (this.currentEngine === 'cassandra') rwLabel = '⚡ Seed Peer';
        svcRw.textContent = `${data.primary_pod} (${rwLabel})`;
      }

      const svcRo = document.getElementById('svc-ro-target');
      if (svcRo) {
        let roLabel = '📦 Standby';
        if (this.currentEngine === 'redis') roLabel = '📦 Replica';
        else if (this.currentEngine === 'mongodb' || this.currentEngine === 'mongo') roLabel = '🍃 Secondary';
        else if (this.currentEngine === 'etcd') roLabel = '📦 Follower';
        else if (this.currentEngine === 'cassandra') roLabel = '🧬 Ring Peer';
        svcRo.textContent = `${standbyPods.join(', ') || 'Standby'} (${roLabel})`;
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

      const mRedis = document.getElementById('metric-redis-conns');
      if (mRedis) mRedis.textContent = data.metrics.redis_connections ? data.metrics.redis_connections.toLocaleString() : '1,840';

      const mMongo = document.getElementById('metric-mongo-conns');
      if (mMongo) mMongo.textContent = data.metrics.mongo_connections ? data.metrics.mongo_connections.toLocaleString() : '1,420';

      const mCassandra = document.getElementById('metric-cassandra-conns');
      if (mCassandra) mCassandra.textContent = data.metrics.cassandra_connections ? data.metrics.cassandra_connections.toLocaleString() : '1,650';

      const mEtcd = document.getElementById('metric-etcd-conns');
      if (mEtcd) mEtcd.textContent = data.metrics.etcd_connections ? data.metrics.etcd_connections.toLocaleString() : '980';

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
window.selectCluster = (c) => App.selectCluster(c);
window.toggleClusterDropdown = (e) => App.toggleClusterDropdown(e);
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

// Close dropdown on click outside
document.addEventListener('click', (e) => {
  const container = document.getElementById('cluster-selector-container');
  if (container && !container.contains(e.target)) {
    App.closeClusterDropdown();
  }
});

// Initialize on DOM load
document.addEventListener('DOMContentLoaded', () => App.init());
