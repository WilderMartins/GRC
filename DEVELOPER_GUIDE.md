# Guia do Desenvolvedor - Phoenix GRC

Este guia contém informações técnicas detalhadas para desenvolvedores que desejam contribuir ou entender a fundo a arquitetura do Phoenix GRC.

## Stack Tecnológica

- **Backend:** Go (Golang) com Gin Gonic
- **Frontend:** Next.js com TypeScript
- **Banco de Dados:** PostgreSQL 16
- **ORM (Go):** GORM
- **Autenticação:** JWT (JSON Web Tokens), OAuth2 (Google, GitHub), MFA (TOTP, Códigos de Backup). SAML.
- **UI (Frontend):** Tailwind CSS
- **Containerização:** Docker

## Ambiente de Desenvolvimento

### Rodando o Frontend em Modo de Desenvolvimento (Hot-Reloading)

Para uma experiência de desenvolvimento mais fluida com hot-reloading:

1.  **Navegue até o diretório do frontend:**
    ```bash
    cd frontend
    ```
2.  **Instale as dependências:**
    ```bash
    npm install
    ```
3.  **Configure as Variáveis de Ambiente do Frontend:**
    Crie um arquivo `.env.local` no diretório `frontend` com o seguinte conteúdo:
    ```env
    NEXT_PUBLIC_API_BASE_URL=http://localhost
    NEXT_PUBLIC_APP_ROOT_URL=http://localhost:3000
    ```
4.  **Inicie o Servidor de Desenvolvimento:**
    ```bash
    npm run dev
    ```
    O frontend estará acessível em `http://localhost:3000`.

### Detalhes do Ambiente Docker

O `docker-compose.yml` define três serviços: `db`, `backend`, e `nginx`.

*   **`db`**: Container PostgreSQL. Os dados são persistidos no volume `postgres_data`.
*   **`backend`**: A API em Go.
*   **`nginx`**: Proxy reverso para o backend e serve o frontend estático.

### Considerações para Produção

*   **Variáveis de Ambiente**: Use chaves fortes e secretas em produção.
*   **HTTPS**: Habilite HTTPS no Nginx. É obrigatório para produção.
*   **Logging**: Configure um sistema de logging centralizado.
*   **Backups**: Implemente uma estratégia de backup para o banco de dados.
*   **Migrações**: Para produção, considere usar uma ferramenta de migração dedicada como `golang-migrate/migrate`.

## Estrutura do Projeto

```
.
├── .env.example
├── Dockerfile.backend
├── README.md
├── DEVELOPER_GUIDE.md
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── auth/
│   │   ├── database/
│   │   ├── handlers/
│   │   ├── models/
│   │   └── ...
│   └── pkg/
├── frontend/
└── docker-compose.yml
```

## Endpoints da API

A documentação completa da API foi movida para `API_DOCUMENTATION.md` para manter este guia focado no desenvolvimento.

## Gerenciamento de Migrações de Banco de Dados

Utilizamos `golang-migrate/migrate` para gerenciar o schema do banco.

### Pré-requisitos

Instale a CLI `migrate`:
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### Comandos

*   **Criar uma nova migração:**
    ```bash
    migrate create -ext sql -dir backend/internal/database/migrations -seq <migration_name>
    ```
*   **Aplicar migrações:**
    ```bash
    migrate -database "${DB_URL_MIGRATE}" -path backend/internal/database/migrations up
    ```
*   **Reverter a última migração:**
    ```bash
    migrate -database "${DB_URL_MIGRATE}" -path backend/internal/database/migrations down 1
    ```

## CI/CD

Utilizamos GitHub Actions para CI/CD. Os workflows estão em `.github/workflows/`.

*   `backend-ci.yml`: Roda linters e testes para o backend.
*   `backend-docker.yml`: Builda e publica a imagem Docker do backend no GitHub Container Registry.

## Próximos Passos (Desenvolvimento Futuro)

*   Finalização e polimento do Frontend Next.js.
*   Concluir a implementação da lógica do Assertion Consumer Service (ACS) para SAML.
*   Implementação da funcionalidade de exclusão de arquivos no `filestorage`.
*   Testes de integração e E2E abrangentes.
*   Melhorias na paginação e filtros da API.
*   Configuração de logging mais robusto.
