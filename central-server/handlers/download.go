package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"central-server/config"
)

// DownloadHandler gerencia o download do instalador e binários do client-probe
type DownloadHandler struct {
	cfg *config.AppConfig
}

// ClientFileInfo representa as informações sobre o arquivo do instalador disponível
type ClientFileInfo struct {
	Available     bool   `json:"available"`
	Filename      string `json:"filename"`
	SizeBytes     int64  `json:"size_bytes"`
	SizeFormatted string `json:"size_formatted"`
	UpdatedAt     string `json:"updated_at,omitempty"`
	Message       string `json:"message,omitempty"`
	ServerURL     string `json:"server_url,omitempty"`
}

// NewDownloadHandler inicializa o handler de download
func NewDownloadHandler(cfg *config.AppConfig) *DownloadHandler {
	return &DownloadHandler{cfg: cfg}
}

// locateClientFile procura pelo arquivo do instalador ou executável em caminhos prioritários
func (h *DownloadHandler) locateClientFile() (string, os.FileInfo, error) {
	candidatePaths := []string{
		filepath.Join(h.cfg.StaticDir, "downloads", "UEPG-Net-Tools-Instalador.exe"),
		filepath.Join(h.cfg.StaticDir, "downloads", "uepg-net-tools.exe"),
		filepath.Join("..", "client-probe", "build", "bin", "UEPG-Net-Tools-Instalador.exe"),
		filepath.Join("..", "client-probe", "build", "bin", "uepg-net-tools.exe"),
		filepath.Join("client-probe", "build", "bin", "UEPG-Net-Tools-Instalador.exe"),
		filepath.Join("client-probe", "build", "bin", "uepg-net-tools.exe"),
	}

	for _, p := range candidatePaths {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Size() > 0 {
			return p, fi, nil
		}
	}

	// Se não achou pelos caminhos exatos, tenta listar a pasta downloads
	downloadsDir := filepath.Join(h.cfg.StaticDir, "downloads")
	if entries, err := os.ReadDir(downloadsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".exe") {
				target := filepath.Join(downloadsDir, e.Name())
				if fi, err := os.Stat(target); err == nil && fi.Size() > 0 {
					return target, fi, nil
				}
			}
		}
	}

	return "", nil, os.ErrNotExist
}

// formatBytes converte bytes para formato legível (MB / KB)
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// getServerURL determina a URL pública do servidor (priorizando .env, com fallback para Host HTTP)
func (h *DownloadHandler) getServerURL(r *http.Request) string {
	if h.cfg.PublicServerURL != "" {
		return h.cfg.PublicServerURL
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// HandleDownloadClient serve o arquivo instalador executável para o navegador
func (h *DownloadHandler) HandleDownloadClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	filePath, fi, err := h.locateClientFile()
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Instalador não disponível",
			"message": "O instalador do client-probe ainda não foi gerado ou compilado. Por favor, execute o script 'gerar-instalador.bat' na pasta do client-probe.",
		})
		return
	}

	filename := fi.Name()
	// Nome amigável padrão se for uepg-net-tools.exe
	if strings.EqualFold(filename, "uepg-net-tools.exe") {
		filename = "UEPG-Net-Tools-Instalador.exe"
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Type", "application/vnd.microsoft.portable-executable")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	http.ServeFile(w, r, filePath)
}

// HandleClientInfo retorna metadados JSON sobre o arquivo disponível para download e a URL do servidor
func (h *DownloadHandler) HandleClientInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	serverURL := h.getServerURL(r)

	_, fi, err := h.locateClientFile()
	if err != nil {
		_ = json.NewEncoder(w).Encode(ClientFileInfo{
			Available: false,
			Message:   "Instalador Windows não localizado no servidor.",
			ServerURL: serverURL,
		})
		return
	}

	filename := fi.Name()
	if strings.EqualFold(filename, "uepg-net-tools.exe") {
		filename = "UEPG-Net-Tools-Instalador.exe"
	}

	_ = json.NewEncoder(w).Encode(ClientFileInfo{
		Available:     true,
		Filename:      filename,
		SizeBytes:     fi.Size(),
		SizeFormatted: formatBytes(fi.Size()),
		UpdatedAt:     fi.ModTime().UTC().Format(time.RFC3339),
		ServerURL:     serverURL,
	})
}

// HandleInstallScript serve o script bash install-probe.sh dinâmico com a URL do servidor do .env embutida
func (h *DownloadHandler) HandleInstallScript(w http.ResponseWriter, r *http.Request) {
	candidates := []string{
		filepath.Join(h.cfg.StaticDir, "downloads", "install-probe.sh"),
		filepath.Join("public", "downloads", "install-probe.sh"),
	}

	var data []byte
	var err error
	for _, p := range candidates {
		if data, err = os.ReadFile(p); err == nil {
			break
		}
	}

	if err != nil || len(data) == 0 {
		http.Error(w, "Script de instalação não encontrado", http.StatusNotFound)
		return
	}

	serverURL := h.getServerURL(r)
	content := string(data)
	content = strings.Replace(content, `SUGGESTED_URL="https://nettools.acon.uepg.br"`, fmt.Sprintf(`SUGGESTED_URL="%s"`, serverURL), 1)
	content = strings.Replace(content, `SUGGESTED_URL="https://akvorado.acon.dev.uepg.br"`, fmt.Sprintf(`SUGGESTED_URL="%s"`, serverURL), 1)

	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

// locateRawBinary localiza o binário executável puro (não instalador) para a plataforma requisitada
func (h *DownloadHandler) locateRawBinary(platform string) (string, os.FileInfo, error) {
	norm := strings.ToLower(strings.TrimSpace(platform))
	var candidates []string

	switch norm {
	case "windows", "windows-amd64", "win":
		candidates = []string{
			filepath.Join(h.cfg.StaticDir, "downloads", "uepg-net-tools.exe"),
			filepath.Join("client-probe", "cmd", "installer", "payload", "uepg-net-tools.exe"),
			filepath.Join("..", "client-probe", "cmd", "installer", "payload", "uepg-net-tools.exe"),
			filepath.Join("..", "client-probe", "build", "bin", "uepg-net-tools.exe"),
			filepath.Join("..", "client-probe", "probe-app.exe"),
			filepath.Join("client-probe", "build", "bin", "uepg-net-tools.exe"),
			filepath.Join("client-probe", "probe-app.exe"),
		}
	case "linux-amd64", "linux_amd64", "linux":
		candidates = []string{
			filepath.Join(h.cfg.StaticDir, "downloads", "uepg-probe-linux-amd64"),
			filepath.Join("..", "client-probe", "build", "linux", "uepg-probe-linux-amd64"),
			filepath.Join("client-probe", "build", "linux", "uepg-probe-linux-amd64"),
		}
	case "linux-arm64", "linux_arm64":
		candidates = []string{
			filepath.Join(h.cfg.StaticDir, "downloads", "uepg-probe-linux-arm64"),
			filepath.Join("..", "client-probe", "build", "linux", "uepg-probe-linux-arm64"),
			filepath.Join("client-probe", "build", "linux", "uepg-probe-linux-arm64"),
		}
	default:
		return "", nil, fmt.Errorf("plataforma desconhecida: %s", platform)
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Size() > 0 {
			return p, fi, nil
		}
	}

	return "", nil, os.ErrNotExist
}

// PlatformArtifactInfo detalha o binário de uma plataforma
type PlatformArtifactInfo struct {
	Available     bool   `json:"available"`
	Filename      string `json:"filename"`
	URL           string `json:"url"`
	SizeBytes     int64  `json:"size_bytes"`
	SizeFormatted string `json:"size_formatted"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

// VersionInfoResponse resposta do endpoint GET /api/v1/download/version
type VersionInfoResponse struct {
	Version   string                          `json:"version"`
	ServerURL string                          `json:"server_url"`
	Platforms map[string]PlatformArtifactInfo `json:"platforms"`
}

// HandleVersionInfo fornece a versão mais atual do client-probe e links para download automático dos binários
func (h *DownloadHandler) HandleVersionInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	serverURL := h.getServerURL(r)

	// Tenta ler versão de public/downloads/version.txt ou usa padrão "2.4.0"
	version := "2.4.0"
	versionPath := filepath.Join(h.cfg.StaticDir, "downloads", "version.txt")
	if data, err := os.ReadFile(versionPath); err == nil {
		vStr := strings.TrimSpace(string(data))
		if vStr != "" {
			version = vStr
		}
	}

	platforms := make(map[string]PlatformArtifactInfo)

	for _, plat := range []string{"windows", "linux-amd64", "linux-arm64"} {
		pPath, fi, err := h.locateRawBinary(plat)
		if err == nil && fi != nil {
			platforms[plat] = PlatformArtifactInfo{
				Available:     true,
				Filename:      fi.Name(),
				URL:           fmt.Sprintf("%s/api/v1/download/binary/%s", serverURL, plat),
				SizeBytes:     fi.Size(),
				SizeFormatted: formatBytes(fi.Size()),
				UpdatedAt:     fi.ModTime().UTC().Format(time.RFC3339),
			}
		} else {
			platforms[plat] = PlatformArtifactInfo{
				Available: false,
				URL:       fmt.Sprintf("%s/api/v1/download/binary/%s", serverURL, plat),
			}
		}
		_ = pPath
	}

	_ = json.NewEncoder(w).Encode(VersionInfoResponse{
		Version:   version,
		ServerURL: serverURL,
		Platforms: platforms,
	})
}

// HandleDownloadBinary serve o executável puro diretamente para auto-update da sonda
func (h *DownloadHandler) HandleDownloadBinary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Extrai plataforma da URL (ex: /api/v1/download/binary/windows)
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 0 {
		http.Error(w, "Plataforma não informada", http.StatusBadRequest)
		return
	}
	platform := parts[len(parts)-1]

	filePath, fi, err := h.locateRawBinary(platform)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":    "Binário não disponível",
			"platform": platform,
			"message":  "O binário compilado para esta plataforma ainda não está presente na pasta de downloads.",
		})
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fi.Name()))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")

	http.ServeFile(w, r, filePath)
}
