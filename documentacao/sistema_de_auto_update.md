# 🔄 Sistema de Auto-Update e Gestão de Patches

Com dezenas ou centenas de computadores executando a sonda na universidade, ir fisicamente em cada máquina para atualizar executáveis é inviável. Por isso, foi implementado um mecanismo nativo de **Auto-Update Transparente (Auto-Patching)**.

---

## 1. Como Funciona o Auto-Update

O processo ocorre de forma segura e autônoma sem interromper a rotina dos usuários dos computadores:

```
[ Central Server ]                              [ Client Probe ]
       │                                               │
       │  1. GET /api/v1/download/version              │
       │ <──────────────────────────────────────────── │
       │  { "version": "2.4.0", "platforms": ... }     │
       │ ────────────────────────────────────────────> │
       │                                               │
       │                                        2. Compara versão:
       │                                           remoteVer > localVer ?
       │                                               │
       │  3. GET /api/v1/download/binary/{platform}    │
       │ <──────────────────────────────────────────── │
       │     (Download do executável compilado)        │
       │ ────────────────────────────────────────────> │
       │                                               │
       │                                        4. Salva como .new.tmp
       │                                        5. Swap Atômico:
       │                                           app.exe -> app.exe.old
       │                                           app.exe.new -> app.exe
       │                                        6. Executa novo processo
       │                                        7. Encerra processo antigo
```

---

## 2. Swap Atômico no Windows e Linux

### 2.1. O desafio do Windows
No Windows, um executável em execução não pode ser sobrescrito diretamente (erro de arquivo bloqueado por outro processo).  
**Solução Técnica:** O Windows permite que arquivos abertos sejam **renomeados** no mesmo disco.
1. O executável antigo `uepg-net-tools.exe` é renomeado para `uepg-net-tools.exe.old`.
2. O arquivo recém-baixado `uepg-net-tools.exe.new` é renomeado para `uepg-net-tools.exe`.
3. O novo executável é iniciado em segundo plano com `exec.Command(novoExec, args...).Start()`.
4. O processo antigo chama `os.Exit(0)` e se encerra.
5. Na próxima inicialização, o método `CleanupOldBinaries()` remove silenciosamente o arquivo `.old`.

### 2.2. No Linux
No Linux, a substituição via chamada de sistema `rename(2)` é atômica mesmo com o processo em execução (o inode anterior continua alocado até a finalização do processo). A sonda substitui o arquivo com permissão de execução (`0755`) e reexecuta o processo ou é reiniciada pelo `systemd`.

---

## 3. Endpoints Disponíveis no Central Server

| Método | Endpoint | Descrição |
| :--- | :--- | :--- |
| `GET` | `/api/v1/download/version` | Retorna a versão oficial atual e URLs dos binários de cada plataforma. |
| `GET` | `/api/v1/download/binary/windows` | Baixa o executável puro do Windows (`uepg-net-tools.exe`). |
| `GET` | `/api/v1/download/binary/linux-amd64` | Baixa o binário Linux para arquiteturas x86_64. |
| `GET` | `/api/v1/download/binary/linux-arm64` | Baixa o binário Linux para Raspberry Pi e ARM64. |
| `GET` | `/api/v1/download/client` | Baixa o instalador completo para novas máquinas (`UEPG-Net-Tools-Instalador.exe`). |

---

## 4. Como Lançar uma Nova Atualização para as Máquinas

Quando você fizer melhorias ou correções no código da sonda:

1. **Atualize a versão**:
   No arquivo [`client-probe/updater.go`](../client-probe/updater.go), atualize a constante:
   ```go
   const CurrentClientVersion = "2.4.1" // nova versão
   ```
   E atualize o arquivo [`central-server/public/downloads/version.txt`](../central-server/public/downloads/version.txt) com `2.4.1`.

2. **Gere os novos binários**:
   - Para Windows: execute `client-probe/gerar-instalador.ps1`.
   - Para Linux: execute `client-probe/compilar-linux.ps1`.
   *(Os scripts compilam e já movem os binários automaticamente para `central-server/public/downloads/`).*

3. **Pronto!**
   Assim que as sondas nas outras máquinas fizerem o próximo ciclo de checagem, elas detectarão a versão `2.4.1`, farão o download automático e se reiniciarão com o código atualizado, **sem necessidade de intervenção manual**.

---

## 5. Atualização Silenciosa em Massa (Alternativa via Script PowerShell)

Se por algum motivo preferir disparar uma atualização forçada em massa em um laboratório via Active Directory (GPO) ou PsExec:

```powershell
# 1. Encerra processo atual
Stop-Process -Name "uepg-net-tools" -Force -ErrorAction SilentlyContinue

# 2. Baixa novo binário diretamente do Central Server
$server = "https://akvorado.acon.dev.uepg.br"
$dest = "$env:ProgramData\UEPG-Net-Tools\uepg-net-tools.exe"
Invoke-WebRequest -Uri "$server/api/v1/download/binary/windows" -OutFile $dest

# 3. Reinicia processo da sonda em segundo plano
Start-Process -FilePath $dest -WindowStyle Hidden
```
