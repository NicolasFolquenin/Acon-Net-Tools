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
let selectedChartNode = 'all';
let selectedChartGroup = 'all';

// Configurações globais dinâmicas puxadas do .env através da API /api/v1/config
window.APP_CONFIG = {
  port: '8085',
  public_server_url: '',
  dns_server_target: '177.101.22.40',
  dns_domain_target: 'uepg.br',
  dns_server_name: 'DNS Institucional',
  ping_core_target: '177.101.22.40',
  ping_core_name: 'Core NTI (Anel)',
  default_ping_target: 'auto_gateway',
  speedtest_download_mb: 5,
  speedtest_upload_mb: 2
};

async function loadSystemConfig() {
  try {
    const res = await fetch('/api/v1/config');
    if (res.ok) {
      const data = await res.json();
      window.APP_CONFIG = { ...window.APP_CONFIG, ...data };
      const smDnsLabel = document.querySelector('#sm-dns-meta .sm-meta-label');
      if (smDnsLabel && window.APP_CONFIG.dns_server_name) {
        smDnsLabel.textContent = `${window.APP_CONFIG.dns_server_name}:`;
      }
    }
  } catch (e) {
    console.warn('[CONFIG] Usando configurações padrão de ambiente:', e);
  }
}
loadSystemConfig();

// Gerenciamento de Sessão JWT e Fetch Resiliente
let authToken = '';
try {
  authToken = localStorage.getItem('noc_token') || sessionStorage.getItem('noc_token') || '';
} catch {}

async function apiFetch(url, options = {}) {
  const opts = { ...options };
  const headers = new Headers(opts.headers || {});
  if (authToken && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${authToken}`);
  }
  opts.headers = headers;
  return fetch(url, opts);
}

const tabDashboard = document.getElementById('tab-dashboard');
const tabMachines = document.getElementById('tab-machines');
const tabGroups = document.getElementById('tab-groups');
const tabModules = document.getElementById('tab-modules');
const viewDashboard = document.getElementById('view-dashboard');
const viewMachines = document.getElementById('view-machines');
const viewGroups = document.getElementById('view-groups');
const viewModules = document.getElementById('view-modules');

// Elementos da Aba Máquinas
const machinesSearchInput = document.getElementById('machines-search-input');
const machinesGroupFilter = document.getElementById('machines-group-filter');
const machinesModuleFilter = document.getElementById('machines-module-filter');
const machinesStatusFilter = document.getElementById('machines-status-filter');
const machinesFilterCount = document.getElementById('machines-filter-count');
const machinesTableBody = document.getElementById('machines-table-body');
const btnRefreshMachines = document.getElementById('btn-refresh-machines');

// Elementos do Modal de Métricas da Máquina
const machineMetricsModal = document.getElementById('machine-metrics-modal');
const modalBtnCloseMm = document.getElementById('modal-btn-close-mm');
const modalBtnCloseMmBottom = document.getElementById('modal-btn-close-mm-bottom');
const mmBtnFocusDashboard = document.getElementById('mm-btn-focus-dashboard');

// Elementos do Modal de Configuração da Máquina
const machineConfigModal = document.getElementById('machine-config-modal');
const modalBtnCloseMc = document.getElementById('modal-btn-close-mc');
const machineConfigForm = document.getElementById('machine-config-form');
const mcInputHostname = document.getElementById('mc-input-hostname');
const mcHostnameTitle = document.getElementById('mc-hostname-title');
const mcSelectGroup = document.getElementById('mc-select-group');
const mcToggleCustom = document.getElementById('mc-toggle-custom');
const mcCustomModulesContainer = document.getElementById('mc-custom-modules-container');
const mcModulesCheckboxesList = document.getElementById('mc-modules-checkboxes-list');

// Elementos do Catálogo de Módulos & Upload JSON
const modulesCardsGrid = document.getElementById('modules-cards-grid');
const modulesCountTag = document.getElementById('modules-count-tag');
const btnOpenUploadModule = document.getElementById('btn-open-upload-module');
const moduleUploadModal = document.getElementById('module-upload-modal');
const modalBtnCloseUploadMod = document.getElementById('modal-btn-close-upload-mod');
const btnCancelUploadMod = document.getElementById('btn-cancel-upload-mod');
const btnSubmitUploadMod = document.getElementById('btn-submit-upload-mod');
const jsonDropzone = document.getElementById('json-dropzone');
const modFileInput = document.getElementById('mod-file-input');
const modJsonTextarea = document.getElementById('mod-json-textarea');
const modJsonErrorAlert = document.getElementById('mod-json-error-alert');
const modJsonErrorText = document.getElementById('mod-json-error-text');

// Estado de Máquinas e Módulos
let allAdminMachines = [];
let allCatalogModules = [];
// Graficos no modal desativados em favor do Dashboard
let currentInspectedMachine = null;
const machineFilterState = {
  search: '',
  group: 'all',
  module: 'all',
  status: 'all',
  date: ''
};

const machinesPaginationState = {
  page: 1,
  itemsPerPage: 15
};

let allTelemetrySeries = [];
const telemetryPageState = {
  page: 1,
  itemsPerPage: 15,
  date: ''
};
const telemetryPaginationState = telemetryPageState;

let currentPingTarget = {
  hostname: '',
  ip: ''
};


// Elementos DOM principais
const selectNode = document.getElementById('select-node');
const btnRefresh = document.getElementById('btn-refresh');
const nodesSummaryGrid = document.getElementById('nodes-summary-grid');
const nodesTableBody = document.getElementById('nodes-table-body');
const nodesSearchInput = document.getElementById('nodes-search-input');
const nodesStatusFilter = document.getElementById('nodes-status-filter');
const nodesGroupFilter = document.getElementById('nodes-group-filter');
const nodesFilterCount = document.getElementById('nodes-filter-count');
const telemetryTableBody = document.getElementById('telemetry-table-body');
const activeNodesCount = document.getElementById('active-nodes-count');
const lastIngestTime = document.getElementById('last-ingest-time');
const autoRefreshTimer = document.getElementById('auto-refresh-timer');

// Elementos de Autenticação / Gateway / Perfil
const loginModalOverlay = document.getElementById('login-modal-overlay');
const welcomeGatewayCard = document.getElementById('welcome-gateway-card');
const loginCard = document.getElementById('login-card');
const btnGatewayGoLogin = document.getElementById('btn-gateway-go-login');
const btnGatewayDownloadClient = document.getElementById('btn-gateway-download-client');
const btnBackToGateway = document.getElementById('btn-back-to-gateway');
const btnToggleGatewayInstructions = document.getElementById('btn-toggle-gateway-instructions');
const gatewayInstructionsPanel = document.getElementById('gateway-instructions-panel');
const gatewayClientFilesize = document.getElementById('gateway-client-filesize');
const btnLoginDownloadClient = document.getElementById('btn-login-download-client');
const btnSidebarDownloadClient = document.getElementById('btn-sidebar-download-client');
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

// Elementos de Gerenciamento de Usuários (Admin)
const btnOpenUsers = document.getElementById('btn-open-users');
const usersModal = document.getElementById('users-modal');
const modalBtnCloseUsers = document.getElementById('modal-btn-close-users');
const modalBtnCloseUsersBottom = document.getElementById('modal-btn-close-users-bottom');
const createUserForm = document.getElementById('create-user-form');
const newUsernameInput = document.getElementById('new-user-username');
const newUserPasswordInput = document.getElementById('new-user-password');
const newUserIsAdminSwitch = document.getElementById('new-user-is-admin');
const btnSubmitCreateUser = document.getElementById('btn-submit-create-user');
const createUserErrorAlert = document.getElementById('create-user-error-alert');
const createUserErrorText = document.getElementById('create-user-error-text');
const usersTableBody = document.getElementById('users-table-body');
const usersCountTag = document.getElementById('users-count-tag');

// Elementos de Gerenciamento de Grupos & VLANs (Admin)
const btnOpenGroups = document.getElementById('btn-open-groups');
const groupsTableBody = document.getElementById('groups-table-body');
const groupsCountTag = document.getElementById('groups-count-tag');
const groupsSearchInput = document.getElementById('groups-search-input');
const btnOpenCreateGroup = document.getElementById('btn-open-create-group');

// Elementos do Modal de Criar Grupo
const groupCreateModal = document.getElementById('group-create-modal');
const modalBtnCloseCreateGroup = document.getElementById('modal-btn-close-create-group');
const btnCancelCreateGroup = document.getElementById('btn-cancel-create-group');
const createGroupForm = document.getElementById('create-group-form');
const createGroupNameInput = document.getElementById('create-group-name');
const createGroupIdInput = document.getElementById('create-group-id');
const createGroupVlansInput = document.getElementById('create-group-vlans');
const createGroupSubnetsInput = document.getElementById('create-group-subnets');
const createGroupIntervalInput = document.getElementById('create-group-interval');
const createGroupDescriptionInput = document.getElementById('create-group-description');
const createGroupModulesList = document.getElementById('create-group-modules-list');
const createGroupModulesCount = document.getElementById('create-group-modules-count');
const createGroupErrorAlert = document.getElementById('create-group-error-alert');
const createGroupErrorText = document.getElementById('create-group-error-text');

// Elementos do Modal de Detalhes do Grupo
const groupDetailsModal = document.getElementById('group-details-modal');
const modalBtnCloseGroupDetails = document.getElementById('modal-btn-close-group-details');
const gdModalTitle = document.getElementById('gd-modal-title');
const gdModalBadge = document.getElementById('gd-modal-badge');
const gdCountMachines = document.getElementById('gd-count-machines');
const gdCountModules = document.getElementById('gd-count-modules');
const gdMachinesTableBody = document.getElementById('gd-machines-table-body');
const gdModulesCardsGrid = document.getElementById('gd-modules-cards-grid');
const editGroupForm = document.getElementById('edit-group-form');
const egGroupId = document.getElementById('eg-group-id');
const egGroupName = document.getElementById('eg-group-name');
const egGroupVlans = document.getElementById('eg-group-vlans');
const egGroupSubnets = document.getElementById('eg-group-subnets');
const egGroupInterval = document.getElementById('eg-group-interval');
const egGroupDescription = document.getElementById('eg-group-description');
const egModulesCheckboxesList = document.getElementById('eg-modules-checkboxes-list');
const egModulesCountTag = document.getElementById('eg-modules-count-tag');
const egErrorAlert = document.getElementById('eg-error-alert');
const egErrorText = document.getElementById('eg-error-text');
const btnEgDeleteGroup = document.getElementById('btn-eg-delete-group');

// Elementos do Modal de Edição de Módulo
const moduleEditModal = document.getElementById('module-edit-modal');
const modalBtnCloseEditMod = document.getElementById('modal-btn-close-edit-mod');
const btnCancelEditMod = document.getElementById('btn-cancel-edit-mod');
const moduleEditForm = document.getElementById('module-edit-form');
const modEditId = document.getElementById('mod-edit-id');
const modEditType = document.getElementById('mod-edit-type');
const modEditName = document.getElementById('mod-edit-name');
const modEditTarget = document.getElementById('mod-edit-target');
const labelEditTarget = document.getElementById('label-edit-target');
const hintEditTarget = document.getElementById('hint-edit-target');
const groupEditTarget = document.getElementById('group-edit-target');
const groupEditCount = document.getElementById('group-edit-count');
const modEditCount = document.getElementById('mod-edit-count');
const groupEditSizeMb = document.getElementById('group-edit-size-mb');
const modEditSizeMb = document.getElementById('mod-edit-size-mb');
const groupEditTimeout = document.getElementById('group-edit-timeout');
const modEditTimeout = document.getElementById('mod-edit-timeout');
const groupEditProtocol = document.getElementById('group-edit-protocol');
const modEditProtocol = document.getElementById('mod-edit-protocol');
const modEditEnabled = document.getElementById('mod-edit-enabled');
const modEditErrorAlert = document.getElementById('mod-edit-error-alert');
const modEditErrorText = document.getElementById('mod-edit-error-text');


// Elementos OIDC / Keycloak
const oidcSSOWrapper = document.getElementById('oidc-sso-wrapper');
const oidcButtonText = document.getElementById('oidc-button-text');

// Elementos de Tipo de Conta no Modal de Usuários
const tabTypeLocal = document.getElementById('tab-type-local');
const tabTypeKeycloak = document.getElementById('tab-type-keycloak');
const newUserAuthTypeInput = document.getElementById('new-user-auth-type');
const fieldGroupPassword = document.getElementById('field-group-password');
const fieldKeycloakInfo = document.getElementById('field-keycloak-info');
const labelNewUserUsername = document.getElementById('label-new-user-username');
const btnCreateUserText = document.getElementById('btn-create-user-text');

// Toast Container
const toastContainer = document.getElementById('toast-container');

// Paleta de cores para gráficos comparativos multi-nós (Gradientes e bordas de alta definição)
const nodeColors = [
  { border: '#0284c7', fillTop: 'rgba(2, 132, 199, 0.24)', fillBottom: 'rgba(2, 132, 199, 0.01)' }, // Sky Blue
  { border: '#4f46e5', fillTop: 'rgba(79, 70, 229, 0.20)', fillBottom: 'rgba(79, 70, 229, 0.01)' }, // Royal Indigo
  { border: '#059669', fillTop: 'rgba(5, 150, 105, 0.22)', fillBottom: 'rgba(5, 150, 105, 0.01)' }, // Emerald
  { border: '#d97706', fillTop: 'rgba(217, 119, 6, 0.20)', fillBottom: 'rgba(217, 119, 6, 0.01)' }, // Amber
  { border: '#e11d48', fillTop: 'rgba(225, 29, 72, 0.20)', fillBottom: 'rgba(225, 29, 72, 0.01)' }, // Rose
  { border: '#0d9488', fillTop: 'rgba(13, 148, 136, 0.20)', fillBottom: 'rgba(13, 148, 136, 0.01)' }, // Teal
  { border: '#ea580c', fillTop: 'rgba(234, 88, 12, 0.20)', fillBottom: 'rgba(234, 88, 12, 0.01)' }, // Orange
  { border: '#475569', fillTop: 'rgba(71, 85, 105, 0.20)', fillBottom: 'rgba(71, 85, 105, 0.01)' }, // Slate
];

let refreshCountdown = 15;
let refreshInterval = null;

// Controle de Métrica de Throughput (Download / Upload / Ambos)
let currentBandwidthMetric = 'download';
let currentLatencyMetric = 'gateway'; // 'gateway' | 'core' | 'both'

window.handleLatencyMetricBtnClick = function(metric) {
  if (metric === currentLatencyMetric) return;
  currentLatencyMetric = metric;

  const toggleContainer = document.getElementById('latency-metric-toggle');
  if (toggleContainer) {
    const buttons = toggleContainer.querySelectorAll('.segment-btn');
    buttons.forEach(b => {
      b.classList.toggle('active', b.getAttribute('data-latency-metric') === metric);
    });
  }

  if (cachedMetricsSeries && cachedMetricsSeries.length > 0) {
    renderComparativeCharts(cachedMetricsSeries, cachedNodesList);
  } else {
    loadDashboardData();
  }
};
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
  const selectedNode = selectedChartNode || (selectNode && selectNode.value) || 'all';
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

// Inicialização Segura da Aplicação
function initApp() {
  try {
    initTheme();
  } catch (err) {
    console.warn('[INIT] Erro ao inicializar tema:', err);
  }

  try {
    setupEventListeners();
  } catch (err) {
    console.error('[INIT] Erro ao configurar listeners:', err);
  }

  try {
    initCharts();
  } catch (err) {
    console.warn('[INIT] Erro ao inicializar gráficos:', err);
  }

  checkAuthAndInitialize().catch(err => {
    console.warn('[INIT] Falha na verificação de sessão inicial:', err);
  });
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initApp);
} else {
  initApp();
}

/* ==========================================================================
   1. Autenticação & Sessão
   ========================================================================== */

async function checkAuthAndInitialize() {
  // 1. Verifica se o servidor possui integração com Keycloak/OIDC ativa
  try {
    await checkOIDCConfig();
  } catch (oidcErr) {
    console.warn('[AUTH] Falha ao verificar OIDC:', oidcErr);
  }

  // 2. Notifica caso haja erro retornado pelo redirecionamento do SSO
  const urlParams = new URLSearchParams(window.location.search);
  const authError = urlParams.get('auth_error');
  if (authError) {
    showToast(`Falha no login institucional: ${authError}`, 'toast-error');
    window.history.replaceState({}, document.title, window.location.pathname);
  }

  try {
    const res = await apiFetch('/api/v1/auth/me');
    if (res.ok) {
      const data = await res.json();
      if (data.authenticated && data.user) {
        setAuthenticatedUser(data.user);
        if (window.location.hash === '#machines') {
          switchView('machines');
        } else if (window.location.hash === '#groups' && data.user.role === 'admin') {
          switchView('groups');
        } else if (window.location.hash === '#modules' && data.user.role === 'admin') {
          switchView('modules');
        } else {
          switchView('dashboard');
        }
        try {
          await loadDashboardData();
          startAutoRefresh();
        } catch (dashErr) {
          console.warn('[AUTH] Aviso ao carregar dashboard após checkAuth:', dashErr);
        }
        return;
      }
    }
  } catch (err) {
    console.warn('Verificação de sessão:', err);
  }

  // Se não estiver logado, exibe o portal de entrada (Gateway de opções ou Login direto)
  const shouldOpenDirectLogin = Boolean(authError || window.location.hash === '#login' || urlParams.get('action') === 'login');
  showLoginModal(shouldOpenDirectLogin ? 'login' : 'gateway');
}

async function checkOIDCConfig() {
  if (!oidcSSOWrapper) return;
  try {
    const res = await apiFetch('/api/v1/auth/oidc/config');
    if (res.ok) {
      const data = await res.json();
      if (data.enabled) {
        oidcSSOWrapper.style.display = 'flex';
        if (oidcButtonText && data.button_text) {
          oidcButtonText.textContent = data.button_text;
        }
        return;
      }
    }
  } catch {}
  oidcSSOWrapper.style.display = 'none';
}

function setAuthenticatedUser(user) {
  currentUser = user;
  document.body.classList.remove('auth-mode');
  if (loginModalOverlay) loginModalOverlay.style.display = 'none';

  if (navUsername) navUsername.textContent = user.username;
  if (navRoleBadge) {
    navRoleBadge.textContent = user.role === 'admin' ? 'Administrador' : 'Operador';
    navRoleBadge.className = `role-badge ${user.role}`;
  }

  // Aba Máquinas visível para todos os usuários autenticados
  if (tabMachines) tabMachines.style.display = 'flex';

  // Se for admin, as abas e botões de administração estarão visíveis
  const adminSection = document.getElementById('sidebar-admin-section');
  if (user.role === 'admin') {
    if (modalBtnDeletePc) modalBtnDeletePc.style.display = 'inline-flex';
    if (btnOpenUsers) btnOpenUsers.style.display = 'flex';
    if (tabGroups) tabGroups.style.display = 'flex';
    if (tabModules) tabModules.style.display = 'flex';
    if (adminSection) adminSection.style.display = 'block';
  } else {
    if (modalBtnDeletePc) modalBtnDeletePc.style.display = 'none';
    if (btnOpenUsers) btnOpenUsers.style.display = 'none';
    if (tabGroups) tabGroups.style.display = 'none';
    if (tabModules) tabModules.style.display = 'none';
    if (adminSection) adminSection.style.display = 'none';
    if (window.location.hash === '#groups' || window.location.hash === '#modules') {
      switchView('dashboard');
    }
  }
}

function showLoginModal(mode = 'gateway') {
  currentUser = null;
  document.body.classList.add('auth-mode');
  if (loginModalOverlay) loginModalOverlay.style.display = 'flex';

  if (mode === 'login') {
    if (welcomeGatewayCard) welcomeGatewayCard.style.display = 'none';
    if (loginCard) loginCard.style.display = 'flex';
    setTimeout(() => {
      if (loginUsernameInput) loginUsernameInput.focus();
    }, 50);
  } else {
    // Exibe o Portal Inicial com as opções de Login ou Baixar Client
    if (welcomeGatewayCard) welcomeGatewayCard.style.display = 'flex';
    if (loginCard) loginCard.style.display = 'none';
    fetchClientDownloadInfo();
  }

  if (navUsername) navUsername.textContent = 'Não conectado';
  if (navRoleBadge) {
    navRoleBadge.textContent = '--';
    navRoleBadge.className = 'role-badge';
  }
  if (loginErrorAlert) loginErrorAlert.style.display = 'none';
  if (loginUsernameInput) loginUsernameInput.value = '';
  if (loginPasswordInput) loginPasswordInput.value = '';
  const adminSection = document.getElementById('sidebar-admin-section');
  if (adminSection) adminSection.style.display = 'none';
  if (btnOpenUsers) btnOpenUsers.style.display = 'none';
  if (usersModal) usersModal.style.display = 'none';
  if (tabMachines) tabMachines.style.display = 'none';
  if (btnOpenGroups) btnOpenGroups.style.display = 'none';
  if (tabGroups) tabGroups.style.display = 'none';
  if (tabModules) tabModules.style.display = 'none';
  if (groupsModal) groupsModal.style.display = 'none';
  switchView('dashboard');
  if (refreshInterval) clearInterval(refreshInterval);
}
window.showLoginModal = showLoginModal;

let configuredServerURL = '';

async function fetchClientDownloadInfo() {
  try {
    const res = await apiFetch('/api/v1/download/client/info');
    if (res.ok) {
      const data = await res.json();
      if (data.server_url) {
        configuredServerURL = data.server_url;
        updateLinuxInstallCommand();
      }
      if (gatewayClientFilesize) {
        if (data.available && data.size_formatted) {
          gatewayClientFilesize.textContent = `Instalador Windows • ${data.size_formatted}`;
        } else {
          gatewayClientFilesize.textContent = 'Instalador Windows 64-bit';
        }
      }
    }
  } catch (err) {
    console.warn('[DOWNLOAD] Falha ao consultar metadados do instalador:', err);
  }
}
window.fetchClientDownloadInfo = fetchClientDownloadInfo;

function toggleGatewayInstructions() {
  const panel = document.getElementById('gateway-instructions-panel');
  const btn = document.getElementById('btn-toggle-gateway-instructions');
  if (!panel || !btn) return;
  const isHidden = panel.style.display === 'none';
  if (isHidden) {
    panel.style.display = 'block';
    btn.classList.add('expanded');
    btn.setAttribute('aria-expanded', 'true');
  } else {
    panel.style.display = 'none';
    btn.classList.remove('expanded');
    btn.setAttribute('aria-expanded', 'false');
  }
}
window.toggleGatewayInstructions = toggleGatewayInstructions;

function selectDownloadOS(os) {
  const btnWin = document.getElementById('btn-tab-windows');
  const btnLin = document.getElementById('btn-tab-linux');
  const contentWin = document.getElementById('download-content-windows');
  const contentLin = document.getElementById('download-content-linux');
  const tutWin = document.getElementById('instructions-tutorial-windows');
  const tutLin = document.getElementById('instructions-tutorial-linux');
  const badgeText = document.getElementById('download-badge-text');

  if (os === 'linux') {
    if (btnWin) { btnWin.classList.remove('active', 'pill-emerald'); }
    if (btnLin) { btnLin.classList.add('active', 'pill-emerald'); }
    if (contentWin) contentWin.style.display = 'none';
    if (contentLin) contentLin.style.display = 'block';
    if (tutWin) tutWin.style.display = 'none';
    if (tutLin) tutLin.style.display = 'block';
    if (badgeText) badgeText.textContent = 'Sonda Linux / CLI';
    updateLinuxInstallCommand();
  } else {
    if (btnLin) { btnLin.classList.remove('active', 'pill-emerald'); }
    if (btnWin) { btnWin.classList.add('active', 'pill-emerald'); }
    if (contentLin) contentLin.style.display = 'none';
    if (contentWin) contentWin.style.display = 'block';
    if (tutLin) tutLin.style.display = 'none';
    if (tutWin) tutWin.style.display = 'block';
    if (badgeText) badgeText.textContent = 'Sonda Windows';
  }
}
window.selectDownloadOS = selectDownloadOS;

function getEffectiveServerOrigin() {
  if (configuredServerURL && configuredServerURL.trim() !== '') {
    return configuredServerURL.trim().replace(/\/+$/, '');
  }
  return window.location.origin || 'http://localhost:8080';
}

function updateLinuxInstallCommand() {
  const origin = getEffectiveServerOrigin();
  const cmd = `curl -fsSL ${origin}/downloads/install-probe.sh | bash -s -- ${origin}`;
  const cmdSpan = document.getElementById('linux-install-cmd');
  if (cmdSpan) cmdSpan.textContent = cmd;
  const modalCmdSpan = document.getElementById('modal-linux-cmd');
  if (modalCmdSpan) modalCmdSpan.textContent = cmd;
}
setTimeout(updateLinuxInstallCommand, 200);
window.addEventListener('load', () => {
  fetchClientDownloadInfo();
  updateLinuxInstallCommand();
});

function copyLinuxInstallCommand() {
  const origin = getEffectiveServerOrigin();
  const cmd = `curl -fsSL ${origin}/downloads/install-probe.sh | bash -s -- ${origin}`;
  const label = document.getElementById('copy-btn-label');

  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(cmd).then(() => {
      if (label) label.textContent = 'Copiado!';
      showToast('Comando copiado! Cole no terminal do seu Debian.', 'toast-success');
      setTimeout(() => { if (label) label.textContent = 'Copiar'; }, 3000);
    }).catch(() => fallbackCopy(cmd, label));
  } else {
    fallbackCopy(cmd, label);
  }
}
window.copyLinuxInstallCommand = copyLinuxInstallCommand;

function copyLinuxModalCommand() {
  const origin = getEffectiveServerOrigin();
  const cmd = `curl -fsSL ${origin}/downloads/install-probe.sh | bash -s -- ${origin}`;
  const label = document.getElementById('modal-copy-btn-label');

  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(cmd).then(() => {
      if (label) label.textContent = 'Copiado!';
      showToast('Comando copiado! Cole no terminal do seu Debian.', 'toast-success');
      setTimeout(() => { if (label) label.textContent = 'Copiar'; }, 3000);
    }).catch(() => fallbackCopy(cmd, label));
  } else {
    fallbackCopy(cmd, label);
  }
}
window.copyLinuxModalCommand = copyLinuxModalCommand;

function fallbackCopy(text, label) {
  const ta = document.createElement('textarea');
  ta.value = text;
  document.body.appendChild(ta);
  ta.select();
  try {
    document.execCommand('copy');
    if (label) label.textContent = 'Copiado!';
    showToast('Comando copiado! Cole no terminal do seu Debian.', 'toast-success');
    setTimeout(() => { if (label) label.textContent = 'Copiar'; }, 3000);
  } catch (err) {
    prompt('Copie o comando abaixo manualmente:', text);
  }
  document.body.removeChild(ta);
}

function openLinuxInstallModal() {
  const modal = document.getElementById('modal-linux-install');
  if (!modal) return;
  const cmdSpan = document.getElementById('modal-linux-cmd');
  const origin = getEffectiveServerOrigin();
  if (cmdSpan) {
    cmdSpan.textContent = `curl -fsSL ${origin}/downloads/install-probe.sh | bash -s -- ${origin}`;
  }
  modal.style.display = 'flex';
}
window.openLinuxInstallModal = openLinuxInstallModal;

function closeLinuxInstallModal() {
  const modal = document.getElementById('modal-linux-install');
  if (modal) modal.style.display = 'none';
}
window.closeLinuxInstallModal = closeLinuxInstallModal;

function copyLinuxModalCommand() {
  const origin = getEffectiveServerOrigin();
  const cmd = `curl -fsSL ${origin}/downloads/install-probe.sh | sudo bash`;
  const label = document.getElementById('modal-copy-btn-label');

  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(cmd).then(() => {
      if (label) label.textContent = 'Copiado!';
      showToast('Comando copiado! Cole no terminal do seu Debian.', 'toast-success');
      setTimeout(() => { if (label) label.textContent = 'Copiar'; }, 3000);
    }).catch(() => fallbackCopy(cmd, label));
  } else {
    fallbackCopy(cmd, label);
  }
}
window.copyLinuxModalCommand = copyLinuxModalCommand;

/* ==========================================================================
   Controle de Views / Páginas (Dashboard vs Máquinas vs Grupos vs Módulos)
   ========================================================================== */

function switchView(viewName) {
  const pageTitle = document.getElementById('page-title');

  // Remove classe active de todas as abas de navegação
  [tabDashboard, tabMachines, tabGroups, tabModules].forEach(t => {
    if (t) t.classList.remove('active');
  });

  // Oculta todas as views principais e remove classe active
  [viewDashboard, viewMachines, viewGroups, viewModules].forEach(v => {
    if (v) {
      v.style.display = 'none';
      v.classList.remove('active');
    }
  });

  if (viewName === 'machines') {
    if (viewMachines) {
      viewMachines.style.display = 'block';
      viewMachines.classList.add('active');
    }
    if (tabMachines) tabMachines.classList.add('active');
    if (pageTitle) pageTitle.textContent = 'Máquinas & Sondas';
    window.location.hash = '#machines';
    loadAdminMachines();
  } else if (viewName === 'groups') {
    if (!currentUser || currentUser.role !== 'admin') {
      showToast('Acesso restrito a administradores', 'toast-error');
      switchView('dashboard');
      return;
    }
    if (viewGroups) {
      viewGroups.style.display = 'block';
      viewGroups.classList.add('active');
    }
    if (tabGroups) tabGroups.classList.add('active');
    if (pageTitle) pageTitle.textContent = 'Grupos & VLANs';
    window.location.hash = '#groups';
    loadGroupsList();
  } else if (viewName === 'modules') {
    if (!currentUser || currentUser.role !== 'admin') {
      showToast('Acesso restrito a administradores', 'toast-error');
      switchView('dashboard');
      return;
    }
    if (viewModules) {
      viewModules.style.display = 'block';
      viewModules.classList.add('active');
    }
    if (tabModules) tabModules.classList.add('active');
    if (pageTitle) pageTitle.textContent = 'Módulos JSON';
    window.location.hash = '#modules';
    loadModulesCatalog();
  } else {
    if (viewDashboard) {
      viewDashboard.style.display = 'block';
      viewDashboard.classList.add('active');
    }
    if (tabDashboard) tabDashboard.classList.add('active');
    if (pageTitle) pageTitle.textContent = 'Dashboard';
    window.location.hash = '#dashboard';
  }
}
window.switchView = switchView;

async function handleLoginSubmit(e) {
  if (e && e.preventDefault) e.preventDefault();

  const username = loginUsernameInput ? loginUsernameInput.value.trim() : '';
  const password = loginPasswordInput ? loginPasswordInput.value.trim() : '';

  if (!username) {
    if (loginErrorText) loginErrorText.textContent = 'Informe o usuário para autenticar';
    if (loginErrorAlert) loginErrorAlert.style.display = 'flex';
    if (loginUsernameInput) loginUsernameInput.focus();
    return;
  }

  if (!password) {
    if (loginErrorText) loginErrorText.textContent = 'Informe a senha para autenticar';
    if (loginErrorAlert) loginErrorAlert.style.display = 'flex';
    if (loginPasswordInput) loginPasswordInput.focus();
    return;
  }

  if (btnLoginSubmit) {
    btnLoginSubmit.disabled = true;
    btnLoginSubmit.classList.add('loading');
  }
  if (loginErrorAlert) loginErrorAlert.style.display = 'none';

  try {
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });

    let data = {};
    try {
      data = await res.json();
    } catch {
      data = {};
    }

    if (!res.ok) {
      throw new Error(data.error || 'Usuário ou senha incorretos');
    }

    if (data.token) {
      authToken = data.token;
      try {
        localStorage.setItem('noc_token', data.token);
        sessionStorage.setItem('noc_token', data.token);
      } catch (storageErr) {
        console.warn('Falha ao salvar token no storage:', storageErr);
      }
    }

    setAuthenticatedUser(data.user);
    showToast(`Bem-vindo, ${data.user.username}!`, 'toast-success');

    // Carrega dados de forma protegida para não quebrar o login caso a renderização falhe
    try {
      await loadDashboardData();
      startAutoRefresh();
    } catch (dashErr) {
      console.warn('[AUTH] Aviso ao carregar dashboard pós-login:', dashErr);
    }
  } catch (err) {
    console.error('[AUTH] Falha ao autenticar:', err);
    if (loginErrorText) {
      loginErrorText.textContent = err.message || 'Usuário ou senha incorretos';
    }
    if (loginErrorAlert) {
      loginErrorAlert.style.display = 'flex';
    }
  } finally {
    if (btnLoginSubmit) {
      btnLoginSubmit.disabled = false;
      btnLoginSubmit.classList.remove('loading');
    }
  }
}
window.handleLoginSubmit = handleLoginSubmit;

async function handleLogout() {
  authToken = '';
  try {
    localStorage.removeItem('noc_token');
    sessionStorage.removeItem('noc_token');
  } catch {}
  try {
    await apiFetch('/api/v1/auth/logout', { method: 'POST' });
  } catch (err) {
    console.error('Erro ao sair:', err);
  }
  showToast('Sessão encerrada com sucesso', 'toast-info');
  window.history.replaceState(null, '', window.location.pathname);
  showLoginModal('gateway');
}
window.handleLogout = handleLogout;

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
  // Gateway Inicial: Alternância entre Portal de Opções e Formulário de Login
  if (btnGatewayGoLogin) {
    btnGatewayGoLogin.addEventListener('click', () => {
      showLoginModal('login');
    });
  }
  if (btnBackToGateway) {
    btnBackToGateway.addEventListener('click', () => {
      showLoginModal('gateway');
    });
  }
  if (btnToggleGatewayInstructions) {
    btnToggleGatewayInstructions.addEventListener('click', toggleGatewayInstructions);
  }

  // Feedback de download ao clicar nos links do instalador
  const notifyDownload = () => {
    showToast('Download do instalador iniciado! Execute-o no Windows para monitorar este computador.', 'toast-success');
  };
  if (btnGatewayDownloadClient) {
    btnGatewayDownloadClient.addEventListener('click', notifyDownload);
  }
  if (btnLoginDownloadClient) {
    btnLoginDownloadClient.addEventListener('click', notifyDownload);
  }
  if (btnSidebarDownloadClient) {
    btnSidebarDownloadClient.addEventListener('click', notifyDownload);
  }

  if (loginForm) loginForm.addEventListener('submit', handleLoginSubmit);
  if (btnLoginSubmit) btnLoginSubmit.addEventListener('click', handleLoginSubmit);
  if (btnLogout) btnLogout.addEventListener('click', handleLogout);

  if (selectNode) {
    selectNode.addEventListener('change', () => {
      selectedChartNode = selectNode.value || 'all';
      updateChartFocusUI();
      updateMetricToggleVisibility();
      loadDashboardData();
    });
  }

  setupSegmentedControl();

  if (btnRefresh) {
    btnRefresh.addEventListener('click', () => {
      refreshCountdown = 15;
      loadDashboardData();
    });
  }

  // Fechamento do Modal de PC
  if (modalPcClose) modalPcClose.addEventListener('click', closePCModal);
  if (modalBtnCloseBottom) modalBtnCloseBottom.addEventListener('click', closePCModal);
  if (pcModal) {
    pcModal.addEventListener('click', (e) => {
      if (e.target === pcModal) closePCModal();
    });
  }

  // Botão de Excluir dentro do Modal de PC
  if (modalBtnDeletePc) {
    modalBtnDeletePc.addEventListener('click', () => {
      const nodeId = modalBtnDeletePc.getAttribute('data-node-id');
      if (nodeId) promptDeleteNode(nodeId);
    });
  }

  // Confirmação de Exclusão
  if (btnCancelDelete) btnCancelDelete.addEventListener('click', closeDeleteModal);
  if (deleteConfirmModal) {
    deleteConfirmModal.addEventListener('click', (e) => {
      if (e.target === deleteConfirmModal) closeDeleteModal();
    });
  }
  if (btnConfirmDelete) btnConfirmDelete.addEventListener('click', executeDeleteNode);

  // Alternância de Visualização de Senha
  initPasswordToggle('login-password', 'btn-toggle-login-password');
  initPasswordToggle('new-user-password', 'btn-toggle-new-user-password');

  // Gerenciamento de Usuários (Admin)
  if (btnOpenUsers) {
    btnOpenUsers.addEventListener('click', openUsersModal);
  }
  if (modalBtnCloseUsers) {
    modalBtnCloseUsers.addEventListener('click', closeUsersModal);
  }
  if (modalBtnCloseUsersBottom) {
    modalBtnCloseUsersBottom.addEventListener('click', closeUsersModal);
  }
  if (usersModal) {
    usersModal.addEventListener('click', (e) => {
      if (e.target === usersModal) closeUsersModal();
    });
  }
  if (createUserForm) {
    createUserForm.addEventListener('submit', handleCreateUserSubmit);
  }

  // Gerenciamento de Grupos & VLANs (Admin)
  if (tabDashboard) {
    tabDashboard.addEventListener('click', () => switchView('dashboard'));
  }
  if (tabGroups) {
    tabGroups.addEventListener('click', () => switchView('groups'));
  }
  if (btnOpenGroups) {
    btnOpenGroups.addEventListener('click', () => switchView('groups'));
  }
  if (groupsSearchInput) {
    groupsSearchInput.addEventListener('input', renderGroupsList);
  }
  if (btnOpenCreateGroup) {
    btnOpenCreateGroup.addEventListener('click', openCreateGroupModal);
  }
  if (modalBtnCloseCreateGroup) {
    modalBtnCloseCreateGroup.addEventListener('click', closeCreateGroupModal);
  }
  if (btnCancelCreateGroup) {
    btnCancelCreateGroup.addEventListener('click', closeCreateGroupModal);
  }
  if (groupCreateModal) {
    groupCreateModal.addEventListener('click', (e) => {
      if (e.target === groupCreateModal) closeCreateGroupModal();
    });
  }
  if (createGroupForm) {
    createGroupForm.addEventListener('submit', handleCreateGroupSubmit);
  }
  if (createGroupNameInput && createGroupIdInput) {
    createGroupNameInput.addEventListener('input', (e) => {
      if (!createGroupIdInput.dataset.manual) {
        createGroupIdInput.value = e.target.value.toLowerCase().trim().replace(/[^a-z0-9_-]/g, '_');
      }
    });
    createGroupIdInput.addEventListener('input', () => {
      createGroupIdInput.dataset.manual = 'true';
    });
  }
  if (modalBtnCloseGroupDetails) {
    modalBtnCloseGroupDetails.addEventListener('click', closeGroupDetailsModal);
  }
  if (groupDetailsModal) {
    groupDetailsModal.addEventListener('click', (e) => {
      if (e.target === groupDetailsModal) closeGroupDetailsModal();
    });
  }
  if (editGroupForm) {
    editGroupForm.addEventListener('submit', handleSaveGroupEditSubmit);
  }
  if (btnEgDeleteGroup) {
    btnEgDeleteGroup.addEventListener('click', () => {
      if (currentViewingGroupId) {
        const g = currentGroups.find(item => item.id === currentViewingGroupId);
        handleDeleteGroup(currentViewingGroupId, g ? g.name : currentViewingGroupId);
      }
    });
  }


  // Filtros em Tempo Real da Tabela de Nós (Telemetria)
  if (nodesSearchInput) {
    nodesSearchInput.addEventListener('input', (e) => {
      nodeFilterState.search = e.target.value.toLowerCase().trim();
      telemetryPaginationState.page = 1;
      applyNodeFilters();
    });
  }
  if (nodesStatusFilter) {
    nodesStatusFilter.addEventListener('change', (e) => {
      nodeFilterState.status = e.target.value;
      telemetryPaginationState.page = 1;
      applyNodeFilters();
    });
  }
  if (nodesGroupFilter) {
    nodesGroupFilter.addEventListener('change', (e) => {
      nodeFilterState.group = e.target.value;
      telemetryPaginationState.page = 1;
      applyNodeFilters();
    });
  }

  // Filtro de Data e Paginação da Telemetria (Dashboard)
  const telemetryDateFilter = document.getElementById('telemetry-date-filter');
  const btnTelemetryFilterToday = document.getElementById('btn-telemetry-filter-today');
  const btnTelemetryFilterAll = document.getElementById('btn-telemetry-filter-all');
  const btnTelemetryPrev = document.getElementById('btn-telemetry-prev');
  const btnTelemetryNext = document.getElementById('btn-telemetry-next');

  if (telemetryDateFilter) {
    telemetryDateFilter.addEventListener('change', (e) => {
      telemetryPageState.date = e.target.value;
      telemetryPageState.page = 1;
      applyTelemetryFilters();
      loadDashboardData(telemetryPageState.date);
    });
  }
  if (btnTelemetryFilterToday) {
    btnTelemetryFilterToday.addEventListener('click', () => {
      const todayStr = new Date().toLocaleDateString('en-CA');
      telemetryPageState.date = todayStr;
      if (telemetryDateFilter) telemetryDateFilter.value = todayStr;
      telemetryPageState.page = 1;
      applyTelemetryFilters();
      loadDashboardData(todayStr);
    });
  }
  if (btnTelemetryFilterAll) {
    btnTelemetryFilterAll.addEventListener('click', () => {
      telemetryPageState.date = '';
      if (telemetryDateFilter) telemetryDateFilter.value = '';
      telemetryPageState.page = 1;
      applyTelemetryFilters();
      loadDashboardData('');
    });
  }
  if (btnTelemetryPrev) {
    btnTelemetryPrev.addEventListener('click', () => {
      if (telemetryPageState.page > 1) {
        telemetryPageState.page--;
        applyTelemetryFilters();
      }
    });
  }
  if (btnTelemetryNext) {
    btnTelemetryNext.addEventListener('click', () => {
      telemetryPageState.page++;
      applyTelemetryFilters();
    });
  }

  // Botão de Resetar Foco dos Gráficos
  const resetChartBtn = document.getElementById('btn-reset-chart-focus');
  if (resetChartBtn) {
    resetChartBtn.addEventListener('click', () => {
      selectedChartNode = 'all';
      if (selectNode) selectNode.value = 'all';
      updateChartFocusUI();
      updateMetricToggleVisibility();
      loadDashboardData();
    });
  }

  // Botão de Detalhes do PC a partir do cabeçalho da Máquina Única
  const btnSmDetails = document.getElementById('btn-sm-details');
  if (btnSmDetails) {
    btnSmDetails.addEventListener('click', () => {
      if (!selectedChartNode || selectedChartNode === 'all') return;
      let summary = currentSummaries.find(item => item.node_id === selectedChartNode);
      if (!summary) {
        summary = { node_id: selectedChartNode, system_info: {}, metrics: {} };
      }
      openPCDetailsModal(summary);
    });
  }

  // Botão de Sair da Visão de Máquina Única
  const btnExitSm = document.getElementById('btn-exit-single-machine');
  if (btnExitSm) {
    btnExitSm.addEventListener('click', () => {
      selectedChartNode = 'all';
      if (selectNode) selectNode.value = 'all';
      updateChartFocusUI();
      updateMetricToggleVisibility();
      loadDashboardData();
      showToast('Exibindo gráficos consolidados de todas as sondas', 'toast-info');
    });
  }

  // Cliques na Tabela de Nós (Clique na linha ou nome: foca gráficos; Inspecionar: abre modal; Excluir: confirma)
  if (nodesTableBody) {
    nodesTableBody.addEventListener('click', (e) => {
      // 1. Inspecionar Detalhes de Hardware/Rede
      const inspectBtn = e.target.closest('.btn-inspect');
      if (inspectBtn) {
        e.stopPropagation();
        const nodeId = inspectBtn.getAttribute('data-node-id');
        const summary = currentSummaries.find(item => item.node_id === nodeId);
        if (summary) openPCDetailsModal(summary);
        return;
      }

      // 2. Exclusão de Computador
      const deleteBtn = e.target.closest('.btn-delete-node');
      if (deleteBtn) {
        e.stopPropagation();
        const nodeId = deleteBtn.getAttribute('data-node-id');
        if (nodeId) promptDeleteNode(nodeId);
        return;
      }

      // 3. Se o clique foi no botão com o link/nome do nó, ele já possui handler próprio
      if (e.target.closest('.btn-node-link')) {
        return;
      }

      // 4. Clique em qualquer outro lugar da linha: Foca nos gráficos da máquina
      const row = e.target.closest('.node-row');
      if (row) {
        const nodeId = row.getAttribute('data-node-id');
        if (nodeId) selectNodeForCharts(nodeId);
      }
    });
  }

  // Navegação para Novas Abas
  if (tabMachines) {
    tabMachines.addEventListener('click', () => switchView('machines'));
  }
  if (tabModules) {
    tabModules.addEventListener('click', () => switchView('modules'));
  }
  if (btnRefreshMachines) {
    btnRefreshMachines.addEventListener('click', loadAdminMachines);
  }

  // Filtros em Tempo Real da Aba de Máquinas
  if (machinesSearchInput) {
    machinesSearchInput.addEventListener('input', (e) => {
      machineFilterState.search = e.target.value.toLowerCase().trim();
      machinesPaginationState.page = 1;
      applyMachineFilters();
    });
  }
  if (machinesGroupFilter) {
    machinesGroupFilter.addEventListener('change', (e) => {
      machineFilterState.group = e.target.value;
      machinesPaginationState.page = 1;
      applyMachineFilters();
    });
  }
  if (machinesModuleFilter) {
    machinesModuleFilter.addEventListener('change', (e) => {
      machineFilterState.module = e.target.value;
      machinesPaginationState.page = 1;
      applyMachineFilters();
    });
  }
  if (machinesStatusFilter) {
    machinesStatusFilter.addEventListener('change', (e) => {
      machineFilterState.status = e.target.value;
      machinesPaginationState.page = 1;
      applyMachineFilters();
    });
  }

  // Filtro de Data e Paginação da Tabela de Máquinas
  const machinesDateFilter = document.getElementById('machines-date-filter');
  const btnMachinesFilterToday = document.getElementById('btn-machines-filter-today');
  const btnMachinesFilterAll = document.getElementById('btn-machines-filter-all');
  const btnMachinesPrev = document.getElementById('btn-machines-prev');
  const btnMachinesNext = document.getElementById('btn-machines-next');

  if (machinesDateFilter) {
    machinesDateFilter.addEventListener('change', (e) => {
      machineFilterState.date = e.target.value;
      machinesPaginationState.page = 1;
      applyMachineFilters();
    });
  }
  if (btnMachinesFilterToday) {
    btnMachinesFilterToday.addEventListener('click', () => {
      const todayStr = new Date().toLocaleDateString('en-CA');
      machineFilterState.date = todayStr;
      if (machinesDateFilter) machinesDateFilter.value = todayStr;
      machinesPaginationState.page = 1;
      applyMachineFilters();
    });
  }
  if (btnMachinesFilterAll) {
    btnMachinesFilterAll.addEventListener('click', () => {
      machineFilterState.date = '';
      if (machinesDateFilter) machinesDateFilter.value = '';
      machinesPaginationState.page = 1;
      applyMachineFilters();
    });
  }
  if (btnMachinesPrev) {
    btnMachinesPrev.addEventListener('click', () => {
      if (machinesPaginationState.page > 1) {
        machinesPaginationState.page--;
        applyMachineFilters();
      }
    });
  }
  if (btnMachinesNext) {
    btnMachinesNext.addEventListener('click', () => {
      machinesPaginationState.page++;
      applyMachineFilters();
    });
  }

  // Modal de Diagnóstico de Ping Forçado
  const btnClosePingModal = document.getElementById('btn-close-ping-modal');
  const btnClosePingModalFooter = document.getElementById('btn-close-ping-modal-footer');
  const modalPingDiagnostic = document.getElementById('modal-ping-diagnostic');
  const btnRetestPing = document.getElementById('btn-retest-ping');
  const btnSmPing = document.getElementById('btn-sm-ping');

  if (btnClosePingModal) btnClosePingModal.addEventListener('click', closePingModal);
  if (btnClosePingModalFooter) btnClosePingModalFooter.addEventListener('click', closePingModal);
  if (modalPingDiagnostic) {
    modalPingDiagnostic.addEventListener('click', (e) => {
      if (e.target === modalPingDiagnostic) closePingModal();
    });
  }
  if (btnRetestPing) {
    btnRetestPing.addEventListener('click', () => {
      if (currentPingTarget.hostname || currentPingTarget.ip) {
        executeAdminPing(currentPingTarget.hostname, currentPingTarget.ip);
      }
    });
  }
  if (btnSmPing) {
    btnSmPing.addEventListener('click', triggerSingleMachinePing);
  }

  // Modal de Métricas da Máquina
  if (modalBtnCloseMm) modalBtnCloseMm.addEventListener('click', closeMachineMetricsModal);
  if (modalBtnCloseMmBottom) modalBtnCloseMmBottom.addEventListener('click', closeMachineMetricsModal);
  if (machineMetricsModal) {
    machineMetricsModal.addEventListener('click', (e) => {
      if (e.target === machineMetricsModal) closeMachineMetricsModal();
    });
  }
  if (mmBtnFocusDashboard) {
    mmBtnFocusDashboard.addEventListener('click', () => {
      if (currentInspectedMachine) {
        selectNodeForCharts(currentInspectedMachine);
        closeMachineMetricsModal();
        switchView('dashboard');
      }
    });
  }

  // Modal de Configuração da Máquina
  if (modalBtnCloseMc) modalBtnCloseMc.addEventListener('click', closeMachineConfigModal);
  if (machineConfigModal) {
    machineConfigModal.addEventListener('click', (e) => {
      if (e.target === machineConfigModal) closeMachineConfigModal();
    });
  }
  if (machineConfigForm) {
    machineConfigForm.addEventListener('submit', handleSaveMachineConfigSubmit);
  }
  const btnMcResetGroup = document.getElementById('btn-mc-reset-group');
  if (btnMcResetGroup) {
    btnMcResetGroup.addEventListener('click', () => {
      const activeGrp = currentGroups.find(g => g.id === mcSelectGroup.value);
      const grpTests = (activeGrp && activeGrp.profile && activeGrp.profile.tests) ? activeGrp.profile.tests : [];
      const grpIds = new Set(grpTests.map(t => t.id));
      renderMachineModulesCheckboxes(mcSelectGroup.value, grpIds);
      showToast('Módulos redefinidos para o padrão do grupo!', 'toast-info');
    });
  }
  if (mcSelectGroup) {
    mcSelectGroup.addEventListener('change', () => {
      const activeGrp = currentGroups.find(g => g.id === mcSelectGroup.value);
      const grpTests = (activeGrp && activeGrp.profile && activeGrp.profile.tests) ? activeGrp.profile.tests : [];
      const grpIds = new Set(grpTests.map(t => t.id));
      renderMachineModulesCheckboxes(mcSelectGroup.value, grpIds);
    });
  }
  if (mcModulesCheckboxesList) {
    mcModulesCheckboxesList.addEventListener('change', (e) => {
      if (e.target.classList.contains('mc-module-cb')) {
        updateMachineConfigStatusSummary(mcSelectGroup.value);
      }
    });
  }

  // Modal de Upload de Módulo JSON
  if (btnOpenUploadModule) btnOpenUploadModule.addEventListener('click', openModuleUploadModal);
  if (modalBtnCloseUploadMod) modalBtnCloseUploadMod.addEventListener('click', closeModuleUploadModal);
  if (btnCancelUploadMod) btnCancelUploadMod.addEventListener('click', closeModuleUploadModal);
  if (moduleUploadModal) {
    moduleUploadModal.addEventListener('click', (e) => {
      if (e.target === moduleUploadModal) closeModuleUploadModal();
    });
  }
  if (btnSubmitUploadMod) btnSubmitUploadMod.addEventListener('click', handleSaveModuleUploadSubmit);

  // Modal de Edição de Módulo
  if (modalBtnCloseEditMod) modalBtnCloseEditMod.addEventListener('click', closeEditModuleModal);
  if (btnCancelEditMod) btnCancelEditMod.addEventListener('click', closeEditModuleModal);
  if (moduleEditModal) {
    moduleEditModal.addEventListener('click', (e) => {
      if (e.target === moduleEditModal) closeEditModuleModal();
    });
  }
  if (moduleEditForm) moduleEditForm.addEventListener('submit', handleSaveModuleEditSubmit);


  // Dropzone de Arquivos JSON
  if (jsonDropzone && modFileInput) {
    jsonDropzone.addEventListener('click', () => modFileInput.click());
    modFileInput.addEventListener('change', handleFileSelect);

    jsonDropzone.addEventListener('dragover', (e) => {
      e.preventDefault();
      jsonDropzone.classList.add('dragover');
    });
    jsonDropzone.addEventListener('dragleave', () => {
      jsonDropzone.classList.remove('dragover');
    });
    jsonDropzone.addEventListener('drop', (e) => {
      e.preventDefault();
      jsonDropzone.classList.remove('dragover');
      if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
        processUploadedJsonFile(e.dataTransfer.files[0]);
      }
    });
  }

  window.addEventListener('hashchange', () => {
    if (window.location.hash === '#machines') {
      switchView('machines');
    } else if (window.location.hash === '#groups') {
      switchView('groups');
    } else if (window.location.hash === '#modules') {
      switchView('modules');
    } else {
      switchView('dashboard');
    }
  });

  if (tabTypeLocal && tabTypeKeycloak) {
    tabTypeLocal.addEventListener('click', () => setAccountType('local'));
    tabTypeKeycloak.addEventListener('click', () => setAccountType('keycloak'));
  }

  // Fechar modais ao pressionar tecla ESC
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      if (deleteConfirmModal && deleteConfirmModal.style.display !== 'none') {
        closeDeleteModal();
      } else if (usersModal && usersModal.style.display !== 'none') {
        closeUsersModal();
      } else if (groupsModal && groupsModal.style.display !== 'none') {
        closeGroupsModal();
      } else if (pcModal && pcModal.style.display !== 'none') {
        closePCModal();
      } else if (machineMetricsModal && machineMetricsModal.style.display !== 'none') {
        closeMachineMetricsModal();
      } else if (machineConfigModal && machineConfigModal.style.display !== 'none') {
        closeMachineConfigModal();
      } else if (moduleUploadModal && moduleUploadModal.style.display !== 'none') {
        closeModuleUploadModal();
      }
    }
  });
}

/* ==========================================================================
   Funções de Visualização de Senha e Gerenciamento de Usuários (Admin)
   ========================================================================== */

function escapeHtml(str) {
  if (!str) return '';
  return String(str).replace(/[&<>"']/g, m => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#039;'
  })[m]);
}

function initPasswordToggle(inputId, toggleBtnId) {
  const input = document.getElementById(inputId);
  const btn = document.getElementById(toggleBtnId);
  if (!input || !btn) return;

  const showIcon = btn.querySelector('.eye-icon-show');
  const hideIcon = btn.querySelector('.eye-icon-hide');

  btn.addEventListener('click', () => {
    const isPassword = input.type === 'password';
    input.type = isPassword ? 'text' : 'password';

    if (showIcon && hideIcon) {
      if (isPassword) {
        showIcon.style.display = 'none';
        hideIcon.style.display = 'inline-block';
        btn.title = 'Ocultar senha';
      } else {
        showIcon.style.display = 'inline-block';
        hideIcon.style.display = 'none';
        btn.title = 'Mostrar senha';
      }
    }
  });
}

function setAccountType(type) {
  if (!tabTypeLocal || !tabTypeKeycloak) return;
  if (type === 'keycloak') {
    tabTypeKeycloak.classList.add('active');
    tabTypeLocal.classList.remove('active');
    if (newUserAuthTypeInput) newUserAuthTypeInput.value = 'keycloak';
    if (fieldGroupPassword) fieldGroupPassword.style.display = 'none';
    if (fieldKeycloakInfo) fieldKeycloakInfo.style.display = 'flex';
    if (newUserPasswordInput) newUserPasswordInput.required = false;
    if (labelNewUserUsername) labelNewUserUsername.textContent = 'Usuário Institucional (Keycloak)';
    if (newUsernameInput) newUsernameInput.placeholder = 'Ex: nicolas.fonseca';
    if (btnCreateUserText) btnCreateUserText.textContent = 'Vincular Usuário Keycloak';
  } else {
    tabTypeLocal.classList.add('active');
    tabTypeKeycloak.classList.remove('active');
    if (newUserAuthTypeInput) newUserAuthTypeInput.value = 'local';
    if (fieldGroupPassword) fieldGroupPassword.style.display = 'flex';
    if (fieldKeycloakInfo) fieldKeycloakInfo.style.display = 'none';
    if (newUserPasswordInput) newUserPasswordInput.required = true;
    if (labelNewUserUsername) labelNewUserUsername.textContent = 'Nome de Usuário';
    if (newUsernameInput) newUsernameInput.placeholder = 'Ex: joao.silva';
    if (btnCreateUserText) btnCreateUserText.textContent = 'Cadastrar Usuário';
  }
}

async function openUsersModal() {
  if (!usersModal) return;
  usersModal.style.display = 'flex';
  if (createUserForm) createUserForm.reset();
  setAccountType('local');
  if (createUserErrorAlert) createUserErrorAlert.style.display = 'none';
  await loadUsersList();
}

function closeUsersModal() {
  if (usersModal) usersModal.style.display = 'none';
}

async function loadUsersList() {
  if (!usersTableBody) return;
  usersTableBody.innerHTML = `<tr><td colspan="5" class="text-center py-4" style="color: var(--text-muted);"><div class="spinner" style="width:20px;height:20px;margin:0 auto 8px;"></div>Carregando usuários...</td></tr>`;

  try {
    const res = await apiFetch('/api/v1/users');
    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || 'Falha ao carregar lista de usuários');
    }

    const users = data.users || [];
    if (usersCountTag) {
      usersCountTag.textContent = `${users.length} usuário${users.length !== 1 ? 's' : ''}`;
    }

    if (users.length === 0) {
      usersTableBody.innerHTML = `<tr><td colspan="5" class="text-center py-4" style="color: var(--text-muted);">Nenhum usuário cadastrado.</td></tr>`;
      return;
    }

    usersTableBody.innerHTML = users.map(u => {
      const isCur = currentUser && currentUser.username.toLowerCase() === u.username.toLowerCase();

      // Badge de Origem
      let originBadge = '<span class="badge-origin-local">Local</span>';
      if (u.is_system) {
        originBadge = '<span class="badge-origin-system">Sistema</span>';
      } else if (u.auth_type === 'keycloak') {
        originBadge = '<span class="badge-origin-keycloak">Keycloak</span>';
      }

      // Badge de Perfil + Botão de Alteração Rápida
      const roleBadge = u.role === 'admin'
        ? `<span class="role-badge-admin">Admin</span>`
        : `<span class="role-badge-operator">Operador</span>`;

      let roleActionBtn = '';
      if (!u.is_system && !isCur) {
        if (u.role === 'admin') {
          roleActionBtn = `<button class="btn-role-toggle to-operator" onclick="handleToggleRole('${escapeHtml(u.username)}', 'operator')" title="Rebaixar para Operador">Tornar Operador</button>`;
        } else {
          roleActionBtn = `<button class="btn-role-toggle to-admin" onclick="handleToggleRole('${escapeHtml(u.username)}', 'admin')" title="Promover a Administrador">Tornar Admin</button>`;
        }
      }

      // Formatação de data
      let dateFormatted = '--';
      const dateToFormat = u.last_login || u.created_at;
      if (dateToFormat) {
        try {
          const d = new Date(dateToFormat);
          if (!isNaN(d.getTime())) {
            dateFormatted = d.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' });
          }
        } catch {}
      }

      let actionHtml = '';
      if (u.is_system) {
        actionHtml = `<span class="system-tag">Conta do Sistema</span>`;
      } else if (isCur) {
        actionHtml = `<span class="system-tag">Sua Conta</span>`;
      } else {
        actionHtml = `<button class="btn-delete-user" onclick="handleDeleteUser('${escapeHtml(u.username)}')">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
          <span>Excluir</span>
        </button>`;
      }

      const displayName = u.full_name
        ? `<strong>${escapeHtml(u.username)}</strong><div style="font-size:0.72rem;color:var(--text-muted);">${escapeHtml(u.full_name)}</div>`
        : `<strong>${escapeHtml(u.username)}</strong>`;

      return `<tr>
        <td>${displayName}</td>
        <td>${originBadge}</td>
        <td>
          <div style="display:flex;align-items:center;flex-wrap:wrap;gap:4px;">
            ${roleBadge}
            ${roleActionBtn}
          </div>
        </td>
        <td style="color: var(--text-muted); font-size: 0.78rem;">${dateFormatted}</td>
        <td style="text-align: right;">${actionHtml}</td>
      </tr>`;
    }).join('');
  } catch (err) {
    usersTableBody.innerHTML = `<tr><td colspan="5" class="text-center py-4" style="color: var(--accent-rose);">${escapeHtml(err.message || 'Erro ao carregar usuários')}</td></tr>`;
  }
}

async function handleCreateUserSubmit(e) {
  e.preventDefault();
  if (!newUsernameInput) return;

  const username = newUsernameInput.value.trim();
  const password = newUserPasswordInput ? newUserPasswordInput.value.trim() : '';
  const authType = newUserAuthTypeInput ? newUserAuthTypeInput.value : 'local';
  const isAdmin = newUserIsAdminSwitch ? newUserIsAdminSwitch.checked : false;

  if (!username) return;
  if (authType === 'local' && !password) return;

  if (createUserErrorAlert) createUserErrorAlert.style.display = 'none';
  if (btnSubmitCreateUser) btnSubmitCreateUser.disabled = true;

  try {
    const res = await apiFetch('/api/v1/users', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username,
        password,
        auth_type: authType,
        is_admin: isAdmin
      })
    });

    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || 'Erro ao criar usuário');
    }

    const typeMsg = authType === 'keycloak' ? 'institucional Keycloak' : 'local';
    showToast(`Usuário ${typeMsg} '${data.user?.username || username}' cadastrado com sucesso!`, 'toast-success');
    createUserForm.reset();
    setAccountType('local');
    if (newUserIsAdminSwitch) newUserIsAdminSwitch.checked = false;
    await loadUsersList();
  } catch (err) {
    if (createUserErrorText) createUserErrorText.textContent = err.message || 'Erro ao cadastrar usuário';
    if (createUserErrorAlert) createUserErrorAlert.style.display = 'flex';
  } finally {
    if (btnSubmitCreateUser) btnSubmitCreateUser.disabled = false;
  }
}

async function handleToggleRole(username, newRole) {
  if (!username || !newRole) return;
  const actionLabel = newRole === 'admin' ? 'promover a Administrador' : 'rebaixar para Operador';
  if (!confirm(`Deseja realmente ${actionLabel} o usuário '${username}'?`)) {
    return;
  }

  try {
    const res = await apiFetch('/api/v1/users', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, role: newRole })
    });
    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || 'Falha ao alterar perfil');
    }
    showToast(data.message || `Perfil de '${username}' atualizado para ${newRole}`, 'toast-success');
    await loadUsersList();
  } catch (err) {
    showToast(err.message || 'Erro ao alterar permissão', 'toast-error');
  }
}
window.handleToggleRole = handleToggleRole;

async function handleDeleteUser(username) {
  if (!username) return;
  if (!confirm(`Deseja realmente remover o usuário '${username}' do sistema?`)) {
    return;
  }

  try {
    const res = await apiFetch(`/api/v1/users?username=${encodeURIComponent(username)}`, {
      method: 'DELETE'
    });
    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || 'Erro ao excluir usuário');
    }

    showToast(data.message || `Usuário '${username}' excluído com sucesso`, 'toast-info');
    await loadUsersList();
  } catch (err) {
    showToast(err.message || 'Falha ao excluir usuário', 'toast-error');
  }
}
window.handleDeleteUser = handleDeleteUser;

/* ==========================================================================
   Gerenciamento de Grupos de Testes & VLANs (Admin)
   ========================================================================== */

async function openGroupsModal() {
  switchView('groups');
}

function closeGroupsModal() {
  switchView('dashboard');
}

function resetGroupForm() {
  if (groupForm) groupForm.reset();
  if (groupIdInput) groupIdInput.value = '';
  if (groupFormTitle) groupFormTitle.textContent = 'Cadastrar Novo Grupo de Testes';
  if (btnSubmitGroupText) btnSubmitGroupText.textContent = 'Salvar Grupo';
  if (btnCancelEditGroup) btnCancelEditGroup.style.display = 'none';
  if (groupErrorAlert) groupErrorAlert.style.display = 'none';
  if (groupIntervalInput) groupIntervalInput.value = '60';

  // Reseta defaults dos módulos
  if (modPingEnabled) modPingEnabled.checked = true;
  if (modPingTarget) modPingTarget.value = '1.1.1.1';
  if (modPingCount) modPingCount.value = '4';

  if (modPacketlossEnabled) modPacketlossEnabled.checked = true;
  if (modPacketlossTarget) modPacketlossTarget.value = '1.1.1.1';
  if (modPacketlossCount) modPacketlossCount.value = '10';

  if (modDownloadEnabled) modDownloadEnabled.checked = true;
  if (modDownloadBytes) modDownloadBytes.value = '10485760';

  if (modUploadEnabled) modUploadEnabled.checked = true;
  if (modUploadBytes) modUploadBytes.value = '5242880';

  if (modDnsEnabled) modDnsEnabled.checked = true;
  if (modDnsDomains) modDnsDomains.value = 'google.com, cloudflare.com';

  if (modHttpEnabled) modHttpEnabled.checked = true;
  if (modHttpUrl) modHttpUrl.value = 'https://www.google.com';
  if (modHttpStatus) modHttpStatus.value = '200';

  if (modTraceEnabled) modTraceEnabled.checked = false;
  if (modTraceTarget) modTraceTarget.value = '1.1.1.1';
  if (modTraceHops) modTraceHops.value = '15';

  if (modLldpEnabled) modLldpEnabled.checked = true;
}

// ==========================================================================
// 2. Gerenciamento de Grupos & VLANs (Visão Limpa, Inspeção e Edição Completa)
// ==========================================================================

let currentGroups = [];
let currentViewingGroupId = null;

async function loadGroupsList() {
  if (!groupsTableBody) return;
  groupsTableBody.innerHTML = `<tr><td colspan="6" class="text-center py-4" style="color: var(--text-muted);"><div class="spinner" style="width:20px;height:20px;margin:0 auto 8px;"></div>Carregando grupos e associações...</td></tr>`;

  try {
    const [grpRes, machRes, modRes] = await Promise.all([
      apiFetch('/api/v1/admin/groups'),
      allAdminMachines.length > 0 ? Promise.resolve(null) : apiFetch('/api/v1/admin/machines').catch(() => null),
      allCatalogModules.length > 0 ? Promise.resolve(null) : apiFetch('/api/v1/admin/modules').catch(() => null)
    ]);

    if (grpRes.ok) {
      currentGroups = await grpRes.json() || [];
    } else {
      const fallbackRes = await apiFetch('/api/v1/groups');
      currentGroups = fallbackRes.ok ? await fallbackRes.json() : [];
    }

    if (machRes && machRes.ok) {
      allAdminMachines = await machRes.json() || [];
    }
    if (modRes && modRes.ok) {
      allCatalogModules = await modRes.json() || [];
    }

    renderGroupsList();
  } catch (err) {
    if (groupsTableBody) {
      groupsTableBody.innerHTML = `<tr><td colspan="6" class="text-center py-4 text-rose">Erro ao carregar grupos: ${escapeHtml(err.message)}</td></tr>`;
    }
  }
}
window.loadGroupsList = loadGroupsList;

function renderGroupsList() {
  if (!groupsTableBody) return;

  const query = groupsSearchInput ? groupsSearchInput.value.toLowerCase().trim() : '';
  const filtered = currentGroups.filter(g => {
    if (!query) return true;
    const nameMatch = (g.name || '').toLowerCase().includes(query);
    const idMatch = (g.id || '').toLowerCase().includes(query);
    const descMatch = (g.description || '').toLowerCase().includes(query);
    return nameMatch || idMatch || descMatch;
  });

  if (groupsCountTag) {
    groupsCountTag.textContent = `${filtered.length} grupo${filtered.length !== 1 ? 's' : ''}`;
  }

  if (filtered.length === 0) {
    groupsTableBody.innerHTML = `<tr><td colspan="6" class="text-center py-4" style="color: var(--text-muted);">${query ? 'Nenhum grupo encontrado para a busca.' : 'Nenhum grupo cadastrado.'}</td></tr>`;
    return;
  }

  groupsTableBody.innerHTML = filtered.map(g => {
    const isDefault = g.id === 'default' || !!g.is_default;
    const vlans = (g.rules && g.rules.vlans) ? g.rules.vlans : (g.vlans || []);
    const vlanTags = vlans.length > 0
      ? vlans.map(v => `<span class="vlan-pill">VLAN ${v}</span>`).join('')
      : '<span class="text-subtle font-mono text-xs">Padrão / Todas</span>';

    // Máquinas vinculadas a este grupo
    const machinesInGroup = (allAdminMachines || []).filter(m => m && m.group_id === g.id);

    // Módulos configurados
    let activeTestsCount = 0;
    if (g.profile && Array.isArray(g.profile.tests)) {
      activeTestsCount = g.profile.tests.filter(t => t.enabled !== false).length;
    } else if (g.modules) {
      activeTestsCount = Object.values(g.modules).filter(m => m && m.enabled).length;
    }

    const intervalMinutes = (g.profile && g.profile.interval_minutes) || g.interval_minutes || 30;

    return `
      <tr>
        <td>
          <div style="display: flex; align-items: center; gap: 6px;">
            <strong class="cursor-pointer" style="color: #f8fafc;" onclick="openGroupDetailsModal('${escapeHtml(g.id)}', 'machines')">${escapeHtml(g.name || g.id)}</strong>
            ${isDefault ? '<span class="default-group-badge">Padrão</span>' : ''}
          </div>
          <div class="text-subtle font-mono text-xs mt-1">ID: ${escapeHtml(g.id)}</div>
          ${g.description ? `<div style="font-size: 0.74rem; color: var(--text-muted); margin-top: 2px;">${escapeHtml(g.description)}</div>` : ''}
        </td>
        <td>${vlanTags}</td>
        <td>
          <span class="badge-counter-clickable badge-machines" title="Ver máquinas deste grupo" onclick="openGroupDetailsModal('${escapeHtml(g.id)}', 'machines')">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect><line x1="8" y1="21" x2="16" y2="21"></line><line x1="12" y1="17" x2="12" y2="21"></line></svg>
            ${machinesInGroup.length} Máquina${machinesInGroup.length !== 1 ? 's' : ''}
          </span>
        </td>
        <td>
          <span class="badge-counter-clickable badge-modules" title="Ver módulos ativos do grupo" onclick="openGroupDetailsModal('${escapeHtml(g.id)}', 'modules')">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path></svg>
            ${activeTestsCount} Módulo${activeTestsCount !== 1 ? 's' : ''}
          </span>
        </td>
        <td><span class="font-mono text-xs">${intervalMinutes}m</span></td>
        <td style="text-align: right; white-space: nowrap;">
          <button type="button" class="btn btn-glass btn-sm" onclick="openGroupDetailsModal('${escapeHtml(g.id)}', 'machines')" title="Inspecionar Máquinas">
            Inspecionar
          </button>
          <button type="button" class="btn btn-glass btn-sm" onclick="openGroupDetailsModal('${escapeHtml(g.id)}', 'edit')" title="Editar Grupo">
            Editar
          </button>
          ${!isDefault ? `
          <button type="button" class="btn btn-outline-danger btn-sm" onclick="handleDeleteGroup('${escapeHtml(g.id)}', '${escapeHtml(g.name || g.id)}')" title="Excluir Grupo">
            Excluir
          </button>` : ''}
        </td>
      </tr>
    `;
  }).join('');
}

function openCreateGroupModal() {
  if (!groupCreateModal) return;
  if (createGroupErrorAlert) createGroupErrorAlert.style.display = 'none';
  if (createGroupForm) createGroupForm.reset();
  if (createGroupIdInput) createGroupIdInput.value = '';
  if (createGroupIntervalInput) createGroupIntervalInput.value = '30';

  renderCreateGroupModulesCheckboxes();
  groupCreateModal.style.display = 'flex';
}
window.openCreateGroupModal = openCreateGroupModal;

function closeCreateGroupModal() {
  if (groupCreateModal) groupCreateModal.style.display = 'none';
}
window.closeCreateGroupModal = closeCreateGroupModal;

function renderCreateGroupModulesCheckboxes() {
  if (!createGroupModulesList) return;
  const allModules = getAllAvailableModules();
  if (allModules.length === 0) {
    createGroupModulesList.innerHTML = '<div class="text-subtle text-xs p-2">Nenhum módulo no catálogo.</div>';
    return;
  }

  createGroupModulesList.innerHTML = allModules.map(mod => {
    const checked = mod.id.includes('ping') || mod.id.includes('discovery');
    return `
      <label class="mc-module-check-item">
        <div class="mc-module-check-left">
          <input type="checkbox" class="cg-module-cb" value="${escapeHtml(mod.id)}" ${checked ? 'checked' : ''} />
          <div>
            <div class="mc-module-title">${escapeHtml(mod.name || mod.id)}</div>
            <div class="mc-module-sub">${escapeHtml(mod.id)} • ${escapeHtml(mod.type)}</div>
          </div>
        </div>
      </label>
    `;
  }).join('');

  updateCreateGroupModulesCount();
  createGroupModulesList.querySelectorAll('.cg-module-cb').forEach(cb => {
    cb.addEventListener('change', updateCreateGroupModulesCount);
  });
}

function updateCreateGroupModulesCount() {
  if (!createGroupModulesCount || !createGroupModulesList) return;
  const count = createGroupModulesList.querySelectorAll('.cg-module-cb:checked').length;
  createGroupModulesCount.textContent = `${count} selecionado${count !== 1 ? 's' : ''}`;
}

async function handleCreateGroupSubmit(e) {
  e.preventDefault();
  if (createGroupErrorAlert) createGroupErrorAlert.style.display = 'none';

  const name = createGroupNameInput.value.trim();
  let id = createGroupIdInput.value.trim().toLowerCase().replace(/[^a-z0-9_-]/g, '_');
  if (!id) id = name.toLowerCase().replace(/[^a-z0-9_-]/g, '_');

  if (!name || !id) {
    if (createGroupErrorAlert) {
      createGroupErrorAlert.style.display = 'flex';
      createGroupErrorText.textContent = 'Informe o nome e o ID do grupo.';
    }
    return;
  }

  const rawVlans = createGroupVlansInput.value.split(',')
    .map(v => parseInt(v.trim(), 10))
    .filter(v => !isNaN(v) && v > 0);

  const rawSubnets = createGroupSubnetsInput.value.split(',')
    .map(s => s.trim())
    .filter(s => s.length > 0);

  const interval = parseInt(createGroupIntervalInput.value, 10) || 30;
  const description = createGroupDescriptionInput.value.trim();

  // Módulos selecionados
  const allModules = getAllAvailableModules();
  const selectedModuleIds = new Set(
    Array.from(createGroupModulesList.querySelectorAll('.cg-module-cb:checked')).map(cb => cb.value)
  );
  const selectedTests = allModules.filter(m => selectedModuleIds.has(m.id));

  const payload = {
    id: id,
    name: name,
    description: description,
    rules: {
      vlans: rawVlans,
      ip_subnets: rawSubnets
    },
    profile: {
      version: new Date().toISOString().replace(/[-:T.Z]/g, '').slice(0, 14),
      updated_at: new Date().toISOString(),
      interval_minutes: interval,
      sync_schedule: "fixed_half_hour",
      description: `Perfil do grupo ${name}`,
      tests: selectedTests
    }
  };

  try {
    const res = await apiFetch('/api/v1/admin/groups', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao criar grupo');

    showToast(`Grupo "${name}" cadastrado com sucesso!`, 'toast-success');
    closeCreateGroupModal();
    await loadGroupsList();
  } catch (err) {
    if (createGroupErrorAlert) {
      createGroupErrorAlert.style.display = 'flex';
      createGroupErrorText.textContent = err.message || 'Erro ao criar grupo';
    }
  }
}

function isNodeOnline(lastSeen, hostname) {
  if (hostname && typeof currentSummaries !== 'undefined' && Array.isArray(currentSummaries)) {
    const summary = currentSummaries.find(s => s && s.node_id === hostname);
    if (summary && typeof summary.is_healthy === 'boolean') {
      return summary.is_healthy;
    }
  }
  if (!lastSeen) return false;
  const d = new Date(lastSeen);
  if (isNaN(d.getTime())) return false;
  return (Date.now() - d.getTime()) < 15 * 60 * 1000;
}
window.isNodeOnline = isNodeOnline;

function openGroupDetailsModal(groupId, tab = 'machines') {
  try {
    const g = (currentGroups || []).find(item => item && item.id === groupId);
    if (!g || !groupDetailsModal) return;

    currentViewingGroupId = groupId;

    if (gdModalTitle) gdModalTitle.textContent = g.name || g.id;
    if (gdModalBadge) gdModalBadge.textContent = g.id + (g.id === 'default' ? ' (Padrão)' : '');

    // 1. Popula Aba Máquinas
    renderGroupMachinesTab(g);

    // 2. Popula Aba Módulos
    renderGroupModulesTab(g);

    // 3. Popula Aba Edição
    populateGroupEditForm(g);

    // Ativa a aba inicial
    switchGroupDetailsTab(tab);

    groupDetailsModal.style.display = 'flex';
  } catch (err) {
    console.error('Erro ao abrir detalhes do grupo:', err);
    if (typeof showToast === 'function') {
      showToast(`Erro ao abrir grupo: ${err.message}`, 'toast-error');
    }
  }
}
window.openGroupDetailsModal = openGroupDetailsModal;

function closeGroupDetailsModal() {
  if (groupDetailsModal) groupDetailsModal.style.display = 'none';
  currentViewingGroupId = null;
}
window.closeGroupDetailsModal = closeGroupDetailsModal;

function switchGroupDetailsTab(tabName) {
  const tabs = {
    machines: { btn: document.getElementById('tab-gd-machines'), pane: document.getElementById('gd-pane-machines') },
    modules: { btn: document.getElementById('tab-gd-modules'), pane: document.getElementById('gd-pane-modules') },
    edit: { btn: document.getElementById('tab-gd-edit'), pane: document.getElementById('gd-pane-edit') }
  };

  Object.keys(tabs).forEach(k => {
    if (tabs[k].btn) tabs[k].btn.classList.toggle('active', k === tabName);
    if (tabs[k].pane) tabs[k].pane.style.display = (k === tabName) ? 'block' : 'none';
  });
}
window.switchGroupDetailsTab = switchGroupDetailsTab;

function renderGroupMachinesTab(g) {
  if (!gdMachinesTableBody) return;
  const machines = (allAdminMachines || []).filter(m => m && m.group_id === g.id);

  if (gdCountMachines) gdCountMachines.textContent = machines.length;

  if (machines.length === 0) {
    gdMachinesTableBody.innerHTML = `
      <tr>
        <td colspan="7" class="text-center py-4 text-muted">
          Nenhuma máquina vinculada ao grupo <strong>${escapeHtml(g.name || g.id)}</strong> no momento.<br/>
          <span class="text-xs">Sondas detectadas nas VLANs deste grupo ou configuradas manualmente na aba Máquinas aparecerão aqui.</span>
        </td>
      </tr>
    `;
    return;
  }

  gdMachinesTableBody.innerHTML = machines.map(m => {
    const isOnline = isNodeOnline(m.last_seen, m.hostname);
    const statusBadge = isOnline
      ? '<span class="status-badge-inline online"><span class="status-dot-sm online"></span> Online</span>'
      : '<span class="status-badge-inline offline"><span class="status-dot-sm offline"></span> Offline</span>';

    const customBadge = m.is_custom
      ? '<span class="mc-module-tag-state state-added">Customizado</span>'
      : '<span class="mc-module-tag-state state-default">Padrão do Grupo</span>';

    return `
      <tr>
        <td>
          <strong class="text-cyan font-mono">${escapeHtml(m.hostname)}</strong>
        </td>
        <td><span class="font-mono text-xs">${escapeHtml(m.ip_address || '--')}</span></td>
        <td>
          <div style="font-weight: 500;">${escapeHtml(m.switch_name || 'Desconhecido')}</div>
          <div class="text-subtle font-mono text-xs">Porta: ${escapeHtml(m.switch_port || '--')}</div>
        </td>
        <td><span class="vlan-pill">${m.vlan ? `VLAN ${m.vlan}` : '--'}</span></td>
        <td>${statusBadge}</td>
        <td>${customBadge}</td>
        <td style="text-align: right;">
          <button type="button" class="btn btn-glass btn-sm" onclick="closeGroupDetailsModal(); openMachineConfigModal('${escapeHtml(m.hostname)}');" title="Configurar Módulos da Máquina">
            Configurar
          </button>
        </td>
      </tr>
    `;
  }).join('');
}

function renderGroupModulesTab(g) {
  if (!gdModulesCardsGrid) return;

  const tests = (g.profile && Array.isArray(g.profile.tests)) ? g.profile.tests : [];
  if (gdCountModules) gdCountModules.textContent = tests.length;

  if (tests.length === 0) {
    gdModulesCardsGrid.innerHTML = `
      <div class="text-subtle text-center py-4 col-span-full">
        Nenhum módulo de teste configurado para este grupo.<br/>
        <button type="button" class="btn btn-glass btn-sm mt-2" onclick="switchGroupDetailsTab('edit')">Configurar Módulos</button>
      </div>
    `;
    return;
  }

  gdModulesCardsGrid.innerHTML = tests.map(t => {
    let target = t.target || t.domain || t.endpoint || t.protocol || '--';
    let extra = '';
    if (t.size_mb) extra = `${t.size_mb} MB`;
    else if (t.count) extra = `${t.count} pacotes`;

    return `
      <div class="module-card">
        <div class="module-card-top">
          <div class="module-card-title">${escapeHtml(t.name || t.id)}</div>
          <span class="module-type-badge ${escapeHtml(t.type || '')}">${escapeHtml(t.type || 'teste')}</span>
        </div>
        <div class="module-card-id">${escapeHtml(t.id)}</div>
        <div class="module-card-details">
          <div class="module-card-details-row">
            <span>Alvo:</span>
            <span class="font-semibold text-white font-mono text-xs">${escapeHtml(target)}</span>
          </div>
          ${extra ? `
          <div class="module-card-details-row">
            <span>Parâmetro:</span>
            <span class="font-mono">${escapeHtml(extra)}</span>
          </div>` : ''}
          <div class="module-card-details-row">
            <span>Timeout:</span>
            <span class="font-mono">${t.timeout_seconds || 15}s</span>
          </div>
        </div>
      </div>
    `;
  }).join('');
}

function populateGroupEditForm(g) {
  if (egErrorAlert) egErrorAlert.style.display = 'none';
  if (egGroupId) egGroupId.value = g.id;
  if (egGroupName) egGroupName.value = g.name || '';
  if (egGroupDescription) egGroupDescription.value = g.description || '';

  const vlans = (g.rules && g.rules.vlans) ? g.rules.vlans : (g.vlans || []);
  if (egGroupVlans) egGroupVlans.value = vlans.join(', ');

  const subnets = (g.rules && g.rules.ip_subnets) ? g.rules.ip_subnets : [];
  if (egGroupSubnets) egGroupSubnets.value = subnets.join(', ');

  const interval = (g.profile && g.profile.interval_minutes) || g.interval_minutes || 30;
  if (egGroupInterval) egGroupInterval.value = interval;

  // Botão excluir oculto para o grupo padrão
  if (btnEgDeleteGroup) {
    btnEgDeleteGroup.style.display = (g.id === 'default') ? 'none' : 'inline-flex';
  }

  // Renderiza checklist de módulos do catálogo
  if (egModulesCheckboxesList) {
    const allModules = getAllAvailableModules();
    const existingTestIds = new Set(
      (g.profile && Array.isArray(g.profile.tests)) ? g.profile.tests.map(t => t.id) : []
    );

    egModulesCheckboxesList.innerHTML = allModules.map(mod => {
      const isChecked = existingTestIds.has(mod.id);
      return `
        <label class="mc-module-check-item">
          <div class="mc-module-check-left">
            <input type="checkbox" class="eg-module-cb" value="${escapeHtml(mod.id)}" ${isChecked ? 'checked' : ''} />
            <div>
              <div class="mc-module-title">${escapeHtml(mod.name || mod.id)}</div>
              <div class="mc-module-sub">${escapeHtml(mod.id)} • ${escapeHtml(mod.type)}</div>
            </div>
          </div>
          <span class="mc-module-tag-state ${isChecked ? 'state-default' : 'state-removed'}">
            ${isChecked ? 'Ativo' : 'Desativado'}
          </span>
        </label>
      `;
    }).join('');

    updateEditGroupModulesCount();
    egModulesCheckboxesList.querySelectorAll('.eg-module-cb').forEach(cb => {
      cb.addEventListener('change', () => {
        const item = cb.closest('.mc-module-check-item');
        const tag = item.querySelector('.mc-module-tag-state');
        if (tag) {
          tag.className = `mc-module-tag-state ${cb.checked ? 'state-default' : 'state-removed'}`;
          tag.textContent = cb.checked ? 'Ativo' : 'Desativado';
        }
        updateEditGroupModulesCount();
      });
    });
  }
}

function updateEditGroupModulesCount() {
  if (!egModulesCountTag || !egModulesCheckboxesList) return;
  const count = egModulesCheckboxesList.querySelectorAll('.eg-module-cb:checked').length;
  egModulesCountTag.textContent = `${count} ativo${count !== 1 ? 's' : ''}`;
}

async function handleSaveGroupEditSubmit(e) {
  e.preventDefault();
  if (egErrorAlert) egErrorAlert.style.display = 'none';

  const groupId = egGroupId.value.trim();
  const name = egGroupName.value.trim();
  if (!groupId || !name) {
    if (egErrorAlert) {
      egErrorAlert.style.display = 'flex';
      egErrorText.textContent = 'Nome do grupo é obrigatório.';
    }
    return;
  }

  const rawVlans = egGroupVlans.value.split(',')
    .map(v => parseInt(v.trim(), 10))
    .filter(v => !isNaN(v) && v > 0);

  const rawSubnets = egGroupSubnets.value.split(',')
    .map(s => s.trim())
    .filter(s => s.length > 0);

  const interval = parseInt(egGroupInterval.value, 10) || 30;
  const description = egGroupDescription.value.trim();

  // Módulos selecionados
  const allModules = getAllAvailableModules();
  const selectedModuleIds = new Set(
    Array.from(egModulesCheckboxesList.querySelectorAll('.eg-module-cb:checked')).map(cb => cb.value)
  );
  const selectedTests = allModules.filter(m => selectedModuleIds.has(m.id));

  const existing = currentGroups.find(g => g.id === groupId) || {};

  const payload = {
    id: groupId,
    name: name,
    description: description,
    rules: {
      vlans: rawVlans,
      ip_subnets: rawSubnets
    },
    profile: {
      version: new Date().toISOString().replace(/[-:T.Z]/g, '').slice(0, 14),
      updated_at: new Date().toISOString(),
      interval_minutes: interval,
      sync_schedule: (existing.profile && existing.profile.sync_schedule) || "fixed_half_hour",
      description: `Perfil do grupo ${name}`,
      tests: selectedTests
    }
  };

  try {
    const res = await apiFetch('/api/v1/admin/groups', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao atualizar grupo');

    showToast(`Grupo "${name}" atualizado com sucesso!`, 'toast-success');
    closeGroupDetailsModal();
    await loadGroupsList();
  } catch (err) {
    if (egErrorAlert) {
      egErrorAlert.style.display = 'flex';
      egErrorText.textContent = err.message || 'Erro ao atualizar grupo';
    }
  }
}

async function handleDeleteGroup(groupId, groupName) {
  if (groupId === 'default') {
    alert('O grupo padrão (default) não pode ser excluído.');
    return;
  }
  if (!confirm(`Deseja realmente excluir o grupo "${groupName || groupId}"?\nAs máquinas deste grupo serão reatribuídas para o grupo padrão.`)) return;

  try {
    const res = await apiFetch(`/api/v1/admin/groups/${encodeURIComponent(groupId)}`, {
      method: 'DELETE'
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao excluir grupo');

    showToast(`Grupo "${groupName || groupId}" excluído com sucesso!`, 'toast-info');
    closeGroupDetailsModal();
    await loadGroupsList();
  } catch (err) {
    showToast(`Erro ao excluir: ${err.message}`, 'toast-error');
  }
}
window.handleDeleteGroup = handleDeleteGroup;


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

async function loadDashboardData(requestedDate) {
  if (!currentUser) return;

  const selected = selectedChartNode || (selectNode && selectNode.value) || 'all';
  let url = '/api/v1/dashboard/metrics';
  const params = [];
  if (selected !== 'all') {
    params.push(`node_id=${encodeURIComponent(selected)}`);
  }
  const dateParam = (typeof requestedDate === 'string' ? requestedDate : telemetryPageState.date).trim();
  if (dateParam) {
    params.push(`date=${encodeURIComponent(dateParam)}`);
  }
  if (params.length > 0) {
    url += '?' + params.join('&');
  }

  try {
    const res = await apiFetch(url);
    if (res.status === 401) {
      showLoginModal();
      return;
    }
    if (!res.ok) throw new Error(`HTTP error ${res.status}`);

    const data = await res.json();
    currentSummaries = data.summaries || [];

    if (currentGroups.length === 0) {
      apiFetch('/api/v1/admin/groups')
        .then(r => r.ok ? r.json() : apiFetch('/api/v1/groups').then(r2 => r2.ok ? r2.json() : []))
        .then(grps => {
          if (grps && grps.length > 0) {
            currentGroups = grps;
            updateChartGroupSelector();
          }
        }).catch(() => {});
    }
    if (allAdminMachines.length === 0) {
      apiFetch('/api/v1/admin/machines')
        .then(r => r.ok ? r.json() : [])
        .then(machs => {
          if (machs && machs.length > 0) {
            allAdminMachines = machs;
            updateChartGroupSelector();
          }
        }).catch(() => {});
    }
    updateChartGroupSelector();

    updateNodeSelector(data.nodes_list || []);
    renderSummaryCards(currentSummaries);
    renderComparativeCharts(data.metrics_series || [], data.nodes_list || []);
    renderTelemetryTable(data.metrics_series || [], data.node_diagnostic);
    updateChartFocusUI();
    renderSingleMachineHeaderAndAlert(data.node_diagnostic, currentSummaries);
    renderSingleMachineDetailsPanel((currentSummaries || []).find(s => s.node_id === selectedChartNode), data.node_diagnostic);

    activeNodesCount.textContent = data.total_nodes || 0;
    if (data.generated_at) {
      lastIngestTime.textContent = `Atualizado: ${new Date(data.generated_at).toLocaleTimeString()}`;
    }
  } catch (err) {
    console.error('Falha ao carregar métricas:', err);
    if (nodesTableBody) {
      nodesTableBody.innerHTML = `
        <tr>
          <td colspan="10" class="nodes-empty-state">
            <p class="text-rose font-bold">Falha ao conectar com o servidor central.</p>
            <p class="text-subtle">${escapeHtml(err.message)}</p>
          </td>
        </tr>
      `;
    }
  }
}

function updateNodeSelector(nodes) {
  if (!selectNode) return;
  const currentVal = selectNode.value;
  selectNode.innerHTML = '<option value="all">Todos os Nós (Consolidado)</option>';

  nodes.forEach(node => {
    const opt = document.createElement('option');
    opt.value = node;
    opt.textContent = node;
    selectNode.appendChild(opt);
  });

  selectNode.value = nodes.includes(currentVal) ? currentVal : 'all';
  updateMetricToggleVisibility();
}

function selectNodeForCharts(nodeId, forceFocus = false) {
  if (!forceFocus && selectedChartNode === nodeId) {
    // Se clicar na mesma máquina já selecionada (e não for foco forçado), desseleciona e volta para consolidado
    selectedChartNode = 'all';
    showToast('Exibindo gráficos consolidados de todas as sondas', 'toast-info');
    const smHeader = document.getElementById('single-machine-header');
    if (smHeader) smHeader.style.display = 'none';
  } else {
    selectedChartNode = nodeId;
    showToast(`Gráficos focados na sonda: ${nodeId}`, 'toast-info');

    // Exibe imediatamente o cabeçalho preliminar para feedback visual instantâneo
    const smHeader = document.getElementById('single-machine-header');
    if (smHeader) {
      smHeader.style.display = 'block';
      const nameEl = document.getElementById('sm-machine-name');
      if (nameEl) nameEl.textContent = nodeId;
      const summary = currentSummaries.find(s => s.node_id === nodeId);
      if (summary) {
        const ipEl = document.getElementById('sm-ip');
        if (ipEl && summary.system_info) {
          ipEl.textContent = summary.system_info.ip_address || summary.system_info.IPAddress || '--';
        }
      }
    }

    // Rola suavemente para o topo do dashboard para exibir o cabeçalho e diagnóstico
    setTimeout(() => {
      const topTarget = document.getElementById('single-machine-header') || document.getElementById('view-dashboard');
      if (topTarget) {
        topTarget.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    }, 50);
  }

  updateChartFocusUI();
  updateMetricToggleVisibility();
  loadDashboardData();
}
window.selectNodeForCharts = selectNodeForCharts;

function getNodesInGroup(groupId) {
  const set = new Set();
  if (!groupId || groupId === 'all') return set;

  const grp = (currentGroups || []).find(g => g.id === groupId || g.name === groupId);
  const targetId = grp ? grp.id : groupId;
  const targetName = grp ? (grp.name || '').toLowerCase() : groupId.toLowerCase();

  (allAdminMachines || []).forEach(m => {
    if (!m) return;
    if (m.group_id === targetId || (m.group_id && m.group_id.toLowerCase() === targetName)) {
      set.add((m.hostname || '').toLowerCase());
    }
  });

  (currentSummaries || []).forEach(s => {
    if (!s) return;
    const sGrpName = (s.group_name || s.GroupName || '').toLowerCase();
    if (sGrpName === targetName || sGrpName === targetId.toLowerCase()) {
      set.add((s.node_id || s.NodeID || '').toLowerCase());
    }
  });

  return set;
}

function updateChartGroupSelector() {
  const selectEl = document.getElementById('chart-group-filter-select');
  if (!selectEl) return;

  const groupsMap = new Map();
  (currentGroups || []).forEach(g => {
    if (g && g.id) groupsMap.set(g.id, g.name || g.id);
  });
  (allAdminMachines || []).forEach(m => {
    if (m && m.group_id && !groupsMap.has(m.group_id)) {
      groupsMap.set(m.group_id, m.group_id);
    }
  });
  (currentSummaries || []).forEach(s => {
    const grpName = s.group_name || s.GroupName;
    if (grpName && !groupsMap.has(grpName)) {
      groupsMap.set(grpName, grpName);
    }
  });

  const prevVal = selectedChartGroup || selectEl.value || 'all';
  let html = `<option value="all">Todos os Grupos</option>`;
  for (const [gid, gname] of groupsMap.entries()) {
    html += `<option value="${escapeHtml(gid)}" ${gid === prevVal ? 'selected' : ''}>${escapeHtml(gname)}</option>`;
  }
  selectEl.innerHTML = html;
  selectEl.value = groupsMap.has(prevVal) ? prevVal : 'all';
}

function handleChartGroupChange(newGroupId) {
  selectedChartGroup = newGroupId || 'all';
  // Se estiver focado em uma sonda individual, desmarca para analisar o grupo
  if (selectedChartNode !== 'all') {
    selectedChartNode = 'all';
    if (selectNode) selectNode.value = 'all';
    const smHeader = document.getElementById('single-machine-header');
    const smAlert = document.getElementById('sm-offline-alert');
    if (smHeader) smHeader.style.display = 'none';
    if (smAlert) smAlert.style.display = 'none';
  }

  updateChartFocusUI();
  updateMetricToggleVisibility();
  renderComparativeCharts(cachedMetricsSeries, cachedNodesList);

  if (selectedChartGroup !== 'all') {
    const grp = (currentGroups || []).find(g => g.id === selectedChartGroup);
    const gName = grp ? grp.name : selectedChartGroup;
    showToast(`Filtrando gráficos pelo grupo: ${gName}`, 'toast-info');
  } else {
    showToast('Exibindo todas as máquinas no gráfico consolidado', 'toast-info');
  }
}
window.handleChartGroupChange = handleChartGroupChange;

function clearChartGroupFilter() {
  selectedChartGroup = 'all';
  const selectEl = document.getElementById('chart-group-filter-select');
  if (selectEl) selectEl.value = 'all';
  handleChartGroupChange('all');
}
window.clearChartGroupFilter = clearChartGroupFilter;

function updateChartFocusUI() {
  const nameEl = document.getElementById('chart-focus-name');
  const resetBtn = document.getElementById('btn-reset-chart-focus');
  const clearGroupBtn = document.getElementById('btn-clear-chart-group');
  const indicator = document.getElementById('chart-focus-indicator');
  const groupSelect = document.getElementById('chart-group-filter-select');
  const nodesSection = document.getElementById('nodes-table-section');
  const grpFilterWrap = document.querySelector('.chart-group-filter-wrap');
  const smDetailsPanel = document.getElementById('single-machine-details-panel');

  if (groupSelect && groupSelect.value !== selectedChartGroup) {
    groupSelect.value = selectedChartGroup;
  }

  if (clearGroupBtn) {
    clearGroupBtn.style.display = (selectedChartGroup !== 'all') ? 'inline-flex' : 'none';
  }

  if (selectedChartNode !== 'all') {
    if (nameEl) nameEl.textContent = selectedChartNode;
    if (indicator) indicator.className = 'chart-focus-pill focused';
    if (resetBtn) resetBtn.style.display = 'inline-flex';

    // Oculta a lista redundante e filtro de grupo quando monitorando apenas 1 maquina
    if (nodesSection) nodesSection.style.display = 'none';
    if (grpFilterWrap) grpFilterWrap.style.display = 'none';
    if (clearGroupBtn) clearGroupBtn.style.display = 'none';
    if (smDetailsPanel) smDetailsPanel.style.display = 'block';
  } else if (selectedChartGroup !== 'all') {
    const grp = (currentGroups || []).find(g => g.id === selectedChartGroup);
    const gName = grp ? grp.name : selectedChartGroup;
    const groupNodes = getNodesInGroup(selectedChartGroup);
    if (nameEl) nameEl.textContent = "Grupo: " + gName + " (" + groupNodes.size + " máquinas)";
    if (indicator) indicator.className = 'chart-focus-pill group-focused';
    if (resetBtn) resetBtn.style.display = 'none';

    if (nodesSection) nodesSection.style.display = 'block';
    if (grpFilterWrap) grpFilterWrap.style.display = 'flex';
    if (smDetailsPanel) smDetailsPanel.style.display = 'none';
  } else {
    if (nameEl) nameEl.textContent = 'Todas as Sondas (Consolidado)';
    if (indicator) indicator.className = 'chart-focus-pill all';
    if (resetBtn) resetBtn.style.display = 'none';

    if (nodesSection) nodesSection.style.display = 'block';
    if (grpFilterWrap) grpFilterWrap.style.display = 'flex';
    if (smDetailsPanel) smDetailsPanel.style.display = 'none';
  }

  // Atualiza destaque visual das linhas da tabela
  if (nodesTableBody) {
    const rows = nodesTableBody.querySelectorAll('.node-row');
    rows.forEach(row => {
      const rowNodeId = row.getAttribute('data-node-id');
      const isSel = (selectedChartNode !== 'all' && rowNodeId === selectedChartNode);
      row.classList.toggle('selected-chart-node', isSel);
    });
  }
}

function renderSingleMachineHeaderAndAlert(diagnostic, summaries) {
  const smHeader = document.getElementById('single-machine-header');
  const smAlert = document.getElementById('sm-offline-alert');
  const chartHistBadge = document.getElementById('chart-historical-badge');

  if (!smHeader || !smAlert) return;

  const isSingleNode = (selectedChartNode && selectedChartNode !== 'all');

  if (!isSingleNode) {
    smHeader.style.display = 'none';
    smAlert.style.display = 'none';
    if (chartHistBadge) chartHistBadge.style.display = 'none';
    return;
  }

  // 1. Preenche o Cabeçalho Superior de Máquina Única
  smHeader.style.display = 'block';

  const nameEl = document.getElementById('sm-machine-name');
  const statusBadge = document.getElementById('sm-status-badge');
  const ipEl = document.getElementById('sm-ip');
  const groupEl = document.getElementById('sm-group');
  const vlanEl = document.getElementById('sm-vlan');
  const switchEl = document.getElementById('sm-switch');
  const portEl = document.getElementById('sm-port');
  const histTag = document.getElementById('sm-historical-tag');

  const nodeName = (diagnostic && diagnostic.node_id) || selectedChartNode;
  if (nameEl) nameEl.textContent = nodeName;

  // Busca resumo correspondente em memória se houver
  const summary = (summaries || []).find(s => s.node_id === selectedChartNode);

  const isOnline = diagnostic ? !!diagnostic.is_online : (summary ? !!summary.is_healthy : false);
  if (statusBadge) {
    statusBadge.className = `dc-health-badge ${isOnline ? 'online' : 'stale'}`;
    statusBadge.textContent = isOnline ? 'Online na Rede' : 'Desconectada';
  }

  // IP
  let ipStr = '--';
  if (diagnostic && diagnostic.system_info && (diagnostic.system_info.ip_address || diagnostic.system_info.IPAddress)) {
    ipStr = diagnostic.system_info.ip_address || diagnostic.system_info.IPAddress;
  } else if (summary && summary.system_info && (summary.system_info.ip_address || summary.system_info.IPAddress)) {
    ipStr = summary.system_info.ip_address || summary.system_info.IPAddress;
  }
  if (ipEl) ipEl.textContent = ipStr;

  // Grupo
  const groupStr = (diagnostic && diagnostic.group_name) || (summary && summary.group_name) || 'Padrão / Desconhecido';
  if (groupEl) groupEl.textContent = groupStr;

  // VLAN
  let vlanVal = (diagnostic && diagnostic.vlan) || (summary && summary.vlan) || 0;
  if (!vlanVal && diagnostic && diagnostic.switch_info && diagnostic.switch_info.vlan) {
    vlanVal = diagnostic.switch_info.vlan;
  } else if (!vlanVal && summary && summary.switch_info && summary.switch_info.vlan) {
    vlanVal = summary.switch_info.vlan;
  }
  if (vlanEl) vlanEl.textContent = vlanVal > 0 ? `VLAN ${vlanVal}` : '--';

  // Switch & Porta (LLDP)
  let swName = '--';
  let swPort = '--';
  const swInfo = (diagnostic && diagnostic.switch_info) || (summary && summary.switch_info);
  if (swInfo && swInfo.switch_name) {
    swName = swInfo.switch_name;
    swPort = swInfo.switch_port || '--';
  }
  if (switchEl) switchEl.textContent = swName;
  if (portEl) portEl.textContent = swPort;

  // Metadados Wi-Fi e DNS no Cabeçalho de Máquina Única
  const smWifiMeta = document.getElementById('sm-wifi-meta');
  const smWifiVal = document.getElementById('sm-wifi-val');
  const smDnsMeta = document.getElementById('sm-dns-meta');
  const smDnsVal = document.getElementById('sm-dns-val');

  const modResults = (diagnostic && diagnostic.modular_results) || (summary && summary.modular_results);
  if (smWifiMeta && smWifiVal) {
    if (modResults && modResults.wifi && modResults.wifi.status && modResults.wifi.status !== 'skipped') {
      const wf = modResults.wifi;
      smWifiMeta.style.display = 'inline-flex';
      if (wf.status === 'disconnected') {
        smWifiVal.textContent = 'Desconectado';
        smWifiVal.className = 'sm-meta-val text-subtle';
      } else {
        const pct = wf.signal_quality_percent || 0;
        const ssid = wf.ssid || 'Wi-Fi';
        smWifiVal.textContent = `${ssid} (${pct}%)`;
        smWifiVal.className = 'sm-meta-val text-emerald font-bold';
      }
    } else {
      smWifiMeta.style.display = 'none';
    }
  }

  if (smDnsMeta && smDnsVal) {
    if (modResults && modResults.multi_dns && modResults.multi_dns.probes && modResults.multi_dns.probes.length > 0) {
      smDnsMeta.style.display = 'inline-flex';
      const instDnsIp = (window.APP_CONFIG && window.APP_CONFIG.dns_server_target) || '177.101.22.40';
      const uepgProbe = modResults.multi_dns.probes.find(p => p.server_ip && (p.server_ip === instDnsIp || p.server_ip.startsWith(instDnsIp)));
      if (uepgProbe && uepgProbe.status === 'success') {
        smDnsVal.textContent = `${uepgProbe.latency_ms.toFixed(1)} ms`;
      } else if (modResults.multi_dns.fastest_server) {
        const fastest = modResults.multi_dns.probes.find(p => p.server_ip === modResults.multi_dns.fastest_server);
        smDnsVal.textContent = fastest ? `${fastest.latency_ms.toFixed(1)} ms (${fastest.server_ip})` : '--';
      } else {
        smDnsVal.textContent = '--';
      }
    } else {
      smDnsMeta.style.display = 'none';
    }
  }

  // Modo Histórico
  const isHistorical = diagnostic ? !!diagnostic.is_historical_fallback : false;
  if (histTag) histTag.style.display = isHistorical ? 'inline-flex' : 'none';
  if (chartHistBadge) chartHistBadge.style.display = isHistorical ? 'inline-flex' : 'none';

  // 2. Painel de Alerta de Desconexão
  if (!isOnline) {
    smAlert.style.display = 'flex';

    const downtimeCounter = document.getElementById('sm-downtime-counter');
    const lastMetricTime = document.getElementById('sm-last-metric-time');
    const lastMetricAgo = document.getElementById('sm-last-metric-ago');
    const lostConnectionTime = document.getElementById('sm-lost-connection-time');
    const lostConnectionDetail = document.getElementById('sm-lost-connection-detail');
    const totalOfflineDuration = document.getElementById('sm-total-offline-duration');

    const dtDuration = (diagnostic && diagnostic.downtime_duration) || (summary && summary.last_seen ? formatTimeAgo(new Date(summary.last_seen)) : 'poucos instantes');
    if (downtimeCounter) downtimeCounter.textContent = `Offline há ${dtDuration}`;
    if (totalOfflineDuration) totalOfflineDuration.textContent = dtDuration;

    // Última Métrica Enviada
    const lastTs = (diagnostic && diagnostic.last_metric_timestamp) || (summary && summary.last_seen);
    if (lastTs) {
      const dLast = new Date(lastTs);
      if (lastMetricTime) {
        lastMetricTime.textContent = `${dLast.toLocaleDateString('pt-BR')} às ${dLast.toLocaleTimeString('pt-BR')}`;
      }
      if (lastMetricAgo) {
        lastMetricAgo.textContent = `(${formatTimeAgo(dLast)})`;
      }
    } else {
      if (lastMetricTime) lastMetricTime.textContent = 'Nenhuma medição registrada';
      if (lastMetricAgo) lastMetricAgo.textContent = '--';
    }

    // Horário de Perda de Conexão
    if (diagnostic && diagnostic.lost_connection_at) {
      const dLost = new Date(diagnostic.lost_connection_at);
      if (lostConnectionTime) {
        lostConnectionTime.textContent = `${dLost.toLocaleDateString('pt-BR')} às ${dLost.toLocaleTimeString('pt-BR')}`;
      }
      if (lostConnectionDetail) {
        const interval = diagnostic.expected_interval_minutes || 30;
        lostConnectionDetail.textContent = `Falha no ciclo agendado (+${interval} min)`;
      }
    } else if (lastTs) {
      // Fallback: Última medição + 30 minutos
      const dLostFallback = new Date(new Date(lastTs).getTime() + 30 * 60 * 1000);
      if (lostConnectionTime) {
        lostConnectionTime.textContent = `${dLostFallback.toLocaleDateString('pt-BR')} às ${dLostFallback.toLocaleTimeString('pt-BR')}`;
      }
      if (lostConnectionDetail) {
        lostConnectionDetail.textContent = 'Falha no ciclo agendado (+30 min)';
      }
    } else {
      if (lostConnectionTime) lostConnectionTime.textContent = '--';
      if (lostConnectionDetail) lostConnectionDetail.textContent = 'Aguardando primeira medição';
    }
  } else {
    smAlert.style.display = 'none';
  }
}

/* ==========================================================================
   4. Tabela de Nós com Filtros em Tempo Real (Substituição dos Cards)
   ========================================================================== */

let nodeFilterState = {
  search: '',
  status: 'all',
  group: 'all',
  date: ''
};

function populateGroupFilter(summaries) {
  if (!nodesGroupFilter) return;
  const currentVal = nodesGroupFilter.value;
  const groupsSet = new Set();
  summaries.forEach(s => {
    if (s.group_name) {
      groupsSet.add(s.group_name);
    }
  });

  const sortedGroups = Array.from(groupsSet).sort();
  nodesGroupFilter.innerHTML = '<option value="all">Todos os Grupos</option>';
  sortedGroups.forEach(gName => {
    const opt = document.createElement('option');
    opt.value = gName;
    opt.textContent = gName;
    nodesGroupFilter.appendChild(opt);
  });

  if (sortedGroups.includes(currentVal)) {
    nodesGroupFilter.value = currentVal;
  } else {
    nodesGroupFilter.value = 'all';
    nodeFilterState.group = 'all';
  }
}

function applyNodeFilters() {
  if (!currentSummaries) return;

  const query = (nodeFilterState.search || '').trim().toLowerCase();
  const statusFilter = nodeFilterState.status || 'all';
  const groupFilter = nodeFilterState.group || 'all';

  const filtered = currentSummaries.filter(s => {
    // 1. Filtro de Status
    if (statusFilter === 'online' && !s.is_healthy) return false;
    if (statusFilter === 'stale' && s.is_healthy) return false;

    // 2. Filtro de Grupo
    if (groupFilter !== 'all') {
      if ((s.group_name || '') !== groupFilter) return false;
    }

    // 3. Filtro de Texto (hostname, local, ip, mac, switch, porta, vlan)
    if (query) {
      const q = query;
      const cleanQ = q.replace(/[:-]/g, '');
      const matchNodeId = (s.node_id || '').toLowerCase().includes(q);
      const matchLoc = (s.location || '').toLowerCase().includes(q);
      const matchGroup = (s.group_name || '').toLowerCase().includes(q);
      const matchIp = (s.system_info && s.system_info.ip_address) ? s.system_info.ip_address.toLowerCase().includes(q) : false;
      const rawMac = (s.system_info && s.system_info.mac_address) ? s.system_info.mac_address.toLowerCase() : '';
      const cleanMac = rawMac.replace(/[:-]/g, '');
      const matchMac = rawMac.includes(q) || (cleanQ.length >= 3 && cleanMac.includes(cleanQ));
      const matchSwitch = (s.switch_info && s.switch_info.switch_name) ? s.switch_info.switch_name.toLowerCase().includes(q) : false;
      const matchPort = (s.switch_info && s.switch_info.switch_port) ? s.switch_info.switch_port.toLowerCase().includes(q) : false;
      const matchVlan = s.vlan ? String(s.vlan).includes(q) : false;

      if (!matchNodeId && !matchLoc && !matchGroup && !matchIp && !matchMac && !matchSwitch && !matchPort && !matchVlan) {
        return false;
      }
    }

    return true;
  });

  // Ordena pelo último sinal mais recente/próximo por primeiro
  filtered.sort((a, b) => {
    const timeA = a.last_seen ? new Date(a.last_seen).getTime() : 0;
    const timeB = b.last_seen ? new Date(b.last_seen).getTime() : 0;
    if (timeB !== timeA) return timeB - timeA;
    return (a.node_id || '').localeCompare(b.node_id || '');
  });

  if (nodesFilterCount) {
    nodesFilterCount.textContent = `${filtered.length} de ${currentSummaries.length} sondas`;
  }

  renderNodesTable(filtered);
}

function renderSummaryCards(summaries) {
  // Mantido para integração e compatibilidade com loadDashboardData
  populateGroupFilter(summaries);
  applyNodeFilters();
  renderKpiCards(summaries);
}

function renderKpiCards(summaries) {
  const totalEl = document.getElementById('kpi-total-nodes');
  const healthEl = document.getElementById('kpi-health-rate');
  const subEl = document.getElementById('kpi-nodes-sub');
  const downEl = document.getElementById('kpi-avg-download');
  const upEl = document.getElementById('kpi-avg-upload');
  const pingEl = document.getElementById('kpi-avg-ping');
  const pingSubEl = document.getElementById('kpi-ping-status');
  const card1Label = document.querySelector('.kpi-card:nth-child(1) .kpi-label');

  const isSingleNode = (selectedChartNode && selectedChartNode !== 'all');

  if (!summaries || summaries.length === 0) {
    if (card1Label) card1Label.textContent = isSingleNode ? 'Perda de Pacotes Gateway' : 'Sondas Monitoradas';
    if (totalEl) totalEl.textContent = isSingleNode ? '0.0%' : '0';
    if (healthEl) { healthEl.textContent = isSingleNode ? 'Sem Perda' : '0% Online'; healthEl.className = 'kpi-badge warning'; }
    if (subEl) subEl.textContent = 'Nenhuma medição detectada';
    if (downEl) downEl.textContent = '0.0';
    if (upEl) upEl.textContent = '0.0';
    if (pingEl) pingEl.textContent = '0.0';
    return;
  }

  if (isSingleNode) {
    // === MODO MÁQUINA INDIVIDUAL: CARD 1 = PERDA DE PACOTES GATEWAY ===
    if (card1Label) card1Label.textContent = 'Perda de Pacotes Gateway';

    const s = summaries.find(item => item.node_id === selectedChartNode) || summaries[0];
    let lossVal = 0.0;
    let targetGateway = '';
    let hasLossMetric = false;

    // 1. Tenta s.metrics.packet_loss_gateway
    if (s && s.metrics) {
      const pl = s.metrics.packet_loss_gateway || s.metrics.packet_loss;
      if (pl !== undefined && pl !== null) {
        hasLossMetric = true;
        if (typeof pl === 'object') {
          lossVal = (pl.value !== undefined) ? Number(pl.value) : (pl.loss_percent || 0);
          if (pl.details) targetGateway = pl.details;
        } else {
          lossVal = Number(pl);
        }
      }
    }
    // 2. Tenta s.modular_results.packet_loss
    if (!hasLossMetric && s && s.modular_results && s.modular_results.packet_loss) {
      const pl = s.modular_results.packet_loss;
      hasLossMetric = true;
      lossVal = pl.loss_percent !== undefined ? pl.loss_percent : (pl.lossPercent || 0);
      if (pl.target) targetGateway = 'Alvo: ' + pl.target;
    }
    // 3. Fallback no histórico recente
    if (!hasLossMetric && cachedMetricsSeries && cachedMetricsSeries.length > 0) {
      const lastItem = cachedMetricsSeries.filter(it => (it.node_id || it.NodeID) === selectedChartNode).pop();
      if (lastItem) {
        if (lastItem.metrics && (lastItem.metrics.packet_loss_gateway || lastItem.metrics.packet_loss)) {
          const pl = lastItem.metrics.packet_loss_gateway || lastItem.metrics.packet_loss;
          hasLossMetric = true;
          lossVal = (typeof pl === 'object' && pl.value !== undefined) ? Number(pl.value) : Number(pl);
        } else if (lastItem.modular_results && lastItem.modular_results.packet_loss) {
          hasLossMetric = true;
          lossVal = lastItem.modular_results.packet_loss.loss_percent || 0;
        }
      }
    }

    if (totalEl) totalEl.textContent = lossVal.toFixed(1) + '%';
    if (healthEl) {
      if (lossVal === 0) {
        healthEl.textContent = '0% Perda';
        healthEl.className = 'kpi-badge success';
      } else if (lossVal <= 5) {
        healthEl.textContent = lossVal.toFixed(1) + '% Perda';
        healthEl.className = 'kpi-badge warning';
      } else {
        healthEl.textContent = lossVal.toFixed(1) + '% Perda';
        healthEl.className = 'kpi-badge danger';
      }
    }

    if (subEl) {
      if (targetGateway) {
        subEl.textContent = targetGateway;
      } else {
        const gwIp = (s && s.system_info && s.system_info.gateway) || 'Gateway Local';
        subEl.textContent = 'Alvo: ' + gwIp + ' (10 amostras ICMP)';
      }
    }

    // Atualiza throughput e ping do nó focado
    if (downEl) downEl.textContent = (s.last_download || 0).toFixed(1);
    if (upEl) upEl.textContent = (s.last_upload || 0).toFixed(1);
    if (pingEl) pingEl.textContent = (s.last_ping || 0).toFixed(1);
    if (pingSubEl) {
      const p = s.last_ping || 0;
      if (p <= 5) pingSubEl.textContent = 'Latência excelente (< 5ms)';
      else if (p <= 20) pingSubEl.textContent = 'Latência estável (< 20ms)';
      else pingSubEl.textContent = 'Atenção (> 20ms)';
    }
  } else {
    // === MODO CONSOLIDADO: CARD 1 = SONDAS MONITORADAS ===
    if (card1Label) card1Label.textContent = 'Sondas Monitoradas';

    const total = summaries.length;
    const onlineCount = summaries.filter(s => s.is_healthy).length;
    const healthPct = Math.round((onlineCount / total) * 100);

    if (totalEl) totalEl.textContent = total;
    if (healthEl) {
      healthEl.textContent = healthPct + '% Online';
      healthEl.className = 'kpi-badge ' + (healthPct >= 80 ? 'success' : (healthPct >= 50 ? 'warning' : 'danger'));
    }
    if (subEl) subEl.textContent = onlineCount + ' de ' + total + ' ativas no campus';

    let sumDown = 0, sumUp = 0, sumPing = 0;
    summaries.forEach(s => {
      sumDown += (s.last_download || 0);
      sumUp += (s.last_upload || 0);
      sumPing += (s.last_ping || 0);
    });

    const avgDown = (sumDown / total).toFixed(1);
    const avgUp = (sumUp / total).toFixed(1);
    const avgPing = (sumPing / total).toFixed(1);

    if (downEl) downEl.textContent = avgDown;
    if (upEl) upEl.textContent = avgUp;
    if (pingEl) pingEl.textContent = avgPing;
    if (pingSubEl) {
      const p = parseFloat(avgPing);
      if (p <= 5) pingSubEl.textContent = 'Latência excelente (< 5ms)';
      else if (p <= 20) pingSubEl.textContent = 'Latência estável (< 20ms)';
      else pingSubEl.textContent = 'Atenção (> 20ms)';
    }
  }
}

let isTelemetryCollapsed = false;
function toggleTelemetryFeed() {
  const container = document.getElementById('telemetry-table-container');
  const label = document.getElementById('telemetry-toggle-text');
  if (!container) return;
  isTelemetryCollapsed = !isTelemetryCollapsed;
  if (isTelemetryCollapsed) {
    container.style.display = 'none';
    if (label) label.textContent = 'Expandir Feed';
  } else {
    container.style.display = 'block';
    if (label) label.textContent = 'Recolher';
  }
}

function renderNodesTable(summaries) {
  if (!nodesTableBody) return;

  if (!summaries || summaries.length === 0) {
    nodesTableBody.innerHTML = `
      <tr>
        <td colspan="10" class="nodes-empty-state">
          <p class="font-bold text-white">Nenhuma máquina encontrada com os filtros selecionados.</p>
          <p class="text-subtle">Ajuste o termo de busca ou limpe os filtros para visualizar as sondas.</p>
        </td>
      </tr>
    `;
    return;
  }

  const isAdmin = currentUser && currentUser.role === 'admin';

  nodesTableBody.innerHTML = summaries.map(s => {
    const lastDate = new Date(s.last_seen);
    const timeAgo = formatTimeAgo(lastDate);
    const healthClass = s.is_healthy ? 'online' : 'stale';
    const healthText = s.is_healthy ? 'Online' : 'Sem sinal';
    const isSelected = (selectedChartNode !== 'all' && s.node_id === selectedChartNode);

    // Switch & Porta
    let switchHtml = '<span class="text-subtle">--</span>';
    if (s.switch_info && s.switch_info.switch_name) {
      switchHtml = `
        <div class="switch-cell">
          <span class="switch-name" title="${escapeHtml(s.switch_info.switch_model || '')}">${escapeHtml(s.switch_info.switch_name)}</span>
          <span class="switch-port">${escapeHtml(s.switch_info.switch_port || '--')}</span>
        </div>
      `;
    }

    // Grupo e VLAN
    let groupHtml = '<span class="text-subtle">--</span>';
    if (s.group_name) {
      groupHtml = `<span class="vlan-pill">${escapeHtml(s.group_name)}${s.vlan ? ' (VLAN ' + escapeHtml(s.vlan) + ')' : ''}</span>`;
    }

    // IP
    const ipHtml = (s.system_info && s.system_info.ip_address)
      ? `<span class="node-sub-ip">${escapeHtml(s.system_info.ip_address)}</span>`
      : '';

    return `
      <tr class="node-row ${isSelected ? 'selected-chart-node' : ''}" data-node-id="${escapeHtml(s.node_id)}" title="Clique para focar gráficos nesta máquina">
        <td>
          <span class="status-indicator-badge ${healthClass}">
            <span class="status-indicator-dot"></span>
            ${healthText}
          </span>
        </td>
        <td>
          <div class="node-cell-main">
            <button type="button" class="btn-node-link font-bold font-mono" onclick="event.stopPropagation(); selectNodeForCharts('${escapeHtml(s.node_id)}', true)">
              ${escapeHtml(s.node_id)}
            </button>
            ${ipHtml}
          </div>
        </td>
        <td>
          <div class="node-cell-location">
            <svg class="location-pin-icon" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 2a8 8 0 0 0-8 8c0 5.25 8 12 8 12s8-6.75 8-12a8 8 0 0 0-8-8z"></path>
              <circle cx="12" cy="10" r="3"></circle>
            </svg>
            <span>${escapeHtml(s.location || '--')}</span>
          </div>
        </td>
        <td>${groupHtml}</td>
        <td>${switchHtml}</td>
        <td class="text-right">
          ${s.last_download > 0 ? `<span class="metric-num cyan">${s.last_download.toFixed(1)} <small>Mbps</small></span>` : `<span class="text-subtle font-mono" title="Não configurado no grupo">--</span>`}
        </td>
        <td class="text-right">
          ${s.last_upload > 0 ? `<span class="metric-num purple">${s.last_upload.toFixed(1)} <small>Mbps</small></span>` : `<span class="text-subtle font-mono" title="Não configurado no grupo">--</span>`}
        </td>
        <td class="text-right">
          <span class="metric-num emerald">${s.last_ping.toFixed(1)} <small>ms</small></span>
        </td>
        <td class="text-right">
          <span class="text-subtle font-mono">${timeAgo}</span>
        </td>
        <td class="text-right">
          <div class="table-actions" style="justify-content: flex-end;">
            ${isAdmin ? `
            <button type="button" class="btn-table-action" onclick="openPingModal('${escapeHtml(s.node_id)}', '${escapeHtml((s.system_info && s.system_info.ip_address) || '')}')" title="Forçar teste de Ping ICMP" style="color: var(--accent-cyan);">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
                <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
              </svg>
              <span>Ping</span>
            </button>
            ` : ''}
            <button type="button" class="btn-table-action btn-inspect" data-node-id="${escapeHtml(s.node_id)}" title="Inspecionar detalhes da máquina">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="3"></circle>
                <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
              </svg>
              <span>Detalhes</span>
            </button>
            ${isAdmin ? `
            <button type="button" class="btn-table-action btn-delete-node" data-node-id="${escapeHtml(s.node_id)}" title="Excluir máquina">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6"></polyline>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
              </svg>
            </button>
            ` : ''}
          </div>
        </td>
      </tr>
    `;
  }).join('');
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

  // 3. Rede Local (Resumo Primário)
  document.getElementById('modal-net-ip').textContent = sys.ip_address || sys.IPAddress || '--';
  document.getElementById('modal-net-mac').textContent = sys.mac_address || sys.MACAddress || '--';
  document.getElementById('modal-net-gateway').textContent = sys.gateway || sys.Gateway || '--';
  document.getElementById('modal-net-adapter').textContent = sys.network_adapter || sys.NetworkAdapter || '--';

  // 3.0 Múltiplas Interfaces e MACs
  const ifacesTbody = document.getElementById('modal-interfaces-body');
  const ifacesCount = document.getElementById('modal-interfaces-count');
  const ifacesList = sys.interfaces || sys.Interfaces || [];

  if (ifacesTbody) {
    if (ifacesList.length > 0) {
      if (ifacesCount) ifacesCount.textContent = `${ifacesList.length} placa(s)`;
      ifacesTbody.innerHTML = ifacesList.map(iface => {
        const isUp = !!iface.is_up;
        const isPrimary = !!iface.is_primary;
        const ifaceType = (iface.type || 'ethernet').toLowerCase();
        let badgeTypeClass = 'badge-iface-eth';
        let typeLabel = 'Ethernet';
        if (ifaceType === 'wifi' || ifaceType === 'wi-fi') {
          badgeTypeClass = 'badge-iface-wifi';
          typeLabel = 'Wi-Fi';
        } else if (ifaceType === 'virtual') {
          badgeTypeClass = 'badge-iface-virtual';
          typeLabel = 'Virtual';
        }

        const ipsStr = (iface.ip_addresses && iface.ip_addresses.length > 0)
          ? iface.ip_addresses.join('<br>')
          : '<span class="iface-no-ip">Sem IP</span>';

        return `
          <tr>
            <td>
              <span class="${isUp ? 'status-dot-up' : 'status-dot-down'}"></span>
              <span class="${isUp ? 'status-iface-active' : 'status-iface-inactive'} font-bold">${isUp ? 'Ativa' : 'Inativa'}</span>
            </td>
            <td>
              <div class="iface-name font-bold">${escapeHtml(iface.name || 'Interface')}</div>
              ${iface.description ? `<div class="iface-desc">${escapeHtml(iface.description)}</div>` : ''}
            </td>
            <td>
              <span class="badge-iface-type ${badgeTypeClass}">${typeLabel}</span>
            </td>
            <td>
              <span class="font-mono iface-mac">${escapeHtml(iface.mac_address || '--')}</span>
            </td>
            <td>
              <span class="font-mono ${isUp ? 'text-cyan font-bold' : ''}">${ipsStr}</span>
            </td>
            <td>
              ${isPrimary
                ? '<span class="badge-primary-route"><svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor" style="display:inline-block; vertical-align:-1px; margin-right:3px;"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon></svg>Principal</span>'
                : '<span class="iface-role-secondary">Secundária</span>'}
            </td>
          </tr>
        `;
      }).join('');
    } else {
      if (ifacesCount) ifacesCount.textContent = '1 placa';
      const fallbackMac = sys.mac_address || sys.MACAddress || '--';
      const fallbackIp = sys.ip_address || sys.IPAddress || '--';
      const fallbackName = sys.network_adapter || sys.NetworkAdapter || 'Interface Ativa';
      ifacesTbody.innerHTML = `
        <tr>
          <td>
            <span class="status-dot-up"></span>
            <span class="status-iface-active font-bold">Ativa</span>
          </td>
          <td>
            <div class="iface-name font-bold">${escapeHtml(fallbackName)}</div>
          </td>
          <td>
            <span class="badge-iface-type badge-iface-eth">Ethernet</span>
          </td>
          <td>
            <span class="font-mono iface-mac">${escapeHtml(fallbackMac)}</span>
          </td>
          <td>
            <span class="font-mono text-cyan font-bold">${escapeHtml(fallbackIp)}</span>
          </td>
          <td>
            <span class="badge-primary-route"><svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor" style="display:inline-block; vertical-align:-1px; margin-right:3px;"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon></svg>Principal</span>
          </td>
        </tr>
      `;
    }
  }

  // 3.1 Status Wi-Fi
  const wifiSection = document.getElementById('modal-wifi-section');
  const modResults = s.modular_results || {};
  const wf = modResults.wifi;

  if (wifiSection) {
    if (wf && wf.status && wf.status !== 'skipped') {
      wifiSection.style.display = 'block';
      const signalPct = Math.max(0, Math.min(100, wf.signal_quality_percent || 0));
      const signalDbm = wf.signal_strength_dbm ? `(${wf.signal_strength_dbm} dBm)` : '';

      const pctEl = document.getElementById('modal-wifi-signal-pct');
      const dbmEl = document.getElementById('modal-wifi-signal-dbm');
      const fillEl = document.getElementById('modal-wifi-signal-fill');
      const descEl = document.getElementById('modal-wifi-signal-desc');
      const stateBadge = document.getElementById('modal-wifi-state-badge');

      if (pctEl) pctEl.textContent = `${signalPct}%`;
      if (dbmEl) dbmEl.textContent = signalDbm;
      if (fillEl) {
        fillEl.style.width = `${signalPct}%`;
        fillEl.className = 'wifi-signal-bar-fill';
        if (signalPct < 40) fillEl.classList.add('poor');
        else if (signalPct < 70) fillEl.classList.add('warning');
      }

      let qualityText = 'Sinal Excelente';
      if (signalPct < 30) qualityText = 'Sinal Muito Fraco';
      else if (signalPct < 50) qualityText = 'Sinal Fraco';
      else if (signalPct < 75) qualityText = 'Sinal Regular/Bom';

      if (descEl) {
        descEl.textContent = wf.status === 'disconnected'
          ? 'Adaptador sem fio desconectado'
          : `${qualityText} • Canal ${wf.channel || '--'} (${wf.radio_type || 'Wi-Fi'})`;
      }

      if (stateBadge) {
        const isConn = wf.status === 'success' || wf.status === 'connected';
        stateBadge.className = `wifi-signal-state-badge ${isConn ? '' : 'disconnected'}`;
        stateBadge.textContent = isConn ? 'Conectado' : 'Desconectado';
      }

      const ssidEl = document.getElementById('modal-wifi-ssid');
      const bssidEl = document.getElementById('modal-wifi-bssid');
      const rcEl = document.getElementById('modal-wifi-radio-channel');
      const ratesEl = document.getElementById('modal-wifi-rates');

      if (ssidEl) ssidEl.textContent = wf.ssid || (wf.status === 'disconnected' ? 'Desconectado' : '--');
      if (bssidEl) bssidEl.textContent = wf.bssid || '--';
      if (rcEl) rcEl.textContent = `${wf.radio_type || '--'} • Canal ${wf.channel || '--'}`;
      if (ratesEl) {
        const tx = wf.tx_rate_mbps ? `${wf.tx_rate_mbps} Mbps` : '--';
        const rx = wf.rx_rate_mbps ? `${wf.rx_rate_mbps} Mbps` : '--';
        ratesEl.textContent = `TX: ${tx} / RX: ${rx}`;
      }
    } else {
      wifiSection.style.display = 'none';
    }
  }

  // 3.2 Benchmark Multi-DNS
  const dnsSection = document.getElementById('modal-dns-section');
  const dnsTbody = document.getElementById('modal-dns-tbody');
  const mdns = modResults.multi_dns;

  if (dnsSection && dnsTbody) {
    if (mdns && mdns.probes && mdns.probes.length > 0) {
      dnsSection.style.display = 'block';
      const fastestIp = mdns.fastest_server || '';

      dnsTbody.innerHTML = mdns.probes.map(probe => {
        const isFastest = (probe.server_ip === fastestIp && probe.status === 'success');
        let providerName = 'Servidor DNS';
        const instDnsIp = (window.APP_CONFIG && window.APP_CONFIG.dns_server_target) || '177.101.22.40';
        const instDnsName = (window.APP_CONFIG && window.APP_CONFIG.dns_server_name) || 'DNS Institucional';
        if (probe.server_ip && (probe.server_ip === instDnsIp || probe.server_ip.startsWith(instDnsIp))) {
          providerName = instDnsName;
        } else if (probe.server_ip === '1.1.1.1' || probe.server_ip === '1.0.0.1') {
          providerName = 'Cloudflare';
        } else if (probe.server_ip === '8.8.8.8' || probe.server_ip === '8.8.4.4') {
          providerName = 'Google Public DNS';
        } else if (probe.server_ip === '9.9.9.9') {
          providerName = 'Quad9';
        }

        let latClass = 'dns-lat-fast';
        if (probe.latency_ms > 80) latClass = 'dns-lat-slow';
        else if (probe.latency_ms > 30) latClass = 'dns-lat-mid';

        const latText = probe.status === 'success' ? `${probe.latency_ms.toFixed(1)} ms` : 'Timeout / Falha';

        return `
          <tr>
            <td>
              <span class="font-mono font-bold">${escapeHtml(probe.server_ip)}</span>
              ${isFastest ? '<span class="dns-fastest-badge"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" style="display:inline-block; vertical-align:-1px; margin-right:3px;"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg>Mais Rápido</span>' : ''}
            </td>
            <td>
              <span class="text-white">${providerName}</span>
            </td>
            <td>
              <span class="${latClass}">${latText}</span>
            </td>
            <td>
              <span class="${probe.status === 'success' ? 'text-emerald' : 'text-rose'} font-bold">
                ${probe.status === 'success' ? 'Resolvido' : 'Erro'}
              </span>
            </td>
            <td>
              <span class="font-mono text-cyan">${escapeHtml(probe.resolved_ip || '--')}</span>
            </td>
          </tr>
        `;
      }).join('');
    } else {
      dnsSection.style.display = 'none';
    }
  }

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
    const res = await apiFetch(`/api/v1/nodes?node_id=${encodeURIComponent(nodeId)}`, {
      method: 'DELETE'
    });

    const data = await res.json();

    if (!res.ok) {
      throw new Error(data.error || 'Falha ao excluir computador');
    }

    closeDeleteModal();
    closePCModal();

    showToast(`Computador "${nodeId}" foi excluído com sucesso!`, 'toast-success');

    if (selectedChartNode === nodeId) {
      selectedChartNode = 'all';
      updateChartFocusUI();
    }
    if (selectNode && selectNode.value === nodeId) {
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
  if (typeof Chart === 'undefined') {
    console.warn('[CHARTS] Chart.js ainda não disponível. Inicialização postergada.');
    return;
  }

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

  const canvasBandwidth = document.getElementById('centralBandwidthChart');
  if (canvasBandwidth && !bandwidthChart) {
    try {
      const ctxBandwidth = canvasBandwidth.getContext('2d');
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
    } catch (chartErr) {
      console.warn('Erro ao criar bandwidthChart:', chartErr);
    }
  }

  const canvasLatency = document.getElementById('centralLatencyChart');
  if (canvasLatency && !latencyChart) {
    try {
      const ctxLatency = canvasLatency.getContext('2d');
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
    } catch (chartErr) {
      console.warn('Erro ao criar latencyChart:', chartErr);
    }
  }
}

function renderComparativeCharts(series, nodesList) {
  cachedMetricsSeries = series || [];
  cachedNodesList = nodesList || [];

  updateMetricToggleVisibility();

  if (!bandwidthChart || !latencyChart) {
    try {
      initCharts();
    } catch {}
    if (!bandwidthChart || !latencyChart) return;
  }

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

  const selectedNode = selectedChartNode || (selectNode && selectNode.value) || 'all';
  const ctxBandwidth = document.getElementById('centralBandwidthChart').getContext('2d');
  const ctxLatency = document.getElementById('centralLatencyChart').getContext('2d');

  // Modo Sonda Individual: Foco nas métricas dessa sonda
  if (selectedNode !== 'all') {
    const allNodeItems = series
      .filter(s => (s.node_id || s.NodeID) === selectedNode)
      .sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));

    // Limita estritamente aos últimos 20 testes
    const nodeItems = allNodeItems.slice(-20);

    const isDifferentDay = nodeItems.some(item => {
      const d = new Date(item.timestamp);
      const today = new Date();
      return d.toDateString() !== today.toDateString();
    });

    const nodeLabels = nodeItems.map(item => {
      const d = new Date(item.timestamp);
      if (isDifferentDay) {
        return `${d.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })} ${d.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })}`;
      }
      return d.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });
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
        borderColor: '#4f46e5',
        backgroundColor: createFillGradient(ctxBandwidth, 'rgba(79, 70, 229, 0.20)', 'rgba(79, 70, 229, 0.01)'),
        fill: true,
        tension: 0.38,
        cubicInterpolationMode: 'monotone',
        spanGaps: true,
        borderWidth: 2.5,
        pointRadius: 3,
        pointHoverRadius: 6,
        pointBackgroundColor: '#4f46e5',
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
          borderColor: '#0284c7',
          backgroundColor: createFillGradient(ctxBandwidth, 'rgba(2, 132, 199, 0.22)', 'rgba(2, 132, 199, 0.01)'),
          fill: true,
          tension: 0.38,
          cubicInterpolationMode: 'monotone',
          spanGaps: true,
          borderWidth: 2.5,
          pointRadius: 3,
          pointHoverRadius: 6,
          pointBackgroundColor: '#0284c7',
          pointBorderColor: '#ffffff',
          pointBorderWidth: 1.5
        },
        {
          label: 'Upload',
          data: upData,
          borderColor: '#4f46e5',
          backgroundColor: createFillGradient(ctxBandwidth, 'rgba(79, 70, 229, 0.20)', 'rgba(79, 70, 229, 0.01)'),
          fill: true,
          tension: 0.38,
          cubicInterpolationMode: 'monotone',
          spanGaps: true,
          borderWidth: 2.5,
          pointRadius: 3,
          pointHoverRadius: 6,
          pointBackgroundColor: '#4f46e5',
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

  // Modo Consolidado (Todos os Nós ou Grupo Filtrado):
  let effectiveSeries = series;
  let activeGroupName = '';

  if (selectedChartGroup !== 'all') {
    const groupNodes = getNodesInGroup(selectedChartGroup);
    const grp = (currentGroups || []).find(g => g.id === selectedChartGroup);
    activeGroupName = grp ? grp.name : selectedChartGroup;

    effectiveSeries = series.filter(item => {
      const id = (item.node_id || item.NodeID || '').toLowerCase();
      return groupNodes.has(id);
    });

    if (effectiveSeries.length === 0) {
      if (bandwidthChartTitle) bandwidthChartTitle.textContent = `Histórico de Throughput - Grupo ${activeGroupName}`;
      if (bandwidthChartDesc) bandwidthChartDesc.textContent = `Nenhuma medição recente encontrada para as máquinas do grupo "${activeGroupName}"`;
      if (bandwidthLegendEl) bandwidthLegendEl.innerHTML = `<span class="text-subtle font-mono text-xs">Sem medições para o grupo ${escapeHtml(activeGroupName)}</span>`;
      if (latencyLegendEl) latencyLegendEl.innerHTML = `<span class="text-subtle font-mono text-xs">Sem medições para o grupo ${escapeHtml(activeGroupName)}</span>`;

      bandwidthChart.data.labels = [];
      bandwidthChart.data.datasets = [];
      bandwidthChart.update();

      latencyChart.data.labels = [];
      latencyChart.data.datasets = [];
      latencyChart.update();
      return;
    }
  }

  // Agrupa os timestamps por minuto para eliminar timestamps duplicados no eixo X
  const minuteMap = new Map();
  const byNode = {};

  effectiveSeries.forEach(item => {
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
    bandwidthChartTitle.textContent = activeGroupName
      ? `Histórico de Throughput (${metricLabel}) - Grupo ${activeGroupName}`
      : `Histórico de Throughput (${metricLabel})`;
  }
  if (bandwidthChartDesc) {
    bandwidthChartDesc.textContent = activeGroupName
      ? `Taxas de ${metricLabel} por Computador do grupo ${activeGroupName} (últimos 20 testes em Mbps)`
      : `Taxas de ${metricLabel} por Computador (últimos 20 testes em Mbps)`;
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
   8. Tabela de Feed de Telemetria com Paginação e Filtro por Dia
   ========================================================================== */

function renderTelemetryTable(series, diagnostic) {
  allTelemetrySeries = series || [];
  applyTelemetryFilters();
}

function applyTelemetryFilters() {
  if (!telemetryTableBody) return;

  let filtered = allTelemetrySeries || [];

  // 1. Filtro por Dia
  if (telemetryPageState.date) {
    filtered = filtered.filter(item => {
      if (!item.timestamp) return false;
      const d = new Date(item.timestamp);
      if (isNaN(d.getTime())) return false;
      return d.toLocaleDateString('en-CA') === telemetryPageState.date;
    });
  }

  // 2. Ordenação mais recente primeiro
  const sorted = [...filtered].sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));

  // 3. Paginação (15 itens por página)
  const total = sorted.length;
  const totalPages = Math.ceil(total / telemetryPageState.itemsPerPage) || 1;
  if (telemetryPageState.page > totalPages) telemetryPageState.page = totalPages;
  if (telemetryPageState.page < 1) telemetryPageState.page = 1;

  const startIdx = (telemetryPageState.page - 1) * telemetryPageState.itemsPerPage;
  const endIdx = Math.min(startIdx + telemetryPageState.itemsPerPage, total);
  const pagedItems = sorted.slice(startIdx, endIdx);

  // Atualiza controles visuais da barra e rodapé de telemetria
  const totalBadge = document.getElementById('telemetry-total-badge');
  const rangeEl = document.getElementById('telemetry-pagination-range');
  const pageDisplayEl = document.getElementById('telemetry-page-display');
  const totalPagesEl = document.getElementById('telemetry-total-pages-display');
  const btnPrev = document.getElementById('btn-telemetry-prev');
  const btnNext = document.getElementById('btn-telemetry-next');

  if (totalBadge) {
    totalBadge.textContent = `${total} mediç${total === 1 ? 'ão' : 'ões'}`;
  }
  if (rangeEl) {
    rangeEl.textContent = total > 0 ? `Exibindo ${startIdx + 1} a ${endIdx} de ${total}` : 'Exibindo 0 a 0 de 0';
  }
  if (pageDisplayEl) pageDisplayEl.textContent = String(telemetryPageState.page);
  if (totalPagesEl) totalPagesEl.textContent = String(totalPages);
  if (btnPrev) btnPrev.disabled = telemetryPageState.page <= 1;
  if (btnNext) btnNext.disabled = telemetryPageState.page >= totalPages;

  if (pagedItems.length === 0) {
    telemetryTableBody.innerHTML = `
      <tr>
        <td colspan="7" class="text-center text-muted py-4">
          Nenhuma medição encontrada para os filtros selecionados.
        </td>
      </tr>
    `;
    return;
  }

  telemetryTableBody.innerHTML = pagedItems.map(item => {
    const d = new Date(item.timestamp);
    const dateFormatted = `${d.toLocaleDateString('pt-BR')} ${d.toLocaleTimeString('pt-BR')}`;
    const id = item.node_id || item.NodeID;
    const isOld = (Date.now() - d.getTime() > 3 * 3600 * 1000);

    const badgeHtml = isOld
      ? '<span class="dc-health-badge stale" style="display:inline-flex;">● Histórico</span>'
      : '<span class="dc-health-badge online" style="display:inline-flex;">● Sincronizado</span>';

    return `
      <tr>
        <td style="font-family: 'JetBrains Mono', monospace; font-size: 0.8rem;">${dateFormatted}</td>
        <td><span class="telemetry-node-tag">${escapeHtml(id)}</span></td>
        <td>${escapeHtml(item.location || 'Desconhecido')}</td>
        <td class="text-right">${item.download_mbps > 0 ? `<strong style="color: var(--accent-cyan);">${item.download_mbps.toFixed(1)}</strong> <small>Mbps</small>` : `<span class="text-subtle font-mono">--</span>`}</td>
        <td class="text-right">${item.upload_mbps > 0 ? `<strong style="color: #c084fc;">${item.upload_mbps.toFixed(1)}</strong> <small>Mbps</small>` : `<span class="text-subtle font-mono">--</span>`}</td>
        <td class="text-right"><strong style="color: #34d399;">${item.ping_gateway.toFixed(1)}</strong> <small>ms</small></td>
        <td class="text-right">${badgeHtml}</td>
      </tr>
    `;
  }).join('');
}
window.applyTelemetryFilters = applyTelemetryFilters;

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

function showToast(message, type = 'toast-info') {
  const toast = document.createElement('div');
  toast.className = `toast ${type}`;

  let dotClass = 'toast-dot-info';
  if (type === 'toast-success') dotClass = 'toast-dot-success';
  if (type === 'toast-error') dotClass = 'toast-dot-error';

  toast.innerHTML = `<span class="toast-dot ${dotClass}"></span><span>${escapeHtml(message)}</span>`;
  toastContainer.appendChild(toast);

  setTimeout(() => {
    toast.remove();
  }, 3500);
}

/* ==========================================================================
   10. Gerenciamento Avançado de Máquinas & Sondas
   ========================================================================== */

function getMachineEffectiveLastDate(m) {
  if (!m) return null;
  const summary = currentSummaries ? currentSummaries.find(s => s.node_id === m.hostname) : null;
  let effectiveDate = null;
  if (summary && summary.last_seen) {
    const ds = new Date(summary.last_seen);
    if (!isNaN(ds.getTime())) effectiveDate = ds;
  }
  if (m.last_seen) {
    const dm = new Date(m.last_seen);
    if (!isNaN(dm.getTime()) && (!effectiveDate || dm > effectiveDate)) {
      effectiveDate = dm;
    }
  }
  return effectiveDate;
}

function getMachineLastSeenTimestamp(m) {
  const d = getMachineEffectiveLastDate(m);
  return d ? d.getTime() : 0;
}

async function loadAdminMachines() {
  if (!machinesTableBody) return;
  machinesTableBody.innerHTML = `<tr><td colspan="8" class="text-center py-4" style="color: var(--text-muted);"><div class="spinner" style="width:20px;height:20px;margin:0 auto 8px;"></div>Carregando máquinas e perfis...</td></tr>`;

  try {
    const [machRes, grpRes, modRes, dashRes] = await Promise.all([
      apiFetch('/api/v1/admin/machines'),
      apiFetch('/api/v1/admin/groups'),
      apiFetch('/api/v1/admin/modules'),
      apiFetch('/api/v1/dashboard/metrics')
    ]);

    if (dashRes.ok) {
      const dashData = await dashRes.json();
      currentSummaries = dashData.summaries || [];
    }
    currentGroups = grpRes.ok ? await grpRes.json() : [];
    allCatalogModules = modRes.ok ? await modRes.json() : [];

    if (machRes.ok) {
      allAdminMachines = await machRes.json() || [];
    } else {
      allAdminMachines = [];
    }

    // Garante que todas as máquinas com telemetria (detectadas no Dashboard) apareçam também na lista de gerenciamento
    const registeredHostnames = new Set(allAdminMachines.map(m => (m.hostname || '').toLowerCase()));
    currentSummaries.forEach(s => {
      if (!s.node_id) return;
      if (!registeredHostnames.has(s.node_id.toLowerCase())) {
        registeredHostnames.add(s.node_id.toLowerCase());
        allAdminMachines.push({
          hostname: s.node_id,
          ip_address: (s.system_info && s.system_info.ip_address) || '',
          mac_address: (s.system_info && s.system_info.mac_address) || '',
          switch_name: (s.switch_info && s.switch_info.switch_name) || '',
          switch_port: (s.switch_info && s.switch_info.switch_port) || '',
          vlan: (s.switch_info && s.switch_info.vlan) || s.vlan || '',
          last_seen: s.last_seen || '',
          group_id: s.group_id || 'acon',
          is_custom: false,
          custom_tests: []
        });
      }
    });

    // Desduplicação inteligente por MAC Address:
    // Máquinas com o mesmo endereço MAC físico são a mesma máquina; a leitura mais recente prevalece
    const macMap = new Map();
    const withoutMac = [];
    allAdminMachines.forEach(m => {
      const mac = (m.mac_address || '').trim().toUpperCase().replace(/-/g, ':');
      if (mac && mac !== '00:00:00:00:00:00' && mac.length >= 17) {
        if (!macMap.has(mac)) {
          macMap.set(mac, m);
        } else {
          const existing = macMap.get(mac);
          const timeM = getMachineLastSeenTimestamp(m);
          const timeEx = getMachineLastSeenTimestamp(existing);
          if (timeM > timeEx) {
            macMap.set(mac, m);
          }
        }
      } else {
        withoutMac.push(m);
      }
    });
    allAdminMachines = [...macMap.values(), ...withoutMac];

    // Ordena pelo último sinal mais recente/próximo por primeiro
    allAdminMachines.sort((a, b) => {
      const timeA = getMachineLastSeenTimestamp(a);
      const timeB = getMachineLastSeenTimestamp(b);
      if (timeB !== timeA) return timeB - timeA;
      return (a.hostname || '').localeCompare(b.hostname || '');
    });

    populateMachineGroupFilterOptions();
    applyMachineFilters();
  } catch (err) {
    machinesTableBody.innerHTML = `<tr><td colspan="8" class="text-center py-4 text-rose">Erro ao carregar máquinas: ${escapeHtml(err.message)}</td></tr>`;
  }
}
window.loadAdminMachines = loadAdminMachines;

function populateMachineGroupFilterOptions() {
  if (!machinesGroupFilter) return;
  const currentVal = machinesGroupFilter.value;
  machinesGroupFilter.innerHTML = '<option value="all">Todos os Grupos</option>';

  const groupIds = new Set();
  currentGroups.forEach(g => groupIds.add(g.id));
  allAdminMachines.forEach(m => { if (m.group_id) groupIds.add(m.group_id); });

  groupIds.forEach(gid => {
    const opt = document.createElement('option');
    opt.value = gid;
    const grp = currentGroups.find(g => g.id === gid);
    opt.textContent = grp ? `${grp.name} (${gid})` : gid;
    machinesGroupFilter.appendChild(opt);
  });

  if (groupIds.has(currentVal)) {
    machinesGroupFilter.value = currentVal;
  }
}

function applyMachineFilters() {
  if (!allAdminMachines) return;

  const query = (machineFilterState.search || '').trim().toLowerCase();
  const statusFilter = machineFilterState.status || 'all';
  const groupFilter = machineFilterState.group || 'all';
  const moduleFilter = machineFilterState.module || 'all';

  const filtered = allAdminMachines.filter(m => {
    // 1. Status Filter
    const summary = currentSummaries.find(s => s.node_id === m.hostname);
    const isOnline = summary ? summary.is_healthy : (m.last_seen && (Date.now() - new Date(m.last_seen).getTime()) < 15 * 60 * 1000);

    if (statusFilter === 'online' && !isOnline) return false;
    if (statusFilter === 'offline' && isOnline) return false;

    // 2. Group Filter
    if (groupFilter !== 'all' && m.group_id !== groupFilter) return false;

    // 3. Module Filter
    if (moduleFilter !== 'all') {
      let effectiveTests = [];
      if (m.is_custom && m.custom_tests && m.custom_tests.length > 0) {
        effectiveTests = m.custom_tests;
      } else {
        const grp = currentGroups.find(g => g.id === m.group_id);
        if (grp && grp.profile && grp.profile.tests) {
          effectiveTests = grp.profile.tests;
        }
      }
      const hasModule = effectiveTests.some(t => t.type === moduleFilter || t.id.includes(moduleFilter));
      if (!hasModule) return false;
    }

    // 4. Filtro por Dia (Último Sinal)
    if (machineFilterState.date) {
      const lastDate = getMachineEffectiveLastDate(m);
      if (!lastDate) return false;
      const dStr = lastDate.toLocaleDateString('en-CA');
      if (dStr !== machineFilterState.date) return false;
    }

    // 5. Text Search Multifuncional (hostname, IP, MAC, switch, porta, VLAN, grupo, local, módulos, status)
    if (query) {
      const q = query;
      const cleanQ = q.replace(/[:-]/g, '');

      const matchHost = (m.hostname || '').toLowerCase().includes(q);
      const matchIp = (m.ip_address || '').toLowerCase().includes(q) ||
                      (summary && summary.system_info && (summary.system_info.ip_address || '').toLowerCase().includes(q));

      const rawMac = (m.mac_address || (summary && summary.system_info && summary.system_info.mac_address) || '').toLowerCase();
      const cleanMac = rawMac.replace(/[:-]/g, '');
      const matchMac = rawMac.includes(q) || (cleanQ.length >= 3 && cleanMac.includes(cleanQ));

      const swName = (m.switch_name || (summary && summary.switch_info && summary.switch_info.switch_name) || '').toLowerCase();
      const matchSwitchName = swName.includes(q);

      const swPort = (m.switch_port || (summary && summary.switch_info && summary.switch_info.switch_port) || '').toLowerCase();
      const matchSwitchPort = swPort.includes(q);

      const swIp = (m.switch_ip || (summary && summary.switch_info && summary.switch_info.switch_ip) || '').toLowerCase();
      const matchSwitchIp = swIp.includes(q);

      const vlanVal = m.vlan || (summary && summary.vlan) || (summary && summary.switch_info && summary.switch_info.vlan) || '';
      const matchVlan = vlanVal ? String(vlanVal).includes(q) : false;

      const grp = currentGroups ? currentGroups.find(g => g.id === m.group_id) : null;
      const grpName = grp ? grp.name.toLowerCase() : '';
      const grpId = (m.group_id || '').toLowerCase();
      const matchGroup = grpId.includes(q) || grpName.includes(q);

      const loc = (m.location || (summary && summary.location) || '').toLowerCase();
      const matchLocation = loc.includes(q);

      let effectiveTests = [];
      if (m.is_custom && m.custom_tests && m.custom_tests.length > 0) {
        effectiveTests = m.custom_tests;
      } else if (grp && grp.profile && grp.profile.tests) {
        effectiveTests = grp.profile.tests;
      }
      const matchModule = effectiveTests.some(t =>
        (t.id || '').toLowerCase().includes(q) ||
        (t.name || '').toLowerCase().includes(q) ||
        (t.type || '').toLowerCase().includes(q)
      );

      const statusText = isOnline ? 'online conectado ativo' : 'sem sinal offline desconectado';
      const matchStatus = statusText.includes(q);

      if (!matchHost && !matchIp && !matchMac && !matchSwitchName && !matchSwitchPort &&
          !matchSwitchIp && !matchVlan && !matchGroup && !matchLocation && !matchModule && !matchStatus) {
        return false;
      }
    }

    return true;
  });

  // Ordena com o último sinal mais recente/próximo por primeiro
  filtered.sort((a, b) => {
    const timeA = getMachineLastSeenTimestamp(a);
    const timeB = getMachineLastSeenTimestamp(b);
    if (timeB !== timeA) {
      return timeB - timeA;
    }
    return (a.hostname || '').localeCompare(b.hostname || '');
  });

  const total = filtered.length;
  const totalPages = Math.ceil(total / machinesPaginationState.itemsPerPage) || 1;
  if (machinesPaginationState.page > totalPages) {
    machinesPaginationState.page = totalPages;
  }
  if (machinesPaginationState.page < 1) {
    machinesPaginationState.page = 1;
  }

  const startIdx = (machinesPaginationState.page - 1) * machinesPaginationState.itemsPerPage;
  const endIdx = Math.min(startIdx + machinesPaginationState.itemsPerPage, total);
  const pagedMachines = filtered.slice(startIdx, endIdx);

  // Controles de Paginação de Máquinas
  const rangeEl = document.getElementById('machines-pagination-range');
  const pageDisplayEl = document.getElementById('machines-page-display');
  const totalPagesEl = document.getElementById('machines-total-pages-display');
  const btnPrev = document.getElementById('btn-machines-prev');
  const btnNext = document.getElementById('btn-machines-next');

  if (rangeEl) {
    rangeEl.textContent = total > 0 ? `Exibindo ${startIdx + 1} a ${endIdx} de ${total}` : 'Exibindo 0 a 0 de 0';
  }
  if (pageDisplayEl) pageDisplayEl.textContent = String(machinesPaginationState.page);
  if (totalPagesEl) totalPagesEl.textContent = String(totalPages);
  if (btnPrev) btnPrev.disabled = machinesPaginationState.page <= 1;
  if (btnNext) btnNext.disabled = machinesPaginationState.page >= totalPages;

  if (machinesFilterCount) {
    machinesFilterCount.textContent = `${total} de ${allAdminMachines.length} máquina${allAdminMachines.length !== 1 ? 's' : ''}`;
  }

  renderMachinesTable(pagedMachines);
}

function renderMachinesTable(machines) {
  if (!machinesTableBody) return;

  if (machines.length === 0) {
    machinesTableBody.innerHTML = `
      <tr>
        <td colspan="8" class="text-center py-5 text-subtle">
          <p class="font-bold text-white mb-1">Nenhuma máquina encontrada com os filtros selecionados.</p>
          <p>Tente ajustar os termos de busca ou limpar os filtros de grupo/módulo.</p>
        </td>
      </tr>
    `;
    return;
  }

  const isAdmin = currentUser && currentUser.role === 'admin';

  machinesTableBody.innerHTML = machines.map(m => {
    const summary = currentSummaries.find(s => s.node_id === m.hostname);
    const effectiveLastDate = getMachineEffectiveLastDate(m);
    const isOnline = summary ? summary.is_healthy : (effectiveLastDate && (Date.now() - effectiveLastDate.getTime()) < 15 * 60 * 1000);
    const statusClass = isOnline ? 'online' : 'stale';
    const statusText = isOnline ? 'Online' : 'Sem sinal';

    // Grupo e Módulos vinculados
    const grp = currentGroups.find(g => g.id === m.group_id);
    const grpName = grp ? grp.name : (m.group_id || 'default');
    const grpTests = (grp && grp.profile && grp.profile.tests) ? grp.profile.tests : [];

    let effectiveTests = [];
    if (m.is_custom && m.custom_tests && m.custom_tests.length > 0) {
      effectiveTests = m.custom_tests;
    } else {
      effectiveTests = grpTests;
    }

    const grpTestIds = new Set(grpTests.map(t => t.id));
    const activeTestIds = new Set(effectiveTests.map(t => t.id));

    const addedModules = effectiveTests.filter(t => !grpTestIds.has(t.id));
    const removedModules = grpTests.filter(t => !activeTestIds.has(t.id));
    const isModified = m.is_custom && (addedModules.length > 0 || removedModules.length > 0);

    const tooltipTests = effectiveTests.map(t => t.name || t.id).join(' • ');

    let modulesBadgeHtml = '';
    if (effectiveTests.length === 0) {
      modulesBadgeHtml = `<span class="text-subtle font-mono text-xs">Nenhum</span>`;
    } else if (!isModified) {
      modulesBadgeHtml = `
        <span class="badge-group-standard" title="Executa todos os ${effectiveTests.length} módulos padrão do grupo (${escapeHtml(tooltipTests)})">
          <span style="display:inline-block;width:6px;height:6px;border-radius:50%;background:#34d399;"></span>
          Padrão do Grupo (${effectiveTests.length})
        </span>
      `;
    } else {
      const diffParts = [];
      if (addedModules.length > 0) diffParts.push(`+${addedModules.length} a mais`);
      if (removedModules.length > 0) diffParts.push(`-${removedModules.length} a menos`);
      const diffText = diffParts.join(', ') || `${effectiveTests.length} ativos`;
      modulesBadgeHtml = `
        <span class="badge-custom-tag" title="Módulos ativos (${effectiveTests.length}): ${escapeHtml(tooltipTests)}">
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="display:inline-block; vertical-align:-1px; margin-right:3px;"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>
          Customizado (${diffText})
        </span>
      `;
    }

    // Switch & Porta
    const switchName = m.switch_name || (summary && summary.switch_info && summary.switch_info.switch_name) || '--';
    const switchPort = m.switch_port || (summary && summary.switch_info && summary.switch_info.switch_port) || '';
    const switchHtml = switchName !== '--'
      ? `<div class="switch-cell"><span class="switch-name">${escapeHtml(switchName)}</span><span class="switch-port">${escapeHtml(switchPort)}</span></div>`
      : '<span class="text-subtle">--</span>';

    // Tempo decorrido
    const timeAgo = effectiveLastDate ? formatTimeAgo(effectiveLastDate) : '--';

    // Select de troca rápida de grupo (ou badge)
    let groupSelectHtml = `<span class="badge-group-tag">${escapeHtml(grpName)}</span>`;
    if (isAdmin) {
      groupSelectHtml = `
        <select class="form-input form-input-sm group-quick-select" onchange="handleQuickChangeGroup('${escapeHtml(m.hostname)}', this.value)">
          ${currentGroups.map(g => `<option value="${escapeHtml(g.id)}" ${g.id === m.group_id ? 'selected' : ''}>${escapeHtml(g.name)}</option>`).join('')}
        </select>
      `;
    }

    return `
      <tr class="node-row" data-hostname="${escapeHtml(m.hostname)}">
        <td>
          <span class="status-indicator-badge ${statusClass}">
            <span class="status-indicator-dot"></span>
            ${statusText}
          </span>
        </td>
        <td>
          <div class="node-cell-main">
            <button type="button" class="btn-node-link font-bold font-mono" onclick="navigateToMachineDashboard('${escapeHtml(m.hostname)}')" title="Abrir Dashboard desta máquina">
              ${escapeHtml(m.hostname)}
            </button>
          </div>
        </td>
        <td>
          <div style="font-size: 0.76rem; font-family: 'JetBrains Mono', monospace; line-height: 1.35; white-space: nowrap;">
            <div style="font-weight: 600; color: var(--text-primary);">${escapeHtml(m.ip_address || '--')}</div>
            <div style="color: var(--text-muted); font-size: 0.70rem; letter-spacing: 0.2px;">${escapeHtml(m.mac_address || '--')}</div>
          </div>
        </td>
        <td>
          ${groupSelectHtml}
        </td>
        <td>
          <div class="modules-compact-cell">
            ${modulesBadgeHtml}
          </div>
        </td>
        <td style="text-align: right;">${switchHtml}</td>
        <td style="text-align: right; white-space: nowrap;"><span class="text-subtle font-mono">${timeAgo}</span></td>
        <td class="col-actions" style="text-align: right;">
          <div class="table-actions" style="display: inline-flex; justify-content: flex-end; align-items: center; gap: 4px; width: 100%;">
            ${isAdmin ? `
            <button type="button" class="btn btn-outline-cyan btn-sm btn-action-compact" onclick="openPingModal('${escapeHtml(m.hostname)}', '${escapeHtml(m.ip_address || '')}')" title="Forçar Ping ICMP nesta máquina">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg>
              <span>Ping</span>
            </button>
            ` : ''}
            <button type="button" class="btn btn-glass btn-sm btn-action-compact" onclick="openMachineMetricsModal('${escapeHtml(m.hostname)}')" title="Visualizar principais informações e métricas da máquina">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
              <span>Detalhes</span>
            </button>
            <button type="button" class="btn btn-glass btn-sm btn-action-compact" onclick="openMachineConfigModal('${escapeHtml(m.hostname)}')" title="Alterar grupo e módulos">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>
              <span>Editar</span>
            </button>
            ${isAdmin ? `
            <button type="button" class="btn btn-outline-danger btn-sm btn-icon-only" onclick="promptDeleteNode('${escapeHtml(m.hostname)}')" title="Excluir máquina">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
            </button>
            ` : ''}
          </div>
        </td>
      </tr>
    `;
  }).join('');
}

async function handleQuickChangeGroup(hostname, newGroupId) {
  if (!confirm(`Deseja alterar a máquina "${hostname}" para o grupo "${newGroupId}"?`)) {
    loadAdminMachines();
    return;
  }
  try {
    const res = await apiFetch(`/api/v1/admin/machines/${encodeURIComponent(hostname)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        group_id: newGroupId,
        is_custom: false
      })
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao alterar grupo');
    showToast(`Máquina "${hostname}" movida para o grupo "${newGroupId}"!`, 'toast-success');
    await loadAdminMachines();
  } catch (err) {
    showToast(`Erro ao alterar grupo: ${err.message}`, 'toast-error');
    await loadAdminMachines();
  }
}
window.handleQuickChangeGroup = handleQuickChangeGroup;

/* ==========================================================================
   11. Modal de Métricas e Gráficos da Máquina
   ========================================================================== */

function openMachineMetricsModal(hostname) {
  if (!machineMetricsModal) return;
  currentInspectedMachine = hostname;

  const machine = allAdminMachines.find(m => m.hostname === hostname) || { hostname };
  const summary = currentSummaries ? currentSummaries.find(s => s.node_id === hostname) : null;
  const effectiveLastDate = getMachineEffectiveLastDate(machine);

  const grp = currentGroups ? currentGroups.find(g => g.id === machine.group_id) : null;
  const grpName = grp ? grp.name : (machine.group_id || 'default');

  document.getElementById('mm-hostname').textContent = hostname;
  document.getElementById('mm-subtitle').textContent = `Grupo: ${grpName} • ${machine.is_custom ? 'Perfil Customizado' : 'Perfil Herdado'}`;

  document.getElementById('mm-stat-ping').textContent = summary ? summary.last_ping.toFixed(1) : '--';
  document.getElementById('mm-stat-down').textContent = summary ? summary.last_download.toFixed(1) : '--';
  document.getElementById('mm-stat-up').textContent = summary ? summary.last_upload.toFixed(1) : '--';
  document.getElementById('mm-stat-readings').textContent = summary ? summary.total_readings_24h : '0';

  const subStatus = document.getElementById('mm-sub-status');
  const isOnline = summary ? summary.is_healthy : (effectiveLastDate && (Date.now() - effectiveLastDate.getTime()) < 15 * 60 * 1000);
  if (subStatus) {
    subStatus.textContent = isOnline ? 'Online na rede' : 'Sem sinal recente';
    subStatus.style.color = isOnline ? '#34d399' : '#f87171';
  }

  // Hardware e Rede
  const sys = (summary && summary.system_info) || {};
  document.getElementById('mm-hw-ip').textContent = machine.ip_address || sys.ip_address || '--';
  document.getElementById('mm-hw-mac').textContent = machine.mac_address || sys.mac_address || '--';
  document.getElementById('mm-hw-os').textContent = sys.os || '--';
  document.getElementById('mm-hw-specs').textContent = `${sys.cpu || '--'} (${sys.ram_total_gb ? sys.ram_total_gb.toFixed(1) + 'GB' : '--'})`;

  const lastSeenEl = document.getElementById('mm-hw-last-seen');
  if (lastSeenEl) {
    if (effectiveLastDate) {
      const dateStr = effectiveLastDate.toLocaleString('pt-BR');
      const timeAgo = formatTimeAgo(effectiveLastDate);
      lastSeenEl.textContent = `${dateStr} (${timeAgo})`;
    } else {
      lastSeenEl.textContent = 'Nunca registrado';
    }
  }

  // Wi-Fi (se presente na telemetria ou modular_results)
  const wifiRow = document.getElementById('mm-wifi-row');
  const wifiValEl = document.getElementById('mm-hw-wifi');
  if (wifiRow && wifiValEl) {
    const modResults = (summary && summary.modular_results) || {};
    const wf = modResults.wifi;
    const wsMetric = (summary && summary.metrics && summary.metrics.wifi_status);

    if (wf && wf.status && wf.status !== 'skipped' && wf.status !== 'no_adapter') {
      wifiRow.style.display = 'flex';
      if (wf.status === 'disconnected') {
        wifiValEl.textContent = 'Adaptador desconectado';
        wifiValEl.className = 'detail-val font-semibold text-subtle';
      } else {
        const pct = wf.signal_quality_percent || 0;
        const dbmStr = wf.signal_strength_dbm ? ` (${wf.signal_strength_dbm} dBm)` : '';
        wifiValEl.textContent = `${wf.ssid || 'Conectado'}: ${pct}%${dbmStr}`;
        wifiValEl.className = 'detail-val font-semibold text-emerald';
      }
    } else if (wsMetric && wsMetric.status === 'ok') {
      wifiRow.style.display = 'flex';
      const pct = Math.round(wsMetric.value || 0);
      const dbm = Math.round((pct / 2) - 100);
      wifiValEl.textContent = `${pct}% (${dbm} dBm)`;
      wifiValEl.className = 'detail-val font-semibold text-emerald';
    } else {
      wifiRow.style.display = 'none';
    }
  }

  // Switch LLDP
  const sw = (summary && summary.switch_info) || {};
  document.getElementById('mm-sw-name').textContent = machine.switch_name || sw.switch_name || 'Não descoberto';
  document.getElementById('mm-sw-port').textContent = machine.switch_port || sw.switch_port || '--';
  document.getElementById('mm-sw-vlan').textContent = machine.vlan ? `VLAN ${machine.vlan}` : (sw.vlan ? `VLAN ${sw.vlan}` : '--');
  document.getElementById('mm-sw-proto').textContent = sw.protocol || 'LLDP/CDP';

  const swGroupEl = document.getElementById('mm-sw-group');
  if (swGroupEl) {
    swGroupEl.textContent = grpName;
  }

  // Módulos Ativos
  const modulesListEl = document.getElementById('mm-modules-list');
  if (modulesListEl) {
    let effectiveTests = [];
    if (machine.is_custom && machine.custom_tests && machine.custom_tests.length > 0) {
      effectiveTests = machine.custom_tests;
    } else if (grp && grp.profile && grp.profile.tests) {
      effectiveTests = grp.profile.tests;
    }

    if (effectiveTests.length === 0) {
      modulesListEl.innerHTML = '<span class="text-subtle font-mono text-xs">Nenhum módulo configurado</span>';
    } else {
      modulesListEl.innerHTML = effectiveTests.map(t => {
        const testName = t.name || t.id;
        return `<span class="badge-group-standard" style="margin-right: 4px; margin-bottom: 4px;"><span style="display:inline-block;width:6px;height:6px;border-radius:50%;background:#38bdf8;margin-right:4px;"></span>${escapeHtml(testName)}</span>`;
      }).join('');
    }
  }

  machineMetricsModal.style.display = 'flex';
}
window.openMachineMetricsModal = openMachineMetricsModal;

function closeMachineMetricsModal() {
  if (machineMetricsModal) machineMetricsModal.style.display = 'none';
}

/* ==========================================================================
   12. Modal de Configuração Customizada da Máquina
   ========================================================================== */

function getAllAvailableModules() {
  const modMap = new Map();

  // 1. Módulos padrão do sistema (puxados dinamicamente do .env através de window.APP_CONFIG)
  const instDnsIp = (window.APP_CONFIG && window.APP_CONFIG.dns_server_target) || '177.101.22.40';
  const instDnsDomain = (window.APP_CONFIG && window.APP_CONFIG.dns_domain_target) || 'uepg.br';
  const instDnsName = (window.APP_CONFIG && window.APP_CONFIG.dns_server_name) || 'Resolução DNS Institucional';
  const coreTarget = (window.APP_CONFIG && window.APP_CONFIG.ping_core_target) || '177.101.22.40';
  const coreName = (window.APP_CONFIG && window.APP_CONFIG.ping_core_name) || 'Latência Core NTI (Anel)';
  const defaultPing = (window.APP_CONFIG && window.APP_CONFIG.default_ping_target) || 'auto_gateway';
  const dlMb = (window.APP_CONFIG && window.APP_CONFIG.speedtest_download_mb) || 5;
  const ulMb = (window.APP_CONFIG && window.APP_CONFIG.speedtest_upload_mb) || 2;

  const systemDefaults = [
    { id: 'ping_gateway', name: 'Latência Gateway Local', type: 'ping', target: defaultPing, timeout_seconds: 15 },
    { id: 'ping_nti_core', name: coreName, type: 'ping', target: coreTarget, timeout_seconds: 15 },
    { id: 'packet_loss_gateway', name: 'Perda de Pacotes Gateway', type: 'packet_loss', target: defaultPing, timeout_seconds: 20 },
    { id: 'dns_uepg', name: instDnsName, type: 'dns_lookup', domain: instDnsDomain, timeout_seconds: 10 },
    { id: 'throughput_download', name: 'Throughput Download', type: 'http_throughput_download', endpoint: '/api/v1/speedtest/download', size_mb: dlMb, timeout_seconds: 45 },
    { id: 'throughput_upload', name: 'Throughput Upload', type: 'http_throughput_upload', endpoint: '/api/v1/speedtest/upload', size_mb: ulMb, timeout_seconds: 45 },
    { id: 'switch_discovery', name: 'Descoberta de Switch LLDP/CDP', type: 'discovery_protocol', protocol: 'LLDP_CDP', timeout_seconds: 60 },
    { id: 'ping_central_server', name: 'Ping Servidor Central', type: 'ping', target: defaultPing, timeout_seconds: 10 },
    { id: 'http_check_portal', name: 'Disponibilidade Portal Web', type: 'http_check', endpoint: `https://${instDnsDomain}`, timeout_seconds: 15 }
  ];
  systemDefaults.forEach(m => modMap.set(m.id, m));

  // 2. Módulos de todos os grupos carregados
  currentGroups.forEach(g => {
    if (g.profile && g.profile.tests) {
      g.profile.tests.forEach(t => {
        if (t && t.id) modMap.set(t.id, t);
      });
    }
  });

  // 3. Módulos do catálogo importados
  if (allCatalogModules && allCatalogModules.length > 0) {
    allCatalogModules.forEach(m => {
      if (m && m.id) modMap.set(m.id, m);
    });
  }

  return Array.from(modMap.values());
}

function openMachineConfigModal(hostname) {
  if (!machineConfigModal) return;
  const machine = allAdminMachines.find(m => m.hostname === hostname);
  if (!machine) {
    showToast(`Máquina ${hostname} não encontrada`, 'toast-error');
    return;
  }

  mcInputHostname.value = machine.hostname;
  mcHostnameTitle.textContent = `Configurações da sonda: ${machine.hostname}`;

  mcSelectGroup.innerHTML = '';
  const availableGroups = currentGroups.length > 0 ? currentGroups : [
    { id: 'default', name: 'Grupo Padrão' },
    { id: 'reitoria', name: 'Grupo Reitoria' },
    { id: 'acon', name: 'Grupo ACON' }
  ];

  availableGroups.forEach(g => {
    const opt = document.createElement('option');
    opt.value = g.id;
    opt.textContent = `${g.name} (${g.id})`;
    if (g.id === machine.group_id) opt.selected = true;
    mcSelectGroup.appendChild(opt);
  });

  // Determina os IDs inicialmente marcados
  const initialCheckedIds = new Set();
  const activeGrp = currentGroups.find(g => g.id === machine.group_id);
  const grpTests = (activeGrp && activeGrp.profile && activeGrp.profile.tests) ? activeGrp.profile.tests : [];

  if (machine.is_custom && machine.custom_tests && machine.custom_tests.length > 0) {
    machine.custom_tests.forEach(t => initialCheckedIds.add(t.id));
  } else {
    grpTests.forEach(t => initialCheckedIds.add(t.id));
  }

  renderMachineModulesCheckboxes(machine.group_id, initialCheckedIds);
  machineConfigModal.style.display = 'flex';
}
window.openMachineConfigModal = openMachineConfigModal;

function closeMachineConfigModal() {
  if (machineConfigModal) machineConfigModal.style.display = 'none';
}

function renderMachineModulesCheckboxes(groupId, checkedModuleIds) {
  if (!mcModulesCheckboxesList) return;

  const activeGrp = currentGroups.find(g => g.id === groupId);
  const grpTests = (activeGrp && activeGrp.profile && activeGrp.profile.tests) ? activeGrp.profile.tests : [];
  const grpTestIds = new Set(grpTests.map(t => t.id));

  const allModules = getAllAvailableModules();

  const groupModules = allModules.filter(m => grpTestIds.has(m.id));
  const otherModules = allModules.filter(m => !grpTestIds.has(m.id));

  let html = '';

  if (groupModules.length > 0) {
    html += `<div class="mc-section-heading"><span>Módulos do Grupo (${escapeHtml(activeGrp ? activeGrp.name : groupId)})</span></div>`;
    groupModules.forEach(mod => {
      const isChecked = checkedModuleIds.has(mod.id);
      const stateClass = isChecked ? '' : 'is-unchecked-group';
      const stateBadge = isChecked
        ? '<span class="mc-module-tag-state state-default">Padrão do Grupo</span>'
        : '<span class="mc-module-tag-state state-removed">- Removido</span>';

      html += `
        <label class="mc-module-check-item ${stateClass}" data-module-id="${escapeHtml(mod.id)}" data-is-group="true">
          <div class="mc-module-check-left">
            <input type="checkbox" class="mc-module-cb" data-module-id="${escapeHtml(mod.id)}" ${isChecked ? 'checked' : ''} />
            <div>
              <div class="mc-module-title">${escapeHtml(mod.name)}</div>
              <div class="mc-module-sub">${escapeHtml(mod.id)} • ${escapeHtml(mod.type)}</div>
            </div>
          </div>
          <div style="display: flex; align-items: center; gap: 8px;">
            ${stateBadge}
            <span class="module-type-badge ${escapeHtml(mod.type)}">${escapeHtml(mod.type)}</span>
          </div>
        </label>
      `;
    });
  }

  if (otherModules.length > 0) {
    html += `<div class="mc-section-heading"><span>Módulos Adicionais do Catálogo (Opcionais)</span></div>`;
    otherModules.forEach(mod => {
      const isChecked = checkedModuleIds.has(mod.id);
      const stateClass = isChecked ? 'is-added-extra' : '';
      const stateBadge = isChecked
        ? '<span class="mc-module-tag-state state-added">+ Adicionado a Mais</span>'
        : '<span class="text-subtle font-mono" style="font-size: 0.68rem;">Opcional</span>';

      html += `
        <label class="mc-module-check-item ${stateClass}" data-module-id="${escapeHtml(mod.id)}" data-is-group="false">
          <div class="mc-module-check-left">
            <input type="checkbox" class="mc-module-cb" data-module-id="${escapeHtml(mod.id)}" ${isChecked ? 'checked' : ''} />
            <div>
              <div class="mc-module-title">${escapeHtml(mod.name)}</div>
              <div class="mc-module-sub">${escapeHtml(mod.id)} • ${escapeHtml(mod.type)}</div>
            </div>
          </div>
          <div style="display: flex; align-items: center; gap: 8px;">
            ${stateBadge}
            <span class="module-type-badge ${escapeHtml(mod.type)}">${escapeHtml(mod.type)}</span>
          </div>
        </label>
      `;
    });
  }

  mcModulesCheckboxesList.innerHTML = html;
  updateMachineConfigStatusSummary(groupId);
}

function updateMachineConfigStatusSummary(groupId) {
  if (!mcModulesCheckboxesList) return;

  const activeGrp = currentGroups.find(g => g.id === groupId);
  const grpTests = (activeGrp && activeGrp.profile && activeGrp.profile.tests) ? activeGrp.profile.tests : [];
  const grpTestIds = new Set(grpTests.map(t => t.id));

  const checkedBoxes = mcModulesCheckboxesList.querySelectorAll('.mc-module-cb:checked');
  const checkedIds = new Set(Array.from(checkedBoxes).map(cb => cb.getAttribute('data-module-id')));

  // Atualiza tags de estado dinamicamente
  const allItems = mcModulesCheckboxesList.querySelectorAll('.mc-module-check-item');
  allItems.forEach(item => {
    const modId = item.getAttribute('data-module-id');
    const isGrp = item.getAttribute('data-is-group') === 'true';
    const isChecked = checkedIds.has(modId);
    const cb = item.querySelector('.mc-module-cb');
    if (cb) cb.checked = isChecked;

    const stateContainer = item.querySelector('.mc-module-tag-state') || item.querySelector('.text-subtle');

    if (isGrp) {
      if (isChecked) {
        item.classList.remove('is-unchecked-group');
        if (stateContainer) stateContainer.outerHTML = '<span class="mc-module-tag-state state-default">Padrão do Grupo</span>';
      } else {
        item.classList.add('is-unchecked-group');
        if (stateContainer) stateContainer.outerHTML = '<span class="mc-module-tag-state state-removed">- Removido</span>';
      }
    } else {
      if (isChecked) {
        item.classList.add('is-added-extra');
        if (stateContainer) stateContainer.outerHTML = '<span class="mc-module-tag-state state-added">+ Adicionado a Mais</span>';
      } else {
        item.classList.remove('is-added-extra');
        if (stateContainer) stateContainer.outerHTML = '<span class="text-subtle font-mono" style="font-size: 0.68rem;">Opcional</span>';
      }
    }
  });

  const added = Array.from(checkedIds).filter(id => !grpTestIds.has(id));
  const removed = Array.from(grpTestIds).filter(id => !checkedIds.has(id));
  const isCustom = (added.length > 0 || removed.length > 0);

  const mcStatusBadge = document.getElementById('mc-status-badge');
  const mcStatusDiffText = document.getElementById('mc-status-diff-text');
  const mcModulesCountTag = document.getElementById('mc-modules-count-tag');

  if (mcModulesCountTag) {
    mcModulesCountTag.textContent = `${checkedIds.size} ativo${checkedIds.size !== 1 ? 's' : ''}`;
  }

  if (!isCustom) {
    if (mcStatusBadge) {
      mcStatusBadge.className = 'badge-group-standard';
      mcStatusBadge.innerHTML = '● Padrão do Grupo';
    }
    if (mcStatusDiffText) {
      mcStatusDiffText.textContent = `Herda todos os ${grpTestIds.size} módulos do grupo sem alterações`;
    }
  } else {
    if (mcStatusBadge) {
      mcStatusBadge.className = 'badge-custom-tag';
      mcStatusBadge.innerHTML = '<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="display:inline-block; vertical-align:-1px; margin-right:3px;"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg> Perfil Customizado';
    }
    const diffParts = [];
    if (added.length > 0) diffParts.push(`+${added.length} a mais`);
    if (removed.length > 0) diffParts.push(`-${removed.length} a menos`);
    if (mcStatusDiffText) {
      mcStatusDiffText.textContent = `Alteração: ${diffParts.join(', ')} (${checkedIds.size} módulos ativos no total)`;
    }
  }
}

async function handleSaveMachineConfigSubmit(e) {
  e.preventDefault();
  const hostname = mcInputHostname.value.trim();
  if (!hostname) return;

  const groupId = mcSelectGroup.value;
  const activeGrp = currentGroups.find(g => g.id === groupId);
  const grpTests = (activeGrp && activeGrp.profile && activeGrp.profile.tests) ? activeGrp.profile.tests : [];
  const grpTestIds = new Set(grpTests.map(t => t.id));

  const allModules = getAllAvailableModules();
  const checkedBoxes = mcModulesCheckboxesList.querySelectorAll('.mc-module-cb:checked');
  const selectedIds = new Set(Array.from(checkedBoxes).map(cb => cb.getAttribute('data-module-id')));

  const added = Array.from(selectedIds).filter(id => !grpTestIds.has(id));
  const removed = Array.from(grpTestIds).filter(id => !selectedIds.has(id));
  const isCustom = (added.length > 0 || removed.length > 0);

  let customTests = [];
  if (isCustom) {
    customTests = allModules.filter(m => selectedIds.has(m.id));
  }

  const btnSubmit = document.getElementById('btn-save-machine-config');
  if (btnSubmit) { btnSubmit.disabled = true; btnSubmit.innerHTML = '<span>Salvando...</span>'; }

  try {
    const res = await apiFetch(`/api/v1/admin/machines/${encodeURIComponent(hostname)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        group_id: groupId,
        is_custom: isCustom,
        custom_tests: customTests
      })
    });

    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao atualizar máquina');

    showToast(`Configuração da máquina "${hostname}" salva com sucesso!`, 'toast-success');
    closeMachineConfigModal();
    await loadAdminMachines();
  } catch (err) {
    showToast(`Erro ao salvar: ${err.message}`, 'toast-error');
  } finally {
    if (btnSubmit) { btnSubmit.disabled = false; btnSubmit.innerHTML = '<span>Salvar Alterações</span>'; }
  }
}

/* ==========================================================================
   13. Catálogo de Módulos & Upload de Arquivos JSON
   ========================================================================== */

async function loadModulesCatalog() {
  if (!modulesCardsGrid) return;
  modulesCardsGrid.innerHTML = '<div class="module-skeleton-loader">Carregando catálogo de módulos...</div>';

  try {
    const res = await apiFetch('/api/v1/admin/modules');
    if (!res.ok) throw new Error('Falha ao carregar catálogo');
    const modules = await res.json() || [];
    allCatalogModules = modules;

    if (modulesCountTag) {
      modulesCountTag.textContent = `${modules.length} módulo${modules.length !== 1 ? 's' : ''}`;
    }

    renderModulesCards(modules);
  } catch (err) {
    modulesCardsGrid.innerHTML = `<div class="text-rose text-center py-4">Erro ao carregar módulos: ${escapeHtml(err.message)}</div>`;
  }
}
window.loadModulesCatalog = loadModulesCatalog;

function renderModulesCards(modules) {
  if (!modulesCardsGrid) return;
  if (modules.length === 0) {
    modulesCardsGrid.innerHTML = '<div class="text-subtle text-center py-4 col-span-full">Nenhum módulo cadastrado. Importe um arquivo .json para começar.</div>';
    return;
  }

  modulesCardsGrid.innerHTML = modules.map(mod => {
    let targetInfo = '--';
    if (mod.target) targetInfo = `Alvo: ${mod.target}`;
    else if (mod.domain) targetInfo = `Domínio: ${mod.domain}`;
    else if (mod.endpoint) targetInfo = `Endpoint: ${mod.endpoint}`;
    else if (mod.protocol) targetInfo = `Protocolo: ${mod.protocol}`;

    let extraInfo = '';
    if (mod.size_mb) extraInfo = `Tamanho: ${mod.size_mb} MB`;
    else if (mod.count) extraInfo = `Pacotes: ${mod.count}`;
    else if (mod.timeout_seconds) extraInfo = `Timeout: ${mod.timeout_seconds}s`;

    return `
      <div class="module-card">
        <div class="module-card-top">
          <div class="module-card-title">${escapeHtml(mod.name)}</div>
          <span class="module-type-badge ${escapeHtml(mod.type)}">${escapeHtml(mod.type)}</span>
        </div>
        <div class="module-card-id">${escapeHtml(mod.id)}</div>
        <div class="module-card-details">
          <div class="module-card-details-row">
            <span>Alvo / Destino:</span>
            <span class="font-semibold text-white">${escapeHtml(targetInfo)}</span>
          </div>
          ${extraInfo ? `
          <div class="module-card-details-row">
            <span>Parâmetro:</span>
            <span class="font-mono">${escapeHtml(extraInfo)}</span>
          </div>
          ` : ''}
          <div class="module-card-details-row">
            <span>Timeout:</span>
            <span class="font-mono">${mod.timeout_seconds || 15}s</span>
          </div>
        </div>
        <div class="module-card-footer">
          <button type="button" class="btn btn-glass btn-sm" onclick="openEditModuleModal('${escapeHtml(mod.id)}')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9"></path><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"></path></svg>
            <span>Editar</span>
          </button>
          <button type="button" class="btn btn-glass btn-sm" onclick="downloadModuleJson('${escapeHtml(mod.id)}')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
            <span>Baixar .json</span>
          </button>
        </div>
      </div>
    `;
  }).join('');
}

let currentEditingModuleId = null;

function openEditModuleModal(moduleId) {
  const mod = allCatalogModules.find(m => m.id === moduleId);
  if (!mod || !moduleEditModal) return;

  currentEditingModuleId = moduleId;
  if (modEditErrorAlert) modEditErrorAlert.style.display = 'none';

  if (modEditId) modEditId.value = mod.id;
  if (modEditType) modEditType.value = mod.type || '';
  if (modEditName) modEditName.value = mod.name || mod.id;
  if (modEditTimeout) modEditTimeout.value = mod.timeout_seconds || 15;
  if (modEditEnabled) modEditEnabled.checked = mod.enabled !== false;

  const type = (mod.type || '').toLowerCase();

  if (type === 'ping' || type === 'packet_loss') {
    if (groupEditTarget) groupEditTarget.style.display = 'block';
    if (labelEditTarget) labelEditTarget.textContent = 'Alvo / IP do Destino';
    if (hintEditTarget) hintEditTarget.textContent = 'Endereço IP (ex: 1.1.1.1, auto_gateway) ou hostname';
    if (modEditTarget) modEditTarget.value = mod.target || '';

    if (groupEditCount) groupEditCount.style.display = 'block';
    if (modEditCount) modEditCount.value = mod.count || (type === 'packet_loss' ? 10 : 4);

    if (groupEditSizeMb) groupEditSizeMb.style.display = 'none';
    if (groupEditProtocol) groupEditProtocol.style.display = 'none';
  } else if (type === 'throughput_download' || type === 'throughput_upload') {
    if (groupEditTarget) groupEditTarget.style.display = 'block';
    if (labelEditTarget) labelEditTarget.textContent = 'Endpoint da API';
    if (hintEditTarget) hintEditTarget.textContent = 'Caminho HTTP do teste (ex: /api/v1/speedtest/download)';
    if (modEditTarget) modEditTarget.value = mod.endpoint || mod.target || '';

    if (groupEditCount) groupEditCount.style.display = 'none';
    if (groupEditSizeMb) groupEditSizeMb.style.display = 'block';
    if (modEditSizeMb) modEditSizeMb.value = mod.size_mb || 5;
    if (groupEditProtocol) groupEditProtocol.style.display = 'none';
  } else if (type === 'dns_lookup') {
    if (groupEditTarget) groupEditTarget.style.display = 'block';
    if (labelEditTarget) labelEditTarget.textContent = 'Domínio para Resolução';
    if (hintEditTarget) hintEditTarget.textContent = 'FQDN (ex: uepg.br, google.com)';
    if (modEditTarget) modEditTarget.value = mod.domain || mod.target || '';

    if (groupEditCount) groupEditCount.style.display = 'none';
    if (groupEditSizeMb) groupEditSizeMb.style.display = 'none';
    if (groupEditProtocol) groupEditProtocol.style.display = 'none';
  } else if (type === 'http_check') {
    if (groupEditTarget) groupEditTarget.style.display = 'block';
    if (labelEditTarget) labelEditTarget.textContent = 'URL Completa de Destino';
    if (hintEditTarget) hintEditTarget.textContent = 'URL HTTP/HTTPS (ex: https://portal.uepg.br)';
    if (modEditTarget) modEditTarget.value = mod.target || mod.url || '';

    if (groupEditCount) groupEditCount.style.display = 'none';
    if (groupEditSizeMb) groupEditSizeMb.style.display = 'none';
    if (groupEditProtocol) groupEditProtocol.style.display = 'none';
  } else if (type === 'traceroute') {
    if (groupEditTarget) groupEditTarget.style.display = 'block';
    if (labelEditTarget) labelEditTarget.textContent = 'Destino do Traceroute';
    if (hintEditTarget) hintEditTarget.textContent = 'Endereço IP ou hostname de destino';
    if (modEditTarget) modEditTarget.value = mod.target || '';

    if (groupEditCount) groupEditCount.style.display = 'block';
    if (modEditCount) modEditCount.value = mod.max_hops || mod.count || 15;
    if (groupEditSizeMb) groupEditSizeMb.style.display = 'none';
    if (groupEditProtocol) groupEditProtocol.style.display = 'none';
  } else if (type === 'discovery_protocol') {
    if (groupEditTarget) groupEditTarget.style.display = 'none';
    if (groupEditCount) groupEditCount.style.display = 'none';
    if (groupEditSizeMb) groupEditSizeMb.style.display = 'none';
    if (groupEditProtocol) groupEditProtocol.style.display = 'block';
    if (modEditProtocol) modEditProtocol.value = mod.protocol || 'LLDP_CDP';
  } else {
    if (groupEditTarget) groupEditTarget.style.display = 'block';
    if (labelEditTarget) labelEditTarget.textContent = 'Alvo / Destino';
    if (modEditTarget) modEditTarget.value = mod.target || mod.endpoint || mod.domain || '';
    if (groupEditCount) groupEditCount.style.display = 'none';
    if (groupEditSizeMb) groupEditSizeMb.style.display = 'none';
    if (groupEditProtocol) groupEditProtocol.style.display = 'none';
  }

  moduleEditModal.style.display = 'flex';
}
window.openEditModuleModal = openEditModuleModal;

function closeEditModuleModal() {
  if (moduleEditModal) moduleEditModal.style.display = 'none';
  currentEditingModuleId = null;
}
window.closeEditModuleModal = closeEditModuleModal;

async function handleSaveModuleEditSubmit(e) {
  e.preventDefault();
  if (!currentEditingModuleId) return;

  const originalMod = allCatalogModules.find(m => m.id === currentEditingModuleId);
  if (!originalMod) return;

  const type = (originalMod.type || '').toLowerCase();
  const name = modEditName ? modEditName.value.trim() : originalMod.name;
  const timeout = modEditTimeout ? (parseInt(modEditTimeout.value, 10) || 15) : 15;
  const enabled = modEditEnabled ? modEditEnabled.checked : true;

  const updatedMod = {
    ...originalMod,
    id: currentEditingModuleId,
    type: originalMod.type,
    name: name || currentEditingModuleId,
    timeout_seconds: timeout,
    enabled: enabled
  };

  if (modEditTarget && groupEditTarget && groupEditTarget.style.display !== 'none') {
    const val = modEditTarget.value.trim();
    if (type === 'dns_lookup') {
      updatedMod.domain = val;
    } else if (type === 'throughput_download' || type === 'throughput_upload') {
      updatedMod.endpoint = val;
    } else {
      updatedMod.target = val;
    }
  }

  if (modEditCount && groupEditCount && groupEditCount.style.display !== 'none') {
    const countVal = parseInt(modEditCount.value, 10) || 4;
    if (type === 'traceroute') {
      updatedMod.max_hops = countVal;
    } else {
      updatedMod.count = countVal;
    }
  }

  if (modEditSizeMb && groupEditSizeMb && groupEditSizeMb.style.display !== 'none') {
    updatedMod.size_mb = parseInt(modEditSizeMb.value, 10) || 5;
  }

  if (modEditProtocol && groupEditProtocol && groupEditProtocol.style.display !== 'none') {
    updatedMod.protocol = modEditProtocol.value.trim() || 'LLDP_CDP';
  }

  try {
    const res = await apiFetch(`/api/v1/admin/modules/${encodeURIComponent(currentEditingModuleId)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(updatedMod)
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao atualizar módulo');

    showToast(`Módulo "${updatedMod.name}" atualizado com sucesso!`, 'toast-success');
    closeEditModuleModal();
    await loadModulesCatalog();
  } catch (err) {
    if (modEditErrorAlert) {
      modEditErrorAlert.style.display = 'flex';
      modEditErrorText.textContent = err.message || 'Erro ao atualizar módulo';
    }
  }
}

function downloadModuleJson(moduleId) {
  const mod = allCatalogModules.find(m => m.id === moduleId);
  if (!mod) return;
  const jsonStr = JSON.stringify(mod, null, 2);
  const blob = new Blob([jsonStr], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${mod.id}.json`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}
window.downloadModuleJson = downloadModuleJson;

async function handleDeleteModule(moduleId) {
  showToast('A exclusão de módulos não é permitida pelo frontend do sistema.', 'toast-warning');
}
window.handleDeleteModule = handleDeleteModule;


function openModuleUploadModal() {
  if (!moduleUploadModal) return;
  if (modJsonErrorAlert) modJsonErrorAlert.style.display = 'none';
  if (modJsonTextarea) modJsonTextarea.value = '';
  moduleUploadModal.style.display = 'flex';
}

function closeModuleUploadModal() {
  if (moduleUploadModal) moduleUploadModal.style.display = 'none';
}

function handleFileSelect(e) {
  if (e.target.files && e.target.files.length > 0) {
    processUploadedJsonFile(e.target.files[0]);
  }
}

function processUploadedJsonFile(file) {
  if (!file) return;
  const reader = new FileReader();
  reader.onload = (event) => {
    try {
      const content = event.target.result;
      const parsed = JSON.parse(content);
      if (modJsonTextarea) {
        modJsonTextarea.value = JSON.stringify(parsed, null, 2);
      }
      if (modJsonErrorAlert) modJsonErrorAlert.style.display = 'none';
      showToast(`Arquivo "${file.name}" carregado. Revise e confirme para salvar.`, 'toast-info');
    } catch (err) {
      if (modJsonErrorText) modJsonErrorText.textContent = `Erro ao processar JSON: ${err.message}`;
      if (modJsonErrorAlert) modJsonErrorAlert.style.display = 'flex';
    }
  };
  reader.readAsText(file);
}

async function handleSaveModuleUploadSubmit() {
  if (!modJsonTextarea) return;
  const raw = modJsonTextarea.value.trim();
  if (!raw) {
    if (modJsonErrorText) modJsonErrorText.textContent = 'Cole ou selecione um arquivo JSON válido.';
    if (modJsonErrorAlert) modJsonErrorAlert.style.display = 'flex';
    return;
  }

  let parsedMod;
  try {
    parsedMod = JSON.parse(raw);
  } catch (e) {
    if (modJsonErrorText) modJsonErrorText.textContent = 'JSON malformado: ' + e.message;
    if (modJsonErrorAlert) modJsonErrorAlert.style.display = 'flex';
    return;
  }

  if (!parsedMod.id || !parsedMod.type) {
    if (modJsonErrorText) modJsonErrorText.textContent = 'Os campos "id" e "type" são obrigatórios no JSON do módulo.';
    if (modJsonErrorAlert) modJsonErrorAlert.style.display = 'flex';
    return;
  }

  if (btnSubmitUploadMod) { btnSubmitUploadMod.disabled = true; btnSubmitUploadMod.innerHTML = '<span>Salvando...</span>'; }

  try {
    const res = await apiFetch('/api/v1/admin/modules', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(parsedMod)
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao salvar módulo');

    showToast(`Módulo "${parsedMod.id}" salvo com sucesso no catálogo!`, 'toast-success');
    closeModuleUploadModal();
    await loadModulesCatalog();
  } catch (err) {
    if (modJsonErrorText) modJsonErrorText.textContent = err.message;
    if (modJsonErrorAlert) modJsonErrorAlert.style.display = 'flex';
  } finally {
    if (btnSubmitUploadMod) { btnSubmitUploadMod.disabled = false; btnSubmitUploadMod.innerHTML = '<span>Salvar no Catálogo</span>'; }
  }
}

/* ==========================================================================
   14. Navegação para Dashboard da Máquina & Diagnóstico de Ping ICMP
   ========================================================================== */

function navigateToHomeDashboard() {
  selectedChartNode = 'all';
  selectedChartGroup = 'all';
  if (selectNode) selectNode.value = 'all';
  const chartGrpSelect = document.getElementById('chart-group-filter-select');
  if (chartGrpSelect) chartGrpSelect.value = 'all';
  if (machinesSearchInput) machinesSearchInput.value = '';
  const nodesSearch = document.getElementById('nodes-search-input');
  if (nodesSearch) nodesSearch.value = '';
  switchView('dashboard');
  updateChartFocusUI();
  updateMetricToggleVisibility();
  loadDashboardData();
  showToast('Exibindo visão consolidada de todas as sondas', 'toast-info');
}
window.navigateToHomeDashboard = navigateToHomeDashboard;

function navigateToMachineDashboard(hostname) {
  if (!hostname) return;
  // Alterna para a visualização do Dashboard
  switchView('dashboard');

  // Foca imediatamente na máquina selecionada e rola até o cabeçalho
  selectNodeForCharts(hostname, true);
}
window.navigateToMachineDashboard = navigateToMachineDashboard;

function openPingModal(hostname, ipAddress) {
  const modal = document.getElementById('modal-ping-diagnostic');
  if (!modal) return;

  // Se não foi passado IP, tenta buscar nos resumos ou lista de máquinas
  let targetIp = (ipAddress || '').trim();
  if (!targetIp && hostname) {
    const summary = currentSummaries.find(s => s.node_id === hostname);
    if (summary && summary.system_info && summary.system_info.ip_address) {
      targetIp = summary.system_info.ip_address;
    } else {
      const m = allAdminMachines.find(item => item.hostname === hostname);
      if (m && m.ip_address) targetIp = m.ip_address;
    }
  }

  currentPingTarget = {
    hostname: hostname || '',
    ip: targetIp || hostname || ''
  };

  const hostEl = document.getElementById('ping-modal-hostname');
  const ipEl = document.getElementById('ping-modal-ip');
  if (hostEl) hostEl.textContent = hostname || '--';
  if (ipEl) ipEl.textContent = targetIp || hostname || '--';

  modal.style.display = 'flex';
  executeAdminPing(currentPingTarget.hostname, currentPingTarget.ip);
}
window.openPingModal = openPingModal;

function closePingModal() {
  const modal = document.getElementById('modal-ping-diagnostic');
  if (modal) modal.style.display = 'none';
}
window.closePingModal = closePingModal;

async function executeAdminPing(hostname, target) {
  const loadingEl = document.getElementById('ping-loading-state');
  const resultEl = document.getElementById('ping-result-content');
  const retestBtn = document.getElementById('btn-retest-ping');
  const terminalOutput = document.getElementById('ping-terminal-output');
  const terminalTime = document.getElementById('ping-terminal-time');

  if (loadingEl) loadingEl.style.display = 'block';
  if (resultEl) resultEl.style.display = 'none';
  if (retestBtn) retestBtn.disabled = true;

  try {
    const res = await apiFetch('/api/v1/admin/ping', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        hostname: hostname || target,
        target: target || hostname
      })
    });

    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || 'Falha ao executar teste de ping');
    }

    // Atualiza estatísticas do Modal
    const statStatus = document.getElementById('ping-stat-status');
    const statSub = document.getElementById('ping-stat-sub');
    const statAvg = document.getElementById('ping-stat-avg');
    const statMin = document.getElementById('ping-stat-min');
    const statMax = document.getElementById('ping-stat-max');
    const statLoss = document.getElementById('ping-stat-loss');
    const statLossDesc = document.getElementById('ping-stat-loss-desc');

    const isAlive = (data.is_alive !== undefined) ? !!data.is_alive : !!data.success;
    if (statStatus) {
      statStatus.textContent = isAlive ? 'Conectado' : 'Sem Resposta';
      statStatus.className = `dc-health-badge ${isAlive ? 'online' : 'stale'}`;
    }

    const loss = typeof data.packet_loss === 'number' ? data.packet_loss :
                 typeof data.packet_loss_percent === 'number' ? data.packet_loss_percent :
                 (isAlive ? 0 : 100);
    const packetsRecv = typeof data.packets_received === 'number' ? data.packets_received : Math.round(4 * (1 - loss / 100));
    const packetsTotal = typeof data.packets_sent === 'number' ? data.packets_sent : 4;
    if (statSub) {
      statSub.textContent = `${packetsRecv} de ${packetsTotal} pacotes recebidos`;
    }

    const avgMs = typeof data.avg_ms === 'number' ? data.avg_ms : data.avg_latency_ms;
    const minMs = typeof data.min_ms === 'number' ? data.min_ms : data.min_latency_ms;
    const maxMs = typeof data.max_ms === 'number' ? data.max_ms : data.max_latency_ms;

    if (statAvg) statAvg.textContent = (avgMs > 0 ? avgMs.toFixed(1) : '--');
    if (statMin) statMin.textContent = (minMs > 0 ? minMs.toFixed(1) : '--');
    if (statMax) statMax.textContent = (maxMs > 0 ? maxMs.toFixed(1) : '--');

    if (statLoss) statLoss.textContent = loss.toFixed(0);
    if (statLossDesc) {
      if (loss === 0) statLossDesc.textContent = 'Sem perda de pacotes (100% íntegro)';
      else if (loss === 100) statLossDesc.textContent = '100% de perda (Host inacessível)';
      else statLossDesc.textContent = `${loss.toFixed(0)}% de perda intermitente`;
    }

    if (terminalTime) {
      terminalTime.textContent = new Date().toLocaleTimeString('pt-BR');
    }
    if (terminalOutput) {
      terminalOutput.textContent = data.raw_output || 'Sem saída de comando retornada.';
    }

    if (loadingEl) loadingEl.style.display = 'none';
    if (resultEl) resultEl.style.display = 'block';
  } catch (err) {
    if (loadingEl) loadingEl.style.display = 'none';
    if (resultEl) resultEl.style.display = 'block';
    if (terminalOutput) {
      terminalOutput.textContent = `Erro ao disparar ping: ${err.message}\nCertifique-se de que o Central Server tem privilégios de rede ICMP.`;
    }
    const statStatus = document.getElementById('ping-stat-status');
    if (statStatus) {
      statStatus.textContent = 'Erro de Execução';
      statStatus.className = 'dc-health-badge stale';
    }
    showToast(`Erro no diagnóstico de ping: ${err.message}`, 'toast-error');
  } finally {
    if (retestBtn) retestBtn.disabled = false;
  }
}
window.executeAdminPing = executeAdminPing;

function triggerSingleMachinePing() {
  if (!selectedChartNode || selectedChartNode === 'all') {
    showToast('Selecione uma máquina específica no Dashboard para disparar o ping.', 'toast-warning');
    return;
  }
  openPingModal(selectedChartNode, '');
}
window.triggerSingleMachinePing = triggerSingleMachinePing;




/* ==========================================================================
   PAINEL TÉCNICO EMBUTIDO & DIRETRIZES DE UX/UI (MÁQUINA ÚNICA)
   ========================================================================== */

function renderSingleMachineDetailsPanel(summary, diagnostic) {
  const panel = document.getElementById('single-machine-details-panel');
  if (!panel) return;

  if (!selectedChartNode || selectedChartNode === 'all') {
    panel.style.display = 'none';
    return;
  }
  panel.style.display = 'block';

  const nodeName = selectedChartNode;
  const sys = (diagnostic && diagnostic.system_info) || (summary && summary.system_info) || {};
  const modResults = (diagnostic && diagnostic.modular_results) || (summary && summary.modular_results) || {};
  const swInfo = (diagnostic && diagnostic.switch_info) || (summary && summary.switch_info) || {};

  // 1. Status Wi-Fi & Potência
  const wifiBadge = document.getElementById('sm-panel-wifi-status-badge');
  const wifiContent = document.getElementById('sm-panel-wifi-content');
  const wf = modResults.wifi;

  if (wifiContent) {
    if (wf && (wf.status && wf.status !== 'skipped' || wf.has_wifi_adapter)) {
      if (wf.connected) {
        const pct = Math.max(0, Math.min(100, wf.signal_percent || wf.signal_quality_percent || 0));
        const dbmStr = wf.signal_dbm ? (wf.signal_dbm + " dBm") : (wf.signal_strength_dbm ? (wf.signal_strength_dbm + " dBm") : '');
        let barClass = 'wifi-signal-bar-fill';
        if (pct < 40) barClass += ' poor';
        else if (pct < 70) barClass += ' warning';

        if (wifiBadge) {
          wifiBadge.className = 'badge-pill badge-primary-route';
          wifiBadge.textContent = 'Wi-Fi Conectado';
        }

        wifiContent.innerHTML = 
          '<div class="wifi-meter-box" style="margin-bottom: 10px;">' +
            '<div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:4px;">' +
              '<span class="font-bold text-white">' + escapeHtml(wf.ssid || 'Rede Wi-Fi') + '</span>' +
              '<span class="font-mono font-bold text-cyan">' + pct + '% ' + (dbmStr ? ('(' + dbmStr + ')') : '') + '</span>' +
            '</div>' +
            '<div class="wifi-signal-bar-track" style="height: 8px; border-radius: 4px; background: rgba(255,255,255,0.08); overflow:hidden;">' +
              '<div class="' + barClass + '" style="width: ' + pct + '%; height: 100%; transition: width 0.4s ease;"></div>' +
            '</div>' +
          '</div>' +
          '<div class="sm-wifi-stats-row">' +
            '<div class="sm-wifi-stat-item">' +
              '<span class="sm-wifi-stat-label">BSSID / AP:</span>' +
              '<span class="sm-wifi-stat-val font-mono">' + escapeHtml(wf.bssid || '--') + '</span>' +
            '</div>' +
            '<div class="sm-wifi-stat-item">' +
              '<span class="sm-wifi-stat-label">Canal & Frequência:</span>' +
              '<span class="sm-wifi-stat-val">' + (wf.channel ? ('Canal ' + wf.channel) : '--') + (wf.band ? (' (' + wf.band + ')') : '') + '</span>' +
            '</div>' +
            '<div class="sm-wifi-stat-item">' +
              '<span class="sm-wifi-stat-label">Rádio:</span>' +
              '<span class="sm-wifi-stat-val">' + escapeHtml(wf.radio_type || '802.11') + '</span>' +
            '</div>' +
            '<div class="sm-wifi-stat-item">' +
              '<span class="sm-wifi-stat-label">Taxa Tx/Rx:</span>' +
              '<span class="sm-wifi-stat-val font-mono">' + (wf.tx_rate_mbps ? (wf.tx_rate_mbps + ' / ' + (wf.rx_rate_mbps || wf.tx_rate_mbps) + ' Mbps') : '--') + '</span>' +
            '</div>' +
          '</div>';
      } else {
        if (wifiBadge) {
          wifiBadge.className = 'badge-pill status-iface-inactive';
          wifiBadge.textContent = 'Wi-Fi Desconectado';
        }
        wifiContent.innerHTML = 
          '<div class="sm-wifi-ethernet-notice" style="border-color: rgba(245, 158, 11, 0.3); color: #f59e0b;">' +
            '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">' +
              '<path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>' +
              '<line x1="12" y1="9" x2="12" y2="13"></line>' +
              '<line x1="12" y1="17" x2="12.01" y2="17"></line>' +
            '</svg>' +
            '<div>' +
              '<strong>Adaptador Wi-Fi detectado (desconectado)</strong>' +
              '<div class="text-xs text-muted">A placa wireless do computador está inativa ou fora do alcance de redes conhecidas.</div>' +
            '</div>' +
          '</div>';
      }
    } else {
      if (wifiBadge) {
        wifiBadge.className = 'badge-pill';
        wifiBadge.textContent = 'Cabo Ethernet';
      }
      wifiContent.innerHTML = 
        '<div class="sm-wifi-ethernet-notice">' +
          '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">' +
            '<rect x="2" y="2" width="20" height="8" rx="2" ry="2"></rect>' +
            '<rect x="2" y="14" width="20" height="8" rx="2" ry="2"></rect>' +
            '<line x1="6" y1="6" x2="6.01" y2="6"></line>' +
            '<line x1="6" y1="18" x2="6.01" y2="18"></line>' +
          '</svg>' +
          '<div>' +
            '<strong>Conexão Cabeada Exclusiva (Ethernet)</strong>' +
            '<div class="text-xs text-muted">Esta sonda opera em enlace de par trançado. Adaptador Wi-Fi ausente ou desativado.</div>' +
          '</div>' +
        '</div>';
    }
  }

  // 2. Módulos Carregados na Sonda
  const modulesListEl = document.getElementById('sm-panel-modules-list');
  const profileBadgeEl = document.getElementById('sm-panel-modules-profile-badge');
  const machAdmin = (allAdminMachines || []).find(m => m.hostname === nodeName);

  if (profileBadgeEl) {
    if (machAdmin && machAdmin.custom_tests && machAdmin.custom_tests.length > 0) {
      profileBadgeEl.textContent = 'Customizado (+' + machAdmin.custom_tests.length + ')';
      profileBadgeEl.className = 'badge-pill warning';
    } else {
      profileBadgeEl.textContent = 'Padrão do Grupo';
      profileBadgeEl.className = 'badge-pill success';
    }
  }

  if (modulesListEl) {
    const activeModules = [
      { id: 'throughput_download', name: 'Throughput Download' },
      { id: 'throughput_upload', name: 'Throughput Upload' },
      { id: 'ping_gateway', name: 'Ping Gateway Local' },
      { id: 'ping_nti_core', name: 'Ping Core NTI (Anel)' },
      { id: 'packet_loss_gateway', name: 'Perda de Pacotes Gateway' },
      { id: 'switch_discovery', name: 'Descoberta Switch LLDP' },
      { id: 'wifi_status', name: 'Telemetria Wi-Fi & RSSI' },
      { id: 'dns_uepg', name: 'Resolução DNS UEPG' }
    ];

    modulesListEl.innerHTML = activeModules.map(mod => 
      '<div class="sm-module-chip active" title="Módulo ativo no ciclo agendado da sonda">' +
        '<span class="sm-module-dot"></span>' +
        '<span>' + escapeHtml(mod.name) + '</span>' +
      '</div>'
    ).join('');
  }

  // 3. Switch LLDP & Conexão de Rede
  const switchContentEl = document.getElementById('sm-panel-switch-content');
  const switchVlanBadge = document.getElementById('sm-panel-switch-vlan');
  const swName = swInfo.switch_name || '--';
  const swPort = swInfo.switch_port || '--';
  const vlanNum = swInfo.vlan || (summary && summary.vlan) || (diagnostic && diagnostic.vlan) || 0;

  if (switchVlanBadge) {
    switchVlanBadge.textContent = vlanNum > 0 ? ('VLAN ' + vlanNum) : 'Sem VLAN';
  }

  if (switchContentEl) {
    const hasSwitch = (swName !== '--' && swName !== '' && swName !== '{}');
    switchContentEl.innerHTML = 
      '<div class="sm-switch-detail-row">' +
        '<span class="text-muted">Nome do Switch:</span>' +
        '<strong class="font-mono text-cyan">' + escapeHtml(swName) + '</strong>' +
      '</div>' +
      '<div class="sm-switch-detail-row">' +
        '<span class="text-muted">Porta de Enlace:</span>' +
        '<strong class="font-mono">' + escapeHtml(swPort) + '</strong>' +
      '</div>' +
      '<div class="sm-switch-detail-row">' +
        '<span class="text-muted">Protocolo de Descoberta:</span>' +
        '<span>' + escapeHtml(swInfo.protocol || 'LLDP / 802.1AB') + '</span>' +
      '</div>' +
      (hasSwitch ? (
        '<div class="sm-switch-action-btn-wrap">' +
          '<button type="button" class="btn btn-sm btn-outline-cyan w-full" onclick="openCurrentMachineSwitchModal()">' +
            '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-right:6px;">' +
              '<polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline>' +
            '</svg>' +
            'Ver Vizinhos Conectados neste Switch' +
          '</button>' +
        '</div>'
      ) : '<div class="text-xs text-muted mt-2">Nenhum pacote LLDP recente detectado nesta porta.</div>');
  }

  // 4. Inventário Completo de Placas de Rede & MACs
  const ifacesTbody = document.getElementById('sm-panel-ifaces-body');
  const ifacesCount = document.getElementById('sm-panel-ifaces-count');
  const ifacesList = sys.interfaces || sys.Interfaces || [];

  if (ifacesTbody) {
    if (ifacesList.length > 0) {
      if (ifacesCount) ifacesCount.textContent = ifacesList.length + ' placa(s) detectada(s)';
      ifacesTbody.innerHTML = ifacesList.map(iface => {
        const isUp = !!iface.is_up;
        const isPrimary = !!iface.is_primary;
        const ifaceType = (iface.type || 'ethernet').toLowerCase();
        let badgeTypeClass = 'badge-iface-eth';
        let typeLabel = 'Ethernet';
        if (ifaceType === 'wifi' || ifaceType === 'wi-fi') {
          badgeTypeClass = 'badge-iface-wifi';
          typeLabel = 'Wi-Fi';
        } else if (ifaceType === 'virtual') {
          badgeTypeClass = 'badge-iface-virtual';
          typeLabel = 'Virtual';
        }

        const ipsStr = (iface.ip_addresses && iface.ip_addresses.length > 0)
          ? iface.ip_addresses.join('<br>')
          : '<span class="iface-no-ip">Sem IP atribuído</span>';

        return '<tr>' +
          '<td>' +
            '<span class="' + (isUp ? 'status-dot-up' : 'status-dot-down') + '"></span>' +
            '<span class="' + (isUp ? 'status-iface-active' : 'status-iface-inactive') + ' font-bold">' + (isUp ? 'Ativa' : 'Inativa') + '</span>' +
          '</td>' +
          '<td>' +
            '<div class="iface-name font-bold">' + escapeHtml(iface.name || 'Interface') + '</div>' +
            (iface.description ? ('<div class="iface-desc text-xs text-muted">' + escapeHtml(iface.description) + '</div>') : '') +
          '</td>' +
          '<td>' +
            '<span class="badge-iface-type ' + badgeTypeClass + '">' + typeLabel + '</span>' +
          '</td>' +
          '<td>' +
            '<span class="font-mono iface-mac font-bold">' + escapeHtml(iface.mac_address || '--') + '</span>' +
          '</td>' +
          '<td>' +
            '<span class="font-mono ' + (isUp ? 'text-cyan font-bold' : '') + '">' + ipsStr + '</span>' +
          '</td>' +
          '<td style="text-align: center;">' +
            (isPrimary
              ? '<span class="badge-primary-route font-bold">Principal (Gateway)</span>'
              : '<span class="iface-role-secondary text-xs text-muted">Secundária</span>') +
          '</td>' +
        '</tr>';
      }).join('');
    } else {
      const fallbackMac = sys.mac_address || sys.MACAddress || '--';
      const fallbackIp = sys.ip_address || sys.IPAddress || '--';
      const fallbackName = sys.network_adapter || sys.NetworkAdapter || 'Interface Ativa Padrão';
      if (ifacesCount) ifacesCount.textContent = '1 placa detectada';
      ifacesTbody.innerHTML = 
        '<tr>' +
          '<td>' +
            '<span class="status-dot-up"></span>' +
            '<span class="status-iface-active font-bold">Ativa</span>' +
          '</td>' +
          '<td>' +
            '<div class="iface-name font-bold">' + escapeHtml(fallbackName) + '</div>' +
          '</td>' +
          '<td>' +
            '<span class="badge-iface-type badge-iface-eth">Ethernet</span>' +
          '</td>' +
          '<td>' +
            '<span class="font-mono iface-mac font-bold">' + escapeHtml(fallbackMac) + '</span>' +
          '</td>' +
          '<td>' +
            '<span class="font-mono text-cyan font-bold">' + escapeHtml(fallbackIp) + '</span>' +
          '</td>' +
          '<td style="text-align: center;">' +
            '<span class="badge-primary-route font-bold">Principal (Gateway)</span>' +
          '</td>' +
        '</tr>';
    }
  }
}
window.renderSingleMachineDetailsPanel = renderSingleMachineDetailsPanel;

/* ==========================================================================
   MODAL DE MÁQUINAS DO SWITCH LLDP & COMPARAÇÃO
   ========================================================================== */

let currentSwitchNameForModal = '';

function openSwitchMachinesModal(switchName) {
  if (!switchName || switchName === '--') {
    const summary = currentSummaries.find(s => s.node_id === selectedChartNode);
    if (summary && summary.switch_info && summary.switch_info.switch_name) {
      switchName = summary.switch_info.switch_name;
    }
  }
  if (!switchName || switchName === '--') {
    showToast('Nenhum switch LLDP identificado para esta máquina', 'toast-warning');
    return;
  }

  currentSwitchNameForModal = switchName;
  const modal = document.getElementById('modal-switch-machines');
  const titleEl = document.getElementById('modal-switch-title');
  const pillEl = document.getElementById('modal-switch-name-pill');
  const totalCountEl = document.getElementById('modal-switch-total-count');
  const tbody = document.getElementById('modal-switch-table-body');

  if (titleEl) titleEl.textContent = 'Switch LLDP: ' + switchName;
  if (pillEl) pillEl.textContent = switchName;

  const matchedSummaries = (currentSummaries || []).filter(s => {
    return s.switch_info && s.switch_info.switch_name && s.switch_info.switch_name.toLowerCase() === switchName.toLowerCase();
  });

  if (totalCountEl) totalCountEl.textContent = matchedSummaries.length + ' máquina(s) conectada(s)';

  if (tbody) {
    if (matchedSummaries.length === 0) {
      tbody.innerHTML = '<tr><td colspan="7" class="text-center text-muted py-4">Nenhuma outra máquina reportou conexão neste switch no ciclo recente.</td></tr>';
    } else {
      tbody.innerHTML = matchedSummaries.map(s => {
        const isSelected = (s.node_id === selectedChartNode);
        const portStr = (s.switch_info && s.switch_info.switch_port) || '--';
        const ipStr = (s.system_info && (s.system_info.ip_address || s.system_info.IPAddress)) || '--';
        const macStr = (s.system_info && (s.system_info.mac_address || s.system_info.MACAddress)) || '--';
        const isOnline = !!s.is_healthy;

        return '<tr style="' + (isSelected ? 'background: rgba(6, 182, 212, 0.08);' : '') + '">' +
          '<td>' +
            '<span class="dc-health-badge ' + (isOnline ? 'online' : 'stale') + '">' + (isOnline ? 'Online' : 'Sem sinal') + '</span>' +
          '</td>' +
          '<td>' +
            '<strong class="font-mono text-white">' + escapeHtml(s.node_id) + '</strong>' +
            (isSelected ? '<span class="badge-pill ml-1" style="font-size:0.65rem;">(Atual)</span>' : '') +
          '</td>' +
          '<td>' +
            '<span class="font-mono text-cyan font-bold">' + escapeHtml(portStr) + '</span>' +
          '</td>' +
          '<td>' +
            '<div class="font-mono">' + escapeHtml(ipStr) + '</div>' +
            '<div class="font-mono text-xs text-muted">' + escapeHtml(macStr) + '</div>' +
          '</td>' +
          '<td style="text-align: right;" class="font-mono">' + (s.last_download ? (s.last_download.toFixed(1) + ' Mbps') : '--') + '</td>' +
          '<td style="text-align: right;" class="font-mono">' + (s.last_ping ? (s.last_ping.toFixed(1) + ' ms') : '--') + '</td>' +
          '<td style="text-align: center;">' +
            '<button type="button" class="btn btn-sm btn-glass" data-node="' + escapeHtml(s.node_id) + '" onclick="handleSwitchRowFocus(this)">' +
              'Focar Sonda' +
            '</button>' +
          '</td>' +
        '</tr>';
      }).join('');
    }
  }

  if (modal) modal.style.display = 'flex';
}
window.openSwitchMachinesModal = openSwitchMachinesModal;

function openCurrentMachineSwitchModal() {
  const summary = currentSummaries.find(s => s.node_id === selectedChartNode);
  const swName = (summary && summary.switch_info && summary.switch_info.switch_name) || '';
  openSwitchMachinesModal(swName);
}
window.openCurrentMachineSwitchModal = openCurrentMachineSwitchModal;

function closeSwitchMachinesModal() {
  const modal = document.getElementById('modal-switch-machines');
  if (modal) modal.style.display = 'none';
}
window.closeSwitchMachinesModal = closeSwitchMachinesModal;

function compareSwitchMachinesOnCharts() {
  closeSwitchMachinesModal();
  if (!currentSwitchNameForModal) return;

  const matchedNodes = (currentSummaries || [])
    .filter(s => s.switch_info && s.switch_info.switch_name && s.switch_info.switch_name.toLowerCase() === currentSwitchNameForModal.toLowerCase())
    .map(s => s.node_id);

  if (matchedNodes.length <= 1) {
    showToast('Apenas uma máquina conectada ao switch ' + currentSwitchNameForModal, 'toast-info');
    return;
  }

  showToast('Comparando ' + matchedNodes.length + ' máquinas do switch ' + currentSwitchNameForModal, 'toast-info');
  selectedChartNode = 'all';
  updateChartFocusUI();

  const nameEl = document.getElementById('chart-focus-name');
  if (nameEl) nameEl.textContent = 'Switch: ' + currentSwitchNameForModal + ' (' + matchedNodes.length + ' máquinas)';
  const resetBtn = document.getElementById('btn-reset-chart-focus');
  if (resetBtn) resetBtn.style.display = 'inline-flex';

  const switchSeries = (cachedMetricsSeries || []).filter(item => {
    return matchedNodes.includes(item.node_id || item.NodeID);
  });

  renderComparativeCharts(switchSeries, matchedNodes);
}
window.compareSwitchMachinesOnCharts = compareSwitchMachinesOnCharts;

function handleSwitchRowFocus(btn) {
  const node = btn ? btn.getAttribute('data-node') : null;
  if (node) selectNodeForCharts(node, true);
  closeSwitchMachinesModal();
}
window.handleSwitchRowFocus = handleSwitchRowFocus;
