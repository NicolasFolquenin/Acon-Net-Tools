package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// CurrentClientVersion versão atual embutida da sonda client
const CurrentClientVersion = "2.4.0"

// RemoteVersionResponse estrutura recebida do endpoint GET /api/v1/download/version
type RemoteVersionResponse struct {
	Version   string                          `json:"version"`
	ServerURL string                          `json:"server_url"`
	Platforms map[string]RemotePlatformArtifact `json:"platforms"`
}

type RemotePlatformArtifact struct {
	Available bool   `json:"available"`
	Filename  string `json:"filename"`
	URL       string `json:"url"`
	SizeBytes int64  `json:"size_bytes"`
}

// CleanupOldBinaries remove arquivos temporários .old gerados em atualizações anteriores
func CleanupOldBinaries() {
	execPath, err := os.Executable()
	if err != nil {
		return
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return
	}

	oldPath := execPath + ".old"
	if _, err := os.Stat(oldPath); err == nil {
		_ = os.Remove(oldPath)
		log.Printf("[UPDATER] Limpeza de binário antigo concluída: %s", filepath.Base(oldPath))
	}

	tmpPath := execPath + ".new"
	if _, err := os.Stat(tmpPath); err == nil {
		_ = os.Remove(tmpPath)
	}
}

// getPlatformKey retorna a chave da plataforma para consulta de binário
func getPlatformKey() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "linux-arm64"
		}
		return "linux-amd64"
	default:
		return fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	}
}

// isNewerVersion verifica se remoteVer é estritamente mais recente que localVer
func isNewerVersion(remoteVer, localVer string) bool {
	remoteClean := strings.TrimPrefix(strings.TrimSpace(remoteVer), "v")
	localClean := strings.TrimPrefix(strings.TrimSpace(localVer), "v")

	if remoteClean == "" || localClean == "" || remoteClean == localClean {
		return false
	}

	// Remove sufixos como -cli, -beta, etc.
	rBase := strings.Split(remoteClean, "-")[0]
	lBase := strings.Split(localClean, "-")[0]

	rParts := strings.Split(rBase, ".")
	lParts := strings.Split(lBase, ".")

	maxLen := len(rParts)
	if len(lParts) > maxLen {
		maxLen = len(lParts)
	}

	for i := 0; i < maxLen; i++ {
		var rNum, lNum int
		if i < len(rParts) {
			rNum, _ = strconv.Atoi(rParts[i])
		}
		if i < len(lParts) {
			lNum, _ = strconv.Atoi(lParts[i])
		}

		if rNum > lNum {
			return true
		} else if rNum < lNum {
			return false
		}
	}

	return false
}

// CheckAndApplyAutoUpdate verifica se há uma versão mais nova no servidor e executa o patch automático
func CheckAndApplyAutoUpdate(ctx context.Context, serverURL, apiToken string) (bool, error) {
	cleanURL := strings.TrimRight(strings.TrimSpace(serverURL), "/")
	if cleanURL == "" || cleanURL == "none" || cleanURL == "disabled" {
		return false, nil
	}

	versionURL := fmt.Sprintf("%s/api/v1/download/version", cleanURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionURL, nil)
	if err != nil {
		return false, err
	}
	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("falha ao consultar versao no servidor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("servidor respondeu com status %d ao consultar versao", resp.StatusCode)
	}

	var verResp RemoteVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&verResp); err != nil {
		return false, fmt.Errorf("falha ao decodificar resposta de versao: %w", err)
	}

	if !isNewerVersion(verResp.Version, CurrentClientVersion) {
		// Já está na versão mais atual
		return false, nil
	}

	platKey := getPlatformKey()
	artifact, ok := verResp.Platforms[platKey]
	if !ok || !artifact.Available || artifact.URL == "" {
		log.Printf("[UPDATER] Nova versao %s encontrada, mas binario para '%s' nao esta disponivel no servidor.", verResp.Version, platKey)
		return false, nil
	}

	log.Printf("[UPDATER] 🚀 Nova versao identificada: %s (atual: %s). Iniciando auto-patching da sonda...", verResp.Version, CurrentClientVersion)

	downloadURL := artifact.URL
	if !strings.HasPrefix(downloadURL, "http://") && !strings.HasPrefix(downloadURL, "https://") {
		downloadURL = fmt.Sprintf("%s%s", cleanURL, downloadURL)
	}

	if err := performBinarySwap(ctx, downloadURL, apiToken); err != nil {
		log.Printf("[UPDATER] [ERRO] Falha ao aplicar atualizacao automatica: %v", err)
		return false, err
	}

	return true, nil
}

// performBinarySwap baixa o binário e faz o swap seguro em disco
func performBinarySwap(ctx context.Context, downloadURL, apiToken string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("nao foi possivel determinar caminho do executavel: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("falha ao resolver symlinks do executavel: %w", err)
	}

	tmpPath := execPath + ".new"
	oldPath := execPath + ".old"

	// 1. Download do binário atualizado
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("erro no download do binario: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status HTTP %d no download do binario", resp.StatusCode)
	}

	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("falha ao criar arquivo temporario de update: %w", err)
	}

	written, err := io.Copy(out, resp.Body)
	_ = out.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("erro ao gravar binario baixado: %w", err)
	}

	if written < 1024*100 { // Menor que 100KB é suspeito de erro/HTML
		_ = os.Remove(tmpPath)
		return fmt.Errorf("tamanho do binario baixado (%d bytes) muito pequeno ou corrompido", written)
	}

	// Garante permissões de execução no Linux/macOS
	if runtime.GOOS != "windows" {
		_ = os.Chmod(tmpPath, 0755)
	}

	log.Printf("[UPDATER] Download concluido com sucesso (%d bytes). Aplicando swap do executavel...", written)

	// 2. Swap Atômico
	if runtime.GOOS == "windows" {
		_ = os.Remove(oldPath) // Remove qualquer .old remanescente
		if err := os.Rename(execPath, oldPath); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("falha ao mover binario atual para .old: %w", err)
		}
		if err := os.Rename(tmpPath, execPath); err != nil {
			// Restaura backup
			_ = os.Rename(oldPath, execPath)
			return fmt.Errorf("falha ao instalar novo binario (backup restaurado): %w", err)
		}
	} else {
		// Linux: substituição atômica via os.Rename
		if err := os.Rename(tmpPath, execPath); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("falha ao substituir binario no Linux: %w", err)
		}
	}

	log.Printf("[UPDATER] ✨ Binario atualizado com sucesso! Reiniciando sonda com a nova versao...")

	// 3. Reinicia o processo com os mesmos argumentos
	cmd := exec.Command(execPath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		log.Printf("[UPDATER] [AVISO] Novo binario instalado, mas falhou ao reiniciar processo automaticamente: %v", err)
		return nil
	}

	// Encerra processo atual para que o novo assuma
	os.Exit(0)
	return nil
}
