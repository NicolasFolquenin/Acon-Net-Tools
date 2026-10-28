package main

import (
	"log"
	"os"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

func main() {
	rs := winres.ResourceSet{}

	// 1. Carrega o ícone oficial .ico
	f, err := os.Open("build/windows/icon.ico")
	if err != nil {
		log.Fatalf("Erro ao abrir icon.ico: %v", err)
	}
	defer f.Close()

	icon, err := winres.LoadICO(f)
	if err != nil {
		log.Fatalf("Erro ao decodificar icon.ico: %v", err)
	}

	// ID 1 ou ID 7 é o ícone padrão do executável no Windows
	rs.SetIcon(winres.ID(1), icon)
	rs.SetIcon(winres.ID(7), icon)

	// 2. Define metadados e versão do binário
	vi := version.Info{
		FileVersion:    [4]uint16{1, 0, 0, 0},
		ProductVersion: [4]uint16{1, 0, 0, 0},
	}
	vi.Set(0x0416, version.ProductName, "UEPG Net Tools")
	vi.Set(0x0416, version.FileDescription, "UEPG Net Tools - Monitoramento e Diagnóstico de Rede")
	vi.Set(0x0416, version.CompanyName, "UEPG Conectividade")
	vi.Set(0x0416, version.LegalCopyright, "Copyright 2026 UEPG")
	vi.Set(0x0416, version.ProductVersion, "1.0.0")
	vi.Set(0x0416, version.FileVersion, "1.0.0.0")

	// Também define idioma neutro (0) para compatibilidade universal
	vi.Set(0, version.ProductName, "UEPG Net Tools")
	vi.Set(0, version.FileDescription, "UEPG Net Tools - Monitoramento e Diagnóstico de Rede")
	vi.Set(0, version.CompanyName, "UEPG Conectividade")
	vi.Set(0, version.LegalCopyright, "Copyright 2026 UEPG")
	vi.Set(0, version.ProductVersion, "1.0.0")
	vi.Set(0, version.FileVersion, "1.0.0.0")
	rs.SetVersionInfo(vi)

	// 3. Manifest com suporte a temas modernos do Windows, DPI e WebView2
	manifestData, err := os.ReadFile("build/windows/wails.exe.manifest")
	if err == nil {
		if manifest, err := winres.AppManifestFromXML(manifestData); err == nil {
			rs.SetManifest(manifest)
		}
	}

	// 4. Grava o objeto linkável rsrc_windows_amd64.syso na raiz do pacote
	out, err := os.Create("rsrc_windows_amd64.syso")
	if err != nil {
		log.Fatalf("Erro ao criar rsrc_windows_amd64.syso: %v", err)
	}
	defer out.Close()

	if err := rs.WriteObject(out, winres.ArchAMD64); err != nil {
		log.Fatalf("Erro ao gravar objeto COFF: %v", err)
	}

	log.Println("[OK] Objeto de recurso do Windows (rsrc_windows_amd64.syso) gerado com sucesso!")
}
