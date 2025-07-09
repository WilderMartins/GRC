# Documentação da API Phoenix GRC (Backend)

## Visão Geral

Esta documentação descreve os endpoints da API para o sistema Phoenix GRC.
A API é versionada e todos os endpoints protegidos estão sob o prefixo `/api/v1`.

**URL Base (Exemplo Local):** `http://localhost:80` (via Nginx) ou `http://localhost:8080` (direto no backend)

**Autenticação:** Endpoints sob `/api/v1` requerem um token JWT no header `Authorization`:
`Authorization: Bearer <seu_token_jwt>`

## Endpoints

### 1. Saúde do Sistema

*   **`GET /health`**
    *   **Descrição:** Verifica a saúde do servidor e a conexão com o banco de dados.
    *   **Autenticação:** Nenhuma.
    *   **Respostas:**
        *   `200 OK`: `{ "status": "ok", "database": "connected" }`
        *   `503 Service Unavailable`: Se o banco de dados não estiver acessível.

### 2. Autenticação (`/auth`)

*   **`POST /auth/login`**
    *   **Descrição:** Realiza o login de um usuário com email e senha. Pode requerer um segundo fator se o 2FA estiver habilitado.
    *   **Autenticação:** Nenhuma.
    *   **Payload da Requisição (`application/json`):**
        ```json
        {
            "email": "user@example.com", // string, obrigatório, formato email
            "password": "yourpassword"   // string, obrigatório
        }
        ```
    *   **Respostas:**
        *   `200 OK` (Login direto ou 2FA necessário):
            *   Se 2FA não habilitado:
                ```json
                {
                    "token": "jwt.token.string",
                    "user_id": "uuid-string",
                    "email": "user@example.com",
                    "name": "User Name",
                    "role": "admin", // ou "manager", "user"
                    "organization_id": "org-uuid-string"
                }
                ```
            *   Se 2FA habilitado:
                ```json
                {
                    "2fa_required": true,
                    "user_id": "uuid-string",
                    "message": "Password verified. Please provide TOTP token."
                }
                ```
        *   `400 Bad Request`: Payload inválido. Ex: `{ "error": "Invalid request payload: ..." }`
        *   `401 Unauthorized`: Email/senha inválidos ou usuário inativo. Ex: `{ "error": "Invalid email or password" }` ou `{ "error": "User account is inactive" }`
        *   `500 Internal Server Error`: Falha ao gerar token.

*   **`POST /auth/login/2fa/verify`**
    *   **Descrição:** Verifica o código TOTP fornecido pelo usuário como segundo fator de autenticação.
    *   **Autenticação:** Nenhuma (parte do fluxo de login multi-etapa).
    *   **Payload da Requisição (`application/json`):**
        ```json
        {
            "user_id": "uuid-string-do-usuario", // string, obrigatório (obtido da resposta do /auth/login)
            "token": "123456"                   // string, obrigatório (código TOTP)
        }
        ```
    *   **Respostas:**
        *   `200 OK` (Token JWT completo):
            ```json
            {
                "token": "jwt.token.string",
                "user_id": "uuid-string",
                "email": "user@example.com",
                "name": "User Name",
                "role": "admin",
                "organization_id": "org-uuid-string"
            }
            ```
        *   `400 Bad Request`: Payload inválido.
        *   `401 Unauthorized`: Token TOTP inválido, usuário não encontrado, ou TOTP não habilitado para o usuário.
        *   `500 Internal Server Error`: Falha ao gerar token JWT.

*   **`GET /auth/oauth2/google/:idpId/login`**
    *   **Descrição:** Inicia o fluxo de login OAuth2 com Google para o provedor de identidade (`idpId`) configurado. Redireciona o usuário para a página de autorização do Google.
    *   **Autenticação:** Nenhuma.
    *   **Parâmetros de Path:**
        *   `idpId` (string UUID): ID do `IdentityProvider` configurado para Google.
    *   **Respostas:**
        *   `302 Found`: Redirecionamento para o Google.
        *   `404 Not Found`: Configuração do IdP não encontrada.
        *   `500 Internal Server Error`: Falha ao configurar OAuth2 ou gerar estado.

*   **`GET /auth/oauth2/google/:idpId/callback`**
    *   **Descrição:** Endpoint de callback para o Google após autorização do usuário. Processa o código, obtém informações do usuário, provisiona/loga o usuário no Phoenix GRC, gera um token JWT e redireciona para o frontend.
    *   **Autenticação:** Nenhuma.
    *   **Parâmetros de Path:**
        *   `idpId` (string UUID): ID do `IdentityProvider`.
    *   **Query Params (enviados pelo Google):** `code`, `state`.
    *   **Respostas:**
        *   `302 Found`: Redirecionamento para `FRONTEND_OAUTH2_CALLBACK_URL` com o token JWT.
        *   `400 Bad Request`: Estado inválido, email não fornecido pelo Google.
        *   `401 Unauthorized`: Código de autorização não encontrado.
        *   `500 Internal Server Error`: Falha na troca de token, obtenção de user info, criação/atualização de usuário, ou geração de token JWT.

*   **`GET /auth/oauth2/github/:idpId/login`**
    *   **Descrição:** Inicia o fluxo de login OAuth2 com GitHub. Similar ao Google.
    *   **Autenticação:** Nenhuma.
    *   **Parâmetros de Path:** `idpId`.
    *   **Respostas:** Similar ao Google Login.

*   **`GET /auth/oauth2/github/:idpId/callback`**
    *   **Descrição:** Callback para GitHub. Similar ao Google Callback.
    *   **Autenticação:** Nenhuma.
    *   **Parâmetros de Path:** `idpId`.
    *   **Respostas:** Similar ao Google Callback.

---

### 3. Usuário Autenticado (`/api/v1`)

Estes endpoints operam no contexto do usuário autenticado via JWT.

*   **`GET /api/v1/me`**
    *   **Descrição:** Retorna informações sobre o usuário atualmente autenticado.
    *   **Autenticação:** JWT Obrigatório.
    *   **Respostas:**
        *   `200 OK`:
            ```json
            {
                "message": "This is a protected route", // Pode ser omitido ou alterado
                "user_id": "uuid-string",
                "email": "user@example.com",
                "role": "admin", // ou "manager", "user"
                "organization_id": "org-uuid-string"
            }
            ```
        *   `401 Unauthorized`: Token JWT ausente ou inválido.

---

### 4. Gestão de Riscos (`/api/v1/risks`)

Todos os endpoints nesta seção requerem autenticação JWT.

*   **`POST /api/v1/risks`**
    *   **Descrição:** Cria um novo risco. O `owner_id` é opcional; se não fornecido, o criador do risco se torna o proprietário. O `RiskLevel` é calculado automaticamente.
    *   **Payload da Requisição (`application/json`):**
        ```json
        {
            "title": "string (obrigatório, min 3, max 255)",
            "description": "string (opcional)",
            "category": "string (opcional, um de: tecnologico, operacional, legal)",
            "impact": "string (opcional, um de: Baixo, Médio, Alto, Crítico)",
            "probability": "string (opcional, um de: Baixo, Médio, Alto, Crítico)",
            "status": "string (opcional, um de: aberto, em_andamento, mitigado, aceito, default: aberto)",
            "owner_id": "string (UUID, opcional)"
        }
        ```
    *   **Respostas:**
        *   `201 Created`: Objeto do risco criado (inclui `id`, `created_at`, `updated_at`, `organization_id`, `risk_level`).
            ```json
            // Exemplo de models.Risk
            {
                "ID": "uuid-do-risco",
                "OrganizationID": "uuid-da-org",
                "Title": "Risco X",
                "Description": "...",
                "Category": "tecnologico",
                "Impact": "Alto",
                "Probability": "Médio",
                "RiskLevel": "Alto", // Calculado
                "Status": "aberto",
                "OwnerID": "uuid-do-owner",
                "CreatedAt": "timestamp",
                "UpdatedAt": "timestamp",
                "Owner": null // Ou objeto User se pré-carregado
            }
            ```
        *   `400 Bad Request`: Payload inválido.
        *   `500 Internal Server Error`: Falha ao criar o risco.

*   **`GET /api/v1/risks`**
    *   **Descrição:** Lista todos os riscos da organização do usuário autenticado, com paginação e filtros.
    *   **Query Params:**
        *   `page` (int, opcional, default: 1): Número da página.
        *   `page_size` (int, opcional, default: 10): Tamanho da página.
        *   `status` (string, opcional): Filtra por status do risco.
        *   `impact` (string, opcional): Filtra por impacto do risco.
        *   `probability` (string, opcional): Filtra por probabilidade do risco.
        *   `category` (string, opcional): Filtra por categoria do risco.
    *   **Respostas:**
        *   `200 OK`: Objeto de resposta paginada.
            ```json
            {
                "items": [ /* array de models.Risk (com Owner pré-carregado) */ ],
                "total_items": 150,
                "total_pages": 15,
                "page": 1,
                "page_size": 10
            }
            ```
        *   `500 Internal Server Error`: Falha ao listar riscos.

*   **`GET /api/v1/risks/:riskId`**
    *   **Descrição:** Obtém um risco específico pelo ID.
    *   **Parâmetros de Path:**
        *   `riskId` (string UUID): ID do risco.
    *   **Respostas:**
        *   `200 OK`: Objeto `models.Risk` (com `Owner` pré-carregado).
        *   `400 Bad Request`: Formato de `riskId` inválido.
        *   `404 Not Found`: Risco não encontrado ou não pertence à organização do usuário.
        *   `500 Internal Server Error`: Falha ao buscar o risco.

*   **`PUT /api/v1/risks/:riskId`**
    *   **Descrição:** Atualiza um risco existente. O `RiskLevel` é recalculado se impacto/probabilidade mudarem.
    *   **Parâmetros de Path:** `riskId`.
    *   **Payload da Requisição (`application/json`):** Similar ao `POST /api/v1/risks`.
    *   **Respostas:**
        *   `200 OK`: Objeto `models.Risk` atualizado (com `Owner` pré-carregado).
        *   `400 Bad Request`: Payload ou `riskId` inválido.
        *   `404 Not Found`: Risco não encontrado.
        *   `500 Internal Server Error`: Falha ao atualizar.
        *   *(TODO: Adicionar verificação de autorização - apenas owner ou admin/manager da org? Atualmente permite qualquer um da org)*

*   **`DELETE /api/v1/risks/:riskId`**
    *   **Descrição:** Deleta um risco.
    *   **Parâmetros de Path:** `riskId`.
    *   **Respostas:**
        *   `200 OK`: `{ "message": "Risk deleted successfully" }`
        *   `400 Bad Request`: `riskId` inválido.
        *   `404 Not Found`: Risco não encontrado.
        *   `500 Internal Server Error`: Falha ao deletar.
        *   *(TODO: Adicionar verificação de autorização)*

*   **`POST /api/v1/risks/bulk-upload-csv`**
    *   **Descrição:** Upload em massa de riscos via arquivo CSV.
    *   **Requisição:** `multipart/form-data` com um campo `file` contendo o CSV.
    *   **Formato CSV Esperado (Cabeçalhos obrigatórios: title, impact, probability. Opcionais: description, category):**
        ```csv
        title,description,category,impact,probability
        Risco A,"Desc A",tecnologico,Alto,Médio
        Risco B,"Desc B",operacional,Baixo,Baixo
        ```
    *   **Respostas:**
        *   `200 OK`: Se todos os riscos válidos foram importados e não houve erros.
            ```json
            { "successfully_imported": 10, "failed_rows": [] }
            ```
        *   `207 Multi-Status`: Se alguns riscos foram importados e outros falharam.
            ```json
            {
                "successfully_imported": 8,
                "failed_rows": [
                    { "line_number": 3, "errors": ["title is required"] },
                    { "line_number": 5, "errors": ["invalid impact value: 'Muito Alto'"] }
                ]
            }
            ```
        *   `400 Bad Request`: Erros gerais (arquivo vazio, cabeçalhos faltando, etc.).
        *   `500 Internal Server Error`: Falha no processamento.

#### 4.1. Stakeholders de Risco (`/api/v1/risks/:riskId/stakeholders`)

*   **`POST /api/v1/risks/:riskId/stakeholders`**
    *   **Descrição:** Adiciona um usuário como stakeholder a um risco.
    *   **Parâmetros de Path:** `riskId`.
    *   **Payload da Requisição (`application/json`):**
        ```json
        { "user_id": "string (UUID do usuário)" }
        ```
    *   **Respostas:**
        *   `201 Created`: `{ "message": "Stakeholder added successfully" }`
        *   `200 OK`: `{ "message": "Stakeholder association already exists." }`
        *   `400 Bad Request`: IDs ou payload inválidos.
        *   `404 Not Found`: Risco ou usuário não encontrado na organização.
        *   `500 Internal Server Error`.

*   **`GET /api/v1/risks/:riskId/stakeholders`**
    *   **Descrição:** Lista todos os stakeholders de um risco.
    *   **Parâmetros de Path:** `riskId`.
    *   **Respostas:**
        *   `200 OK`: Array de objetos `UserStakeholderResponse`.
            ```json
            [
                { "id": "uuid", "name": "Nome Stakeholder", "email": "stake@example.com", "role": "user" }
            ]
            ```
            (O DTO `UserStakeholderResponse` é: `{ id, name, email, role }`)
        *   `404 Not Found`: Risco não encontrado.
        *   `500 Internal Server Error`.

*   **`DELETE /api/v1/risks/:riskId/stakeholders/:userId`**
    *   **Descrição:** Remove um stakeholder de um risco.
    *   **Parâmetros de Path:** `riskId`, `userId` (ID do stakeholder a ser removido).
    *   **Respostas:**
        *   `200 OK`: `{ "message": "Stakeholder removed successfully" }`
        *   `404 Not Found`: Associação de stakeholder ou risco não encontrada.
        *   `500 Internal Server Error`.

#### 4.2. Workflow de Aceite de Risco (`/api/v1/risks/:riskId/...`)

*   **`POST /api/v1/risks/:riskId/submit-acceptance`**
    *   **Descrição:** Submete um risco para aprovação de aceite. Requer que o usuário seja Admin ou Manager da organização. O risco deve ter um proprietário (`OwnerID`) definido.
    *   **Parâmetros de Path:** `riskId`.
    *   **Respostas:**
        *   `201 Created`: Objeto `ApprovalWorkflow` criado.
        *   `400 Bad Request`: Risco sem proprietário.
        *   `403 Forbidden`: Usuário não autorizado.
        *   `404 Not Found`: Risco não encontrado.
        *   `409 Conflict`: Workflow de aprovação já pendente.
        *   `500 Internal Server Error`.

*   **`POST /api/v1/risks/:riskId/approval/:approvalId/decide`**
    *   **Descrição:** Registra uma decisão (aprovar/rejeitar) para um workflow de aceite. Requer que o usuário autenticado seja o `ApproverID` (proprietário do risco) do workflow.
    *   **Parâmetros de Path:** `riskId`, `approvalId`.
    *   **Payload da Requisição (`application/json`):**
        ```json
        {
            "decision": "string (aprovado ou rejeitado)",
            "comments": "string (opcional)"
        }
        ```
    *   **Respostas:**
        *   `200 OK`: Objeto `ApprovalWorkflow` atualizado. Se aprovado, o status do risco é mudado para "aceito".
        *   `403 Forbidden`: Usuário não é o aprovador.
        *   `404 Not Found`: Workflow não encontrado.
        *   `409 Conflict`: Workflow já decidido.
        *   `500 Internal Server Error`.

*   **`GET /api/v1/risks/:riskId/approval-history`**
    *   **Descrição:** Lista o histórico de workflows de aprovação para um risco.
    *   **Parâmetros de Path:** `riskId`.
    *   **Query Params:** `page`, `page_size`.
    *   **Respostas:**
        *   `200 OK`: Resposta paginada com array de `ApprovalWorkflow` (com `Requester` e `Approver` pré-carregados).
        *   `404 Not Found`: Risco não encontrado.
        *   `500 Internal Server Error`.

---

### 5. Organizações (`/api/v1/organizations/:orgId`)

Endpoints para gerenciar configurações e recursos no nível da organização.
O parâmetro de path `:orgId` refere-se ao ID da organização sendo gerenciada.
A autorização geralmente requer que o usuário autenticado pertença à organização e, para certas operações (como updates), tenha uma role de `admin` ou `manager` dentro dessa organização.

#### 5.1. Branding da Organização

*   **`PUT /api/v1/organizations/:orgId/branding`**
    *   **Descrição:** Atualiza as configurações de branding (logo, cores) da organização.
    *   **Autenticação:** JWT Obrigatório. Usuário deve ser Admin ou Manager da organização especificada por `:orgId`.
    *   **Requisição:** `multipart/form-data`.
        *   Campo `data` (string JSON): Contém `primary_color` e `secondary_color`.
            ```json
            {
                "primary_color": "#RRGGBB",   // string, opcional, formato HEX
                "secondary_color": "#RRGGBB"  // string, opcional, formato HEX
            }
            ```
        *   Campo `logo_file` (arquivo, opcional): Arquivo de imagem para o logo (JPEG, PNG, GIF, SVG, limite 2MB).
    *   **Respostas:**
        *   `200 OK`: Objeto `models.Organization` atualizado.
        *   `400 Bad Request`: Formato de ID inválido, JSON inválido, formato de cor inválido, arquivo de logo muito grande ou tipo não permitido.
        *   `403 Forbidden`: Usuário não autorizado.
        *   `404 Not Found`: Organização não encontrada.
        *   `500 Internal Server Error`: Falha no upload do arquivo ou ao salvar no banco.

*   **`GET /api/v1/organizations/:orgId/branding`**
    *   **Descrição:** Obtém as configurações de branding da organização.
    *   **Autenticação:** JWT Obrigatório. Usuário deve pertencer à organização especificada por `:orgId`.
    *   **Respostas:**
        *   `200 OK`:
            ```json
            {
                "id": "uuid-da-org",
                "name": "Nome da Organização",
                "logo_url": "url-do-logo.png",
                "primary_color": "#RRGGBB",
                "secondary_color": "#RRGGBB"
            }
            ```
        *   `403 Forbidden`: Usuário não autorizado.
        *   `404 Not Found`: Organização não encontrada.
        *   `500 Internal Server Error`.

#### 5.2. Provedores de Identidade (SSO/Social Login) (`/api/v1/organizations/:orgId/identity-providers`)

Gerencia configurações de SSO SAML e Social Login (OAuth2) para uma organização. Requer role de Admin ou Manager da organização.

*   **`POST /`**
    *   **Descrição:** Adiciona um novo provedor de identidade (IdP).
    *   **Payload da Requisição (`application/json`):**
        ```json
        {
            "provider_type": "string (obrigatório, um de: saml, oauth2_google, oauth2_github)",
            "name": "string (obrigatório, min 3, max 100)",
            "is_active": "boolean (opcional, default: true)",
            "config_json": {}, // objeto JSON, obrigatório, estrutura varia por provider_type
            "attribute_mapping_json": {} // objeto JSON, opcional
        }
        ```
        *   Exemplo `config_json` para `saml`: `{ "idp_entity_id": "url", "idp_sso_url": "url", "idp_x509_cert": "pem_string" }`
        *   Exemplo `config_json` para `oauth2_google`: `{ "client_id": "id", "client_secret": "secret" }`
    *   **Respostas:**
        *   `201 Created`: Objeto `models.IdentityProvider` criado.
        *   `400 Bad Request`: Payload inválido.
        *   `403 Forbidden`.
        *   `500 Internal Server Error`.

*   **`GET /`**
    *   **Descrição:** Lista todos os provedores de identidade da organização (paginado).
    *   **Query Params:** `page`, `page_size`.
    *   **Respostas:**
        *   `200 OK`: Resposta paginada com array de `models.IdentityProvider`.
        *   `403 Forbidden`.
        *   `500 Internal Server Error`.

*   **`GET /:idpId`**
    *   **Descrição:** Obtém um provedor de identidade específico.
    *   **Parâmetros de Path:** `idpId`.
    *   **Respostas:**
        *   `200 OK`: Objeto `models.IdentityProvider`.
        *   `403 Forbidden`.
        *   `404 Not Found`.

*   **`PUT /:idpId`**
    *   **Descrição:** Atualiza um provedor de identidade existente.
    *   **Parâmetros de Path:** `idpId`.
    *   **Payload da Requisição (`application/json`):** Similar ao POST.
    *   **Respostas:**
        *   `200 OK`: Objeto `models.IdentityProvider` atualizado.
        *   `400 Bad Request`.
        *   `403 Forbidden`.
        *   `404 Not Found`.
        *   `500 Internal Server Error`.

*   **`DELETE /:idpId`**
    *   **Descrição:** Remove um provedor de identidade.
    *   **Parâmetros de Path:** `idpId`.
    *   **Respostas:**
        *   `200 OK`: `{ "message": "Identity provider deleted successfully" }`
        *   `403 Forbidden`.
        *   `404 Not Found`.
        *   `500 Internal Server Error`.

#### 5.3. Configurações de Webhook (`/api/v1/organizations/:orgId/webhooks`)

Gerencia configurações de webhook para uma organização. Requer role de Admin ou Manager da organização para criar/atualizar/deletar. Listar pode ser permitido para membros da organização.

*   **`POST /`**
    *   **Descrição:** Cria uma nova configuração de webhook.
    *   **Payload da Requisição (`application/json`):**
        ```json
        {
            "name": "string (obrigatório, min 3, max 100)",
            "url": "string (obrigatório, URL válida, max 2048)",
            "event_types": ["string"], // Array de strings, obrigatório, ex: ["risk_created", "risk_status_changed"]
            "is_active": "boolean (opcional, default: true)"
        }
        ```
    *   **Respostas:**
        *   `201 Created`: Objeto `models.WebhookConfiguration`.
        *   `400 Bad Request`.
        *   `403 Forbidden`.
        *   `500 Internal Server Error`.

*   **`GET /`**
    *   **Descrição:** Lista todas as configurações de webhook da organização (paginado).
    *   **Autorização:** Usuário deve pertencer à organização.
    *   **Query Params:** `page`, `page_size`.
    *   **Respostas:**
        *   `200 OK`: Resposta paginada com array de `WebhookResponseItem` (inclui `EventTypesList` como array).
        *   `403 Forbidden`.
        *   `500 Internal Server Error`.

*   **`GET /:webhookId`**
    *   **Descrição:** Obtém uma configuração de webhook específica.
    *   **Autorização:** Usuário deve pertencer à organização.
    *   **Parâmetros de Path:** `webhookId`.
    *   **Respostas:**
        *   `200 OK`: Objeto `models.WebhookConfiguration`.
        *   `403 Forbidden`.
        *   `404 Not Found`.

*   **`PUT /:webhookId`**
    *   **Descrição:** Atualiza uma configuração de webhook existente.
    *   **Parâmetros de Path:** `webhookId`.
    *   **Payload da Requisição (`application/json`):** Similar ao POST.
    *   **Respostas:**
        *   `200 OK`: Objeto `models.WebhookConfiguration` atualizado.
        *   `400 Bad Request`.
        *   `403 Forbidden`.
        *   `404 Not Found`.

*   **`DELETE /:webhookId`**
    *   **Descrição:** Deleta uma configuração de webhook.
    *   **Parâmetros de Path:** `webhookId`.
    *   **Respostas:**
        *   `200 OK`: `{ "message": "Webhook configuration deleted successfully" }`
        *   `403 Forbidden`.
        *   `404 Not Found`.

#### 5.4. Gerenciamento de Usuários da Organização (`/api/v1/organizations/:orgId/users`)

Gerencia usuários dentro de uma organização. Requer role de Admin ou Manager da organização.

*   **`GET /`**
    *   **Descrição:** Lista todos os usuários da organização (paginado).
    *   **Query Params:** `page`, `page_size`.
    *   **Respostas:**
        *   `200 OK`: Resposta paginada com array de `UserResponse` (DTO sem PasswordHash).
        *   `403 Forbidden`.

*   **`GET /:userId`**
    *   **Descrição:** Obtém detalhes de um usuário específico da organização.
    *   **Parâmetros de Path:** `userId`.
    *   **Respostas:**
        *   `200 OK`: Objeto `UserResponse`.
        *   `403 Forbidden`.
        *   `404 Not Found`.

*   **`PUT /:userId/role`**
    *   **Descrição:** Atualiza a role de um usuário.
    *   **Parâmetros de Path:** `userId`.
    *   **Payload da Requisição (`application/json`):**
        ```json
        { "role": "string (um de: admin, manager, user)" }
        ```
    *   **Respostas:**
        *   `200 OK`: Objeto `UserResponse` atualizado.
        *   `400 Bad Request`.
        *   `403 Forbidden` (ex: tentar rebaixar o último admin).
        *   `404 Not Found`.

*   **`PUT /:userId/status`**
    *   **Descrição:** Ativa ou desativa um usuário.
    *   **Parâmetros de Path:** `userId`.
    *   **Payload da Requisição (`application/json`):**
        ```json
        { "is_active": "boolean (obrigatório)" }
        ```
    *   **Respostas:**
        *   `200 OK`: Objeto `UserResponse` atualizado.
        *   `400 Bad Request`.
        *   `403 Forbidden` (ex: tentar desativar o último admin ativo).
        *   `404 Not Found`.

---

### 6. Gestão de Vulnerabilidades (`/api/v1/vulnerabilities`)

Todos os endpoints nesta seção requerem autenticação JWT.

*   **`POST /api/v1/vulnerabilities`**
    *   **Descrição:** Cria uma nova vulnerabilidade.
    *   **Payload da Requisição (`application/json`):**
        ```json
        {
            "title": "string (obrigatório, min 3, max 255)",
            "description": "string (opcional)",
            "cve_id": "string (opcional, max 50)",
            "severity": "string (obrigatório, um de: Baixo, Médio, Alto, Crítico)",
            "status": "string (opcional, um de: descoberta, em_correcao, corrigida, default: descoberta)",
            "asset_affected": "string (opcional, max 255)"
        }
        ```
    *   **Respostas:**
        *   `201 Created`: Objeto `models.Vulnerability` criado.
        *   `400 Bad Request`: Payload inválido.
        *   `500 Internal Server Error`.

*   **`GET /api/v1/vulnerabilities`**
    *   **Descrição:** Lista todas as vulnerabilidades da organização do usuário (paginado).
    *   **Query Params:**
        *   `page` (int, opcional, default: 1)
        *   `page_size` (int, opcional, default: 10)
        *   `status` (string, opcional): Filtra por status.
        *   `severity` (string, opcional): Filtra por severidade.
    *   **Respostas:**
        *   `200 OK`: Resposta paginada com array de `models.Vulnerability`.
        *   `500 Internal Server Error`.

*   **`GET /api/v1/vulnerabilities/:vulnId`**
    *   **Descrição:** Obtém uma vulnerabilidade específica pelo ID.
    *   **Parâmetros de Path:** `vulnId` (string UUID).
    *   **Respostas:**
        *   `200 OK`: Objeto `models.Vulnerability`.
        *   `400 Bad Request`: `vulnId` inválido.
        *   `404 Not Found`.
        *   `500 Internal Server Error`.

*   **`PUT /api/v1/vulnerabilities/:vulnId`**
    *   **Descrição:** Atualiza uma vulnerabilidade existente.
    *   **Parâmetros de Path:** `vulnId`.
    *   **Payload da Requisição (`application/json`):** Similar ao POST.
    *   **Respostas:**
        *   `200 OK`: Objeto `models.Vulnerability` atualizado.
        *   `400 Bad Request`.
        *   `404 Not Found`.
        *   `500 Internal Server Error`.

*   **`DELETE /api/v1/vulnerabilities/:vulnId`**
    *   **Descrição:** Deleta uma vulnerabilidade.
    *   **Parâmetros de Path:** `vulnId`.
    *   **Respostas:**
        *   `200 OK`: `{ "message": "Vulnerability deleted successfully" }`
        *   `400 Bad Request`.
        *   `404 Not Found`.
        *   `500 Internal Server Error`.

---

### 7. Auditoria e Conformidade (`/api/v1/audit`)

Endpoints para interagir com frameworks de auditoria, controles e avaliações.

*   **`GET /api/v1/audit/frameworks`**
    *   **Descrição:** Lista todos os frameworks de auditoria pré-carregados no sistema (ex: NIST CSF, ISO 27001).
    *   **Autenticação:** JWT Obrigatório.
    *   **Respostas:**
        *   `200 OK`: Array de objetos `models.AuditFramework`.
            ```json
            [
                {
                    "ID": "uuid-framework-1",
                    "Name": "NIST Cybersecurity Framework 2.0",
                    "CreatedAt": "timestamp",
                    "UpdatedAt": "timestamp"
                }
            ]
            ```
        *   `500 Internal Server Error`.

*   **`GET /api/v1/audit/frameworks/:frameworkId/controls`**
    *   **Descrição:** Lista todos os controles para um framework de auditoria específico.
    *   **Autenticação:** JWT Obrigatório.
    *   **Parâmetros de Path:** `frameworkId` (string UUID).
    *   **Respostas:**
        *   `200 OK`: Array de objetos `models.AuditControl`.
            ```json
            [
                {
                    "ID": "uuid-controle-1",
                    "FrameworkID": "uuid-framework-1",
                    "ControlID": "GV.OC-1",
                    "Description": "Papéis e responsabilidades...",
                    "Family": "Governança Organizacional (GV.OC)",
                    "CreatedAt": "timestamp",
                    "UpdatedAt": "timestamp"
                }
            ]
            ```
        *   `400 Bad Request`: `frameworkId` inválido.
        *   `404 Not Found`: Framework não encontrado (se nenhum controle for retornado e o framework não existir).
        *   `500 Internal Server Error`.

*   **`POST /api/v1/audit/assessments`**
    *   **Descrição:** Cria ou atualiza uma avaliação para um controle de auditoria específico dentro da organização do usuário autenticado. Realiza um "upsert" baseado em `(OrganizationID, AuditControlID)`.
    *   **Autenticação:** JWT Obrigatório.
    *   **Requisição:** `multipart/form-data`.
        *   Campo `data` (string JSON, obrigatório):
            ```json
            {
                "audit_control_id": "uuid-do-audit-control", // string UUID, obrigatório
                "status": "string (obrigatório, um de: conforme, nao_conforme, parcialmente_conforme)",
                "evidence_url": "string (opcional, URL)", // Usado se evidence_file não for enviado
                "score": "integer (opcional, 0-100, default baseado no status)",
                "assessment_date": "string (opcional, YYYY-MM-DD, default: data atual)"
            }
            ```
        *   Campo `evidence_file` (arquivo, opcional): Arquivo de evidência (limite 10MB, tipos permitidos: JPEG, PNG, PDF, DOC, DOCX, XLS, XLSX, TXT). Se fornecido, sua URL (após upload) substitui `evidence_url` no JSON.
    *   **Respostas:**
        *   `200 OK`: Objeto `models.AuditAssessment` criado ou atualizado.
        *   `400 Bad Request`: Formulário/JSON inválido, `audit_control_id` inválido, data inválida, arquivo muito grande ou tipo não permitido.
        *   `500 Internal Server Error`: Falha no upload ou ao salvar no banco.

*   **`GET /api/v1/audit/assessments/control/:controlId`**
    *   **Descrição:** Obtém a avaliação de um controle específico (`controlId` é o UUID do `AuditControl`) para a organização do usuário autenticado.
    *   **Autenticação:** JWT Obrigatório.
    *   **Parâmetros de Path:** `controlId` (string UUID).
    *   **Respostas:**
        *   `200 OK`: Objeto `models.AuditAssessment` (com `AuditControl` opcionalmente pré-carregado).
        *   `400 Bad Request`: `controlId` inválido.
        *   `404 Not Found`: Nenhuma avaliação encontrada para este controle na organização.
        *   `500 Internal Server Error`.

*   **`GET /api/v1/audit/organizations/:orgId/frameworks/:frameworkId/assessments`**
    *   **Descrição:** Lista todas as avaliações de uma organização específica para um determinado framework (paginado).
    *   **Autenticação:** JWT Obrigatório. Usuário deve pertencer à `:orgId` ou ser Admin.
    *   **Parâmetros de Path:** `orgId`, `frameworkId`.
    *   **Query Params:** `page`, `page_size`.
    *   **Respostas:**
        *   `200 OK`: Resposta paginada com array de `models.AuditAssessment` (com `AuditControl` pré-carregado).
        *   `400 Bad Request`: IDs inválidos.
        *   `403 Forbidden`.
        *   `500 Internal Server Error`.

*   **`GET /api/v1/audit/organizations/:orgId/frameworks/:frameworkId/compliance-score`**
    *   **Descrição:** Calcula e retorna o score geral de conformidade para um framework dentro de uma organização.
    *   **Autenticação:** JWT Obrigatório. Usuário deve pertencer à `:orgId` ou ser Admin.
    *   **Parâmetros de Path:** `orgId`, `frameworkId`.
    *   **Respostas:**
        *   `200 OK`: Objeto `ComplianceScoreResponse`.
            ```json
            {
                "framework_id": "uuid-framework",
                "framework_name": "Nome do Framework",
                "organization_id": "uuid-org",
                "compliance_score": 75.5, // Média dos scores dos controles avaliados
                "total_controls": 120,
                "evaluated_controls": 80,
                "conformant_controls": 60,
                "partially_conformant_controls": 10,
                "non_conformant_controls": 10
            }
            ```
        *   `400 Bad Request`: IDs inválidos.
        *   `403 Forbidden`.
        *   `404 Not Found`: Framework não encontrado.
        *   `500 Internal Server Error`.

---

### 8. Autenticação de Múltiplos Fatores (MFA) (`/api/v1/users/me/2fa`)

Endpoints para o usuário autenticado gerenciar suas configurações de 2FA.

#### 8.1. TOTP (Time-based One-Time Password)

*   **`POST /api/v1/users/me/2fa/totp/setup`**
    *   **Descrição:** Inicia o processo de configuração do TOTP para o usuário autenticado. Gera um novo segredo TOTP, armazena-o (associado ao usuário, `IsTOTPEnabled` ainda é `false`) e retorna o segredo e um QR code para o usuário escanear em seu aplicativo autenticador.
    *   **Autenticação:** JWT Obrigatório.
    *   **Respostas:**
        *   `200 OK`:
            ```json
            {
                "secret": "BASE32_ENCODED_SECRET_KEY",
                "qr_code": "data:image/png;base64,BASE64_ENCODED_PNG_IMAGE_OF_QR_CODE",
                "account": "user@example.com", // Email do usuário
                "issuer": "PhoenixGRC",       // Nome da aplicação (configurável)
                "backup_codes_generated": false // Será true quando os códigos de backup forem implementados e gerados nesta etapa
            }
            ```
        *   `500 Internal Server Error`: Falha ao gerar chave TOTP, QR code, ou salvar segredo.

*   **`POST /api/v1/users/me/2fa/totp/verify`**
    *   **Descrição:** Verifica um código TOTP fornecido pelo usuário. Se for a primeira verificação após o setup, ativa o TOTP para o usuário (`IsTOTPEnabled = true`).
    *   **Autenticação:** JWT Obrigatório.
    *   **Payload da Requisição (`application/json`):**
        ```json
        {
            "token": "123456" // string, código TOTP de 6 dígitos
        }
        ```
    *   **Respostas:**
        *   `200 OK`: `{ "message": "TOTP successfully verified and enabled." }` (se estava desabilitado e agora foi habilitado)
        *   `200 OK`: `{ "message": "TOTP token verified successfully." }` (se já estava habilitado)
        *   `400 Bad Request`: Payload inválido ou TOTP não configurado para o usuário.
        *   `401 Unauthorized`: Token TOTP inválido.
        *   `500 Internal Server Error`: Falha ao salvar o estado do usuário.

*   **`POST /api/v1/users/me/2fa/totp/disable`**
    *   **Descrição:** Desabilita o TOTP para o usuário autenticado. Requer a senha atual do usuário para confirmação.
    *   **Autenticação:** JWT Obrigatório.
    *   **Payload da Requisição (`application/json`):**
        ```json
        {
            "password": "current_user_password" // string, obrigatório
        }
        ```
    *   **Respostas:**
        *   `200 OK`: `{ "message": "TOTP has been successfully disabled." }`
        *   `400 Bad Request`: Payload inválido ou TOTP não está habilitado.
        *   `401 Unauthorized`: Senha inválida.
        *   `500 Internal Server Error`: Falha ao salvar o estado do usuário.

#### 8.2. Códigos de Backup (TODO)

*   **`GET /api/v1/users/me/2fa/backup-codes/generate`** (TODO)
    *   **Descrição:** Gera um novo conjunto de códigos de backup para o usuário. Invalida quaisquer códigos anteriores. Retorna os novos códigos (apenas uma vez).
    *   **Autenticação:** JWT Obrigatório. Requer que TOTP esteja habilitado.
    *   **Respostas:**
        *   `200 OK`: `{ "backup_codes": ["code1", "code2", ...] }`
        *   *(Outros erros a definir)*

*   **`POST /auth/login/2fa/backup-code/verify`** (TODO - parte do fluxo de login)
    *   **Descrição:** Verifica um código de backup fornecido pelo usuário durante o login 2FA.
    *   **Payload:** `{ "user_id": "uuid", "backup_code": "string" }`
    *   **Respostas:** Similar ao `/auth/login/2fa/verify` com TOTP, emitindo JWT se o código for válido e não utilizado.
    *   *(Outros erros a definir)*

---
**TODOs Gerais da Documentação:**
*   Revisar e detalhar as validações específicas para cada campo nos payloads (ex: formatos de data exatos se não YYYY-MM-DD, limites de string específicos além de min/max básicos, etc.).
*   Confirmar e detalhar as regras de autorização para todas as operações de modificação (PUT, POST, DELETE), especialmente quem pode modificar recursos de outros usuários ou da organização.
*   Expandir exemplos de payloads de requisição e resposta onde for útil.
*   Detalhar a estrutura exata do `config_json` para cada `provider_type` de IdentityProvider.
*   Adicionar uma seção sobre códigos de erro comuns e seus significados.
```
