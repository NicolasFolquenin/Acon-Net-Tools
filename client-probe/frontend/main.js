/**
 * Client Probe - Desktop Wails Frontend Logic
 */

let throughputChart = null;
let latencyChart = null;

// Elementos DOM
const btnRunTest = document.getElementById('btn-run-test');
const btnIconPlay = document.getElementById('btn-icon-play');
const btnSpinner = document.getElementById('btn-spinner');
const btnLabel = document.getElementById('btn-label');
const statusBadge = document.getElementById('status-badge');
const statusText = document.getElementById('status-text');

const statDownload = document.getElementById('stat-download');
const statUpload = document.getElementById('stat-upload');
const statPing = document.getElementById('stat-ping');
const statSync = document.getElementById('stat-sync');
const statSyncDetail = document.getElementById('stat-sync-detail');

const nodeIdDisplay = document.getElementById('node-id-display');
const nodeLocationDisplay = document.getElementById('node-location-display');
const nextRunText = document.getElementById('next-run-text');
const metricsTableBody = document.getElementById('metrics-table-body');
const lastUpdatedText = document.getElementById('last-updated-text');

// Modal Settings Elements
const modalSettings = document.getElementById('settings-modal');
const btnOpenSettings = document.getElementById('btn-open-settings');
const btnCloseSettings = document.getElementById('btn-close-settings');
const btnCancelSettings = document.getElementById('btn-cancel-settings');
const settingsForm = document.getElementById('settings-form');

// Helper para invocar métodos Wails com fallback seguro
const WailsApp = {
  async getStatus() {
    if (window.go && window.go.main && window.go.main.App) {
      return await window.go.main.App.GetStatus();
    }
    return {
      is_running: false,
      status_text: 'Agendado (1h)',
      server_status: 'Online',
      total_synced: 24,
      total_pending: 0,
      next_run: new Date(Date.now() + 3600000).toISOString()
    };
  },

  async getHistory() {
    if (window.go && window.go.main && window.go.main.App) {
      return await window.go.main.App.GetHistory();
    }
    // Mock para desenvolvimento/preview no navegador
    return generateMockHistory(24);
  },

  async runTestNow() {
    if (window.go && window.go.main && window.go.main.App) {
      return await window.go.main.App.RunTestNow();
    }
    await new Promise(r => setTimeout(r, 1800));
    return {
      id: Date.now(),
      node_id: "minipc-dc-01",
      location: "Guarapuava-DC1",
      timestamp: new Date().toISOString(),
      download_mbps: +(820 + Math.random() * 80).toFixed(2),
      upload_mbps: +(700 + Math.random() * 60).toFixed(2),
      ping_gateway: +(1.1 + Math.random() * 0.8).toFixed(2),
      synced: true
    };
  },

  async getConfig() {
    if (window.go && window.go.main && window.go.main.App) {
      return await window.go.main.App.GetConfig();
    }
    return {
      node_id: 'minipc-dc-01',
      location: 'Guarapuava-DC1',
      server_url: 'http://localhost:8080',
      api_token: 'super-secret-probe-token-2026',
      interval_minutes: 60,
      ping_target: '192.168.1.1'
    };
  },

  async saveConfig(cfg) {
    if (window.go && window.go.main && window.go.main.App) {
      return await window.go.main.App.SaveConfig(cfg);
    }
    console.log('[MOCK] Configurações salvas:', cfg);
    return true;
  }
};

// Inicialização da aplicação
document.addEventListener('DOMContentLoaded', async () => {
  setupEventListeners();
  initCharts();
  await loadAppConfig();
  await refreshDashboard();

  // Escuta eventos Wails se disponíveis
  if (window.runtime) {
    window.runtime.EventsOn('status_updated', updateStatusBadge);
    window.runtime.EventsOn('metric_added', (record) => {
      refreshDashboard();
    });
  }

  // Polling leve para sincronizar UI caso não haja Wails runtime
  setInterval(async () => {
    const status = await WailsApp.getStatus();
    updateStatusBadge(status);
  }, 10000);
});

function setupEventListeners() {
  btnRunTest.addEventListener('click', onRunTestClicked);
  btnOpenSettings.addEventListener('click', openSettingsModal);
  btnCloseSettings.addEventListener('click', closeSettingsModal);
  btnCancelSettings.addEventListener('click', closeSettingsModal);
  settingsForm.addEventListener('submit', onSaveSettings);
}

// Handler do botão "Rodar Teste Agora" com estado visual de loading
async function onRunTestClicked() {
  setLoadingState(true);
  updateStatusBadge({ is_running: true, status_text: 'Executando teste manual...' });

  try {
    const newMetric = await WailsApp.runTestNow();
    await refreshDashboard();
  } catch (err) {
    console.error('Erro ao executar teste:', err);
    alert('Erro ao executar medição de rede: ' + (err.message || err));
  } finally {
    setLoadingState(false);
    const status = await WailsApp.getStatus();
    updateStatusBadge(status);
  }
}

function setLoadingState(isLoading) {
  if (isLoading) {
    btnRunTest.disabled = true;
    btnIconPlay.classList.add('hidden');
    btnSpinner.classList.remove('hidden');
    btnLabel.textContent = 'Medindo Rede...';
  } else {
    btnRunTest.disabled = false;
    btnIconPlay.classList.remove('hidden');
    btnSpinner.classList.add('hidden');
    btnLabel.textContent = 'Rodar Teste Agora';
  }
}

function updateStatusBadge(status) {
  if (!status) return;

  if (status.is_running) {
    statusBadge.className = 'status-badge running';
    statusText.textContent = status.status_text || 'Executando teste...';
  } else {
    statusBadge.className = 'status-badge scheduled';
    statusText.textContent = status.status_text || 'Agendado (1h)';
  }

  if (status.next_run) {
    const d = new Date(status.next_run);
    nextRunText.textContent = d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  if (status.total_synced !== undefined) {
    statSyncDetail.textContent = `${status.total_synced} sincronizados | ${status.total_pending} pendentes`;
    if (status.total_pending > 0) {
      statSync.textContent = 'Pendente';
      statSync.style.color = '#fbbf24';
    } else {
      statSync.textContent = 'Sincronizado';
      statSync.style.color = '#10b981';
    }
  }
}

// Inicialização dos Gráficos com Chart.js
function initCharts() {
  const commonOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: '#1f2937',
        titleColor: '#fff',
        bodyColor: '#e5e7eb',
        borderColor: 'rgba(255,255,255,0.1)',
        borderWidth: 1,
        padding: 10,
        displayColors: false,
      }
    },
    scales: {
      x: {
        grid: { color: 'rgba(255, 255, 255, 0.05)' },
        ticks: { color: '#9ca3af', font: { size: 10 } }
      },
      y: {
        grid: { color: 'rgba(255, 255, 255, 0.05)' },
        ticks: { color: '#9ca3af', font: { size: 10 } }
      }
    }
  };

  // 1. Throughput Chart (Download & Upload)
  const ctxThroughput = document.getElementById('throughputChart').getContext('2d');
  throughputChart = new Chart(ctxThroughput, {
    type: 'line',
    data: {
      labels: [],
      datasets: [
        {
          label: 'Download (Mbps)',
          data: [],
          borderColor: '#06b6d4',
          backgroundColor: 'rgba(6, 182, 212, 0.1)',
          fill: true,
          tension: 0.35,
          borderWidth: 2,
          pointRadius: 3,
          pointHoverRadius: 6
        },
        {
          label: 'Upload (Mbps)',
          data: [],
          borderColor: '#a855f7',
          backgroundColor: 'rgba(168, 85, 247, 0.08)',
          fill: true,
          tension: 0.35,
          borderWidth: 2,
          pointRadius: 3,
          pointHoverRadius: 6
        }
      ]
    },
    options: {
      ...commonOptions,
      scales: {
        ...commonOptions.scales,
        y: {
          ...commonOptions.scales.y,
          title: { display: true, text: 'Mbps', color: '#9ca3af', font: { size: 11 } }
        }
      }
    }
  });

  // 2. Latency Chart (Ping)
  const ctxLatency = document.getElementById('latencyChart').getContext('2d');
  latencyChart = new Chart(ctxLatency, {
    type: 'line',
    data: {
      labels: [],
      datasets: [
        {
          label: 'Ping Gateway (ms)',
          data: [],
          borderColor: '#10b981',
          backgroundColor: 'rgba(16, 185, 129, 0.12)',
          fill: true,
          tension: 0.3,
          borderWidth: 2,
          pointRadius: 3,
          pointHoverRadius: 6
        }
      ]
    },
    options: {
      ...commonOptions,
      scales: {
        ...commonOptions.scales,
        y: {
          ...commonOptions.scales.y,
          title: { display: true, text: 'ms', color: '#9ca3af', font: { size: 11 } }
        }
      }
    }
  });
}

// Atualiza Dashboard com métricas do SQLite
async function refreshDashboard() {
  const history = await WailsApp.getHistory();
  const status = await WailsApp.getStatus();

  updateStatusBadge(status);

  if (history && history.length > 0) {
    const latest = history[history.length - 1];

    // Atualiza cards numéricos
    statDownload.textContent = latest.download_mbps.toFixed(1);
    statUpload.textContent = latest.upload_mbps.toFixed(1);
    statPing.textContent = latest.ping_gateway.toFixed(1);

    // Formata timestamps e alimenta gráficos (últimas 24 execuções)
    const labels = history.map(item => {
      const d = new Date(item.timestamp);
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    });

    const downloads = history.map(item => item.download_mbps);
    const uploads = history.map(item => item.upload_mbps);
    const pings = history.map(item => item.ping_gateway);

    // Atualiza Chart Throughput
    throughputChart.data.labels = labels;
    throughputChart.data.datasets[0].data = downloads;
    throughputChart.data.datasets[1].data = uploads;
    throughputChart.update();

    // Atualiza Chart Latência
    latencyChart.data.labels = labels;
    latencyChart.data.datasets[0].data = pings;
    latencyChart.update();

    // Renderiza tabela de log recente (inversa para mostrar mais novo no topo)
    renderTable([...history].reverse());
    lastUpdatedText.textContent = `Última leitura: ${new Date(latest.timestamp).toLocaleTimeString()}`;
  } else {
    statDownload.textContent = '--';
    statUpload.textContent = '--';
    statPing.textContent = '--';
  }
}

function renderTable(records) {
  if (!records || records.length === 0) {
    metricsTableBody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">Nenhum teste registrado ainda.</td></tr>';
    return;
  }

  metricsTableBody.innerHTML = records.map(r => {
    const d = new Date(r.timestamp);
    const dateFormatted = `${d.toLocaleDateString()} ${d.toLocaleTimeString()}`;
    const syncBadge = r.synced
      ? '<span class="sync-badge ok">✓ Sincronizado</span>'
      : '<span class="sync-badge fail" title="' + (r.error_message || 'Falha de conexão') + '">✕ Pendente</span>';

    return `
      <tr>
        <td>${dateFormatted}</td>
        <td><strong style="color: var(--cyan-light)">${r.download_mbps.toFixed(1)}</strong> Mbps</td>
        <td><strong style="color: var(--purple-light)">${r.upload_mbps.toFixed(1)}</strong> Mbps</td>
        <td><strong style="color: var(--emerald-light)">${r.ping_gateway.toFixed(1)}</strong> ms</td>
        <td>${syncBadge}</td>
      </tr>
    `;
  }).join('');
}

// Configurações
async function loadAppConfig() {
  const cfg = await WailsApp.getConfig();
  nodeIdDisplay.textContent = cfg.node_id || 'node-01';
  nodeLocationDisplay.textContent = cfg.location || 'Local';

  document.getElementById('cfg-node-id').value = cfg.node_id || '';
  document.getElementById('cfg-location').value = cfg.location || '';
  document.getElementById('cfg-server-url').value = cfg.server_url || '';
  document.getElementById('cfg-api-token').value = cfg.api_token || '';
  document.getElementById('cfg-interval').value = cfg.interval_minutes || 60;
  document.getElementById('cfg-ping-target').value = cfg.ping_target || '192.168.1.1';
}

function openSettingsModal() {
  modalSettings.classList.remove('hidden');
}

function closeSettingsModal() {
  modalSettings.classList.add('hidden');
}

async function onSaveSettings(e) {
  e.preventDefault();
  const cfg = {
    node_id: document.getElementById('cfg-node-id').value.trim(),
    location: document.getElementById('cfg-location').value.trim(),
    server_url: document.getElementById('cfg-server-url').value.trim(),
    api_token: document.getElementById('cfg-api-token').value.trim(),
    interval_minutes: parseInt(document.getElementById('cfg-interval').value, 10) || 60,
    ping_target: document.getElementById('cfg-ping-target').value.trim()
  };

  try {
    await WailsApp.saveConfig(cfg);
    await loadAppConfig();
    closeSettingsModal();
    alert('Configurações salvas com sucesso!');
  } catch (err) {
    alert('Erro ao salvar configurações: ' + err.message);
  }
}

// Fallback Mock generator
function generateMockHistory(count) {
  const list = [];
  const now = Date.now();
  for (let i = count - 1; i >= 0; i--) {
    const time = new Date(now - i * 3600000);
    list.push({
      id: i + 1,
      node_id: 'minipc-dc-01',
      location: 'Guarapuava-DC1',
      timestamp: time.toISOString(),
      download_mbps: +(810 + Math.sin(i) * 60 + Math.random() * 20).toFixed(2),
      upload_mbps: +(715 + Math.cos(i) * 45 + Math.random() * 15).toFixed(2),
      ping_gateway: +(1.2 + Math.random() * 0.7).toFixed(2),
      synced: true
    });
  }
  return list;
}
