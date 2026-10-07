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
let mmLatencyChart = null;
let mmThroughputChart = null;
let currentInspectedMachine = null;
const machineFilterState = {
  search: '',
  group: 'all',
  module: 'all',
  status: 'all'
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

  // Se não estiver logado, exibe a tela de login
  showLoginModal();
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

function showLoginModal() {
  currentUser = null;
  if (loginModalOverlay) loginModalOverlay.style.display = 'flex';
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
  showLoginModal();
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


  // Filtros em Tempo Real da Tabela de Nós
  if (nodesSearchInput) {
    nodesSearchInput.addEventListener('input', (e) => {
      nodeFilterState.search = e.target.value.toLowerCase().trim();
      applyNodeFilters();
    });
  }
  if (nodesStatusFilter) {
    nodesStatusFilter.addEventListener('change', (e) => {
      nodeFilterState.status = e.target.value;
      applyNodeFilters();
    });
  }
  if (nodesGroupFilter) {
    nodesGroupFilter.addEventListener('change', (e) => {
      nodeFilterState.group = e.target.value;
      applyNodeFilters();
    });
  }

  // Botão de Resetar Foco dos Gráficos
  const resetChartBtn = document.getElementById('btn-reset-chart-focus');
  if (resetChartBtn) {
    resetChartBtn.addEventListener('click', () => {
      selectedChartNode = 'all';
      updateChartFocusUI();
      updateMetricToggleVisibility();
      loadDashboardData();
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

      // 3. Clique em qualquer outro lugar da linha (ou nome do PC): Foca nos gráficos!
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
      applyMachineFilters();
    });
  }
  if (machinesGroupFilter) {
    machinesGroupFilter.addEventListener('change', (e) => {
      machineFilterState.group = e.target.value;
      applyMachineFilters();
    });
  }
  if (machinesModuleFilter) {
    machinesModuleFilter.addEventListener('change', (e) => {
      machineFilterState.module = e.target.value;
      applyMachineFilters();
    });
  }
  if (machinesStatusFilter) {
    machinesStatusFilter.addEventListener('change', (e) => {
      machineFilterState.status = e.target.value;
      applyMachineFilters();
    });
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
    const machinesInGroup = allAdminMachines.filter(m => m.group_id === g.id);

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

function openGroupDetailsModal(groupId, tab = 'machines') {
  const g = currentGroups.find(item => item.id === groupId);
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
  const machines = allAdminMachines.filter(m => m.group_id === g.id);

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
    const isOnline = isNodeOnline(m.last_seen);
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

async function loadDashboardData() {
  if (!currentUser) return;

  const selected = selectedChartNode || (selectNode && selectNode.value) || 'all';
  let url = '/api/v1/dashboard/metrics';
  if (selected !== 'all') {
    url += `?node_id=${encodeURIComponent(selected)}`;
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

    updateNodeSelector(data.nodes_list || []);
    renderSummaryCards(currentSummaries);
    renderComparativeCharts(data.metrics_series || [], data.nodes_list || []);
    renderTelemetryTable(data.metrics_series || []);
    updateChartFocusUI();

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

function selectNodeForCharts(nodeId) {
  if (selectedChartNode === nodeId) {
    // Se clicar na mesma máquina já selecionada, desseleciona e volta para consolidado
    selectedChartNode = 'all';
    showToast('Exibindo gráficos consolidados de todas as sondas', 'toast-info');
  } else {
    selectedChartNode = nodeId;
    showToast(`Gráficos focados na sonda: ${nodeId}`, 'toast-info');
  }

  updateChartFocusUI();
  updateMetricToggleVisibility();
  loadDashboardData();
}
window.selectNodeForCharts = selectNodeForCharts;

function updateChartFocusUI() {
  const nameEl = document.getElementById('chart-focus-name');
  const resetBtn = document.getElementById('btn-reset-chart-focus');
  const indicator = document.getElementById('chart-focus-indicator');

  if (selectedChartNode === 'all') {
    if (nameEl) nameEl.textContent = 'Todas as Sondas (Consolidado)';
    if (indicator) indicator.className = 'chart-focus-pill all';
    if (resetBtn) resetBtn.style.display = 'none';
  } else {
    if (nameEl) nameEl.textContent = selectedChartNode;
    if (indicator) indicator.className = 'chart-focus-pill focused';
    if (resetBtn) resetBtn.style.display = 'inline-flex';
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

/* ==========================================================================
   4. Tabela de Nós com Filtros em Tempo Real (Substituição dos Cards)
   ========================================================================== */

let nodeFilterState = {
  search: '',
  status: 'all',
  group: 'all'
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

    // 3. Filtro de Texto (hostname, local, ip, switch, vlan)
    if (query) {
      const matchNodeId = (s.node_id || '').toLowerCase().includes(query);
      const matchLoc = (s.location || '').toLowerCase().includes(query);
      const matchGroup = (s.group_name || '').toLowerCase().includes(query);
      const matchIp = (s.system_info && s.system_info.ip_address) ? s.system_info.ip_address.toLowerCase().includes(query) : false;
      const matchSwitch = (s.switch_info && s.switch_info.switch_name) ? s.switch_info.switch_name.toLowerCase().includes(query) : false;
      const matchVlan = s.vlan ? String(s.vlan).includes(query) : false;

      if (!matchNodeId && !matchLoc && !matchGroup && !matchIp && !matchSwitch && !matchVlan) {
        return false;
      }
    }

    return true;
  });

  if (nodesFilterCount) {
    nodesFilterCount.textContent = `${filtered.length} de ${currentSummaries.length}`;
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

  if (!summaries || summaries.length === 0) {
    if (totalEl) totalEl.textContent = '0';
    if (healthEl) { healthEl.textContent = '0% Online'; healthEl.className = 'kpi-badge warning'; }
    if (subEl) subEl.textContent = 'Nenhuma sonda detectada';
    if (downEl) downEl.textContent = '0.0';
    if (upEl) upEl.textContent = '0.0';
    if (pingEl) pingEl.textContent = '0.0';
    return;
  }

  const total = summaries.length;
  const onlineCount = summaries.filter(s => s.is_healthy).length;
  const healthPct = Math.round((onlineCount / total) * 100);

  if (totalEl) totalEl.textContent = total;
  if (healthEl) {
    healthEl.textContent = `${healthPct}% Online`;
    healthEl.className = `kpi-badge ${healthPct >= 80 ? 'success' : (healthPct >= 50 ? 'warning' : 'danger')}`;
  }
  if (subEl) subEl.textContent = `${onlineCount} de ${total} ativas no campus`;

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
            <span class="node-name-text">${escapeHtml(s.node_id)}</span>
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
        <td>
          <span class="text-subtle font-mono">${timeAgo}</span>
        </td>
        <td class="text-center">
          <div class="table-actions">
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
        <td>${item.download_mbps > 0 ? `<strong style="color: var(--accent-cyan);">${item.download_mbps.toFixed(1)}</strong> Mbps` : `<span class="text-subtle font-mono">--</span>`}</td>
        <td>${item.upload_mbps > 0 ? `<strong style="color: #c084fc;">${item.upload_mbps.toFixed(1)}</strong> Mbps` : `<span class="text-subtle font-mono">--</span>`}</td>
        <td><strong style="color: #34d399;">${item.ping_gateway.toFixed(1)}</strong> ms</td>
        <td><span class="dc-health-badge online" style="display:inline-flex;">● Sincronizado</span></td>
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
      // Fallback gracioso para usuários sem permissão de admin completa
      allAdminMachines = currentSummaries.map(s => ({
        hostname: s.node_id,
        ip_address: (s.system_info && s.system_info.ip_address) || '',
        mac_address: (s.system_info && s.system_info.mac_address) || '',
        switch_name: (s.switch_info && s.switch_info.switch_name) || '',
        switch_port: (s.switch_info && s.switch_info.switch_port) || '',
        vlan: (s.switch_info && s.switch_info.vlan) || '',
        last_seen: s.last_seen || '',
        group_id: s.group_id || 'acon',
        is_custom: false,
        custom_tests: []
      }));
    }

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

    // 4. Text Search
    if (query) {
      const matchHost = (m.hostname || '').toLowerCase().includes(query);
      const matchIp = (m.ip_address || '').toLowerCase().includes(query);
      const matchMac = (m.mac_address || '').toLowerCase().includes(query);
      const matchSwitch = (m.switch_name || '').toLowerCase().includes(query);
      const matchVlan = m.vlan ? String(m.vlan).includes(query) : false;
      const matchGroup = (m.group_id || '').toLowerCase().includes(query);
      if (!matchHost && !matchIp && !matchMac && !matchSwitch && !matchVlan && !matchGroup) {
        return false;
      }
    }

    return true;
  });

  if (machinesFilterCount) {
    machinesFilterCount.textContent = `${filtered.length} de ${allAdminMachines.length} máquina${allAdminMachines.length !== 1 ? 's' : ''}`;
  }

  renderMachinesTable(filtered);
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
    const isOnline = summary ? summary.is_healthy : (m.last_seen && (Date.now() - new Date(m.last_seen).getTime()) < 15 * 60 * 1000);
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
          ⚙️ Alterado (${diffText})
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
    const lastDate = m.last_seen ? new Date(m.last_seen) : (summary ? new Date(summary.last_seen) : null);
    const timeAgo = lastDate ? formatTimeAgo(lastDate) : '--';

    // Select de troca rápida de grupo (ou badge)
    let groupSelectHtml = `<span class="badge-group-tag">${escapeHtml(grpName)}</span>`;
    if (isAdmin) {
      groupSelectHtml = `
        <select class="form-input form-input-sm" style="min-width: 110px; padding: 2px 8px; font-size: 0.78rem;" onchange="handleQuickChangeGroup('${escapeHtml(m.hostname)}', this.value)">
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
            <span class="node-name-text font-bold">${escapeHtml(m.hostname)}</span>
            <span class="node-sub-ip">${escapeHtml(m.ip_address || '--')}</span>
          </div>
        </td>
        <td>
          <div style="font-size: 0.76rem; font-family: 'JetBrains Mono', monospace; line-height: 1.25;">
            <div>${escapeHtml(m.ip_address || '--')}</div>
            <div style="color: var(--text-muted); font-size: 0.70rem;">${escapeHtml(m.mac_address || '--')}</div>
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
        <td>${switchHtml}</td>
        <td><span class="text-subtle font-mono">${timeAgo}</span></td>
        <td style="text-align: right;">
          <div class="table-actions" style="justify-content: flex-end;">
            <button type="button" class="btn btn-glass btn-sm" onclick="openMachineMetricsModal('${escapeHtml(m.hostname)}')" title="Visualizar gráficos e estatísticas da máquina">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="20" x2="18" y2="10"></line><line x1="12" y1="20" x2="12" y2="4"></line><line x1="6" y1="20" x2="6" y2="14"></line></svg>
              <span>Gráficos</span>
            </button>
            <button type="button" class="btn btn-glass btn-sm" onclick="openMachineConfigModal('${escapeHtml(m.hostname)}')" title="Alterar grupo e módulos">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>
              <span>Editar</span>
            </button>
            ${isAdmin ? `
            <button type="button" class="btn btn-outline-danger btn-sm" onclick="promptDeleteNode('${escapeHtml(m.hostname)}')" title="Excluir máquina">
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

async function openMachineMetricsModal(hostname) {
  if (!machineMetricsModal) return;
  currentInspectedMachine = hostname;

  const machine = allAdminMachines.find(m => m.hostname === hostname) || { hostname };
  const summary = currentSummaries.find(s => s.node_id === hostname);

  document.getElementById('mm-hostname').textContent = hostname;
  document.getElementById('mm-subtitle').textContent = `Grupo: ${machine.group_id || 'default'} • ${machine.is_custom ? 'Perfil Personalizado' : 'Perfil Herdado'}`;

  document.getElementById('mm-stat-ping').textContent = summary ? summary.last_ping.toFixed(1) : '--';
  document.getElementById('mm-stat-down').textContent = summary ? summary.last_download.toFixed(1) : '--';
  document.getElementById('mm-stat-up').textContent = summary ? summary.last_upload.toFixed(1) : '--';
  document.getElementById('mm-stat-readings').textContent = summary ? summary.total_readings_24h : '0';

  const subStatus = document.getElementById('mm-sub-status');
  if (subStatus) {
    const isOnline = summary && summary.is_healthy;
    subStatus.textContent = isOnline ? 'Online na rede' : 'Sem sinal recente';
    subStatus.style.color = isOnline ? '#34d399' : '#f87171';
  }

  // Hardware e Rede
  const sys = (summary && summary.system_info) || {};
  document.getElementById('mm-hw-ip').textContent = machine.ip_address || sys.ip_address || '--';
  document.getElementById('mm-hw-mac').textContent = machine.mac_address || sys.mac_address || '--';
  document.getElementById('mm-hw-os').textContent = sys.os || '--';
  document.getElementById('mm-hw-specs').textContent = `${sys.cpu || '--'} (${sys.ram_total_gb ? sys.ram_total_gb.toFixed(1) + 'GB' : '--'})`;

  // Switch LLDP
  const sw = (summary && summary.switch_info) || {};
  document.getElementById('mm-sw-name').textContent = machine.switch_name || sw.switch_name || 'Não descoberto';
  document.getElementById('mm-sw-port').textContent = machine.switch_port || sw.switch_port || '--';
  document.getElementById('mm-sw-vlan').textContent = machine.vlan ? `VLAN ${machine.vlan}` : (sw.vlan ? `VLAN ${sw.vlan}` : '--');
  document.getElementById('mm-sw-proto').textContent = sw.protocol || 'LLDP/CDP';

  machineMetricsModal.style.display = 'flex';

  try {
    const res = await apiFetch(`/api/v1/dashboard/metrics?node_id=${encodeURIComponent(hostname)}`);
    if (res.ok) {
      const data = await res.json();
      renderMachineMetricsCharts(data.metrics_series || []);
    }
  } catch (err) {
    console.error('Falha ao carregar gráficos da máquina:', err);
  }
}
window.openMachineMetricsModal = openMachineMetricsModal;

function closeMachineMetricsModal() {
  if (machineMetricsModal) machineMetricsModal.style.display = 'none';
  if (mmLatencyChart) { mmLatencyChart.destroy(); mmLatencyChart = null; }
  if (mmThroughputChart) { mmThroughputChart.destroy(); mmThroughputChart = null; }
}

function renderMachineMetricsCharts(metricsSeries) {
  if (typeof Chart === 'undefined') return;
  const ctxLatency = document.getElementById('mmLatencyChart');
  const ctxThroughput = document.getElementById('mmThroughputChart');
  if (!ctxLatency || !ctxThroughput) return;

  const currentTheme = document.documentElement.getAttribute('data-theme') || 'dark';
  const colors = getChartThemeColors(currentTheme);

  if (mmLatencyChart) { mmLatencyChart.destroy(); mmLatencyChart = null; }
  if (mmThroughputChart) { mmThroughputChart.destroy(); mmThroughputChart = null; }

  const sorted = [...metricsSeries].sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
  const labels = sorted.map(m => new Date(m.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }));
  const pings = sorted.map(m => m.ping_gateway || 0);
  const downs = sorted.map(m => m.download_mbps || 0);
  const ups = sorted.map(m => m.upload_mbps || 0);

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    layout: { padding: { left: 8, right: 28, top: 10, bottom: 4 } },
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: colors.tooltipBg,
        titleColor: colors.tooltipTitle,
        bodyColor: colors.tooltipBody,
        borderColor: colors.tooltipBorder,
        borderWidth: 1,
        padding: 10,
        cornerRadius: 6
      }
    },
    scales: {
      x: {
        grid: { color: colors.gridColor, drawBorder: false },
        ticks: { color: colors.textColor, font: { size: 10 }, maxRotation: 0, autoSkip: true, maxTicksLimit: 8 }
      },
      y: {
        grid: { color: colors.gridColor, drawBorder: false },
        ticks: { color: colors.textColor, font: { size: 10 } }
      }
    }
  };

  const ctxLat = ctxLatency.getContext('2d');
  mmLatencyChart = new Chart(ctxLat, {
    type: 'line',
    data: {
      labels,
      datasets: [{
        label: 'Ping Gateway',
        data: pings,
        borderColor: '#10b981',
        backgroundColor: createFillGradient(ctxLat, 'rgba(16, 185, 129, 0.25)', 'rgba(16, 185, 129, 0.01)'),
        fill: true,
        tension: 0.35,
        borderWidth: 2,
        pointRadius: 2.5,
        pointHoverRadius: 5
      }]
    },
    options: {
      ...chartOptions,
      scales: {
        ...chartOptions.scales,
        y: {
          ...chartOptions.scales.y,
          title: { display: true, text: 'ms', color: colors.textColor, font: { size: 10 } }
        }
      }
    }
  });

  const ctxThr = ctxThroughput.getContext('2d');
  mmThroughputChart = new Chart(ctxThr, {
    type: 'line',
    data: {
      labels,
      datasets: [
        {
          label: 'Download',
          data: downs,
          borderColor: '#0284c7',
          backgroundColor: createFillGradient(ctxThr, 'rgba(2, 132, 199, 0.22)', 'rgba(2, 132, 199, 0.01)'),
          fill: true,
          tension: 0.35,
          borderWidth: 2,
          pointRadius: 2.5,
          pointHoverRadius: 5
        },
        {
          label: 'Upload',
          data: ups,
          borderColor: '#4f46e5',
          backgroundColor: createFillGradient(ctxThr, 'rgba(79, 70, 229, 0.20)', 'rgba(79, 70, 229, 0.01)'),
          fill: true,
          tension: 0.35,
          borderWidth: 2,
          pointRadius: 2.5,
          pointHoverRadius: 5
        }
      ]
    },
    options: {
      ...chartOptions,
      scales: {
        ...chartOptions.scales,
        y: {
          ...chartOptions.scales.y,
          title: { display: true, text: 'Mbps', color: colors.textColor, font: { size: 10 } }
        }
      }
    }
  });
}

/* ==========================================================================
   12. Modal de Configuração Customizada da Máquina
   ========================================================================== */

function getAllAvailableModules() {
  const modMap = new Map();

  // 1. Módulos padrão do sistema (garante que nunca venha vazio)
  const systemDefaults = [
    { id: 'ping_gateway', name: 'Latência Gateway Local', type: 'ping', target: 'auto_gateway', timeout_seconds: 15 },
    { id: 'ping_nti_core', name: 'Latência Core NTI (Anel)', type: 'ping', target: '177.101.22.40', timeout_seconds: 15 },
    { id: 'packet_loss_gateway', name: 'Perda de Pacotes Gateway', type: 'packet_loss', target: 'auto_gateway', timeout_seconds: 20 },
    { id: 'dns_uepg', name: 'Resolução DNS Institucional', type: 'dns_lookup', domain: 'uepg.br', timeout_seconds: 10 },
    { id: 'throughput_download', name: 'Throughput Download ACON', type: 'http_throughput_download', endpoint: '/api/v1/speedtest/download', size_mb: 5, timeout_seconds: 45 },
    { id: 'throughput_upload', name: 'Throughput Upload ACON', type: 'http_throughput_upload', endpoint: '/api/v1/speedtest/upload', size_mb: 2, timeout_seconds: 45 },
    { id: 'switch_discovery', name: 'Descoberta de Switch LLDP/CDP', type: 'discovery_protocol', protocol: 'LLDP_CDP', timeout_seconds: 60 },
    { id: 'ping_central_server', name: 'Ping Servidor Central', type: 'ping', target: 'auto_gateway', timeout_seconds: 10 },
    { id: 'http_check_portal', name: 'Disponibilidade Portal UEPG', type: 'http_check', endpoint: 'https://www.uepg.br', timeout_seconds: 15 }
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
      mcStatusBadge.innerHTML = '⚙️ Perfil Customizado';
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
          <button type="button" class="btn btn-outline-danger btn-sm" onclick="handleDeleteModule('${escapeHtml(mod.id)}')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
            <span>Excluir</span>
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
  if (!confirm(`Deseja realmente remover o módulo "${moduleId}" do catálogo?`)) return;
  try {
    const res = await apiFetch(`/api/v1/admin/modules/${encodeURIComponent(moduleId)}`, {
      method: 'DELETE'
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Falha ao excluir módulo');
    showToast(`Módulo "${moduleId}" excluído com sucesso`, 'toast-info');
    await loadModulesCatalog();
  } catch (err) {
    showToast(`Erro ao excluir: ${err.message}`, 'toast-error');
  }
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

