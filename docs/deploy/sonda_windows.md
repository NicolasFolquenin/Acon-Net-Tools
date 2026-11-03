# Instalação e Empacotamento Windows

Nas máquinas de usuários e estações de trabalho institucionais com Windows, a sonda opera via instalador gráfico ou executável direto.

---

## 1. Instalação pelo Usuário Final

1. Abra o navegador e acesse o Dashboard NOC do Servidor Central.
2. No menu lateral, acesse a aba **Downloads**.
3. Baixe o executável `UEPG-Net-Tools-Instalador.exe`.
4. Execute o instalador com duplo clique:
   - A sonda é instalada no diretório padrão do usuário (`%LOCALAPPDATA%\UEPG-Net-Tools`).
   - Um atalho é criado na Inicialização do Windows (`Startup`), garantindo que a sonda inicie junto com o sistema.
   - O ícone é integrado à bandeja do sistema (System Tray).

---

## 2. Geração do Instalador (Ambiente de Build)

Para gerar o instalador do Windows:

1. **Compilar o Binário Wails:**
   ```powershell
   cd client-probe
   wails build -tags desktop,production
   ```
2. **Gerar o Instalador com Inno Setup:**
   Utilize o script `client-probe/build/installer.iss`:
   ```powershell
   iscc client-probe/build/installer.iss
   ```
   O instalador gerado será salvo em `central-server/public/downloads/UEPG-Net-Tools-Instalador.exe`.

