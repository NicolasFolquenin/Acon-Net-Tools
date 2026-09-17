/**
 * Central NOC Dashboard - Web App Logic
 */

let bandwidthChart = null;
let latencyChart = null;

// Elementos DOM
const selectNode = document.getElementById('select-node');
const btnRefresh = document.getElementById('btn-refresh');
const btnSeed = document.getElementById('btn-seed');
const nodesSummaryGrid = document.getElementById('nodes-summary-grid');
const telemetryTableBody = document.getElementById('telemetry-table-body');
const activeNodesCount = document.getElementById('active-nodes-count');
const lastIngestTime = document.getElementById('last-ingest-time');
const autoRefreshTimer = document.getElementById('auto-refresh-timer');

// Paleta de cores para gráficos comparativos multi-nós
const nodeColors = [
  { border: '#06b6d4', bg: 'rgba(6, 182, 212, 0.1)' },    // Cyan
  { border: '#a855f7', bg: 'rgba(168, 85, 247, 0.1)' },   // Purple
  { border: '#10b981', bg: 'rgba(16, 185, 129, 0.1)' },   // Emerald
  { border: '#f59e0b', bg: 'rgba(245, 158, 11, 0.1)' },   // Amber
  { border: '#3b82f6', bg: 'rgba(59, 130, 246, 0.1)' },   // Blue
  { border: '#ec4899', bg: 'rgba(236, 72, 153, 0.1)' },   // Pink
];

let refreshCountdown = 15;
let currentData = null;

document.addEventListener('DOMContentLoaded', async () => {
  initCharts();
  setupEvents();
  await loadDashboardData();
  startAutoRefresh();
});

function setupEvents() {
  selectNode.addEventListener('change', () => {
    loadDashboardData();
  });

  btnRefresh.addEventListener('click', () => {
    refreshCountdown = 15;
    loadDashboardData();
  });

  btnSeed.addEventListener('click', async () => {
    try {
      btnSeed.disabled = true;
      btnSeed.textContent = 'Gerando...';
      const res = await fetch('/api/v1/seed', { method: 'POST' });
      const data = await res.json();
      alert(`Sucesso: ${data.total_inserted} métricas simuladas foram geradas no MongoDB!`);
      await loadDashboardData();
    } catch (err) {
      alert('Erro ao gerar dados de teste: ' + err.message);
    } finally {
      btnSeed.disabled = false;
      btnSeed.textContent = '+ Gerar Test Data';
    }
  });
}

function startAutoRefresh() {
  setInterval(() => {
    refreshCountdown--;
    if (refreshCountdown <= 0) {
      refreshCountdown = 15;
      loadDashboardData();
    }
    autoRefreshTimer.textContent = `Auto: ${refreshCountdown}s`;
  }, 1000);
}

// Carrega dados da API do Central Server
async function loadDashboardData() {
  const selected = selectNode.value || 'all';
  let url = '/api/v1/dashboard/metrics';
  if (selected !== 'all') {
    url += `?node_id=${encodeURIComponent(selected)}`;
  }

  try {
    const res = await fetch(url);
    if (!res.ok) throw new Error(`HTTP error ${res.status}`);
    const data = await res.json();
    currentData = data;

    updateNodeSelector(data.nodes_list || []);
    renderSummaryCards(data.summaries || []);
    renderComparativeCharts(data.metrics_series || [], data.nodes_list || []);
    renderTelemetryTable(data.metrics_series || []);

    activeNodesCount.textContent = data.total_nodes || 0;
    if (data.generated_at) {
      lastIngestTime.textContent = `Atualizado: ${new Date(data.generated_at).toLocaleTimeString()}`;
    }
  } catch (err) {
    console.error('Falha ao carregar métricas:', err);
    // Se não houver servidor rodando ainda, exibe estado mock para navegação estática
    useFallbackMock();
  }
}

// Atualiza o seletor de nós dinamicamente
function updateNodeSelector(nodes) {
  const currentVal = selectNode.value;
  const existingOptions = Array.from(selectNode.options).map(o => o.value);

  nodes.forEach(node => {
    if (!existingOptions.includes(node)) {
      const opt = document.createElement('option');
      opt.value = node;
      opt.textContent = `🖥️ ${node}`;
      selectNode.appendChild(opt);
    }
  });

  selectNode.value = currentVal || 'all';
}

// Renderiza os cartões de resumo com última leitura de cada data center
function renderSummaryCards(summaries) {
  if (!summaries || summaries.length === 0) {
    nodesSummaryGrid.innerHTML = `
      <div class="card-loading">
        <p>Nenhuma sonda enviou métricas ainda.</p>
        <p class="text-subtle">Aguardando telemetria do Desktop Probe ou clique em <strong>"+ Gerar Test Data"</strong> acima.</p>
      </div>
    `;
    return;
  }

  nodesSummaryGrid.innerHTML = summaries.map(s => {
    const lastDate = new Date(s.last_seen);
    const timeAgo = formatTimeAgo(lastDate);
    const healthClass = s.is_healthy ? 'online' : 'stale';
    const healthText = s.is_healthy ? 'Ativo (1h)' : 'Sem sinal recente';

    return `
      <div class="dc-card">
        <div class="dc-card-top">
          <div class="dc-name-group">
            <h4>${s.node_id}</h4>
            <div class="dc-location">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 2a8 8 0 0 0-8 8c0 5.25 8 12 8 12s8-6.75 8-12a8 8 0 0 0-8-8z"></path>
                <circle cx="12" cy="10" r="3"></circle>
              </svg>
              <span>${s.location}</span>
            </div>
          </div>
          <span class="dc-health-badge ${healthClass}">● ${healthText}</span>
        </div>

        <div class="dc-metrics-row">
          <div class="metric-col">
            <div class="metric-lbl">Download</div>
            <div class="metric-val cyan">${s.last_download.toFixed(1)}</div>
            <div class="metric-unit">Mbps</div>
          </div>
          <div class="metric-col">
            <div class="metric-lbl">Upload</div>
            <div class="metric-val purple">${s.last_upload.toFixed(1)}</div>
            <div class="metric-unit">Mbps</div>
          </div>
          <div class="metric-col">
            <div class="metric-lbl">Ping Gateway</div>
            <div class="metric-val emerald">${s.last_ping.toFixed(1)}</div>
            <div class="metric-unit">ms</div>
          </div>
        </div>

        <div class="dc-footer">
          <span>Último sync: ${timeAgo}</span>
          <span>${s.total_readings_24h} leituras / 24h</span>
        </div>
      </div>
    `;
  }).join('');
}

// Inicializa instâncias do Chart.js
function initCharts() {
  const chartOpts = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: 'top',
        labels: {
          color: '#94a3b8',
          font: { size: 11 },
          boxWidth: 12
        }
      },
      tooltip: {
        backgroundColor: '#1e293b',
        titleColor: '#fff',
        bodyColor: '#e2e8f0',
        borderColor: 'rgba(255,255,255,0.1)',
        borderWidth: 1,
        padding: 10
      }
    },
    scales: {
      x: {
        grid: { color: 'rgba(255, 255, 255, 0.05)' },
        ticks: { color: '#64748b', font: { size: 10 } }
      },
      y: {
        grid: { color: 'rgba(255, 255, 255, 0.05)' },
        ticks: { color: '#64748b', font: { size: 10 } }
      }
    }
  };

  // 1. Bandwidth Chart
  const ctxBandwidth = document.getElementById('centralBandwidthChart').getContext('2d');
  bandwidthChart = new Chart(ctxBandwidth, {
    type: 'line',
    data: { labels: [], datasets: [] },
    options: {
      ...chartOpts,
      scales: {
        ...chartOpts.scales,
        y: {
          ...chartOpts.scales.y,
          title: { display: true, text: 'Mbps', color: '#94a3b8' }
        }
      }
    }
  });

  // 2. Latency Chart
  const ctxLatency = document.getElementById('centralLatencyChart').getContext('2d');
  latencyChart = new Chart(ctxLatency, {
    type: 'line',
    data: { labels: [], datasets: [] },
    options: {
      ...chartOpts,
      scales: {
        ...chartOpts.scales,
        y: {
          ...chartOpts.scales.y,
          title: { display: true, text: 'ms (Latência)', color: '#94a3b8' }
        }
      }
    }
  });
}

// Renderiza os gráficos comparativos agrupados por nó
function renderComparativeCharts(series, nodesList) {
  if (!series || series.length === 0) {
    bandwidthChart.data.labels = [];
    bandwidthChart.data.datasets = [];
    bandwidthChart.update();

    latencyChart.data.labels = [];
    latencyChart.data.datasets = [];
    latencyChart.update();
    return;
  }

  // Identifica timestamps únicos ordenados
  const timestamps = Array.from(new Set(series.map(s => s.timestamp))).sort();
  const labels = timestamps.map(t => {
    const d = new Date(t);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  });

  // Agrupa métricas por nó
  const byNode = {};
  series.forEach(item => {
    if (!byNode[item.NodeID || item.node_id]) {
      byNode[item.NodeID || item.node_id] = {};
    }
    byNode[item.NodeID || item.node_id][item.timestamp] = item;
  });

  const selectedNode = selectNode.value;
  const bandwidthDatasets = [];
  const latencyDatasets = [];

  const targetNodes = (selectedNode === 'all')
    ? Object.keys(byNode)
    : [selectedNode];

  targetNodes.forEach((nodeId, idx) => {
    const color = nodeColors[idx % nodeColors.length];
    const nodeDataMap = byNode[nodeId] || {};

    const downData = timestamps.map(t => nodeDataMap[t] ? nodeDataMap[t].download_mbps : null);
    const upData = timestamps.map(t => nodeDataMap[t] ? nodeDataMap[t].upload_mbps : null);
    const pingData = timestamps.map(t => nodeDataMap[t] ? nodeDataMap[t].ping_gateway : null);

    // Se estiver filtrado por 1 nó, mostra Download e Upload do nó
    if (selectedNode !== 'all') {
      bandwidthDatasets.push({
        label: `${nodeId} - Download`,
        data: downData,
        borderColor: '#06b6d4',
        backgroundColor: 'rgba(6, 182, 212, 0.1)',
        tension: 0.3,
        borderWidth: 2,
        pointRadius: 3
      });
      bandwidthDatasets.push({
        label: `${nodeId} - Upload`,
        data: upData,
        borderColor: '#a855f7',
        backgroundColor: 'rgba(168, 85, 247, 0.08)',
        tension: 0.3,
        borderWidth: 2,
        pointRadius: 3
      });
    } else {
      // Comparativo entre todos os nós (Download de cada um)
      bandwidthDatasets.push({
        label: `${nodeId} (Down)`,
        data: downData,
        borderColor: color.border,
        backgroundColor: color.bg,
        tension: 0.3,
        borderWidth: 2,
        pointRadius: 2
      });
    }

    // Latência do nó
    latencyDatasets.push({
      label: `${nodeId}`,
      data: pingData,
      borderColor: color.border,
      backgroundColor: color.bg,
      tension: 0.3,
      borderWidth: 2,
      pointRadius: 2
    });
  });

  // Atualiza Throughput
  bandwidthChart.data.labels = labels;
  bandwidthChart.data.datasets = bandwidthDatasets;
  bandwidthChart.update();

  // Atualiza Latência
  latencyChart.data.labels = labels;
  latencyChart.data.datasets = latencyDatasets;
  latencyChart.update();
}

// Renderiza a tabela de telemetria
function renderTelemetryTable(series) {
  if (!series || series.length === 0) {
    telemetryTableBody.innerHTML = '<tr><td colspan="7" class="text-center text-muted">Nenhuma telemetria recebida até o momento.</td></tr>';
    return;
  }

  // Mostra as últimas 15 leituras em ordem decrescente
  const recent = [...series].reverse().slice(0, 15);

  telemetryTableBody.innerHTML = recent.map(item => {
    const d = new Date(item.timestamp);
    const dateFormatted = `${d.toLocaleDateString()} ${d.toLocaleTimeString()}`;

    return `
      <tr>
        <td style="font-family: 'JetBrains Mono', monospace; font-size: 0.8rem;">${dateFormatted}</td>
        <td><span class="telemetry-node-tag">${item.node_id}</span></td>
        <td>${item.location}</td>
        <td><strong style="color: var(--accent-cyan);">${item.download_mbps.toFixed(1)}</strong> Mbps</td>
        <td><strong style="color: #c084fc;">${item.upload_mbps.toFixed(1)}</strong> Mbps</td>
        <td><strong style="color: #34d399;">${item.ping_gateway.toFixed(1)}</strong> ms</td>
        <td><span class="dc-health-badge online" style="display:inline-flex;">● Ingerido</span></td>
      </tr>
    `;
  }).join('');
}

function formatTimeAgo(date) {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 60) return 'há poucos segundos';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `há ${minutes} min`;
  const hours = Math.floor(minutes / 60);
  return `há ${hours}h`;
}

// Fallback visual caso a API ainda esteja iniciando
function useFallbackMock() {
  const nodes = ['minipc-dc-01', 'minipc-dc-02', 'minipc-dc-03'];
  const locations = {
    'minipc-dc-01': 'Guarapuava-DC1',
    'minipc-dc-02': 'Curitiba-DC2',
    'minipc-dc-03': 'SaoPaulo-DC3'
  };

  const summaries = nodes.map(id => ({
    node_id: id,
    location: locations[id],
    last_seen: new Date().toISOString(),
    last_download: id === 'minipc-dc-01' ? 850.5 : (id === 'minipc-dc-02' ? 920.0 : 780.0),
    last_upload: id === 'minipc-dc-01' ? 720.0 : (id === 'minipc-dc-02' ? 810.0 : 690.0),
    last_ping: id === 'minipc-dc-01' ? 1.2 : (id === 'minipc-dc-02' ? 3.4 : 8.5),
    is_healthy: true,
    total_readings_24h: 24
  }));

  updateNodeSelector(nodes);
  renderSummaryCards(summaries);
  activeNodesCount.textContent = nodes.length;
}
