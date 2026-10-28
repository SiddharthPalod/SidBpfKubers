// SidBpfKubers CLI Console & Slide-Over Drawer Manager

const Terminal = {
  isOpen: false,

  toggle() {
    const drawer = document.getElementById('terminal-drawer');
    this.isOpen = !this.isOpen;
    if (this.isOpen) {
      drawer.classList.remove('translate-x-full');
      const inp = document.getElementById('terminal-input');
      if (inp) inp.focus();
    } else {
      drawer.classList.add('translate-x-full');
    }
  },

  appendLog(cmd, out, ms, success) {
    const logs = document.getElementById('terminal-logs');
    if (!logs) return;

    const time = new Date().toLocaleTimeString();
    const mark = success ? '✔' : '✖';
    const markColor = success ? 'text-cyber-mint' : 'text-cyber-crimson';

    const item = document.createElement('div');
    item.className = 'border-b border-surface-border pb-2 space-y-1';
    item.innerHTML = `
      <div class="flex items-center justify-between text-[10px] text-slate-400">
        <span class="text-white font-bold">$ ${this.escapeHtml(cmd)}</span>
        <span>${time} · ${ms}ms <span class="${markColor}">${mark}</span></span>
      </div>
      <pre class="text-slate-300 text-[11px] whitespace-pre-wrap overflow-x-auto">${this.escapeHtml(out ? out.trim() : '(No output)')}</pre>
    `;
    logs.appendChild(item);
    logs.scrollTop = logs.scrollHeight;
  },

  clear() {
    const logs = document.getElementById('terminal-logs');
    if (logs) {
      logs.innerHTML = '<div class="text-slate-500 italic">// Terminal log cleared.</div>';
    }
  },

  escapeHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }
};
