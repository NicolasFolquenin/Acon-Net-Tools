Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host "  UEPG Net Tools - Compilação da Sonda Client para Linux (Debian)" -ForegroundColor Cyan
Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host ""

$env:CGO_ENABLED = "0"
$env:GOOS = "linux"

Write-Host "[1/3] Compilando binário Linux amd64 (x86_64)..." -ForegroundColor Yellow
$env:GOARCH = "amd64"
go build -tags cli -ldflags="-s -w" -o build/linux/uepg-probe-linux-amd64 .
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERRO] Falha ao compilar para Linux amd64!" -ForegroundColor Red
    exit $LASTEXITCODE
}

Write-Host "[2/3] Compilando binário Linux arm64 (Raspberry Pi / ARM)..." -ForegroundColor Yellow
$env:GOARCH = "arm64"
go build -tags cli -ldflags="-s -w" -o build/linux/uepg-probe-linux-arm64 .
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERRO] Falha ao compilar para Linux arm64!" -ForegroundColor Red
    exit $LASTEXITCODE
}

Write-Host "[3/3] Publicando binários na pasta de downloads do Central Server..." -ForegroundColor Yellow
$destDir = "..\central-server\public\downloads\"
if (Test-Path $destDir) {
    Copy-Item "build\linux\uepg-probe-linux-amd64" -Destination $destDir -Force
    Copy-Item "build\linux\uepg-probe-linux-arm64" -Destination $destDir -Force
    Write-Host "[OK] Binários copiados para $destDir com sucesso!" -ForegroundColor Green
}

Write-Host ""
Write-Host "==================================================================" -ForegroundColor Green
Write-Host "  Compilação Linux concluída com sucesso!" -ForegroundColor Green
Write-Host "==================================================================" -ForegroundColor Green
