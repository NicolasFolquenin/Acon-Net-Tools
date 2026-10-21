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

// HandleClientInfo retorna metadados JSON sobre o arquivo disponível para download
func (h *DownloadHandler) HandleClientInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	_, fi, err := h.locateClientFile()
	if err != nil {
		_ = json.NewEncoder(w).Encode(ClientFileInfo{
			Available: false,
			Message:   "Instalador não localizado no servidor.",
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
	})
}
