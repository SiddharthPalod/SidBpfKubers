// SidBpfKubers Asynchronous HTML Component Loader

const ComponentLoader = {
  components: [
    { target: 'component-header', file: '/components/header.html' },
    { target: 'component-subbar', file: '/components/subbar.html' },
    { target: 'component-sidebar', file: '/components/sidebar.html' },
    { target: 'component-layer-k8s', file: '/components/layer-k8s.html' },
    { target: 'component-layer-kernel', file: '/components/layer-kernel.html' },
    { target: 'component-layer-grafana', file: '/components/layer-grafana.html' },
    { target: 'component-terminal', file: '/components/terminal-drawer.html' }
  ],

  async loadAll() {
    const promises = this.components.map(async (c) => {
      try {
        const res = await fetch(`${c.file}?v=${Date.now()}`);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const html = await res.text();
        const el = document.getElementById(c.target);
        if (el) {
          el.innerHTML = html;
        }
      } catch (err) {
        console.error(`Failed to load component: ${c.file}`, err);
      }
    });

    await Promise.all(promises);
  }
};
