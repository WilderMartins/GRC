# Phoenix GRC - Sistema de Gestão de Governança, Risco e Conformidade de TI

Phoenix GRC é uma plataforma SaaS (Software as a Service), whitelabel, projetada para ser intuitiva, escalável e segura. O objetivo é permitir que equipes de TI e segurança de todos os tamanhos gerenciem riscos, respondam a vulnerabilidades e demonstrem conformidade com os principais frameworks do setor de forma eficiente.

## Stack Tecnológica

- **Backend:** Go (Golang) com Gin Gonic
- **Frontend:** Next.js com TypeScript (a ser desenvolvido)
- **Banco de Dados:** PostgreSQL 16
- **ORM (Go):** GORM
- **Autenticação:** JWT (JSON Web Tokens)
- **UI (Frontend):** Tailwind CSS (a ser desenvolvido)
- **Containerização:** Docker

## Ambiente de Desenvolvimento com Docker

Este projeto utiliza Docker e Docker Compose para facilitar a configuração do ambiente de desenvolvimento.

### Pré-requisitos

- Docker: [Instruções de Instalação](https://docs.docker.com/get-docker/)
- Docker Compose: (Normalmente incluído com Docker Desktop) [Instruções de Instalação](https://docs.docker.com/compose/install/)
- Git: [Instruções de Instalação](https://git-scm.com/downloads)
- `curl` ou um cliente API como Postman/Insomnia (para interagir com a API).

### Configuração Inicial e Setup

1.  **Clone o Repositório:**
    ```bash
    git clone <url_do_repositorio>
    cd phoenix-grc # Ou o nome do diretório do projeto
    ```

2.  **Configure as Variáveis de Ambiente:**
    Copie o arquivo de exemplo `.env.example` para `.env`.
    ```bash
    cp .env.example .env
    ```
    Edite o arquivo `.env` e **certifique-se de definir `JWT_SECRET_KEY`, `SAML_SP_KEY_PEM`, e `SAML_SP_CERT_PEM` com valores seguros e únicos**. Ajuste `APP_ROOT_URL` para o endereço base da sua aplicação (ex: `http://localhost:8080` para desenvolvimento local). Configure também as URLs de callback do frontend (`FRONTEND_SAML_CALLBACK_URL`, `FRONTEND_OAUTH2_CALLBACK_URL`) e as demais configurações do banco de dados e servidor conforme necessário.

    **Geração de Chaves SAML SP:**
    Para `SAML_SP_KEY_PEM` e `SAML_SP_CERT_PEM`, você precisará de um par de chave privada e certificado X.509 para o seu Service Provider (Phoenix GRC). Você pode gerá-los usando OpenSSL:
    ```bash
    # 1. Gerar chave privada RSA de 2048 bits
    openssl genpkey -algorithm RSA -out saml_sp_private.key -pkeyopt rsa_keygen_bits:2048
    # 2. Gerar um certificado público auto-assinado a partir da chave privada (válido por 10 anos)
    openssl req -new -x509 -key saml_sp_private.key -out saml_sp_public.crt -days 3650 -subj "/CN=PhoenixGRC_SP_Local"
    ```
    Copie o conteúdo de `saml_sp_private.key` para a variável `SAML_SP_KEY_PEM` e o conteúdo de `saml_sp_public.crt` para `SAML_SP_CERT_PEM` no seu arquivo `.env`. Certifique-se de formatar corretamente as strings PEM multi-linha no arquivo `.env` (geralmente substituindo novas linhas por `\n` ou usando aspas apropriadas se o seu parser `.env` suportar).

3.  **Construa as Imagens Docker:**
    Este comando irá construir a imagem Docker para o backend.
    ```bash
    docker-compose build
    ```

4.  **Execute o Script de Instalação Interativo (Primeira Vez):**
    O script de instalação configura o banco de dados, executa migrações e cria o primeiro usuário administrador.
    Execute o seguinte comando para rodar o setup de forma interativa:
    ```bash
    docker-compose run --rm backend setup
    ```
    Siga as instruções no terminal:
    *   **Database Host:** `db` (nome do serviço PostgreSQL no `docker-compose.yml`)
    *   **Database Port:** `5432` (porta padrão do PostgreSQL)
    *   **Database User:** O valor de `POSTGRES_USER` do seu `.env` (padrão: `admin`)
    *   **Database Password:** O valor de `POSTGRES_PASSWORD` do seu `.env` (padrão: `password123`)
    *   **Database Name:** O valor de `POSTGRES_DB` do seu `.env` (padrão: `phoenix_grc_dev`)
    *   **Database SSL Mode:** `disable` (para desenvolvimento local)
    *   Nome para a primeira organização.
    *   Nome, email e senha para o usuário administrador.

    O contêiner `backend` finalizará após o script de setup.

### Iniciando o Servidor Backend

Após o setup inicial, você pode iniciar o servidor backend e o banco de dados com:

```bash
docker-compose up
```

O servidor backend estará acessível em `http://localhost:PORTA` (onde `PORTA` é o valor de `SERVER_PORT` no seu `.env`, padrão: `8080`).

## Endpoints da API (Backend)

A API está versionada sob `/api/v1`. Rotas dentro deste grupo requerem autenticação JWT.

### Autenticação

*   **`POST /auth/login`**: Login de usuário.
    *   **Payload:** `{ "email": "user@example.com", "password": "yourpassword" }`
    *   **Resposta (Sucesso - 200 OK):**
        ```json
        {
            "token": "jwt.token.string",
            "user_id": "uuid-string",
            "email": "user@example.com",
            "name": "User Name",
            "role": "admin", // ou manager, user
            "organization_id": "org-uuid-string"
        }
        ```
    *   **Resposta (Erro):** Status `400`, `401` ou `500` com mensagem de erro.

*   **SAML 2.0 Login (Iniciação pelo SP):**
    *   **`GET /auth/saml/{idpId}/login`**: Redireciona o usuário para o IdP SAML configurado para iniciar o login. `{idpId}` é o ID do `IdentityProvider` configurado no sistema.
*   **SAML 2.0 SP Metadata:**
    *   **`GET /auth/saml/{idpId}/metadata`**: Expõe os metadados do Service Provider (Phoenix GRC) para o IdP SAML especificado.
*   **SAML 2.0 Assertion Consumer Service (ACS):**
    *   **`POST /auth/saml/{idpId}/acs`**: Endpoint para onde o IdP SAML redireciona o usuário com a asserção SAML após o login bem-sucedido. O backend processa a asserção, provisiona/loga o usuário e emite um token JWT do Phoenix GRC, redirecionando para `FRONTEND_SAML_CALLBACK_URL`.

*   **OAuth2 Login (Exemplo Google - Iniciação pelo SP):**
    *   **`GET /auth/oauth2/google/{idpId}/login`**: Redireciona o usuário para a página de autorização do Google. `{idpId}` é o ID do `IdentityProvider` configurado para Google.
*   **OAuth2 Callback (Exemplo Google):**
    *   **`GET /auth/oauth2/google/{idpId}/callback`**: Endpoint para onde o Google redireciona após a autorização do usuário. O backend troca o código por um token, obtém informações do usuário, provisiona/loga o usuário, emite um token JWT do Phoenix GRC e redireciona para `FRONTEND_OAUTH2_CALLBACK_URL`.

### Health Check

*   **`GET /health`**: Verifica a saúde do servidor e a conexão com o banco de dados.
    *   **Resposta (Sucesso - 200 OK):** `{ "status": "ok", "database": "connected" }`

### API Protegida (`/api/v1`)

Para acessar os endpoints abaixo, inclua o token JWT no header `Authorization`:
`Authorization: Bearer <seu_token_jwt>`

#### Exemplo: Teste de Autenticação

*   **`GET /api/v1/me`**: Retorna informações do usuário autenticado.
    *   **Resposta (Sucesso - 200 OK):**
        ```json
        {
            "message": "This is a protected route",
            "user_id": "uuid-string",
            "email": "user@example.com",
            "role": "admin",
            "organization_id": "org-uuid-string"
        }
        ```

#### Gestão de Riscos (`/api/v1/risks`)

*   **`POST /api/v1/risks`**: Cria um novo risco.
    *   **Payload:**
        ```json
        {
            "title": "Novo Risco de Teste",
            "description": "Descrição detalhada do risco.",
            "category": "tecnologico", // "operacional", "legal"
            "impact": "medio", // "baixo", "alto", "critico"
            "probability": "baixa", // "media", "alta", "critica"
            "status": "aberto", // "em_andamento", "mitigado", "aceito"
            "owner_id": "uuid-do-usuario-owner" // Opcional, se não informado, o criador é o owner
        }
        ```
    *   **Resposta (Sucesso - 201 Created):** Objeto do risco criado.

*   **`GET /api/v1/risks`**: Lista todos os riscos da organização do usuário autenticado.
    *   **Resposta (Sucesso - 200 OK):** Array de objetos de risco.

*   **`GET /api/v1/risks/{riskId}`**: Obtém um risco específico pelo ID.
    *   **Resposta (Sucesso - 200 OK):** Objeto do risco.

*   **`PUT /api/v1/risks/{riskId}`**: Atualiza um risco existente.
    *   **Payload:** Similar ao de criação.
    *   **Resposta (Sucesso - 200 OK):** Objeto do risco atualizado.

*   **`DELETE /api/v1/risks/{riskId}`**: Deleta um risco.
    *   **Resposta (Sucesso - 200 OK):** `{ "message": "Risk deleted successfully" }`

#### Gestão de Provedores de Identidade (`/api/v1/organizations/{orgId}/identity-providers`)
Endpoints para administradores de organização gerenciarem configurações de SSO SAML e Social Login (OAuth2). Requer autenticação como admin da `{orgId}`.

*   **`POST /api/v1/organizations/{orgId}/identity-providers`**: Adiciona um novo provedor de identidade.
    *   **Payload:**
        ```json
        {
            "provider_type": "saml", // ou "oauth2_google", "oauth2_github"
            "name": "Meu IdP SAML Corporativo",
            "is_active": true,
            "config_json": { // Estrutura varia conforme provider_type
                // Para SAML:
                "idp_entity_id": "http://idp.example.com/entity",
                "idp_sso_url": "http://idp.example.com/sso",
                "idp_x509_cert": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
                "sign_request": true, // opcional
                "want_assertions_signed": true // opcional
                // Para OAuth2 (ex: Google):
                // "client_id": "YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com",
                // "client_secret": "YOUR_GOOGLE_CLIENT_SECRET",
                // "scopes": ["email", "profile"] // opcional
            },
            "attribute_mapping_json": { // Opcional
                "email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
                "name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name"
            }
        }
        ```
    *   **Resposta (Sucesso - 201 Created):** Objeto do provedor de identidade criado.

*   **`GET /api/v1/organizations/{orgId}/identity-providers`**: Lista todos os provedores de identidade da organização.
*   **`GET /api/v1/organizations/{orgId}/identity-providers/{idpId}`**: Obtém um provedor específico.
*   **`PUT /api/v1/organizations/{orgId}/identity-providers/{idpId}`**: Atualiza um provedor. (Payload similar ao POST).
*   **`DELETE /api/v1/organizations/{orgId}/identity-providers/{idpId}`**: Remove um provedor.

#### Gestão de Vulnerabilidades (`/api/v1/vulnerabilities`)
Endpoints para gerenciar vulnerabilidades dentro da organização do usuário autenticado.

*   **`POST /api/v1/vulnerabilities`**: Cria uma nova vulnerabilidade.
    *   **Payload:**
        ```json
        {
            "title": "Vulnerabilidade de Exemplo XSS",
            "description": "Entrada não sanitizada no campo de busca permite XSS.",
            "cve_id": "CVE-2023-99999", // Opcional
            "severity": "alto", // "baixo", "medio", "critico"
            "status": "descoberta", // "em_correcao", "corrigida"
            "asset_affected": "Página de Busca do Portal Principal"
        }
        ```
    *   **Resposta (Sucesso - 201 Created):** Objeto da vulnerabilidade criada.

*   **`GET /api/v1/vulnerabilities`**: Lista todas as vulnerabilidades da organização.
    *   **Resposta (Sucesso - 200 OK):** Array de objetos de vulnerabilidade.

*   **`GET /api/v1/vulnerabilities/{vulnId}`**: Obtém uma vulnerabilidade específica.
    *   **Resposta (Sucesso - 200 OK):** Objeto da vulnerabilidade.

*   **`PUT /api/v1/vulnerabilities/{vulnId}`**: Atualiza uma vulnerabilidade existente.
    *   **Payload:** Similar ao de criação.
    *   **Resposta (Sucesso - 200 OK):** Objeto da vulnerabilidade atualizada.

*   **`DELETE /api/v1/vulnerabilities/{vulnId}`**: Deleta uma vulnerabilidade.
    *   **Resposta (Sucesso - 200 OK):** `{ "message": "Vulnerability deleted successfully" }`

### Exemplo de Uso com `curl`

1.  **Login para obter o token:**
    ```bash
    curl -X POST -H "Content-Type: application/json" \
    -d '{"email":"seu_admin_email@example.com","password":"sua_senha"}' \
    http://localhost:8080/auth/login
    ```
    Copie o valor do campo `token` da resposta.

2.  **Acessar uma rota protegida (ex: listar riscos):**
    Substitua `SEU_TOKEN_AQUI` pelo token obtido.
    ```bash
    curl -X GET -H "Authorization: Bearer SEU_TOKEN_AQUI" http://localhost:8080/api/v1/risks
    ```

## Estrutura do Projeto

```
.
├── .env.example            # Exemplo de variáveis de ambiente
├── Dockerfile.backend      # Dockerfile para a aplicação Go (backend)
├── README.md               # Este arquivo
├── AGENTS.md               # Instruções para agentes de IA
├── backend/
│   ├── cmd/server/main.go  # Ponto de entrada (servidor Gin e comando de setup)
│   ├── internal/
│   │   ├── auth/           # Lógica de autenticação JWT (geração, validação, middleware)
│   │   ├── database/       # Conexão com DB e migrações GORM
│   │   ├── handlers/       # Handlers HTTP (controladores) para Gin (auth, risks, identity providers)
│   │   ├── models/         # Structs GORM (schema do DB, incluindo IdentityProvider)
│   │   ├── samlauth/       # Lógica específica para autenticação SAML 2.0
│   │   ├── oauth2auth/     # Lógica específica para autenticação OAuth2 (ex: Google)
│   │   └── ...
│   ├── pkg/                # Pacotes Go reutilizáveis (se houver)
│   ├── go.mod
│   └── go.sum
├── frontend/               # Código fonte do frontend (Next.js - a ser desenvolvido)
│   └── ...
└── docker-compose.yml      # Orquestração dos contêineres Docker
```

## Próximos Passos (Desenvolvimento Futuro)

*   Desenvolvimento do Frontend Next.js.
*   Implementação dos demais módulos (Vulnerabilidades, Auditoria, etc.).
*   Testes de integração e E2E.
*   Melhorias na paginação e filtros da API.
*   Configuração de logging mais robusto.
*   ...e muito mais!

## Contribuindo

Detalhes sobre como contribuir serão adicionados futuramente.
