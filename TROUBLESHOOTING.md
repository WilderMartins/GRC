# Guia de Solução de Problemas (Troubleshooting)

Este documento lista alguns problemas comuns que podem ocorrer durante o desenvolvimento ou build do Phoenix GRC e como resolvê-los.

## Erros de Build do Docker

### 1. `no space left on device`

- **Sintoma:** O processo de build (`docker compose build` ou `docker build`) falha com uma mensagem indicando que não há espaço em disco.
- **Causa:** O Docker consome muito espaço em disco para armazenar imagens, caches de build e volumes. Ambientes com pouco espaço podem encontrar esse erro.
- **Solução:**
  1.  **Limpeza Agressiva do Docker:** O comando a seguir remove imagens não utilizadas, contêineres parados e, mais importante, todo o cache de build.
      ```bash
      docker system prune -a -f && docker builder prune -a -f
      ```
  2.  **Aumentar o Espaço em Disco do Docker:** Se você estiver usando Docker Desktop, vá em `Settings > Resources` e aumente o limite de espaço em disco alocado para o Docker.
  3.  **Ambiente com Mais Recursos:** Se o problema persistir, pode ser necessário executar o build em uma máquina com mais espaço em disco disponível.

### 2. Build do Frontend falha com erro de ESLint

- **Sintoma:** O build do frontend (`npm run build` dentro do Docker ou localmente) falha com erros relacionados ao ESLint, como `Failed to load plugin` ou `Cannot find module`.
- **Causa:** Faltam dependências de desenvolvimento (`devDependencies`) necessárias para o ESLint analisar o código TypeScript corretamente.
- **Solução:**
  - Certifique-se de que as seguintes dependências estão no `devDependencies` do arquivo `frontend/package.json`:
    ```json
    "devDependencies": {
      ...
      "eslint": "^8.57.0",
      "@typescript-eslint/eslint-plugin": "^7.16.0",
      "@typescript-eslint/parser": "^7.16.0",
      ...
    }
    ```
  - Após adicionar as dependências, delete a pasta `node_modules` e o `package-lock.json` e rode `npm install` novamente. Se o erro for no build do Docker, lembre-se de buildar sem cache (`--no-cache`) para garantir que a nova dependência seja instalada.

### 3. Build do Backend falha com erro de versão do Go

- **Sintoma:** O build do backend falha com uma mensagem como `go: go.mod requires go >= 1.24.3`.
- **Causa:** A versão do Go definida no `Dockerfile.backend` é mais antiga do que a versão exigida pelo `go.mod` do projeto.
- **Solução:**
  - Edite o `Dockerfile.backend`.
  - Encontre a linha `FROM golang:...`
  - Atualize a versão para uma que seja compatível com a do `go.mod`. Por exemplo:
    ```dockerfile
    FROM golang:1.24-alpine AS builder
    ```
  - Reconstrua a imagem do Docker.

## Outros

### `docker-compose: command not found`

- **Sintoma:** Ao tentar rodar `docker-compose`, o terminal retorna "command not found".
- **Causa:** Versões mais recentes do Docker integraram o `compose` diretamente no CLI principal. O comando `docker-compose` (com hífen) foi substituído por `docker compose` (com espaço).
- **Solução:**
  - Use `docker compose` em vez de `docker-compose`. Por exemplo:
    ```bash
    docker compose up -d --build
    ```
