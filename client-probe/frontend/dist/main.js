/**
 * Client Probe - Desktop Wails Frontend Logic
 */

let throughputChart = null;
let latencyChart = null;
let packetLossChart = null;

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

// Elementos de Paginação e Filtro por Data da Tabela (15 itens por página)
const historyDateFilter = document.getElementById('history-date-filter');
const btnFilterToday = document.getElementById('btn-filter-today');
const btnFilterAll = document.getElementById('btn-filter-all');
const historyTotalBadge = document.getElementById('history-total-badge');
const paginationRangeText = document.getElementById('pagination-range-text');
const btnPrevPage = document.getElementById('btn-prev-page');
const btnNextPage = document.getElementById('btn-next-page');
const currentPageDisplay = document.getElementById('current-page-display');
const totalPagesDisplay = document.getElementById('total-pages-display');

// Estado de Paginação
const historyPagination = {
  currentPage: 1,
  pageSize: 15,
  dayFilter: '',
  totalCount: 0,
  totalPages: 1
};

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

  async getActiveProfile() {
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetActiveProfile) {
      return await window.go.main.App.GetActiveProfile();
    }
    return null;
  },

  async getHistory() {
    if (window.go && window.go.main && window.go.main.App) {
      return await window.go.main.App.GetHistory();
    }
    // Mock para desenvolvimento/preview no navegador
    return generateMockHistory(24);
  },

  async getHistoryPage(dayFilter = '', page = 1, pageSize = 15) {
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetHistoryPage) {
      return await window.go.main.App.GetHistoryPage(dayFilter, page, pageSize);
    }
    // Mock para desenvolvimento/preview no navegador
    return generateMockHistoryPage(dayFilter, page, pageSize);
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
      server_url: 'http://localhost:8090',
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
  },

  async checkUpdate() {
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.CheckUpdateNow) {
      return await window.go.main.App.CheckUpdateNow();
    }
    return {
      updated: false,
      current_version: '2.4.1',
      message: 'Sonda já está operando na versão mais recente (v2.4.1).'
    };
  }
};

// Inicialização da aplicação com suporte a carregamento imediato ou diferido (resiliente para WebViews Wails)
async function initApp() {
  console.log('[PROBE] Inicializando aplicação...');
  try { initTheme(); } catch (e) { console.error('Erro initTheme:', e); }
  try { setupEventListeners(); } catch (e) { console.error('Erro setupEventListeners:', e); }
  try { initCharts(); } catch (e) { console.error('Erro initCharts:', e); }
  try { await loadAppConfig(); } catch (e) { console.error('Erro loadAppConfig:', e); }
  try { await refreshDashboard(); } catch (e) { console.error('Erro refreshDashboard:', e); }

  // Escuta eventos Wails se disponíveis
  if (window.runtime) {
    window.runtime.EventsOn('status_updated', updateStatusBadge);
    window.runtime.EventsOn('metric_added', () => {
      refreshDashboard();
    });
  }

  // Polling leve para sincronizar UI caso não haja Wails runtime
  setInterval(async () => {
    try {
      const status = await WailsApp.getStatus();
      updateStatusBadge(status);
    } catch (_) { }
  }, 10000);
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initApp);
} else {
  // O documento já está pronto (muito comum em WebViews locais)
  initApp();
}

function setupEventListeners() {
  btnRunTest.addEventListener('click', onRunTestClicked);
  btnOpenSettings.addEventListener('click', openSettingsModal);
  btnCloseSettings.addEventListener('click', closeSettingsModal);
  btnCancelSettings.addEventListener('click', closeSettingsModal);
  settingsForm.addEventListener('submit', onSaveSettings);

  const btnCheckUpdate = document.getElementById('btn-check-update');
  if (btnCheckUpdate) {
    btnCheckUpdate.addEventListener('click', async () => {
      btnCheckUpdate.classList.add('loading');
      btnCheckUpdate.disabled = true;
      try {
        const res = await WailsApp.checkUpdate();
        alert(res.message || 'Verificação concluída.');
      } catch (err) {
        alert('Erro ao verificar atualizações: ' + (err.message || err));
      } finally {
        btnCheckUpdate.classList.remove('loading');
        btnCheckUpdate.disabled = false;
      }
    });
  }

  // Paginação e Filtro por Data
  if (historyDateFilter) {
    historyDateFilter.addEventListener('change', async (e) => {
      historyPagination.dayFilter = (e.target.value || '').trim();
      historyPagination.currentPage = 1;
      await loadHistoryTablePage();
    });
  }

  if (btnFilterToday) {
    btnFilterToday.addEventListener('click', async () => {
      const now = new Date();
      const year = now.getFullYear();
      const month = String(now.getMonth() + 1).padStart(2, '0');
      const day = String(now.getDate()).padStart(2, '0');
      const todayStr = `${year}-${month}-${day}`;

      if (historyDateFilter) historyDateFilter.value = todayStr;
      historyPagination.dayFilter = todayStr;
      historyPagination.currentPage = 1;
      await loadHistoryTablePage();
    });
  }

  if (btnFilterAll) {
    btnFilterAll.addEventListener('click', async () => {
      if (historyDateFilter) historyDateFilter.value = '';
      historyPagination.dayFilter = '';
      historyPagination.currentPage = 1;
      await loadHistoryTablePage();
    });
  }

  if (btnPrevPage) {
    btnPrevPage.addEventListener('click', async () => {
      if (historyPagination.currentPage > 1) {
        historyPagination.currentPage--;
        await loadHistoryTablePage();
      }
    });
  }

  if (btnNextPage) {
    btnNextPage.addEventListener('click', async () => {
      if (historyPagination.currentPage < historyPagination.totalPages) {
        historyPagination.currentPage++;
        await loadHistoryTablePage();
      }
    });
  }
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
    statusText.textContent = status.status_text || (status.active_task ? `Executando: ${status.active_task}...` : 'Executando teste...');
  } else {
    statusBadge.className = 'status-badge scheduled';
    statusText.textContent = status.status_text || (status.sync_with_central ? 'Sincronizado com o servidor central' : 'Agendado');
  }

  // Atualiza badge de Grupo / VLAN
  const groupBadge = document.getElementById('group-badge-display');
  if (groupBadge) {
    if (status.group_name) {
      groupBadge.textContent = `Grupo: ${status.group_name}${status.current_vlan ? ' (VLAN ' + status.current_vlan + ')' : ''}`;
    } else {
      groupBadge.textContent = 'Grupo: Padrão';
    }
  }

  // Atualiza badge de Perfil de Medição
  const profileBadge = document.getElementById('profile-badge');
  if (profileBadge && status.current_profile) {
    const prof = status.current_profile;
    const activeCount = prof.tests ? prof.tests.filter(t => t.enabled).length : 0;
    profileBadge.textContent = `Perfil ${prof.version} (${activeCount} testes)`;
    profileBadge.title = prof.description || `Perfil ativo com ${activeCount} tarefas`;
  }

  const footerDesc = document.getElementById('footer-agent-desc');
  if (footerDesc) {
    if (status.sync_with_central) {
      footerDesc.textContent = 'Sincronizado com o servidor central (:00 e :30)';
    } else {
      footerDesc.textContent = 'Agente Operando em Background (Agendado)';
    }
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

  // Preenche dados do PC
  if (status.current_pc) {
    const pc = status.current_pc;
    const pcModel = document.getElementById('pc-model');
    const pcCpu = document.getElementById('pc-cpu');
    const pcRam = document.getElementById('pc-ram');
    const pcIpMac = document.getElementById('pc-ip-mac');
    const pcAdapter = document.getElementById('pc-adapter');

    if (pcModel) pcModel.textContent = pc.model || pc.hostname || 'Dell Pro Micro';
    if (pcCpu) pcCpu.textContent = pc.cpu || 'Intel Core Ultra';
    if (pcRam) pcRam.textContent = pc.ram_total_gb ? `${pc.ram_total_gb.toFixed(1)} GB` : '-- GB';
    if (pcIpMac) pcIpMac.textContent = `${pc.ip_address || '--'} / ${pc.mac_address || '--'}`;
    if (pcAdapter) pcAdapter.textContent = pc.network_adapter || 'Ethernet';
  }

  // Preenche dados do Switch LLDP
  if (status.current_switch) {
    const sw = status.current_switch;
    const swName = document.getElementById('switch-name');
    const swPort = document.getElementById('switch-port');
    const swVlan = document.getElementById('switch-vlan');

    if (swName) {
      swName.textContent = sw.switch_name || 'Aguardando frame LLDP...';
      if (sw.switch_model) swName.title = `Modelo: ${sw.switch_model}`;
    }
    if (swPort) swPort.textContent = sw.switch_port || '--';
    if (swVlan) swVlan.textContent = sw.vlan ? `VLAN ${sw.vlan}` : '--';
  }
}

/* ==========================================================================
   Tema Claro (Padrão) / Escuro
   ========================================================================== */

function initTheme() {
  const savedTheme = localStorage.getItem('probe-theme') || 'light';
  applyTheme(savedTheme);

  const btnTheme = document.getElementById('btn-theme-toggle');
  if (btnTheme) {
    btnTheme.onclick = (e) => {
      e.preventDefault();
      const cur = document.documentElement.getAttribute('data-theme') || 'light';
      const next = cur === 'dark' ? 'light' : 'dark';
      applyTheme(next);
    };
  }
}

function applyTheme(theme) {
  const chosen = (theme === 'dark') ? 'dark' : 'light';
  document.documentElement.setAttribute('data-theme', chosen);
  localStorage.setItem('probe-theme', chosen);

  const labelEl = document.getElementById('theme-toggle-label');
  if (labelEl) {
    labelEl.textContent = chosen === 'dark' ? 'Modo Claro' : 'Modo Escuro';
  }
  const btnTheme = document.getElementById('btn-theme-toggle');
  if (btnTheme) {
    btnTheme.setAttribute('title', chosen === 'dark' ? 'Alternar para Modo Claro' : 'Alternar para Modo Escuro');
  }

  updateChartsTheme(chosen);
}

function getChartThemeColors(theme) {
  const currentTheme = theme || document.documentElement.getAttribute('data-theme') || 'light';
  const isLight = currentTheme === 'light';
  return {
    textColor: isLight ? '#475569' : '#9ca3af',
    gridColor: isLight ? 'rgba(0, 0, 0, 0.07)' : 'rgba(255, 255, 255, 0.05)',
    tooltipBg: isLight ? '#ffffff' : '#111827',
    tooltipTitle: isLight ? '#0f172a' : '#ffffff',
    tooltipBody: isLight ? '#334155' : '#e5e7eb',
    tooltipBorder: isLight ? 'rgba(0, 0, 0, 0.15)' : 'rgba(255, 255, 255, 0.12)'
  };
}

function updateChartsTheme(theme) {
  const charts = [throughputChart, latencyChart, packetLossChart].filter(Boolean);
  if (charts.length === 0) return;
  const currentTheme = theme || document.documentElement.getAttribute('data-theme') || 'light';
  const colors = getChartThemeColors(currentTheme);

  charts.forEach(chart => {
    chart.options.scales.x.grid.color = colors.gridColor;
    chart.options.scales.x.ticks.color = colors.textColor;
    chart.options.scales.y.grid.color = colors.gridColor;
    chart.options.scales.y.ticks.color = colors.textColor;
    if (chart.options.scales.y.title) {
      chart.options.scales.y.title.color = colors.textColor;
    }
    chart.options.plugins.tooltip.backgroundColor = colors.tooltipBg;
    chart.options.plugins.tooltip.titleColor = colors.tooltipTitle;
    chart.options.plugins.tooltip.bodyColor = colors.tooltipBody;
    chart.options.plugins.tooltip.borderColor = colors.tooltipBorder;
    chart.update('none');
  });
}

function createFillGradient(ctx, topColor, bottomColor, height = 240) {
  const grad = ctx.createLinearGradient(0, 0, 0, height);
  grad.addColorStop(0, topColor);
  grad.addColorStop(1, bottomColor);
  return grad;
}

// Inicialização dos Gráficos com Chart.js (Com Gradientes, Curvas Suaves e Temas)
function initCharts() {
  if (typeof Chart === 'undefined') {
    console.warn('[CHARTS] Biblioteca Chart.js ainda não disponível no DOM.');
    return;
  }

  const currentTheme = document.documentElement.getAttribute('data-theme') || 'light';
  const colors = getChartThemeColors(currentTheme);

  const commonOptions = {
    responsive: true,
    maintainAspectRatio: false,
    layout: {
      padding: {
        left: 8,
        right: 32, // Garante que a linha e o rótulo do último tick não transbordem na borda direita
        top: 14,
        bottom: 6
      }
    },
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
          maxRotation: 0,
          autoSkip: true,
          maxTicksLimit: 12
        }
      },
      y: {
        grid: { color: colors.gridColor, drawBorder: false },
        ticks: { color: colors.textColor, font: { size: 10, family: "'JetBrains Mono', monospace" } }
      }
    }
  };

  // 1. Throughput Chart (Download & Upload)
  const ctxThroughput = document.getElementById('throughputChart')?.getContext('2d');
  if (ctxThroughput) {
    throughputChart = new Chart(ctxThroughput, {
      type: 'line',
      data: {
        labels: [],
        datasets: [
          {
            label: 'Download',
            data: [],
            borderColor: '#0284c7',
            backgroundColor: createFillGradient(ctxThroughput, 'rgba(2, 132, 199, 0.22)', 'rgba(2, 132, 199, 0.01)'),
            fill: true,
            tension: 0.38,
            borderWidth: 2.5,
            pointRadius: 3,
            pointHoverRadius: 6,
            pointBackgroundColor: '#0284c7',
            pointBorderColor: '#ffffff',
            pointBorderWidth: 1.5
          },
          {
            label: 'Upload',
            data: [],
            borderColor: '#4f46e5',
            backgroundColor: createFillGradient(ctxThroughput, 'rgba(79, 70, 229, 0.20)', 'rgba(79, 70, 229, 0.01)'),
            fill: true,
            tension: 0.38,
            borderWidth: 2.5,
            pointRadius: 3,
            pointHoverRadius: 6,
            pointBackgroundColor: '#4f46e5',
            pointBorderColor: '#ffffff',
            pointBorderWidth: 1.5
          }
        ]
      },
      options: {
        ...commonOptions,
        plugins: {
          ...commonOptions.plugins,
          tooltip: {
            ...commonOptions.plugins.tooltip,
            callbacks: {
              label: (ctx) => ` ${ctx.dataset.label}: ${ctx.parsed.y != null ? ctx.parsed.y.toFixed(1) + ' Mbps' : '--'}`
            }
          }
        },
        scales: {
          ...commonOptions.scales,
          y: {
            ...commonOptions.scales.y,
            title: { display: true, text: 'Mbps', color: colors.textColor, font: { size: 11, weight: '600' } }
          }
        }
      }
    });
  }

  // 2. Latency Chart (Ping)
  const ctxLatency = document.getElementById('latencyChart')?.getContext('2d');
  if (ctxLatency) {
    latencyChart = new Chart(ctxLatency, {
      type: 'line',
      data: {
        labels: [],
        datasets: [
          {
            label: 'Ping Gateway',
            data: [],
            borderColor: '#10b981',
            backgroundColor: createFillGradient(ctxLatency, 'rgba(16, 185, 129, 0.25)', 'rgba(16, 185, 129, 0.01)'),
            fill: true,
            tension: 0.38,
            borderWidth: 2.5,
            pointRadius: 3,
            pointHoverRadius: 6,
            pointBackgroundColor: '#10b981',
            pointBorderColor: '#ffffff',
            pointBorderWidth: 1.5
          }
        ]
      },
      options: {
        ...commonOptions,
        plugins: {
          ...commonOptions.plugins,
          tooltip: {
            ...commonOptions.plugins.tooltip,
            callbacks: {
              label: (ctx) => ` ${ctx.dataset.label}: ${ctx.parsed.y != null ? ctx.parsed.y.toFixed(1) + ' ms' : '--'}`
            }
          }
        },
        scales: {
          ...commonOptions.scales,
          y: {
            ...commonOptions.scales.y,
            title: { display: true, text: 'ms', color: colors.textColor, font: { size: 11, weight: '600' } }
          }
        }
      }
    });
  }

  // 3. Packet Loss Chart (Perda de Pacotes Gateway - 0% a 100%)
  const ctxPacketLoss = document.getElementById('packetLossChart')?.getContext('2d');
  if (ctxPacketLoss) {
    packetLossChart = new Chart(ctxPacketLoss, {
      type: 'line',
      data: {
        labels: [],
        datasets: [
          {
            label: 'Perda (%)',
            data: [],
            borderColor: '#ef4444',
            backgroundColor: createFillGradient(ctxPacketLoss, 'rgba(239, 68, 68, 0.25)', 'rgba(239, 68, 68, 0.01)'),
            fill: true,
            tension: 0.38,
            borderWidth: 2.5,
            pointRadius: 3,
            pointHoverRadius: 6,
            pointBackgroundColor: '#ef4444',
            pointBorderColor: '#ffffff',
            pointBorderWidth: 1.5
          }
        ]
      },
      options: {
        ...commonOptions,
        plugins: {
          ...commonOptions.plugins,
          tooltip: {
            ...commonOptions.plugins.tooltip,
            callbacks: {
              label: (ctx) => ` ${ctx.dataset.label}: ${ctx.parsed.y != null ? ctx.parsed.y.toFixed(1) + '%' : '--'}`
            }
          }
        },
        scales: {
          ...commonOptions.scales,
          y: {
            ...commonOptions.scales.y,
            min: 0,
            max: 100,
            title: { display: true, text: '% Perda', color: colors.textColor, font: { size: 11, weight: '600' } }
          }
        }
      }
    });
  }
}

// Avalia quais medições estão de fato ativas e configuradas no perfil da máquina
let currentCapabilities = {
  hasDownload: true,
  hasUpload: true,
  hasThroughput: true,
  hasPing: true,
  hasDNS: false,
  hasPacketLoss: false,
  hasLLDP: true,
};

function getProfileCapabilities(profile, history) {
  if (profile && Array.isArray(profile.tests) && profile.tests.length > 0) {
    const enabledTests = profile.tests.filter(t => t.enabled !== false);
    const hasDownload = enabledTests.some(t =>
      t.type === 'http_throughput_download' || t.id === 'throughput_download'
    );
    const hasUpload = enabledTests.some(t =>
      t.type === 'http_throughput_upload' || t.id === 'throughput_upload'
    );
    const hasPing = enabledTests.some(t =>
      t.type === 'ping' || t.id.startsWith('ping_')
    );
    const hasPacketLoss = enabledTests.some(t =>
      t.type === 'packet_loss' || t.id.startsWith('packet_loss_')
    );
    const hasDNS = enabledTests.some(t =>
      t.type === 'dns_lookup' || t.id.startsWith('dns_')
    );
    const hasLLDP = enabledTests.some(t =>
      t.type === 'discovery_protocol' || t.id.includes('discovery') || t.id.includes('lldp')
    );

    return {
      hasDownload,
      hasUpload,
      hasThroughput: hasDownload || hasUpload,
      hasPing,
      hasPacketLoss,
      hasDNS,
      hasLLDP,
    };
  }

  // Fallback: se não há perfil definido ainda, verifica histórico de medições reais
  const hasHistoryThroughput = Array.isArray(history) && history.some(r => (r.download_mbps > 0 || r.upload_mbps > 0));
  const hasHistoryPacketLoss = Array.isArray(history) && history.some(r => r.metrics && (r.metrics.packet_loss_gateway != null || r.metrics.packet_loss != null));
  return {
    hasDownload: hasHistoryThroughput,
    hasUpload: hasHistoryThroughput,
    hasThroughput: hasHistoryThroughput,
    hasPing: true,
    hasPacketLoss: hasHistoryPacketLoss,
    hasDNS: false,
    hasLLDP: true,
  };
}

function applyCapabilitiesToUI(caps) {
  currentCapabilities = caps;

  // 1. Cards de métricas no topo
  const cardPacketLoss = document.getElementById('card-packet-loss');
  const cardDownload = document.getElementById('card-download');
  const cardUpload = document.getElementById('card-upload');
  const cardPing = document.getElementById('card-ping');

  if (cardPacketLoss) cardPacketLoss.style.display = caps.hasPacketLoss ? '' : 'none';
  if (cardDownload) cardDownload.style.display = caps.hasDownload ? '' : 'none';
  if (cardUpload) cardUpload.style.display = caps.hasUpload ? '' : 'none';
  if (cardPing) cardPing.style.display = caps.hasPing ? '' : 'none';

  // 2. Gráficos empilhados independentes
  const throughputChartCard = document.getElementById('throughput-chart-card');
  const latencyChartCard = document.getElementById('latency-chart-card');
  const packetLossChartCard = document.getElementById('packet-loss-chart-card');
  const legendDownload = document.getElementById('legend-download');
  const legendUpload = document.getElementById('legend-upload');

  if (throughputChartCard) {
    const wasHidden = throughputChartCard.style.display === 'none';
    throughputChartCard.style.display = caps.hasThroughput ? '' : 'none';
    if (caps.hasThroughput && wasHidden && throughputChart) {
      setTimeout(() => {
        try { throughputChart.resize(); throughputChart.update(); } catch (_) { }
      }, 50);
    }
  }

  if (legendDownload) legendDownload.style.display = caps.hasDownload ? '' : 'none';
  if (legendUpload) legendUpload.style.display = caps.hasUpload ? '' : 'none';

  if (latencyChartCard) {
    latencyChartCard.style.display = caps.hasPing ? '' : 'none';
  }

  if (packetLossChartCard) {
    packetLossChartCard.style.display = caps.hasPacketLoss ? '' : 'none';
  }

  // 3. Colunas da tabela de execuções recentes
  const thDownload = document.getElementById('th-download');
  const thUpload = document.getElementById('th-upload');
  const thPing = document.getElementById('th-ping');

  if (thDownload) thDownload.style.display = caps.hasDownload ? '' : 'none';
  if (thUpload) thUpload.style.display = caps.hasUpload ? '' : 'none';
  if (thPing) thPing.style.display = caps.hasPing ? '' : 'none';
}

// Atualiza Dashboard com métricas do SQLite e adaptação dinâmica ao Perfil
async function refreshDashboard() {
  const history = await WailsApp.getHistory();
  const status = await WailsApp.getStatus();

  // Avalia capacidades do perfil ativo e adapta interface
  const profile = status.current_profile || (await WailsApp.getActiveProfile()) || null;
  const capabilities = getProfileCapabilities(profile, history);
  applyCapabilitiesToUI(capabilities);

  updateStatusBadge(status);

  let latest = null;
  if (history && history.length > 0) {
    latest = history[history.length - 1];

    // Atualiza cards numéricos
    if (capabilities.hasDownload && statDownload) statDownload.textContent = latest.download_mbps.toFixed(1);
    if (capabilities.hasUpload && statUpload) statUpload.textContent = latest.upload_mbps.toFixed(1);
    if (capabilities.hasPing && statPing) statPing.textContent = latest.ping_gateway.toFixed(1);

    // Perda de Pacotes
    let lossVal = 0.0;
    let lossDetail = 'Alvo: Gateway local';
    const plMetric = latest.metrics && (latest.metrics.packet_loss_gateway || latest.metrics.packet_loss);
    if (plMetric) {
      if (typeof plMetric.value === 'number') {
        lossVal = plMetric.value;
      }
      if (plMetric.details) {
        lossDetail = plMetric.details;
      }
    }
    const statLoss = document.getElementById('stat-packet-loss');
    const statBadge = document.getElementById('stat-packet-loss-badge');
    const statDet = document.getElementById('stat-packet-loss-detail');
    if (statLoss) statLoss.textContent = lossVal.toFixed(1);
    if (statDet) statDet.textContent = lossDetail;
    if (statBadge) {
      if (lossVal === 0) {
        statBadge.className = 'kpi-badge success';
        statBadge.textContent = '0% Perda';
      } else if (lossVal < 5) {
        statBadge.className = 'kpi-badge warning';
        statBadge.textContent = `${lossVal.toFixed(1)}% Perda`;
      } else {
        statBadge.className = 'kpi-badge danger';
        statBadge.textContent = `${lossVal.toFixed(1)}% Perda`;
      }
    }

    // Formata timestamps e alimenta gráficos (últimas 24 execuções)
    const labels = history.map(item => {
      const d = new Date(item.timestamp);
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    });

    const downloads = history.map(item => item.download_mbps);
    const uploads = history.map(item => item.upload_mbps);
    const pings = history.map(item => item.ping_gateway);
    const packetLosses = history.map(item => {
      const pl = item.metrics && (item.metrics.packet_loss_gateway || item.metrics.packet_loss);
      if (pl && typeof pl.value === 'number') {
        return pl.value;
      }
      return 0.0;
    });

    // Inicializa charts se ainda não foram criados
    if (!throughputChart && typeof Chart !== 'undefined') {
      try { initCharts(); } catch (_) { }
    }

    // Atualiza Chart Throughput
    if (capabilities.hasThroughput && throughputChart) {
      throughputChart.data.labels = labels;
      throughputChart.data.datasets[0].data = capabilities.hasDownload ? downloads : [];
      throughputChart.data.datasets[1].data = capabilities.hasUpload ? uploads : [];
      throughputChart.update();
    }

    // Atualiza Chart Latência
    if (capabilities.hasPing && latencyChart) {
      latencyChart.data.labels = labels;
      latencyChart.data.datasets[0].data = pings;
      latencyChart.update();
    }

    // Atualiza Chart Perda de Pacotes
    if (capabilities.hasPacketLoss && packetLossChart) {
      packetLossChart.data.labels = labels;
      packetLossChart.data.datasets[0].data = packetLosses;
      packetLossChart.update();
    }

    if (lastUpdatedText) {
      lastUpdatedText.textContent = `Última leitura: ${new Date(latest.timestamp).toLocaleTimeString()}`;
    }
  } else {
    if (statDownload) statDownload.textContent = '--';
    if (statUpload) statUpload.textContent = '--';
    if (statPing) statPing.textContent = '--';
  }

  // Atualiza painel técnico: Wi-Fi, Módulos, Switch e Inventário de Placas
  renderTechnicalSpecs(status, latest);

  // Carrega a página da tabela de execuções com o filtro atual
  await loadHistoryTablePage();
}

// Renderiza especificações técnicas, Wi-Fi, switches, módulos e inventário de rede
function renderTechnicalSpecs(status, latestRecord) {
  const sysInfo = (status && status.current_pc) || (latestRecord && latestRecord.system_info) || null;
  const swInfo = (status && status.current_switch) || (latestRecord && latestRecord.switch_info) || null;
  const profile = (status && status.current_profile) || null;

  // 1. Status Wi-Fi & Potência
  const wifiBadge = document.getElementById('wifi-status-badge');
  const wifiBody = document.getElementById('wifi-content-body');
  if (wifiBody) {
    let wifiData = null;
    if (sysInfo && sysInfo.wifi_info && sysInfo.wifi_info.connected) {
      wifiData = sysInfo.wifi_info;
    } else if (sysInfo && Array.isArray(sysInfo.interfaces)) {
      const wIface = sysInfo.interfaces.find(i => i.is_wireless || (i.name && i.name.toLowerCase().includes('wi-fi')) || (i.name && i.name.toLowerCase().includes('wlan')));
      if (wIface && wIface.wifi_info && wIface.wifi_info.connected) {
        wifiData = wIface.wifi_info;
      }
    }

    if (wifiData) {
      if (wifiBadge) {
        wifiBadge.textContent = 'Conectado';
        wifiBadge.className = 'badge-pill text-emerald';
      }
      const signalPct = wifiData.signal_percent || 0;
      const signalDbm = wifiData.signal_dbm || -50;
      let barClass = '';
      if (signalPct < 40) barClass = 'poor';
      else if (signalPct < 70) barClass = 'warning';

      wifiBody.innerHTML = `
        <div class="wifi-summary-box">
          <div class="wifi-meter-label-row">
            <span class="wifi-meter-title">${escapeHtml(wifiData.ssid || 'Wi-Fi Corporativo')}</span>
            <div class="wifi-signal-value-group">
              <span class="wifi-signal-pct">${signalPct}%</span>
              <span class="wifi-signal-dbm">${signalDbm} dBm</span>
            </div>
          </div>
          <div class="wifi-signal-bar-track">
            <div class="wifi-signal-bar-fill ${barClass}" style="width: ${signalPct}%;"></div>
          </div>
          <div class="wifi-meter-footer">
            <span class="wifi-signal-desc">${escapeHtml(wifiData.channel ? `Canal ${wifiData.channel}` : '5 GHz')}</span>
            <span class="badge-pill">${escapeHtml(wifiData.bssid || 'AP Local')}</span>
          </div>
        </div>
      `;
    } else {
      if (wifiBadge) {
        wifiBadge.textContent = 'Cabo Ethernet';
        wifiBadge.className = 'badge-pill';
      }
      wifiBody.innerHTML = `
        <div class="wifi-eth-notice">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-cyan"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"></rect><rect x="2" y="14" width="20" height="8" rx="2" ry="2"></rect><line x1="6" y1="6" x2="6.01" y2="6"></line><line x1="6" y1="18" x2="6.01" y2="18"></line></svg>
          <div>
            <div>Conexão Cabeada Exclusiva (Ethernet)</div>
            <div class="text-xs text-muted" style="margin-top: 2px;">Esta sonda opera em enlace de par trançado. Adaptador Wi-Fi ausente ou desativado.</div>
          </div>
        </div>
      `;
    }
  }

  // 2. Módulos Ativos Carregados
  const modBadge = document.getElementById('modules-profile-badge');
  const modContainer = document.getElementById('modules-chips-container');
  if (modContainer) {
    if (profile && Array.isArray(profile.tests) && profile.tests.length > 0) {
      const enabledTests = profile.tests.filter(t => t.enabled !== false);
      if (modBadge) modBadge.textContent = `${enabledTests.length} módulos`;
      modContainer.innerHTML = enabledTests.map(t => {
        let name = t.name || t.id;
        name = name.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
        return `<span class="module-chip" title="Tipo: ${t.type || 'teste'}">${escapeHtml(name)}</span>`;
      }).join('');
    } else {
      if (modBadge) modBadge.textContent = 'Padrão (3 módulos)';
      modContainer.innerHTML = `
        <span class="module-chip">Latência Gateway Local</span>
        <span class="module-chip">Perda de Pacotes Gateway</span>
        <span class="module-chip">Throughput ACON</span>
      `;
    }
  }

  // 3. Switch LLDP
  if (swInfo) {
    const swName = document.getElementById('switch-name');
    const swPort = document.getElementById('switch-port');
    const swVlan = document.getElementById('switch-vlan');
    const swProto = document.getElementById('switch-protocol');
    const pcAdapter = document.getElementById('pc-adapter');

    if (swName) swName.textContent = swInfo.switch_name || 'Aguardando frame LLDP...';
    if (swPort) swPort.textContent = swInfo.switch_port || '--';
    if (swVlan) swVlan.textContent = swInfo.vlan ? `VLAN ${swInfo.vlan}` : 'Sem VLAN';
    if (swProto) swProto.textContent = swInfo.protocol || 'LLDP';
    if (pcAdapter && sysInfo) pcAdapter.textContent = sysInfo.network_adapter || '--';
  }

  // 4. Inventário de Placas de Rede & Endereços MAC
  const ifacesTbody = document.getElementById('ifaces-tbody');
  const ifacesCountBadge = document.getElementById('ifaces-count-badge');
  if (ifacesTbody) {
    const ifaces = (sysInfo && Array.isArray(sysInfo.interfaces)) ? sysInfo.interfaces : [];
    if (ifacesCountBadge) {
      ifacesCountBadge.textContent = `${ifaces.length} placa(s) detectada(s)`;
    }

    if (ifaces.length === 0) {
      ifacesTbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted py-3">Nenhum adaptador de rede identificado no momento.</td></tr>';
    } else {
      ifacesTbody.innerHTML = ifaces.map(iface => {
        const isUp = iface.is_up !== false;
        const statusBadge = isUp
          ? '<span class="badge-iface-status active">Ativa</span>'
          : '<span class="badge-iface-status inactive">Inativa</span>';

        let typeBadge = '<span class="badge-iface-type eth">Ethernet</span>';
        if (iface.is_wireless || (iface.type && iface.type.toLowerCase().includes('wifi'))) {
          typeBadge = '<span class="badge-iface-type wifi">Wi-Fi</span>';
        } else if (iface.is_virtual || (iface.type && iface.type.toLowerCase().includes('virtual'))) {
          typeBadge = '<span class="badge-iface-type virtual">Virtual</span>';
        }

        const macFormatted = (iface.mac || '--').toUpperCase();
        const ips = (Array.isArray(iface.ips) && iface.ips.length > 0)
          ? iface.ips.join(', ')
          : (iface.ip || '--');

        const priorityBadge = iface.is_primary
          ? '<span class="badge-iface-priority">Principal (Gateway)</span>'
          : '<span class="text-muted text-xs">Secundária</span>';

        return `
          <tr>
            <td>${statusBadge}</td>
            <td>
              <div class="font-bold">${escapeHtml(iface.name || 'Adaptador')}</div>
              ${iface.description && iface.description !== iface.name ? `<div class="text-xs text-muted">${escapeHtml(iface.description)}</div>` : ''}
            </td>
            <td>${typeBadge}</td>
            <td><code class="font-mono" style="color: var(--cyan-light);">${escapeHtml(macFormatted)}</code></td>
            <td><code class="font-mono">${escapeHtml(ips)}</code></td>
            <td style="text-align: center;">${priorityBadge}</td>
          </tr>
        `;
      }).join('');
    }
  }
}

function escapeHtml(str) {
  if (!str) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

// Carrega página específica da tabela de execuções recentes (15 itens por página com filtro de data)
async function loadHistoryTablePage() {
  try {
    const res = await WailsApp.getHistoryPage(
      historyPagination.dayFilter,
      historyPagination.currentPage,
      historyPagination.pageSize
    );

    const records = (res && res.records) ? res.records : [];
    historyPagination.totalCount = (res && typeof res.total_count === 'number') ? res.total_count : 0;
    historyPagination.totalPages = Math.max(1, (res && typeof res.total_pages === 'number') ? res.total_pages : 1);
    historyPagination.currentPage = Math.min(Math.max(1, (res && typeof res.page === 'number') ? res.page : 1), historyPagination.totalPages);

    renderTable(records, currentCapabilities);
    updatePaginationUI();
  } catch (err) {
    console.error('Erro ao carregar página de histórico:', err);
    metricsTableBody.innerHTML = `<tr><td colspan="5" class="text-center text-muted">Erro ao carregar histórico: ${err.message || err}</td></tr>`;
  }
}

function updatePaginationUI() {
  if (currentPageDisplay) currentPageDisplay.textContent = historyPagination.currentPage;
  if (totalPagesDisplay) totalPagesDisplay.textContent = historyPagination.totalPages;

  if (btnPrevPage) {
    btnPrevPage.disabled = (historyPagination.currentPage <= 1);
  }
  if (btnNextPage) {
    btnNextPage.disabled = (historyPagination.currentPage >= historyPagination.totalPages);
  }

  const start = historyPagination.totalCount === 0 ? 0 : (historyPagination.currentPage - 1) * historyPagination.pageSize + 1;
  const end = Math.min(historyPagination.currentPage * historyPagination.pageSize, historyPagination.totalCount);

  if (paginationRangeText) {
    paginationRangeText.textContent = `Exibindo ${start} a ${end} de ${historyPagination.totalCount}`;
  }

  if (historyTotalBadge) {
    const dayLabel = historyPagination.dayFilter ? ` (${historyPagination.dayFilter})` : '';
    historyTotalBadge.textContent = `${historyPagination.totalCount} execuções${dayLabel}`;
  }
}

function renderTable(records, caps = currentCapabilities) {
  let colSpan = 2; // Data & Hora + Status Sync
  if (caps.hasDownload) colSpan++;
  if (caps.hasUpload) colSpan++;
  if (caps.hasPing) colSpan++;

  if (!records || records.length === 0) {
    const emptyMsg = historyPagination.dayFilter
      ? `Nenhuma execução encontrada para a data ${historyPagination.dayFilter}.`
      : 'Nenhum teste registrado ainda.';
    metricsTableBody.innerHTML = `<tr><td colspan="${colSpan}" class="text-center text-muted">${emptyMsg}</td></tr>`;
    return;
  }

  metricsTableBody.innerHTML = records.map(r => {
    const d = new Date(r.timestamp);
    const dateFormatted = `${d.toLocaleDateString()} ${d.toLocaleTimeString()}`;
    const syncBadge = r.synced
      ? '<span class="sync-badge ok">✓ Sincronizado</span>'
      : '<span class="sync-badge fail" title="' + (r.error_message || 'Falha de conexão') + '">✕ Pendente</span>';

    let extraBadges = '';
    if (r.metrics) {
      if (r.metrics.dns_uepg && r.metrics.dns_uepg.value > 0) {
        extraBadges += `<span class="mono-badge" style="font-size: 0.68rem; margin-left: 4px;" title="Resolução DNS UEPG">DNS ${r.metrics.dns_uepg.value.toFixed(1)}ms</span>`;
      }
      if (r.metrics.ping_nti_core && r.metrics.ping_nti_core.value > 0) {
        extraBadges += `<span class="mono-badge" style="font-size: 0.68rem; margin-left: 4px; color: #c084fc; border-color: rgba(192, 132, 252, 0.3);" title="Ping Core NTI">NTI ${r.metrics.ping_nti_core.value.toFixed(1)}ms</span>`;
      }
      const pl = r.metrics && (r.metrics.packet_loss_gateway || r.metrics.packet_loss);
      if (pl && typeof pl.value === 'number' && pl.value >= 0) {
        extraBadges += `<span class="mono-badge" style="font-size: 0.68rem; margin-left: 4px; color: #f87171;" title="Perda de Pacotes">Perda ${pl.value.toFixed(0)}%</span>`;
      }
    }

    let cells = `<td>${dateFormatted} ${extraBadges}</td>`;
    if (caps.hasDownload) {
      cells += `<td><strong style="color: var(--cyan-light)">${r.download_mbps.toFixed(1)}</strong> Mbps</td>`;
    }
    if (caps.hasUpload) {
      cells += `<td><strong style="color: var(--purple-light)">${r.upload_mbps.toFixed(1)}</strong> Mbps</td>`;
    }
    if (caps.hasPing) {
      cells += `<td><strong style="color: var(--emerald-light)">${r.ping_gateway.toFixed(1)}</strong> ms</td>`;
    }
    cells += `<td>${syncBadge}</td>`;

    return `<tr>${cells}</tr>`;
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
  document.getElementById('cfg-interval').value = cfg.interval_minutes || 30;
  document.getElementById('cfg-ping-target').value = cfg.ping_target || '192.168.1.1';

  const isSync = (cfg.sync_with_central !== false) && (cfg.server_url && cfg.server_url.trim() !== '');
  const hintEl = document.getElementById('cfg-interval-hint');
  if (hintEl) {
    hintEl.style.display = isSync ? 'block' : 'none';
  }
}

function openSettingsModal() {
  modalSettings.classList.remove('hidden');
}

function closeSettingsModal() {
  modalSettings.classList.add('hidden');
}

async function onSaveSettings(e) {
  e.preventDefault();
  const serverUrl = document.getElementById('cfg-server-url').value.trim();
  const cfg = {
    node_id: document.getElementById('cfg-node-id').value.trim(),
    location: document.getElementById('cfg-location').value.trim(),
    server_url: serverUrl,
    api_token: document.getElementById('cfg-api-token').value.trim(),
    interval_minutes: parseInt(document.getElementById('cfg-interval').value, 10) || 30,
    ping_target: document.getElementById('cfg-ping-target').value.trim(),
    sync_with_central: serverUrl !== ''
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

function generateMockHistoryPage(dayFilter, page, pageSize) {
  const all = [];
  const now = Date.now();
  for (let i = 44; i >= 0; i--) {
    const time = new Date(now - i * 3600000 * 2);
    all.push({
      id: 45 - i,
      node_id: 'minipc-dc-01',
      location: 'Guarapuava-DC1',
      timestamp: time.toISOString(),
      download_mbps: +(810 + Math.sin(i) * 60 + Math.random() * 20).toFixed(2),
      upload_mbps: +(715 + Math.cos(i) * 45 + Math.random() * 15).toFixed(2),
      ping_gateway: +(1.2 + Math.random() * 0.7).toFixed(2),
      synced: true
    });
  }

  let filtered = all;
  if (dayFilter && dayFilter.trim() !== '') {
    filtered = all.filter(r => r.timestamp.startsWith(dayFilter.trim()));
  }

  // Mais recentes primeiro
  filtered.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));

  const totalCount = filtered.length;
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));
  const validPage = Math.min(Math.max(1, page), totalPages);
  const start = (validPage - 1) * pageSize;
  const records = filtered.slice(start, start + pageSize);

  return {
    records,
    total_count: totalCount,
    page: validPage,
    page_size: pageSize,
    total_pages: totalPages
  };
}

