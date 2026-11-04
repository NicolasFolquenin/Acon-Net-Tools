# Instalação da Sonda em Linux

Nas máquinas cliente Linux (como Raspberry Pi, estações Debian, Ubuntu ou servidores de borda), a sonda é instalada como serviço nativo do sistema operacional (`systemd`).

---

## 1. Instalação em 1 Comando via Terminal

Execute no terminal da máquina cliente:

```bash
curl -fsSL https://akvorado.acon.dev.uepg.br/downloads/install-probe.sh | sudo bash
```
*(Caso seu servidor utilize outro IP ou domínio, substitua a URL acima).*

---

## 2. Ações do Script de Instalação

- Baixa o binário compilado para a arquitetura local (`linux-amd64` ou `linux-arm64`).
- Instala o binário em `/usr/local/bin/uepg-probe`.
- Instala e habilita o serviço do sistema `uepg-probe.service` em `/etc/systemd/system/`.
- Concede capabilities de rede (`cap_net_raw+ep`) ao executável para envio de ICMP nativo sem exigir permissões de superusuário contínuas.
- Registra o atalho do monitor interativo no terminal:
  ```bash
  uepg-probe-monitor
  ```
  (ou simplesmente `uepg-monitor`).

---

## 3. Gestão do Serviço Systemd

```bash
# Verificar status da sonda
sudo systemctl status uepg-probe

# Visualizar logs em tempo real
sudo journalctl -u uepg-probe -f

# Reiniciar serviço
sudo systemctl restart uepg-probe
```

