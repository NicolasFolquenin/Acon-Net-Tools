# 📌 Guia Rápido: Testes Locais e Atualização das Sondas

> O documento completo com todos os detalhes de arquitetura, compilação e deploy está disponível em:  
> 🔗 **[`docs/deploy/procedimento_atualizacao_sondas.md`](docs/deploy/procedimento_atualizacao_sondas.md)**

---

### 1. Como Testar Agora Localmente (Antes de Mudar a Versão)
Para validar o novo recurso de detecção de usuários ativos e contas da máquina:

1. Acesse a pasta `client-probe` e execute:
   ```cmd
   testar.bat
   ```
2. Na janela que abrir:
   - Verifique o **badge de usuário** no cabeçalho superior.
   - Veja o **Bloco 5** em Especificações Técnicas (usuário em execução, usuário de console e tabela de contas com status **Logado** / **Inativo**).
   - Clique em **"Rodar Teste Agora"** para validar o envio das métricas.

---

### 2. Resumo de Como Atualizar Todas as Máquinas (Quando Estiverem Prontos)

As sondas instaladas **já possuem auto-update automático a cada 30 minutos**. Para que todas se atualizem:

1. **Alterar a versão** (ex: `2.5.0`):
   - [`client-probe/updater.go`](client-probe/updater.go) $\rightarrow$ `const CurrentClientVersion = "2.5.0"`
   - [`central-server/public/downloads/version.txt`](central-server/public/downloads/version.txt) $\rightarrow$ `2.5.0`
2. **Compilar o instalador/executável**:
   - Rodar `client-probe/gerar-instalador.ps1`.
3. **Publicar no Central Server**:
   - Colocar o novo `uepg-net-tools.exe` e `version.txt` na pasta `public/downloads/` do servidor.
4. **Pronto!**
   - As máquinas clientes detectam a versão nova sozinhas, baixam o patch em segundo plano e reiniciam atualizadas.
