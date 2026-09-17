package handlers

import (
	"io"
	"net/http"
	"strconv"
)

// SpeedtestDownloadHandler gera um fluxo contínuo de bytes para medição real de download pelo client
func SpeedtestDownloadHandler(w http.ResponseWriter, r *http.Request) {
	sizeMB := 5 // padrão 5MB
	if s := r.URL.Query().Get("size_mb"); s != "" {
		if val, err := strconv.Atoi(s); err == nil && val > 0 && val <= 50 {
			sizeMB = val
		}
	}

	totalBytes := int64(sizeMB * 1024 * 1024)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(totalBytes, 10))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	chunk := make([]byte, 64*1024) // blocos de 64KB
	remaining := totalBytes

	for remaining > 0 {
		toWrite := int64(len(chunk))
		if remaining < toWrite {
			toWrite = remaining
		}
		if _, err := w.Write(chunk[:toWrite]); err != nil {
			return
		}
		remaining -= toWrite
	}
}

// SpeedtestUploadHandler recebe o fluxo de bytes enviado pelo client para medição real de upload
func SpeedtestUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	written, err := io.Copy(io.Discard, r.Body)
	if err != nil {
		http.Error(w, `{"error": "falha ao ler payload"}`, http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","bytes_received":` + strconv.FormatInt(written, 10) + `}`))
}
