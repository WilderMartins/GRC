# Phoenix GRC - Sistema de Gestão de Governança, Risco e Conformidade de TI

Phoenix GRC é uma plataforma SaaS (Software as a Service), whitelabel, projetada para ser intuitiva, escalável e segura. O objetivo é permitir que equipes de TI e segurança de todos os tamanhos gerenciem riscos, respondam a vulnerabilidades e demonstrem conformidade com os principais frameworks do setor de forma eficiente.

## Stack Tecnológica

- **Backend:** Go (Golang) com Gin Gonic
- **Frontend:** Next.js com TypeScript (em desenvolvimento, funcionalidades principais implementadas)
- **Banco de Dados:** PostgreSQL 16
- **ORM (Go):** GORM
- **Autenticação:** JWT (JSON Web Tokens), OAuth2 (Google, GitHub), MFA (TOTP, Códigos de Backup). SAML (desativado temporariamente).
- **UI (Frontend):** Tailwind CSS
- **Containerização:** Docker

## Ambiente de Desenvolvimento com Docker

Este projeto utiliza Docker e Docker Compose para facilitar a configuração do ambiente de desenvolvimento.

### Pré-requisitos

- Docker: [Instruções de Instalação](https://docs.docker.com/get-docker/)
- Docker Compose: (Normalmente incluído com Docker Desktop) [Instruções de Instalação](https://docs.docker.com/compose/install/)
- Git: [Instruções de Instalação](https://git-scm.com/downloads)
- `curl` ou um cliente API como Postman/Insomnia (para interagir com a API).

### Configuração Inicial e Setup

Recomendamos usar o Wizard de Instalação via Browser para a primeira configuração.

#### Método 1: Wizard de Instalação via Browser (Recomendado)

1.  **Clone o Repositório:**
    ```bash
    git clone <url_do_repositorio>
    cd phoenix-grc # Ou o nome do diretório do projeto
    ```

2.  **Configure as Variáveis de Ambiente Essenciais:**
    Copie o arquivo de exemplo `.env.example` para `.env`.
    ```bash
    cp .env.example .env
    ```
    Edite o arquivo `.env` e configure **inicialmente APENAS as seguintes seções**:
    *   **Conexão com o Banco de Dados:**
        *   `POSTGRES_HOST` (ex: `db` se usando o docker-compose padrão, ou `localhost` se o DB estiver rodando no host)
        *   `POSTGRES_PORT` (ex: `5432`)
        *   `POSTGRES_USER` (ex: `admin`)
        *   `POSTGRES_PASSWORD` (ex: `password123`)
        *   `POSTGRES_DB` (ex: `phoenix_grc_dev`)
        *   `POSTGRES_SSLMODE` (ex: `disable` para desenvolvimento local)
        *   `POSTGRES_SSLMODE_ENABLE` (bool, ex: `false` para desabilitar SSL via DSN, `true` para habilitar. Usado por `config.Cfg.EnableDBSSL`. `POSTGRES_SSLMODE` ainda é usado para construir a DSN string.)
    *   **Configurações Essenciais do Servidor e Segurança:**
        *   `GIN_MODE` (ex: `debug` para desenvolvimento, `release` para produção)
        *   `SERVER_PORT` (opcional, padrão `8080`. Porta interna do container backend)
        *   `NGINX_PORT` (opcional, padrão `80`. Porta externa exposta pelo Nginx)
        *   `APP_ROOT_URL` (ex: `http://localhost:80` ou `http://localhost:${NGINX_PORT}`. URL base da aplicação como vista pelo backend, usada para gerar links SAML/OAuth etc.)
        *   `FRONTEND_BASE_URL` (ex: `http://localhost:3000` em dev, ou `http://localhost:${NGINX_PORT}` se servido pelo Nginx. Usada para links em emails/notificações.)
        *   `JWT_SECRET_KEY`: String longa, aleatória e segura para assinar tokens JWT. **OBRIGATÓRIO.**
        *   `JWT_TOKEN_LIFESPAN_HOURS` (opcional, padrão `24`)
        *   `ENCRYPTION_KEY_HEX`: Chave hexadecimal de 64 caracteres (32 bytes) para criptografia AES-256 (ex: segredos TOTP). **OBRIGATÓRIO PARA PRODUÇÃO.** Use um gerador seguro.
    *   **Configurações de Autenticação Externa (Opcional, configurar via UI/variáveis após setup inicial se necessário):**
        *   `SAML_SP_KEY_PEM`, `SAML_SP_CERT_PEM`: Chaves para SAML SP (se SAML for ativado).
        *   `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`: Para login com Google.
        *   `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`: Para login com GitHub.
    *   **Configurações de Armazenamento de Arquivos (Opcional):**
        *   `FILE_STORAGE_PROVIDER` (opcional, padrão `gcs`. Pode ser `s3` ou `gcs`)
        *   `GCS_PROJECT_ID`, `GCS_BUCKET_NAME` (se usar GCS)
        *   `AWS_S3_BUCKET` (se usar S3, `AWS_REGION` também é necessário)
    *   **Configurações de Email (Opcional, para notificações):**
        *   `AWS_REGION` (se usar AWS SES)
        *   `AWS_SES_EMAIL_SENDER` (email remetente verificado no SES)
    *   **Outras Configurações:**
        *   `TOTP_ISSUER_NAME` (opcional, padrão `PhoenixGRC`. Nome que aparece no app autenticador)

    **Importante:** O Wizard de Instalação (`/setup` no browser) ou o setup via CLI (`docker-compose run --rm backend setup`) cuidará da criação da primeira organização e do usuário administrador. Para o Wizard via browser, apenas as variáveis de conexão com o banco de dados (`POSTGRES_*`) e `JWT_SECRET_KEY`, `APP_ROOT_URL`, `FRONTEND_BASE_URL`, `ENCRYPTION_KEY_HEX` precisam ser configuradas inicialmente no arquivo `.env`. O restante pode ser configurado depois.

    **Nota sobre SAML:** A funcionalidade SAML está atualmente desativada no código devido a desafios técnicos. As variáveis `SAML_SP_KEY_PEM` e `SAML_SP_CERT_PEM` são listadas para referência futura.

3.  **Construa e Inicie os Containers Docker:**
    ```bash
    docker-compose build
    docker-compose up -d # O -d executa em segundo plano
    ```
    Aguarde alguns instantes para os serviços iniciarem.

4.  **Acesse o Wizard de Instalação no Navegador:**
    Abra seu navegador e acesse a URL base da sua aplicação frontend (normalmente `http://localhost:PORTA_DO_FRONTEND` se rodando o frontend localmente, ou `http://localhost:NGINX_PORT` se o Nginx estiver configurado para servir o frontend ou se o frontend e backend estiverem na mesma porta via proxy).
    *   Se o sistema detectar que a configuração inicial não foi concluída, você deverá ser redirecionado para o wizard de instalação (ex: `/setup`) ou verá um link para iniciá-lo.
    *   Siga as instruções na tela:
        *   **Bem-vindo:** Introdução ao processo.
        *   **Verificação da Configuração do Banco de Dados:** O wizard confirmará que o backend conseguiu se conectar ao banco de dados usando as configurações do seu arquivo `.env`. Se houver erro, revise seu `.env` e **REINICIE os containers (`docker-compose restart backend` ou `docker-compose down && docker-compose up -d`)** para que o backend carregue as novas variáveis.
        *   **Executar Migrações:** Um botão para criar as tabelas no banco de dados.
        *   **Criar Administrador:** Formulário para definir o nome da sua organização e criar a primeira conta de administrador.
        *   **Conclusão:** Confirmação e link para a página de login.

5.  **Após o Wizard:**
    *   Acesse a página de login e utilize as credenciais do administrador criadas durante o wizard.
    *   Você pode então configurar o restante das variáveis no arquivo `.env` (SAML, GCS, SES, etc.) conforme necessário e reiniciar os containers para que as novas configurações tenham efeito.

#### Método 2: Setup via Linha de Comando (CLI) - Para Usuários Avançados

Este método permite configurar o sistema interativamente através do terminal. Certifique-se de ter configurado **TODAS** as variáveis de ambiente necessárias no seu arquivo `.env` primeiro, incluindo as de banco de dados e `JWT_SECRET_KEY`.

1.  **Clone o Repositório e Configure o `.env` Completo:** (Conforme passos 1 e 2 do método do Wizard, mas preenchendo todas as variáveis relevantes do `.env.example`).

2.  **Construa as Imagens Docker:**
    ```bash
    docker-compose build
    ```

3.  **Execute o Script de Instalação Interativo via CLI:**
    ```bash
    docker-compose run --rm backend setup
    ```
    Siga as instruções no terminal. Você será solicitado a fornecer:
    *   Credenciais do banco de dados (mesmo que já estejam no `.env`, o script CLI pode pedi-las novamente para confirmar ou se o script não ler diretamente todas as vars do .env para este fluxo específico).
    *   Nome para a primeira organização.
    *   Nome, email e senha para o usuário administrador.

    O contêiner `backend` finalizará após o script de setup.

4.  **Inicie os Servidores:**
    ```bash
    docker-compose up
    ```

### Iniciando a Aplicação (Após Setup)

Se o setup (via Wizard ou CLI) foi concluído com sucesso, para iniciar a aplicação normalmente:

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

*   **Autenticação 2FA (Dois Fatores):**
    *   Após o login com senha bem-sucedido, se o 2FA (TOTP) estiver habilitado para o usuário, a API retornará `{"2fa_required": true, "user_id": "..."}`.
    *   O frontend deverá então solicitar o código TOTP ou um código de backup.
    *   **`POST /auth/login/2fa/verify`**: Para verificar um código TOTP.
    *   **`POST /auth/login/2fa/backup-code/verify`**: Para verificar um código de backup.
    *   Ambos retornam o token JWT completo em caso de sucesso.

*   **SAML 2.0 (Funcionalidade Futura / Temporariamente Desativada):**
    *   A integração com SAML 2.0 está planejada mas encontra-se temporariamente desativada devido a desafios técnicos.
    *   Os endpoints teóricos seriam:
        *   `GET /auth/saml/{idpId}/login`
        *   `GET /auth/saml/{idpId}/metadata`
        *   `POST /auth/saml/{idpId}/acs`

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
            "impact": "Médio", // "Baixo", "Alto", "Crítico"
            "probability": "Baixo", // "Médio", "Alto", "Crítico"
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
            "severity": "Alto", // "Baixo", "Médio", "Crítico"
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

#### Auditoria e Conformidade (`/api/v1/audit`)
Endpoints para listar frameworks de auditoria, seus controles, e gerenciar avaliações de conformidade.

*   **`GET /api/v1/audit/frameworks`**: Lista todos os frameworks de auditoria pré-carregados (ex: NIST CSF, ISO 27001).
    *   **Resposta (Sucesso - 200 OK):** Array de objetos `AuditFramework`.

*   **`GET /api/v1/audit/frameworks/{frameworkId}/controls`**: Lista todos os controles para um framework específico.
    *   **Resposta (Sucesso - 200 OK):** Array de objetos `AuditControl`.

*   **`POST /api/v1/audit/assessments`**: Cria ou atualiza uma avaliação para um controle específico dentro da organização do usuário autenticado.
    *   **Tipo de Conteúdo da Requisição:** `multipart/form-data`.
    *   A organização é inferida a partir do token JWT.
    *   Este endpoint realiza um "upsert": se uma avaliação para o `audit_control_id` e organização já existir, ela é atualizada; caso contrário, uma nova é criada.
    *   **Campos do Formulário:**
        *   `data` (obrigatório): Um JSON string contendo os dados da avaliação:
            ```json
            {
                "audit_control_id": "uuid-do-audit-control", // ID (UUID) do AuditControl
                "status": "conforme", // "nao_conforme", "parcialmente_conforme"
                "evidence_url": "http://example.com/link-externo.pdf", // Opcional, usado se nenhum arquivo for enviado
                "score": 100, // Opcional, entre 0-100. Pode ser inferido do status se não fornecido.
                "assessment_date": "2023-10-26" // Opcional, YYYY-MM-DD. Default para data atual se não fornecido.
            }
            ```
        *   `evidence_file` (opcional): O arquivo de evidência. Se fornecido, a URL deste arquivo (após upload para GCS) substituirá qualquer `evidence_url` fornecida no JSON `data`.
    *   **Resposta (Sucesso - 200 OK):** Objeto da `AuditAssessment` criada ou atualizada, com `EvidenceURL` apontando para o arquivo no GCS se um upload foi feito.

*   **`GET /api/v1/audit/assessments/control/{controlId}`**: Obtém a avaliação de um controle específico (`controlId` é o UUID do `AuditControl`) para a organização do usuário autenticado.
    *   **Resposta (Sucesso - 200 OK):** Objeto `AuditAssessment`.
    *   **Resposta (404 Not Found):** Se nenhuma avaliação existir para o controle na organização.

*   **`GET /api/v1/audit/organizations/{orgId}/frameworks/{frameworkId}/assessments`**: Lista todas as avaliações de uma organização específica para um determinado framework.
    *   Requer que o usuário autenticado pertença à `{orgId}` ou seja um superadmin (lógica de superadmin não implementada).
    *   **Resposta (Sucesso - 200 OK):** Array de objetos `AuditAssessment`, cada um incluindo detalhes do `AuditControl` associado. (Paginado)

#### Gerenciamento de Usuários da Organização (`/api/v1/organizations/{orgId}/users`)
Endpoints para administradores de organização gerenciarem usuários. Requer role `admin` ou `manager` da organização.

*   **`GET /api/v1/organizations/{orgId}/users`**: Lista todos os usuários da organização.
    *   **Query Params:** `page`, `page_size` para paginação.
    *   **Resposta (Sucesso - 200 OK):** Resposta paginada (`PaginatedResponse`) com `items` contendo `UserResponse` (sem `PasswordHash`).

*   **`GET /api/v1/organizations/{orgId}/users/{userId}`**: Obtém detalhes de um usuário específico da organização.
    *   **Resposta (Sucesso - 200 OK):** Objeto `UserResponse`.

*   **`PUT /api/v1/organizations/{orgId}/users/{userId}/role`**: Atualiza a role de um usuário.
    *   **Payload:** `{ "role": "admin" | "manager" | "user" }`
    *   **Resposta (Sucesso - 200 OK):** Objeto `UserResponse` atualizado.
    *   **Nota:** Contém lógica para prevenir auto-rebaixamento do último admin/manager.

*   **`PUT /api/v1/organizations/{orgId}/users/{userId}/status`**: Ativa ou desativa um usuário.
    *   **Payload:** `{ "is_active": true | false }`
    *   **Resposta (Sucesso - 200 OK):** Objeto `UserResponse` atualizado.
    *   **Nota:** Contém lógica para prevenir auto-desativação do último admin/manager ativo.

#### Workflow de Aceite de Risco (`/api/v1/risks/{riskId}/...`)
Endpoints para gerenciar o processo de aprovação para aceite de riscos.

*   **`POST /api/v1/risks/{riskId}/submit-acceptance`**: Submete um risco (identificado por `{riskId}`) para aprovação de aceite.
    *   Requer que o usuário autenticado tenha a role `manager` ou `admin`.
    *   O risco deve ter um `OwnerID` (proprietário) definido, que será o aprovador.
    *   Cria um registro `ApprovalWorkflow` com status `pendente`.
    *   **Resposta (Sucesso - 201 Created):** Objeto do `ApprovalWorkflow` criado.
    *   **Resposta (Erro - 403 Forbidden):** Se o usuário não for manager/admin.
    *   **Resposta (Erro - 400 Bad Request):** Se o risco não tiver proprietário.
    *   **Resposta (Erro - 404 Not Found):** Se o risco não for encontrado.
    *   **Resposta (Erro - 409 Conflict):** Se já existir um workflow de aprovação pendente para este risco.

*   **`POST /api/v1/risks/{riskId}/approval/{approvalId}/decide`**: Registra uma decisão (aprovar/rejeitar) para um workflow de aceite de risco.
    *   Requer que o usuário autenticado seja o `ApproverID` (proprietário do risco) do `ApprovalWorkflow` especificado por `{approvalId}`.
    *   O `{riskId}` na URL é para contexto e verificação.
    *   **Payload:**
        ```json
        {
            "decision": "aprovado", // ou "rejeitado"
            "comments": "Comentários sobre a decisão." // Opcional
        }
        ```
    *   **Resposta (Sucesso - 200 OK):** Objeto do `ApprovalWorkflow` atualizado.
    *   **Resposta (Erro - 403 Forbidden):** Se o usuário não for o aprovador designado.
    *   **Resposta (Erro - 404 Not Found):** Se o workflow de aprovação não for encontrado.
    *   **Resposta (Erro - 409 Conflict):** Se o workflow já tiver sido decidido.

*   **`GET /api/v1/risks/{riskId}/approval-history`**: Lista o histórico de todos os workflows de aprovação para um risco específico.
    *   Requer que o usuário pertença à organização do risco.
    *   **Resposta (Sucesso - 200 OK):** Array de objetos `ApprovalWorkflow`, com detalhes do requisitante e aprovador.

#### Upload em Massa de Riscos (`/api/v1/risks/bulk-upload-csv`)

*   **`POST /api/v1/risks/bulk-upload-csv`**: Permite o upload de múltiplos riscos através de um arquivo CSV.
    *   **Tipo de Conteúdo da Requisição:** `multipart/form-data`.
    *   **Campo do Formulário:** O arquivo CSV deve ser enviado no campo `file`.
    *   **Formato CSV Esperado:**
        *   **Cabeçalhos (obrigatórios em qualquer ordem, case-insensitive):** `title`, `impact`, `probability`.
        *   **Cabeçalhos (opcionais):** `description`, `category`.
        *   **Valores:**
            *   `title`: string (3-255 caracteres).
            *   `description`: string.
            *   `category`: "tecnologico", "operacional", "legal" (default: "tecnologico" se inválido/ausente).
            *   `impact`: "Baixo", "Médio", "Alto", "Crítico".
            *   `probability`: "Baixo", "Médio", "Alto", "Crítico".
    *   **Lógica:**
        *   O `OrganizationID` é inferido do token JWT.
        *   O `OwnerID` dos riscos criados é o `UserID` do usuário que fez o upload.
        *   O `Status` inicial é "aberto".
    *   **Resposta:**
        *   **`200 OK`:** Se todos os riscos válidos foram importados com sucesso e não houve linhas com erro.
            ```json
            {
                "successfully_imported": 10,
                "failed_rows": []
            }
            ```
        *   **`207 Multi-Status`:** Se alguns riscos foram importados e outros falharam na validação.
            ```json
            {
                "successfully_imported": 8,
                "failed_rows": [
                    { "line_number": 3, "errors": ["title is required"] },
                    { "line_number": 5, "errors": ["invalid impact value: 'Muito Alto'. Valid are: Baixo, Médio, Alto, Crítico."] }
                ]
            }
            ```
        *   **`400 Bad Request`:** Para erros como arquivo CSV vazio, cabeçalhos obrigatórios ausentes, ou se todas as linhas de dados forem inválidas.
            ```json
            {
                "successfully_imported": 0,
                "failed_rows": [...], // Se aplicável
                "general_error": "Missing required CSV header: impact" // Exemplo
            }
            ```
        *   **`500 Internal Server Error`:** Para erros no servidor durante o processamento.

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
│   │   ├── handlers/       # Handlers HTTP (controladores) para Gin (auth, risks, identity providers, webhooks, audit, vulnerabilities, user)
│   │   ├── models/         # Structs GORM (schema do DB)
│   │   ├── notifications/  # Lógica para Webhooks e Email (SES)
│   │   ├── samlauth/       # Lógica específica para autenticação SAML 2.0 (atualmente desativado)
│   │   ├── oauth2auth/     # Lógica específica para autenticação OAuth2 (Google, GitHub)
│   │   ├── filestorage/    # Lógica para armazenamento de arquivos (GCS, S3)
│   │   ├── seeders/        # Seeders de dados (ex: AuditFrameworks)
│   │   ├── utils/          # Utilitários gerais (ex: criptografia)
│   │   └── ...
│   ├── pkg/                # Pacotes Go reutilizáveis (config, logger, features)
│   ├── go.mod
│   └── go.sum
├── frontend/               # Código fonte do frontend (Next.js com TypeScript)
│   └── ...
└── docker-compose.yml      # Orquestração dos contêineres Docker
```

## Próximos Passos (Desenvolvimento Futuro)

*   Finalização e polimento do Frontend Next.js.
*   Reativação e finalização da funcionalidade SAML.
*   Implementação da funcionalidade de exclusão de arquivos no `filestorage`.
*   Testes de integração e E2E abrangentes.
*   Melhorias na paginação e filtros da API.
*   Configuração de logging mais robusto.
*   ...e muito mais!

## Contribuindo

Detalhes sobre como contribuir serão adicionados futuramente.
