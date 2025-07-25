# Phoenix GRC - Gestão de Governança, Risco e Conformidade

Bem-vindo ao Phoenix GRC! Uma plataforma de código aberto para ajudar equipes a gerenciar riscos de TI, responder a vulnerabilidades e alcançar a conformidade de forma eficiente.

[![Backend CI](https://github.com/SEU_USUARIO/phoenix-grc/actions/workflows/backend-ci.yml/badge.svg)](https://github.com/SEU_USUARIO/phoenix-grc/actions/workflows/backend-ci.yml)
[![Frontend CI](https://github.com/SEU_USUARIO/phoenix-grc/actions/workflows/frontend-ci.yml/badge.svg)](https://github.com/SEU_USUARIO/phoenix-grc/actions/workflows/frontend-ci.yml)

## Instalação para Produção (Automatizada com Docker)

Este processo foi desenhado para ser robusto e simples. Você não precisa instalar `Node.js` ou `Go`.

### Pré-requisitos

1.  **Git:** Para baixar o código-fonte.
2.  **Docker:** Para construir e rodar a aplicação. O Docker deve ter recursos suficientes (recomendado: 4+ CPUs, 8GB+ RAM, 20GB+ de espaço em disco).

Se precisar de ajuda, siga nosso **[Guia de Preparação do Ambiente](./PREPARACAO_AMBIENTE.md)**.

### Passo a Passo

1.  **Clone o projeto:**
    ```bash
    git clone https://github.com/SEU_USUARIO/phoenix-grc.git
    cd phoenix-grc
    ```

2.  **Configure as variáveis de ambiente:**
    - Copie o arquivo de exemplo:
      ```bash
      cp .env.example .env
      ```
    - **IMPORTANTE PARA PRODUÇÃO:** Abra o arquivo `.env` e altere a senha padrão do banco de dados (`POSTGRES_PASSWORD`) para uma senha forte e segura.

3.  **Inicie a Aplicação:**
    > Certifique-se de que o Docker Desktop (ou Docker Engine) está em execução.
    ```bash
    docker compose up -d --build
    ```
    Este comando irá construir, de forma otimizada, todas as partes da aplicação e iniciá-las. Pode demorar alguns minutos na primeira vez.

4.  **Acesse o Wizard de Instalação:**
    - Abra seu navegador e acesse: **http://localhost**
    - Siga os passos para criar sua organização e a conta de administrador.

### Verificando a Instalação

Para garantir que o build ocorreu corretamente, você pode usar nosso script de verificação:
```bash
./build-verify.sh
```
Este script executa um build limpo (sem cache) e irá falhar se houver qualquer problema no processo. Para mais detalhes sobre solução de problemas, veja o **[TROUBLESHOOTING.md](./TROUBLESHOOTING.md)**.

---

## Para Desenvolvedores
Se você deseja contribuir com o projeto, nosso **[Guia do Desenvolvedor](./DEVELOPER_GUIDE.md)** contém informações detalhadas sobre a arquitetura, como rodar o frontend em modo de desenvolvimento (hot-reload) e os padrões de código.