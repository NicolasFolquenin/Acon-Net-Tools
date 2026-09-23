/**
 * Central NOC Dashboard - Web App Logic
 * Inclui: Autenticação com perfis, Inspeção de configurações do PC ao clicar no card,
 * Exclusão de computadores por Admin e remoção completa de dados simulados.
 */

let bandwidthChart = null;
let latencyChart = null;
let currentUser = null;
let currentSummaries = [];
let pendingDeleteNodeId = null;

// Elementos DOM principais
const selectNode = document.getElementById('select-node');
const btnRefresh = document.getElementById('btn-refresh');
const nodesSummaryGrid = document.getElementById('nodes-summary-grid');
const telemetryTableBody = document.getElementById('telemetry-table-body');
const activeNodesCount = document.getElementById('active-nodes-count');
const lastIngestTime = document.getElementById('last-ingest-time');
const autoRefreshTimer = document.getElementById('auto-refresh-timer');

// Elementos de Autenticação / Perfil
const loginModalOverlay = document.getElementById('login-modal-overlay');
const loginForm = document.getElementById('login-form');
const loginUsernameInput = document.getElementById('login-username');
const loginPasswordInput = document.getElementById('login-password');
const loginErrorAlert = document.getElementById('login-error-alert');
const loginErrorText = document.getElementById('login-error-text');
const btnLoginSubmit = document.getElementById('btn-login-submit');
const navUsername = document.getElementById('nav-username');
const navRoleBadge = document.getElementById('nav-role-badge');
const btnLogout = document.getElementById('btn-logout');

// Elementos do Modal de Configurações do PC
const pcModal = document.getElementById('pc-modal');
const modalPcClose = document.getElementById('modal-pc-close');
const modalBtnCloseBottom = document.getElementById('modal-btn-close-bottom');
const modalBtnDeletePc = document.getElementById('modal-btn-delete-pc');

// Elementos do Modal de Confirmação de Exclusão
const deleteConfirmModal = document.getElementById('delete-confirm-modal');
const deleteTargetNode = document.getElementById('delete-target-node');
const btnCancelDelete = document.getElementById('btn-cancel-delete');
const btnConfirmDelete = document.getElementById('btn-confirm-delete');

// Toast Container
const toastContainer = document.getElementById('toast-container');

// Paleta de cores para gráficos comparativos multi-nós (Gradientes e bordas de alta definição)
const nodeColors = [
  { border: '#06b6d4', fillTop: 'rgba(6, 182, 212, 0.28)', fillBottom: 'rgba(6, 182, 212, 0.01)' }, // Cyan
  { border: '#a855f7', fillTop: 'rgba(168, 85, 247, 0.22)', fillBottom: 'rgba(168, 85, 247, 0.01)' }, // Purple
  { border: '#10b981', fillTop: 'rgba(16, 185, 129, 0.25)', fillBottom: 'rgba(16, 185, 129, 0.01)' }, // Emerald
  { border: '#f59e0b', fillTop: 'rgba(245, 158, 11, 0.22)', fillBottom: 'rgba(245, 158, 11, 0.01)' }, // Amber
  { border: '#3b82f6', fillTop: 'rgba(59, 130, 246, 0.25)', fillBottom: 'rgba(59, 130, 246, 0.01)' }, // Blue
  { border: '#ec4899', fillTop: 'rgba(236, 72, 153, 0.22)', fillBottom: 'rgba(236, 72, 153, 0.01)' }, // Pink
];

let refreshCountdown = 15;
let refreshInterval = null;

// Controle de Métrica de Throughput (Download / Upload / Ambos)
let currentBandwidthMetric = 'download';
let cachedMetricsSeries = [];
let cachedNodesList = [];

window.handleMetricBtnClick = function(metric) {
  if (metric === currentBandwidthMetric) return;
  currentBandwidthMetric = metric;

  const toggleContainer = document.getElementById('bandwidth-metric-toggle');
  if (toggleContainer) {
    const buttons = toggleContainer.querySelectorAll('.segment-btn');
    buttons.forEach(b => {
      b.classList.toggle('active', b.getAttribute('data-metric') === metric);
    });
  }

  if (cachedMetricsSeries && cachedMetricsSeries.length > 0) {
    renderComparativeCharts(cachedMetricsSeries, cachedNodesList);
  } else {
    loadDashboardData();
  }
};

function setupSegmentedControl() {
  const toggleContainer = document.getElementById('bandwidth-metric-toggle');
  if (!toggleContainer) return;

  const buttons = toggleContainer.querySelectorAll('.segment-btn');
  buttons.forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.preventDefault();
      const metric = btn.getAttribute('data-metric');
      window.handleMetricBtnClick(metric);
    });
  });
}

function updateMetricToggleVisibility() {
  const selectedNode = selectNode.value || 'all';
  const btnBoth = document.getElementById('btn-metric-both');
  const toggleContainer = document.getElementById('bandwidth-metric-toggle');
  if (!btnBoth || !toggleContainer) return;

  if (selectedNode === 'all') {
    btnBoth.classList.add('hidden-metric');
    if (currentBandwidthMetric === 'both') {
      currentBandwidthMetric = 'download';
      const btns = toggleContainer.querySelectorAll('.segment-btn');
      btns.forEach(b => b.classList.toggle('active', b.getAttribute('data-metric') === 'download'));
    }
  } else {
    btnBoth.classList.remove('hidden-metric');
  }
}

// Inicialização
document.addEventListener('DOMContentLoaded', async () => {
  initTheme();
  initCharts();
  setupEventListeners();
  await checkAuthAndInitialize();
});

/* ==========================================================================
   1. Autenticação & Sessão
   ========================================================================== */

async function checkAuthAndInitialize() {
  try {
    const res = await fetch('/api/v1/auth/me');
    if (res.ok) {
      const data = await res.json();
      if (data.authenticated && data.user) {
        setAuthenticatedUser(data.user);
        await loadDashboardData();
        startAutoRefresh();
        return;
      }
    }
  } catch (err) {
    console.warn('Verificação de sessão:', err);
  }

  // Se não estiver logado, exibe a tela de login
  showLoginModal();
}

function setAuthenticatedUser(user) {
  currentUser = user;
  loginModalOverlay.style.display = 'none';

  navUsername.textContent = user.username;
  navRoleBadge.textContent = user.role === 'admin' ? 'Administrador' : 'Operador';
  navRoleBadge.className = `role-badge ${user.role}`;

  // Se for admin, o botão de deletar estará disponível
  if (user.role === 'admin') {
    modalBtnDeletePc.style.display = 'inline-flex';
  } else {
    modalBtnDeletePc.style.display = 'none';
  }
}

function showLoginModal() {
  currentUser = null;
  loginModalOverlay.style.display = 'flex';
  navUsername.textContent = 'Não conectado';
  navRoleBadge.textContent = '--';
  navRoleBadge.className = 'role-badge';
  loginErrorAlert.style.display = 'none';
  loginUsernameInput.value = '';
  loginPasswordInput.value = '';
  if (refreshInterval) clearInterval(refreshInterval);
}

async function handleLoginSubmit(e) {
  e.preventDefault();
  const username = loginUsernameInput.value.trim();
  const password = loginPasswordInput.value.trim();

  if (!username || !password) return;

  btnLoginSubmit.disabled = true;
  loginErrorAlert.style.display = 'none';

  try {
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });

    const data = await res.json();

    if (!res.ok) {
      throw new Error(data.error || 'Falha ao autenticar');
    }

    setAuthenticatedUser(data.user);
    showToast(`Bem-vindo, ${data.user.username}!`, 'toast-success');
    await loadDashboardData();
    startAutoRefresh();
  } catch (err) {
    loginErrorText.textContent = err.message || 'Usuário ou senha incorretos';
    loginErrorAlert.style.display = 'flex';
  } finally {
    btnLoginSubmit.disabled = false;
  }
}

async function handleLogout() {
  try {
    await fetch('/api/v1/auth/logout', { method: 'POST' });
  } catch (err) {
    console.error('Erro ao sair:', err);
  }
  showToast('Sessão encerrada com sucesso', 'toast-info');
  showLoginModal();
}

/* ==========================================================================
   Tema Claro / Escuro
   ========================================================================== */

function initTheme() {
  const savedTheme = localStorage.getItem('noc-theme') ||
    (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark');
  applyTheme(savedTheme, false);

  const btnTheme = document.getElementById('btn-theme-toggle');
  if (btnTheme) {
    btnTheme.addEventListener('click', () => {
      const cur = document.documentElement.getAttribute('data-theme') || 'dark';
      const next = cur === 'dark' ? 'light' : 'dark';
      applyTheme(next, true);
    });
  }
}

function applyTheme(theme, notify = false) {
  document.documentElement.setAttribute('data-theme', theme);
  localStorage.setItem('noc-theme', theme);
  updateChartsTheme(theme);
  if (notify) {
    showToast(`Tema alternado para Modo ${theme === 'dark' ? 'Escuro' : 'Claro'}`, 'toast-info');
  }
}

/* ==========================================================================
   2. Configuração de Eventos
   ========================================================================== */

function setupEventListeners() {
  loginForm.addEventListener('submit', handleLoginSubmit);
  btnLogout.addEventListener('click', handleLogout);

  selectNode.addEventListener('change', () => {
    updateMetricToggleVisibility();
    loadDashboardData();
  });

  setupSegmentedControl();

  btnRefresh.addEventListener('click', () => {
    refreshCountdown = 15;
    loadDashboardData();
  });

  // Fechamento do Modal de PC
  modalPcClose.addEventListener('click', closePCModal);
  modalBtnCloseBottom.addEventListener('click', closePCModal);
  pcModal.addEventListener('click', (e) => {
    if (e.target === pcModal) closePCModal();
  });

  // Botão de Excluir dentro do Modal de PC
  modalBtnDeletePc.addEventListener('click', () => {
    const nodeId = modalBtnDeletePc.getAttribute('data-node-id');
    if (nodeId) promptDeleteNode(nodeId);
  });

  // Confirmação de Exclusão
  btnCancelDelete.addEventListener('click', closeDeleteModal);
  deleteConfirmModal.addEventListener('click', (e) => {
    if (e.target === deleteConfirmModal) closeDeleteModal();
  });
  btnConfirmDelete.addEventListener('click', executeDeleteNode);

  // Fechar modais ao pressionar tecla ESC
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      if (deleteConfirmModal.style.display !== 'none') {
        closeDeleteModal();
      } else if (pcModal.style.display !== 'none') {
        closePCModal();
      }
    }
  });
}

function startAutoRefresh() {
  if (refreshInterval) clearInterval(refreshInterval);
  refreshInterval = setInterval(() => {
    refreshCountdown--;
    if (refreshCountdown <= 0) {
      refreshCountdown = 15;
      loadDashboardData();
    }
    autoRefreshTimer.textContent = `Auto: ${refreshCountdown}s`;
  }, 1000);
}

/* ==========================================================================
   3. Carregamento de Métricas do Dashboard
   ========================================================================== */

async function loadDashboardData() {
  if (!currentUser) return;

  const selected = selectNode.value || 'all';
  let url = '/api/v1/dashboard/metrics';
  if (selected !== 'all') {
    url += `?node_id=${encodeURIComponent(selected)}`;
  }

  try {
    const res = await fetch(url);
    if (res.status === 401) {
      showLoginModal();
      return;
    }
    if (!res.ok) throw new Error(`HTTP error ${res.status}`);

    const data = await res.json();
    currentSummaries = data.summaries || [];

    updateNodeSelector(data.nodes_list || []);
    renderSummaryCards(currentSummaries);
    renderComparativeCharts(data.metrics_series || [], data.nodes_list || []);
    renderTelemetryTable(data.metrics_series || []);

    activeNodesCount.textContent = data.total_nodes || 0;
    if (data.generated_at) {
      lastIngestTime.textContent = `Atualizado: ${new Date(data.generated_at).toLocaleTimeString()}`;
    }
  } catch (err) {
    console.error('Falha ao carregar métricas:', err);
    nodesSummaryGrid.innerHTML = `
      <div class="card-loading">
        <p class="text-rose font-bold">Falha ao conectar com o servidor central.</p>
        <p class="text-subtle">${err.message}</p>
      </div>
    `;
  }
}

function updateNodeSelector(nodes) {
  const currentVal = selectNode.value;
  selectNode.innerHTML = '<option value="all">⚡ Todos os Nós (Consolidado)</option>';

  nodes.forEach(node => {
    const opt = document.createElement('option');
    opt.value = node;
    opt.textContent = `🖥️ ${node}`;
    selectNode.appendChild(opt);
  });

  selectNode.value = nodes.includes(currentVal) ? currentVal : 'all';
  updateMetricToggleVisibility();
}

/* ==========================================================================
   4. Renderização dos Cards dos Computadores (com clique interativo)
   ========================================================================== */

function renderSummaryCards(summaries) {
  if (!summaries || summaries.length === 0) {
    nodesSummaryGrid.innerHTML = `
      <div class="card-loading">
        <div style="font-size: 2rem; margin-bottom: 4px;">📡</div>
        <p class="font-bold text-white">Nenhum computador ou sonda registrada ainda.</p>
        <p class="text-subtle">Aguardando a primeira telemetria enviada pelo agente UEPG-Net-Tools instalado nas máquinas.</p>
      </div>
    `;
    return;
  }

  const isAdmin = currentUser && currentUser.role === 'admin';

  nodesSummaryGrid.innerHTML = summaries.map(s => {
    const lastDate = new Date(s.last_seen);
    const timeAgo = formatTimeAgo(lastDate);
    const healthClass = s.is_healthy ? 'online' : 'stale';
    const healthText = s.is_healthy ? 'Ativo (1h)' : 'Sem sinal recente';

    return `
      <div class="dc-card" data-node-id="${escapeHtml(s.node_id)}" title="Clique para ver as configurações detalhadas">
        <div class="dc-card-top">
          <div class="dc-name-group">
            <h4>${escapeHtml(s.node_id)}</h4>
            <div class="dc-location">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 2a8 8 0 0 0-8 8c0 5.25 8 12 8 12s8-6.75 8-12a8 8 0 0 0-8-8z"></path>
                <circle cx="12" cy="10" r="3"></circle>
              </svg>
              <span>${escapeHtml(s.location)}</span>
            </div>
          </div>
          <div class="dc-card-top-actions">
            <span class="dc-health-badge ${healthClass}">● ${healthText}</span>
            ${isAdmin ? `
              <button class="card-quick-delete" data-node-id="${escapeHtml(s.node_id)}" title="Excluir computador do sistema">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"></polyline>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                </svg>
              </button>
            ` : ''}
          </div>
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

        ${s.system_info || s.switch_info ? `
        <div class="dc-hardware-box">
          ${s.system_info ? `
          <div class="hw-line">
            <span class="hw-tag">HOST</span>
            <span class="hw-desc" title="${escapeHtml(s.system_info.cpu || '')}">${escapeHtml(s.system_info.model || s.system_info.hostname || 'PC')} (${s.system_info.ram_total_gb ? s.system_info.ram_total_gb.toFixed(0) + 'GB' : '--'}) • IP: ${escapeHtml(s.system_info.ip_address || '--')}</span>
          </div>` : ''}
          ${s.switch_info && s.switch_info.switch_name ? `
          <div class="hw-line">
            <span class="hw-tag switch">LLDP</span>
            <span class="hw-desc font-bold text-cyan" title="${escapeHtml(s.switch_info.switch_model || '')}">SW: ${escapeHtml(s.switch_info.switch_name)} [${escapeHtml(s.switch_info.switch_port || '--')}] ${s.switch_info.vlan ? '• VLAN ' + s.switch_info.vlan : ''}</span>
          </div>` : ''}
        </div>` : ''}

        <div class="dc-footer">
          <span>Último sync: ${timeAgo}</span>
          <span style="color: var(--accent-cyan); font-weight: 500;">🔍 Ver configs &raquo;</span>
        </div>
      </div>
    `;
  }).join('');

  // Atribui evento de clique em cada card para abrir o modal de configurações
  nodesSummaryGrid.querySelectorAll('.dc-card').forEach(card => {
    card.addEventListener('click', (e) => {
      // Se clicou no botão de exclusão rápida, não abre o modal de detalhes
      if (e.target.closest('.card-quick-delete')) {
        e.stopPropagation();
        const nodeId = e.target.closest('.card-quick-delete').getAttribute('data-node-id');
        promptDeleteNode(nodeId);
        return;
      }

      const nodeId = card.getAttribute('data-node-id');
      const summary = currentSummaries.find(item => item.node_id === nodeId);
      if (summary) {
        openPCDetailsModal(summary);
      }
    });
  });
}

/* ==========================================================================
   5. Modal de Detalhes & Configurações do Computador
   ========================================================================== */

function openPCDetailsModal(s) {
  // Cabeçalho
  document.getElementById('modal-pc-nodeid').textContent = s.node_id;
  document.getElementById('modal-pc-location').textContent = `Localização: ${s.location || 'Não informada'}`;

  const healthBadge = document.getElementById('modal-pc-health');
  healthBadge.className = `dc-health-badge ${s.is_healthy ? 'online' : 'stale'}`;
  healthBadge.textContent = s.is_healthy ? '● Online (Última hora)' : '● Sem sinal recente';

  // 1. Métricas
  document.getElementById('modal-pc-down').textContent = `${s.last_download.toFixed(1)} Mbps`;
  document.getElementById('modal-pc-up').textContent = `${s.last_upload.toFixed(1)} Mbps`;
  document.getElementById('modal-pc-ping').textContent = `${s.last_ping.toFixed(1)} ms`;
  document.getElementById('modal-pc-readings').textContent = `${s.total_readings_24h} medições`;

  // 2. Hardware e SO
  const sys = s.system_info || {};
  document.getElementById('modal-hw-hostname').textContent = sys.hostname || '--';
  document.getElementById('modal-hw-model').textContent = sys.model || 'Mini PC / Computador Genérico';
  document.getElementById('modal-hw-cpu').textContent = sys.cpu || '--';
  document.getElementById('modal-hw-ram').textContent = sys.ram_total_gb ? `${sys.ram_total_gb.toFixed(1)} GB` : '--';
  document.getElementById('modal-hw-os').textContent = sys.os || '--';

  // 3. Rede Local
  document.getElementById('modal-net-ip').textContent = sys.ip_address || '--';
  document.getElementById('modal-net-mac').textContent = sys.mac_address || '--';
  document.getElementById('modal-net-gateway').textContent = sys.gateway || '--';
  document.getElementById('modal-net-adapter').textContent = sys.network_adapter || '--';

  // 4. Switch & LLDP
  const sw = s.switch_info || {};
  document.getElementById('modal-lldp-switch').textContent = sw.switch_name || 'Não descoberto via LLDP';
  document.getElementById('modal-lldp-port').textContent = sw.switch_port || '--';
  document.getElementById('modal-lldp-vlan').textContent = sw.vlan ? `VLAN ${sw.vlan}` : '--';
  document.getElementById('modal-lldp-model').textContent = sw.switch_model || '--';
  document.getElementById('modal-lldp-proto').textContent = sw.protocol ? `${sw.protocol} (${sw.discovered_at ? new Date(sw.discovered_at).toLocaleTimeString() : ''})` : '--';

  // Permissão de exclusão
  if (currentUser && currentUser.role === 'admin') {
    modalBtnDeletePc.style.display = 'inline-flex';
    modalBtnDeletePc.setAttribute('data-node-id', s.node_id);
  } else {
    modalBtnDeletePc.style.display = 'none';
  }

  pcModal.style.display = 'flex';
}

function closePCModal() {
  pcModal.style.display = 'none';
}

/* ==========================================================================
   6. Exclusão Administrativa de Computadores
   ========================================================================== */

function promptDeleteNode(nodeId) {
  pendingDeleteNodeId = nodeId;
  deleteTargetNode.textContent = nodeId;
  deleteConfirmModal.style.display = 'flex';
}

function closeDeleteModal() {
  pendingDeleteNodeId = null;
  deleteConfirmModal.style.display = 'none';
}

async function executeDeleteNode() {
  if (!pendingDeleteNodeId) return;

  const nodeId = pendingDeleteNodeId;
  btnConfirmDelete.disabled = true;
  btnConfirmDelete.innerHTML = '<span>Excluindo...</span>';

  try {
    const res = await fetch(`/api/v1/nodes?node_id=${encodeURIComponent(nodeId)}`, {
      method: 'DELETE'
    });

    const data = await res.json();

    if (!res.ok) {
      throw new Error(data.error || 'Falha ao excluir computador');
    }

    closeDeleteModal();
    closePCModal();

    showToast(`Computador "${nodeId}" foi excluído com sucesso!`, 'toast-success');

    // Se o nó excluído era o filtro selecionado, reseta para "all"
    if (selectNode.value === nodeId) {
      selectNode.value = 'all';
    }

    await loadDashboardData();
  } catch (err) {
    showToast(`Erro ao excluir: ${err.message}`, 'toast-error');
  } finally {
    btnConfirmDelete.disabled = false;
    btnConfirmDelete.innerHTML = `
      <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="3 6 5 6 21 6"></polyline>
        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
      </svg>
      <span>Sim, Excluir Definitivamente</span>
    `;
  }
}

/* ==========================================================================
   7. Gráficos Comparativos Chart.js (Com Gradientes, Curvas Suaves e Temas)
   ========================================================================== */

function getChartThemeColors(theme) {
  const isLight = theme === 'light';
  return {
    textColor: isLight ? '#475569' : '#94a3b8',
    gridColor: isLight ? 'rgba(0, 0, 0, 0.07)' : 'rgba(255, 255, 255, 0.06)',
    tooltipBg: isLight ? '#ffffff' : '#0f172a',
    tooltipTitle: isLight ? '#0f172a' : '#ffffff',
    tooltipBody: isLight ? '#334155' : '#e2e8f0',
    tooltipBorder: isLight ? 'rgba(0, 0, 0, 0.12)' : 'rgba(255, 255, 255, 0.12)'
  };
}

function updateChartsTheme(theme) {
  if (!bandwidthChart || !latencyChart) return;
  const currentTheme = theme || document.documentElement.getAttribute('data-theme') || 'dark';
  const colors = getChartThemeColors(currentTheme);

  [bandwidthChart, latencyChart].forEach(chart => {
    if (!chart) return;
    chart.options.scales.x.grid.color = colors.gridColor;
    chart.options.scales.x.ticks.color = colors.textColor;
    chart.options.scales.y.grid.color = colors.gridColor;
    chart.options.scales.y.ticks.color = colors.textColor;
    if (chart.options.scales.y.title) {
      chart.options.scales.y.title.color = colors.textColor;
    }
    if (chart.options.plugins.legend) {
      chart.options.plugins.legend.labels.color = colors.textColor;
    }
    if (chart.options.plugins.tooltip) {
      chart.options.plugins.tooltip.backgroundColor = colors.tooltipBg;
      chart.options.plugins.tooltip.titleColor = colors.tooltipTitle;
      chart.options.plugins.tooltip.bodyColor = colors.tooltipBody;
      chart.options.plugins.tooltip.borderColor = colors.tooltipBorder;
    }
    chart.update('none');
  });
}

function createFillGradient(ctx, colorTop, colorBottom, height = 280) {
  const grad = ctx.createLinearGradient(0, 0, 0, height);
  grad.addColorStop(0, colorTop);
  grad.addColorStop(1, colorBottom);
  return grad;
}

function initCharts() {
  const currentTheme = document.documentElement.getAttribute('data-theme') || 'dark';
  const colors = getChartThemeColors(currentTheme);

  const chartOpts = {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      mode: 'index',
      intersect: false
    },
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: colors.tooltipBg,
        titleColor: colors.tooltipTitle,
        bodyColor: colors.tooltipBody,
        borderColor: colors.tooltipBorder,
        borderWidth: 1,
        padding: 12,
        cornerRadius: 8,
        titleFont: { size: 12, weight: '700' },
        bodyFont: { size: 12 },
        displayColors: true,
        boxWidth: 8,
        boxHeight: 8,
        usePointStyle: true
      }
    },
    scales: {
      x: {
        grid: { color: colors.gridColor, drawBorder: false },
        ticks: {
          color: colors.textColor,
          font: { size: 10, family: "'JetBrains Mono', monospace" },
          maxTicksLimit: 8,
          maxRotation: 0,
          autoSkip: true
        }
      },
      y: {
        grid: { color: colors.gridColor, drawBorder: false },
        ticks: { color: colors.textColor, font: { size: 10, family: "'JetBrains Mono', monospace" } }
      }
    }
  };

  const ctxBandwidth = document.getElementById('centralBandwidthChart').getContext('2d');
  bandwidthChart = new Chart(ctxBandwidth, {
    type: 'line',
    data: { labels: [], datasets: [] },
    options: {
      ...chartOpts,
      plugins: {
        ...chartOpts.plugins,
        tooltip: {
          ...chartOpts.plugins.tooltip,
          callbacks: {
            label: (ctx) => ` ${ctx.dataset.label}: ${ctx.parsed.y != null ? ctx.parsed.y.toFixed(1) + ' Mbps' : '--'}`
          }
        }
      },
      scales: {
        ...chartOpts.scales,
        y: {
          ...chartOpts.scales.y,
          title: { display: true, text: 'Mbps', color: colors.textColor, font: { size: 11, weight: '600' } }
        }
      }
    }
  });

  const ctxLatency = document.getElementById('centralLatencyChart').getContext('2d');
  latencyChart = new Chart(ctxLatency, {
    type: 'line',
    data: { labels: [], datasets: [] },
    options: {
      ...chartOpts,
      plugins: {
        ...chartOpts.plugins,
        tooltip: {
          ...chartOpts.plugins.tooltip,
          callbacks: {
            label: (ctx) => ` ${ctx.dataset.label}: ${ctx.parsed.y != null ? ctx.parsed.y.toFixed(1) + ' ms' : '--'}`
          }
        }
      },
      scales: {
        ...chartOpts.scales,
        y: {
          ...chartOpts.scales.y,
          title: { display: true, text: 'ms', color: colors.textColor, font: { size: 11, weight: '600' } }
        }
      }
    }
  });
}

function renderComparativeCharts(series, nodesList) {
  cachedMetricsSeries = series || [];
  cachedNodesList = nodesList || [];

  updateMetricToggleVisibility();

  const bandwidthLegendEl = document.getElementById('bandwidth-legend-container');
  const latencyLegendEl = document.getElementById('latency-legend-container');
  const bandwidthChartTitle = document.getElementById('bandwidth-chart-title');
  const bandwidthChartDesc = document.getElementById('bandwidth-chart-desc');

  if (!series || series.length === 0) {
    bandwidthChart.data.labels = [];
    bandwidthChart.data.datasets = [];
    bandwidthChart.update();

    latencyChart.data.labels = [];
    latencyChart.data.datasets = [];
    latencyChart.update();
    return;
  }

  const selectedNode = selectNode.value;
  const ctxBandwidth = document.getElementById('centralBandwidthChart').getContext('2d');
  const ctxLatency = document.getElementById('centralLatencyChart').getContext('2d');

  // Modo Sonda Individual: Foco nas métricas dessa sonda
  if (selectedNode !== 'all') {
    const allNodeItems = series
      .filter(s => (s.node_id || s.NodeID) === selectedNode)
      .sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));

    // Limita estritamente aos últimos 20 testes
    const nodeItems = allNodeItems.slice(-20);

    const nodeLabels = nodeItems.map(item => {
      const d = new Date(item.timestamp);
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    });

    const downData = nodeItems.map(item => item.download_mbps);
    const upData = nodeItems.map(item => item.upload_mbps);
    const pingData = nodeItems.map(item => item.ping_gateway);

    const bandwidthDatasets = [];

    if (currentBandwidthMetric === 'download') {
      if (bandwidthChartTitle) bandwidthChartTitle.textContent = `Throughput - Download (${selectedNode})`;
      if (bandwidthChartDesc) bandwidthChartDesc.textContent = 'Taxa de Download dos últimos 20 testes (Mbps)';
      if (bandwidthLegendEl) {
        bandwidthLegendEl.innerHTML = `<span class="legend-badge cyan-dot">Download</span>`;
      }
      bandwidthDatasets.push({
        label: 'Download',
        data: downData,
        borderColor: '#06b6d4',
        backgroundColor: createFillGradient(ctxBandwidth, 'rgba(6, 182, 212, 0.28)', 'rgba(6, 182, 212, 0.01)'),
        fill: true,
        tension: 0.38,
        cubicInterpolationMode: 'monotone',
        spanGaps: true,
        borderWidth: 2.5,
        pointRadius: 3,
        pointHoverRadius: 6,
        pointBackgroundColor: '#06b6d4',
        pointBorderColor: '#ffffff',
        pointBorderWidth: 1.5
      });
    } else if (currentBandwidthMetric === 'upload') {
      if (bandwidthChartTitle) bandwidthChartTitle.textContent = `Throughput - Upload (${selectedNode})`;
      if (bandwidthChartDesc) bandwidthChartDesc.textContent = 'Taxa de Upload dos últimos 20 testes (Mbps)';
      if (bandwidthLegendEl) {
        bandwidthLegendEl.innerHTML = `<span class="legend-badge purple-dot">Upload</span>`;
      }
      bandwidthDatasets.push({
        label: 'Upload',
        data: upData,
        borderColor: '#a855f7',
        backgroundColor: createFillGradient(ctxBandwidth, 'rgba(168, 85, 247, 0.22)', 'rgba(168, 85, 247, 0.01)'),
        fill: true,
        tension: 0.38,
        cubicInterpolationMode: 'monotone',
        spanGaps: true,
        borderWidth: 2.5,
        pointRadius: 3,
        pointHoverRadius: 6,
        pointBackgroundColor: '#a855f7',
        pointBorderColor: '#ffffff',
        pointBorderWidth: 1.5
      });
    } else {
      // Modo Ambos
      if (bandwidthChartTitle) bandwidthChartTitle.textContent = `Throughput - Comparativo (${selectedNode})`;
      if (bandwidthChartDesc) bandwidthChartDesc.textContent = 'Taxas de Download e Upload dos últimos 20 testes (Mbps)';
      if (bandwidthLegendEl) {
        bandwidthLegendEl.innerHTML = `
          <span class="legend-badge cyan-dot">Download</span>
          <span class="legend-badge purple-dot">Upload</span>
        `;
      }
      bandwidthDatasets.push(
        {
          label: 'Download',
          data: downData,
          borderColor: '#06b6d4',
          backgroundColor: createFillGradient(ctxBandwidth, 'rgba(6, 182, 212, 0.28)', 'rgba(6, 182, 212, 0.01)'),
          fill: true,
          tension: 0.38,
          cubicInterpolationMode: 'monotone',
          spanGaps: true,
          borderWidth: 2.5,
          pointRadius: 3,
          pointHoverRadius: 6,
          pointBackgroundColor: '#06b6d4',
          pointBorderColor: '#ffffff',
          pointBorderWidth: 1.5
        },
        {
          label: 'Upload',
          data: upData,
          borderColor: '#a855f7',
          backgroundColor: createFillGradient(ctxBandwidth, 'rgba(168, 85, 247, 0.22)', 'rgba(168, 85, 247, 0.01)'),
          fill: true,
          tension: 0.38,
          cubicInterpolationMode: 'monotone',
          spanGaps: true,
          borderWidth: 2.5,
          pointRadius: 3,
          pointHoverRadius: 6,
          pointBackgroundColor: '#a855f7',
          pointBorderColor: '#ffffff',
          pointBorderWidth: 1.5
        }
      );
    }

    bandwidthChart.data.labels = nodeLabels;
    bandwidthChart.data.datasets = bandwidthDatasets;
    bandwidthChart.update();

    if (latencyLegendEl) {
      latencyLegendEl.innerHTML = `<span class="legend-badge emerald-dot">Ping Gateway</span>`;
    }

    latencyChart.data.labels = nodeLabels;
    latencyChart.data.datasets = [
      {
        label: 'Ping Gateway',
        data: pingData,
        borderColor: '#10b981',
        backgroundColor: createFillGradient(ctxLatency, 'rgba(16, 185, 129, 0.25)', 'rgba(16, 185, 129, 0.01)'),
        fill: true,
        tension: 0.38,
        cubicInterpolationMode: 'monotone',
        spanGaps: true,
        borderWidth: 2.5,
        pointRadius: 3,
        pointHoverRadius: 6,
        pointBackgroundColor: '#10b981',
        pointBorderColor: '#ffffff',
        pointBorderWidth: 1.5
      }
    ];
    latencyChart.update();
    return;
  }

  // Modo Consolidado (Todos os Nós):
  // Agrupa os timestamps por minuto para eliminar timestamps duplicados no eixo X
  const minuteMap = new Map();
  const byNode = {};

  series.forEach(item => {
    const id = item.node_id || item.NodeID;
    if (!byNode[id]) byNode[id] = {};

    const d = new Date(item.timestamp);
    d.setSeconds(0, 0);
    const minuteKey = d.getTime();

    if (!minuteMap.has(minuteKey)) {
      minuteMap.set(minuteKey, d);
    }
    byNode[id][minuteKey] = item;
  });

  const allMinuteKeys = Array.from(minuteMap.keys()).sort((a, b) => a - b);
  // Limita estritamente aos últimos 20 testes no consolidado
  const sortedMinuteKeys = allMinuteKeys.slice(-20);

  const labels = sortedMinuteKeys.map(k => {
    const d = minuteMap.get(k);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  });

  const targetNodes = Object.keys(byNode).sort();

  const isUploadMode = currentBandwidthMetric === 'upload';
  const metricLabel = isUploadMode ? 'Upload' : 'Download';

  if (bandwidthChartTitle) {
    bandwidthChartTitle.textContent = `Histórico de Throughput (${metricLabel})`;
  }
  if (bandwidthChartDesc) {
    bandwidthChartDesc.textContent = `Taxas de ${metricLabel} por Computador (últimos 20 testes em Mbps)`;
  }

  // Atualiza as legendas HTML com bolinhas sólidas coloridas
  if (bandwidthLegendEl) {
    bandwidthLegendEl.innerHTML = targetNodes.map((nodeId, idx) => {
      const color = nodeColors[idx % nodeColors.length];
      return `<span class="legend-badge"><span class="badge-dot" style="background:${color.border};"></span>${escapeHtml(nodeId)}</span>`;
    }).join('');
  }

  if (latencyLegendEl) {
    latencyLegendEl.innerHTML = targetNodes.map((nodeId, idx) => {
      const color = nodeColors[idx % nodeColors.length];
      return `<span class="legend-badge"><span class="badge-dot" style="background:${color.border};"></span>${escapeHtml(nodeId)}</span>`;
    }).join('');
  }

  const bandwidthDatasets = [];
  const latencyDatasets = [];

  targetNodes.forEach((nodeId, idx) => {
    const color = nodeColors[idx % nodeColors.length];
    const nodeDataMap = byNode[nodeId] || {};

    const metricData = sortedMinuteKeys.map(k => {
      if (!nodeDataMap[k]) return null;
      return isUploadMode ? nodeDataMap[k].upload_mbps : nodeDataMap[k].download_mbps;
    });
    const pingData = sortedMinuteKeys.map(k => nodeDataMap[k] ? nodeDataMap[k].ping_gateway : null);

    bandwidthDatasets.push({
      label: `${nodeId} (${metricLabel})`,
      data: metricData,
      borderColor: color.border,
      backgroundColor: createFillGradient(ctxBandwidth, color.fillTop, color.fillBottom),
      fill: true,
      tension: 0.38,
      cubicInterpolationMode: 'monotone',
      spanGaps: true,
      borderWidth: 2.5,
      pointRadius: 3,
      pointHoverRadius: 6,
      pointBackgroundColor: color.border,
      pointBorderColor: '#ffffff',
      pointBorderWidth: 1.5
    });

    latencyDatasets.push({
      label: `${nodeId}`,
      data: pingData,
      borderColor: color.border,
      backgroundColor: createFillGradient(ctxLatency, color.fillTop, color.fillBottom),
      fill: true,
      tension: 0.38,
      cubicInterpolationMode: 'monotone',
      spanGaps: true,
      borderWidth: 2.5,
      pointRadius: 3,
      pointHoverRadius: 6,
      pointBackgroundColor: color.border,
      pointBorderColor: '#ffffff',
      pointBorderWidth: 1.5
    });
  });

  bandwidthChart.data.labels = labels;
  bandwidthChart.data.datasets = bandwidthDatasets;
  bandwidthChart.update();

  latencyChart.data.labels = labels;
  latencyChart.data.datasets = latencyDatasets;
  latencyChart.update();
}

/* ==========================================================================
   8. Tabela de Feed de Telemetria
   ========================================================================== */

function renderTelemetryTable(series) {
  if (!series || series.length === 0) {
    telemetryTableBody.innerHTML = '<tr><td colspan="7" class="text-center text-muted">Nenhuma telemetria recebida até o momento.</td></tr>';
    return;
  }

  const recent = [...series].reverse().slice(0, 15);

  telemetryTableBody.innerHTML = recent.map(item => {
    const d = new Date(item.timestamp);
    const dateFormatted = `${d.toLocaleDateString()} ${d.toLocaleTimeString()}`;
    const id = item.node_id || item.NodeID;

    return `
      <tr>
        <td style="font-family: 'JetBrains Mono', monospace; font-size: 0.8rem;">${dateFormatted}</td>
        <td><span class="telemetry-node-tag">${escapeHtml(id)}</span></td>
        <td>${escapeHtml(item.location || 'Desconhecido')}</td>
        <td><strong style="color: var(--accent-cyan);">${item.download_mbps.toFixed(1)}</strong> Mbps</td>
        <td><strong style="color: #c084fc;">${item.upload_mbps.toFixed(1)}</strong> Mbps</td>
        <td><strong style="color: #34d399;">${item.ping_gateway.toFixed(1)}</strong> ms</td>
        <td><span class="dc-health-badge online" style="display:inline-flex;">● Ingerido</span></td>
      </tr>
    `;
  }).join('');
}

/* ==========================================================================
   9. Utilitários & Toasts (Snappy)
   ========================================================================== */

function formatTimeAgo(date) {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 60) return 'há poucos segundos';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `há ${minutes} min`;
  const hours = Math.floor(minutes / 60);
  return `há ${hours}h`;
}

function escapeHtml(text) {
  if (!text) return '';
  return String(text)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

function showToast(message, type = 'toast-info') {
  const toast = document.createElement('div');
  toast.className = `toast ${type}`;

  let icon = 'ℹ️';
  if (type === 'toast-success') icon = '✅';
  if (type === 'toast-error') icon = '⚠️';

  toast.innerHTML = `<span>${icon}</span><span>${escapeHtml(message)}</span>`;
  toastContainer.appendChild(toast);

  setTimeout(() => {
    toast.remove();
  }, 3500);
}
