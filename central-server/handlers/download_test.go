package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"central-server/config"
)

func TestDownloadHandler_ClientInfoAndDownload(t *testing.T) {
	tempDir := t.TempDir()
	downloadsDir := filepath.Join(tempDir, "downloads")
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		t.Fatalf("falha ao criar pasta temporária: %v", err)
	}

	testExe := filepath.Join(downloadsDir, "UEPG-Net-Tools-Instalador.exe")
	if err := os.WriteFile(testExe, []byte("MOCK_EXE_CONTENT_FOR_TESTING"), 0644); err != nil {
		t.Fatalf("falha ao criar arquivo mock exe: %v", err)
	}

	cfg := &config.AppConfig{
		StaticDir: tempDir,
	}

	handler := NewDownloadHandler(cfg)

	// Teste 1: HandleClientInfo
	reqInfo := httptest.NewRequest(http.MethodGet, "/api/v1/download/client/info", nil)
	rrInfo := httptest.NewRecorder()
	handler.HandleClientInfo(rrInfo, reqInfo)

	if rrInfo.Code != http.StatusOK {
		t.Fatalf("esperado código 200 em info, obteve %d", rrInfo.Code)
	}

	var info ClientFileInfo
	if err := json.Unmarshal(rrInfo.Body.Bytes(), &info); err != nil {
		t.Fatalf("falha ao deserializar JSON info: %v", err)
	}

	if !info.Available {
		t.Errorf("esperava available=true")
	}
	if info.Filename != "UEPG-Net-Tools-Instalador.exe" {
		t.Errorf("esperava filename UEPG-Net-Tools-Instalador.exe, obteve %s", info.Filename)
	}
	if info.SizeBytes <= 0 {
		t.Errorf("esperava tamanho > 0")
	}

	// Teste 2: HandleDownloadClient
	reqDl := httptest.NewRequest(http.MethodGet, "/api/v1/download/client", nil)
	rrDl := httptest.NewRecorder()
	handler.HandleDownloadClient(rrDl, reqDl)

	if rrDl.Code != http.StatusOK {
		t.Fatalf("esperado código 200 no download, obteve %d", rrDl.Code)
	}

	disposition := rrDl.Header().Get("Content-Disposition")
	if disposition == "" {
		t.Errorf("esperava cabeçalho Content-Disposition no download")
	}

	if rrDl.Body.String() != "MOCK_EXE_CONTENT_FOR_TESTING" {
		t.Errorf("conteúdo baixado não confere com o esperado")
	}

	// Teste 3: Método inválido (POST)
	reqPost := httptest.NewRequest(http.MethodPost, "/api/v1/download/client", nil)
	rrPost := httptest.NewRecorder()
	handler.HandleDownloadClient(rrPost, reqPost)
	if rrPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("esperava código 405 para POST, obteve %d", rrPost.Code)
	}
}

func TestDownloadHandler_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.AppConfig{
		StaticDir: tempDir,
	}

	handler := NewDownloadHandler(cfg)

	// Teste 1: Info quando não existe
	reqInfo := httptest.NewRequest(http.MethodGet, "/api/v1/download/client/info", nil)
	rrInfo := httptest.NewRecorder()
	handler.HandleClientInfo(rrInfo, reqInfo)

	var info ClientFileInfo
	if err := json.Unmarshal(rrInfo.Body.Bytes(), &info); err != nil {
		t.Fatalf("falha ao deserializar JSON info: %v", err)
	}
	if info.Available {
		t.Errorf("esperava available=false quando o arquivo não existe")
	}

	// Teste 2: Download quando não existe
	reqDl := httptest.NewRequest(http.MethodGet, "/api/v1/download/client", nil)
	rrDl := httptest.NewRecorder()
	handler.HandleDownloadClient(rrDl, reqDl)

	if rrDl.Code != http.StatusNotFound {
		t.Errorf("esperado 404, obteve %d", rrDl.Code)
	}
}
