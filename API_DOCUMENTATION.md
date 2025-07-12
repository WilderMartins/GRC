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

---

### 2. Endpoints Públicos Adicionais

*   **`GET /api/public/social-identity-providers`**
    *   **Descrição:** Lista todos os provedores de identidade OAuth2 configurados que podem ser usados para login social. Isso inclui provedores globais (usando credenciais de todo o aplicativo) e provedores específicos da organização que foram marcados como públicos. Este endpoint é destinado a ser usado pela tela de login para exibir dinamicamente as opções de login social.
    *   **Autenticação:** Nenhuma.
    *   **Respostas:**
        *   `200 OK`: Array de objetos `PublicIdentityProviderResponse`.
            ```json
            [
                {
                    "id": "global", // Ou UUID para IdP específico da organização
                    "name": "Google (Global)", // Nome amigável
                    "type": "oauth2_google", // Tipo do provedor (oauth2_google, oauth2_github)
                    "provider_slug": "google", // Slug para construir URLs de login (ex: google, github)
                    "icon_url": "/path/to/google_icon.svg" // URL para um ícone (opcional, pode ser gerenciado pelo frontend)
                },
                {
                    "id": "uuid-org-idp-github",
                    "name": "GitHub (Organização Acme)",
                    "type": "oauth2_github",
                    "provider_slug": "github",
                    "icon_url": "/path/to/github_icon.svg"
                }
            ]
            ```
            *   **Campos da Resposta:**
                *   `id` (string): O identificador do provedor. Será "global" para IdPs globais, ou o UUID do `IdentityProvider` para IdPs de organização. Este `id` deve ser usado no path `{idpId}` das rotas de login OAuth2.
                *   `name` (string): Nome amigável do provedor (ex: "Google", "GitHub da Empresa X").
                *   `type` (string): O tipo técnico do provedor (ex: `oauth2_google`, `oauth2_github`).
                *   `provider_slug` (string): Um slug curto identificando o tipo de provedor (ex: `google`, `github`). Usado para construir as URLs de login como `/auth/oauth2/{provider_slug}/{id}/login`.
                *   `icon_url` (string, opcional): Uma URL para um ícone representando o provedor.
        *   `500 Internal Server Error`: Falha ao buscar os provedores de identidade.

### 3. Autenticação (`/auth`)

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
    *   **Descrição:** Inicia o fluxo de login OAuth2 com Google. Redireciona o usuário para a página de autorização do Google.
        *   Se `{idpId}` for um UUID, usa a configuração do `IdentityProvider` específico da organização.
        *   Se `{idpId}` for a string `"global"`, usa as credenciais OAuth2 globais do Google configuradas nas variáveis de ambiente do backend.
    *   **Autenticação:** Nenhuma.
    *   **Parâmetros de Path:**
        *   `idpId` (string): UUID do `IdentityProvider` da organização OU a string `"global"`.
    *   **Respostas:**
        *   `302 Found`: Redirecionamento para o Google.
        *   `404 Not Found`: Configuração do IdP (para UUID) não encontrada ou inativa.
        *   `500 Internal Server Error`: Falha ao configurar OAuth2 (ex: credenciais globais ausentes para `"global"`, `APP_ROOT_URL` não configurado, ou erro ao gerar estado).

*   **`GET /auth/oauth2/google/:idpId/callback`**
    *   **Descrição:** Endpoint de callback para o Google após autorização do usuário. Processa o código de autorização, obtém informações do usuário do Google, e então:
        *   Provisiona um novo usuário no Phoenix GRC ou loga um usuário existente.
        *   Para IdPs globais (`idpId="global"`):
            *   Se `ALLOW_GLOBAL_SSO_USER_CREATION` for `true`, novos usuários são criados.
            *   Se `DEFAULT_ORGANIZATION_ID_FOR_GLOBAL_SSO` estiver configurado, novos usuários globais são associados a essa organização; caso contrário, são criados sem organização inicial.
            *   Usuários globais existentes são logados.
        *   Para IdPs de organização (UUID): Usuários são provisionados/logados dentro da organização do IdP.
        *   Gera um token JWT para o usuário.
        *   Redireciona para o frontend. A URL exata de redirecionamento do frontend é construída usando `APP_ROOT_URL` (configurado no backend) e um path padrão como `/oauth2/callback`. O resultado final será algo como: `[APP_ROOT_URL]/oauth2/callback?token=[JWT_TOKEN]&sso_success=true&provider=google`. O frontend deve estar preparado para lidar com esta rota e extrair o token.
    *   **Autenticação:** Nenhuma.
    *   **Parâmetros de Path:**
        *   `idpId` (string): UUID do `IdentityProvider` OU a string `"global"`.
    *   **Query Params (enviados pelo Google):** `code`, `state`.
    *   **Respostas:**
        *   `302 Found`: Redirecionamento para o frontend com token JWT.
        *   `400 Bad Request`: Estado OAuth inválido, email não fornecido pelo Google.
        *   `401 Unauthorized`: Código de autorização não encontrado/inválido.
        *   `403 Forbidden`: Se a criação de usuário global estiver desabilitada (`ALLOW_GLOBAL_SSO_USER_CREATION=false`) e um novo usuário global tentar se registrar.
        *   `500 Internal Server Error`: Falha na troca de token, obtenção de user info, criação/atualização de usuário no DB, ou geração de token JWT.

*   **`GET /auth/oauth2/github/:idpId/login`**
    *   **Descrição:** Inicia o fluxo de login OAuth2 com GitHub. Similar ao Google Login, usando configurações de IdP de organização (para UUID) ou credenciais globais do GitHub (para `idpId="global"`).
    *   **Autenticação:** Nenhuma.
    *   **Parâmetros de Path:**
        *   `idpId` (string): UUID do `IdentityProvider` da organização OU a string `"global"`.
    *   **Respostas:** Similar ao Google Login (`404` se IdP de org não encontrado, `500` para falhas de config/estado).

*   **`GET /auth/oauth2/github/:idpId/callback`**
    *   **Descrição:** Callback para GitHub. Similar ao Google Callback, mas obtendo informações do usuário do GitHub.
        *   Provisiona/loga usuários, respeitando `ALLOW_GLOBAL_SSO_USER_CREATION` e `DEFAULT_ORGANIZATION_ID_FOR_GLOBAL_SSO` para IdPs globais.
        *   Redireciona para o frontend. A URL exata de redirecionamento do frontend é construída usando `APP_ROOT_URL` (configurado no backend) e um path padrão como `/oauth2/callback`. O resultado final será algo como: `[APP_ROOT_URL]/oauth2/callback?token=[JWT_TOKEN]&sso_success=true&provider=github`. O frontend deve estar preparado para lidar com esta rota e extrair o token.
    *   **Autenticação:** Nenhuma.
    *   **Parâmetros de Path:**
        *   `idpId` (string): UUID do `IdentityProvider` OU a string `"global"`.
    *   **Query Params (enviados pelo GitHub):** `code`, `state`.
    *   **Respostas:** Similar ao Google Callback (`302` para sucesso, `400` para estado/email inválido, `401` para código inválido, `403` para criação global desabilitada, `500` para outras falhas).

*   **`POST /auth/login/2fa/backup-code/verify`**
    *   **Descrição:** Verifica um código de backup fornecido pelo usuário como segundo fator de autenticação.
    *   **Autenticação:** Nenhuma (parte do fluxo de login multi-etapa).
    *   **Payload da Requisição (`application/json`):**
        ```json
        {
            "user_id": "uuid-string-do-usuario", // string, obrigatório (obtido da resposta do /auth/login)
            "backup_code": "string-codigo-backup" // string, obrigatório
        }
        ```
    *   **Respostas:**
        *   `200 OK` (Token JWT completo): Similar à resposta de `/auth/login/2fa/verify` com TOTP.
        *   `400 Bad Request`: Payload inválido.
        *   `401 Unauthorized`: Código de backup inválido, usuário não encontrado, ou 2FA/códigos de backup não habilitados.
        *   `500 Internal Server Error`: Falha ao gerar token JWT ou atualizar códigos de backup.

*   **Endpoints SAML 2.0 (Implementação Parcial - Requer Teste e Finalização)**
    *   **Nota:** A biblioteca SAML (`github.com/crewjam/saml v0.5.1`) foi adicionada/atualizada e o código relacionado foi descomentado. No entanto, a compilação completa e testes funcionais não puderam ser realizados no ambiente atual. A lógica principal do Assertion Consumer Service (ACS) para processar a asserção, provisionar usuários e emitir tokens JWT ainda é um placeholder e precisa ser implementada. **Esta funcionalidade deve ser considerada experimental e requer testes e desenvolvimento adicionais antes do uso em produção.**
    *   **`GET /auth/saml/:idpId/login`**
        *   **Descrição:** Tenta iniciar o fluxo de login SAML SP-initiated redirecionando o usuário para o IdP SAML configurado.
    *   **`GET /auth/saml/:idpId/metadata`**
        *   **Descrição:** Expõe os metadados do Service Provider (Phoenix GRC) para o IdP SAML especificado.
    *   **`POST /auth/saml/:idpId/acs`**
        *   **Descrição:** Endpoint para onde o IdP SAML redireciona o usuário com a asserção SAML após o login bem-sucedido. **Lógica de processamento da asserção e provisionamento de usuário pendente.**

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

*   **`GET /api/v1/me/dashboard/summary`**
    *   **Descrição:** Retorna um resumo de dados para o dashboard do usuário autenticado.
    *   **Autenticação:** JWT Obrigatório.
    *   **Respostas:**
        *   `200 OK`:
            ```json
            {
                "assigned_risks_open_count": 5,
                "assigned_vulnerabilities_open_count": 2, // Pode ser da organização se não houver atribuição direta
                "pending_approval_tasks_count": 1
            }
            ```
        *   `500 Internal Server Error`: Falha ao buscar dados do resumo.

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
    *   **Autorização:** Requer que o usuário autenticado seja o proprietário (`OwnerID`) do risco, ou tenha a role `admin` ou `manager` na organização do risco.
    *   **Respostas:**
        *   `200 OK`: Objeto `models.Risk` atualizado (com `Owner` pré-carregado).
        *   `400 Bad Request`: Payload ou `riskId` inválido.
        *   `403 Forbidden`: Usuário não autorizado.
        *   `404 Not Found`: Risco não encontrado.
        *   `500 Internal Server Error`: Falha ao atualizar.

*   **`DELETE /api/v1/risks/:riskId`**
    *   **Descrição:** Deleta um risco.
    *   **Parâmetros de Path:** `riskId`.
    *   **Autorização:** Requer que o usuário autenticado seja o proprietário (`OwnerID`) do risco, ou tenha a role `admin` ou `manager` na organização do risco.
    *   **Respostas:**
        *   `200 OK`: `{ "message": "Risk deleted successfully" }`
        *   `400 Bad Request`: `riskId` inválido.
        *   `403 Forbidden`: Usuário não autorizado.
        *   `404 Not Found`: Risco não encontrado.
        *   `500 Internal Server Error`: Falha ao deletar.

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
        *   Campo `logo_file` (arquivo, opcional): Arquivo de imagem para o logo (JPEG, PNG, GIF, SVG, limite 2MB). Se fornecido, o `objectName` do logo armazenado será salvo no campo `LogoURL` do modelo `Organization`.
    *   **Respostas:**
        *   `200 OK`: Objeto `models.Organization` atualizado. O campo `LogoURL` conterá o `objectName` (se um arquivo foi carregado) ou estará vazio/inalterado. Para acessar o logo carregado, use o endpoint `GET /api/v1/files/signed-url`.
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

*   **`GET /api/v1/users/organization-lookup`**
    *   **Descrição:** Retorna uma lista simplificada de usuários (ID, Nome) da organização do usuário autenticado. Útil para preencher dropdowns ou campos de seleção de proprietário/stakeholder. Retorna apenas usuários ativos.
    *   **Autenticação:** JWT Obrigatório.
    *   **Respostas:**
        *   `200 OK`: Array de objetos `UserLookupResponse`.
            ```json
            [
                { "id": "uuid-user1", "name": "Nome do Usuário 1" },
                { "id": "uuid-user2", "name": "Nome do Usuário 2" }
            ]
            ```
        *   `500 Internal Server Error`: Falha ao listar usuários para lookup.

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
        *   `title_like` (string, opcional): Filtra por título (case-insensitive, partial match).
        *   `cve_id` (string, opcional): Filtra por CVE ID (case-insensitive, exact match).
        *   `asset_affected_like` (string, opcional): Filtra por ativo afetado (case-insensitive, partial match).
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
    *   **Autorização:** Requer que o usuário autenticado tenha a role `admin` ou `manager` na organização.
    *   **Respostas:**
        *   `200 OK`: Objeto `models.Vulnerability` atualizado.
        *   `400 Bad Request`.
        *   `403 Forbidden`: Usuário não autorizado.
        *   `404 Not Found`.
        *   `500 Internal Server Error`.

*   **`DELETE /api/v1/vulnerabilities/:vulnId`**
    *   **Descrição:** Deleta uma vulnerabilidade.
    *   **Parâmetros de Path:** `vulnId`.
    *   **Autorização:** Requer que o usuário autenticado tenha a role `admin` ou `manager` na organização.
    *   **Respostas:**
        *   `200 OK`: `{ "message": "Vulnerability deleted successfully" }`
        *   `400 Bad Request`.
        *   `403 Forbidden`: Usuário não autorizado.
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

*   **`GET /api/v1/audit/frameworks/:frameworkId/control-families`**
    *   **Descrição:** Lista todas as famílias de controles únicas para um framework de auditoria específico.
    *   **Autenticação:** JWT Obrigatório.
    *   **Parâmetros de Path:** `frameworkId` (string UUID).
    *   **Respostas:**
        *   `200 OK`:
            ```json
            {
                "families": ["Família de Controle 1", "Família de Controle 2"]
            }
            ```
        *   `400 Bad Request`: `frameworkId` inválido.
        *   `500 Internal Server Error`: Falha ao listar famílias de controles.

*   **`GET /api/v1/audit/frameworks/:frameworkId/controls`**
    *   **Descrição:** Lista todos os controles para um framework de auditoria específico.
    *   **Autenticação:** JWT Obrigatório.
    *   **Parâmetros de Path:** `frameworkId` (string UUID).
    *   **Respostas:**
        *   `200 OK`: Array de objetos `AuditControlWithAssessmentResponse`.
            ```json
            [
                {
                    // Campos de models.AuditControl
                    "ID": "uuid-controle-1",
                    "FrameworkID": "uuid-framework-1",
                    "ControlID": "GV.OC-1",
                    "Description": "Papéis e responsabilidades...",
                    "Family": "Governança Organizacional (GV.OC)",
                    // ... outros campos de AuditControl
                    "assessment": { // Objeto models.AuditAssessment (pode ser null)
                        "ID": "uuid-assessment-1",
                        "OrganizationID": "uuid-da-org-do-usuario",
                        "AuditControlID": "uuid-controle-1",
                        "Status": "conforme",
                        "EvidenceURL": "objectName/ou/urlExterna", // Contém objectName ou URL externa
                        "Score": 100,
                        "AssessmentDate": "timestamp",
                        "Comments": "Comentários da avaliação principal",
                        "c2m2_maturity_level": 2, // Exemplo, pode ser null/omitido
                        "c2m2_assessment_date": "timestamp", // Exemplo, pode ser null/omitido
                        "c2m2_comments": "Comentários da avaliação C2M2" // Exemplo, pode ser null/omitido
                        // ... outros campos de AuditAssessment
                    }
                }
            ]
            ```
        *   `400 Bad Request`: `frameworkId` inválido.
        *   `403 Forbidden`: Se o `organizationID` não puder ser obtido do token.
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
            "status": "string (obrigatório, um de: conforme, nao_conforme, parcialmente_conforme, nao_aplicavel)",
            "evidence_url": "string (opcional, URL externa)",
            "score": "integer (opcional, 0-100)",
            "assessment_date": "string (opcional, YYYY-MM-DD, default: data atual)",
            "comments": "string (opcional, comentários da avaliação principal)",
            // Campos C2M2 (opcionais)
            "c2m2_maturity_level": "integer (opcional, 0-3)",
            "c2m2_assessment_date": "string (opcional, YYYY-MM-DD)",
            "c2m2_comments": "string (opcional, comentários da avaliação C2M2)"
            }
            ```
        *   Campo `evidence_file` (arquivo, opcional): Arquivo de evidência (limite 10MB, tipos permitidos: JPEG, PNG, PDF, DOC, DOCX, XLS, XLSX, TXT). Se fornecido, o `objectName` do arquivo armazenado será salvo no campo `EvidenceURL` do modelo `AuditAssessment`.
    *   **Respostas:**
        *   `200 OK`: Objeto `models.AuditAssessment` criado ou atualizado, podendo incluir campos C2M2. O campo `EvidenceURL` conterá o `objectName` (se um arquivo foi carregado) ou a URL externa fornecida. Para acessar arquivos carregados, use o endpoint `GET /api/v1/files/signed-url`.
        *   `400 Bad Request`: Formulário/JSON inválido, `audit_control_id` inválido, data inválida, `c2m2_maturity_level` fora do range, arquivo muito grande ou tipo não permitido.
        *   `500 Internal Server Error`: Falha no upload ou ao salvar no banco.

*   **`GET /api/v1/audit/assessments/control/:controlId`**
    *   **Descrição:** Obtém a avaliação de um controle específico (`controlId` é o UUID do `AuditControl`) para a organização do usuário autenticado. A resposta pode incluir campos C2M2. O campo `EvidenceURL` conterá o `objectName` (se aplicável) ou uma URL externa.
    *   **Autenticação:** JWT Obrigatório.
    *   **Parâmetros de Path:** `controlId` (string UUID).
    *   **Respostas:**
        *   `200 OK`: Objeto `models.AuditAssessment` (com `AuditControl` opcionalmente pré-carregado).
        *   `400 Bad Request`: `controlId` inválido.
        *   `404 Not Found`: Nenhuma avaliação encontrada para este controle na organização.
        *   `500 Internal Server Error`.

*   **`DELETE /api/v1/audit/assessments/:assessmentId/evidence`**
    *   **Descrição:** Remove o arquivo de evidência associado a uma avaliação específica e limpa o campo `EvidenceURL` no banco de dados. Se a `EvidenceURL` for um link externo, apenas o campo no banco é limpo.
    *   **Autenticação:** JWT Obrigatório. (Autorização: Usuário deve pertencer à organização da avaliação; TODO: refinar para admin/manager ou criador da avaliação).
    *   **Parâmetros de Path:** `assessmentId` (string UUID).
    *   **Respostas:**
        *   `200 OK`: `{ "message": "Evidence deleted successfully from assessment." }` ou `{ "message": "No evidence to delete for this assessment." }`
        *   `403 Forbidden`: Usuário não autorizado.
        *   `404 Not Found`: Avaliação não encontrada.
        *   `500 Internal Server Error`: Falha ao deletar arquivo do storage ou ao atualizar o registro da avaliação.

*   **`GET /api/v1/audit/organizations/:orgId/frameworks/:frameworkId/assessments`**
    *   **Descrição:** Lista todas as avaliações de uma organização específica para um determinado framework (paginado).
    *   **Autenticação:** JWT Obrigatório. O `organization_id` no token do usuário deve corresponder ao `:orgId` no path.
    *   **Parâmetros de Path:** `orgId`, `frameworkId`.
    *   **Query Params:** `page`, `page_size`.
    *   **Respostas:**
        *   `200 OK`: Resposta paginada com array de `models.AuditAssessment` (com `AuditControl` pré-carregado).
        *   `400 Bad Request`: IDs inválidos.
        *   `403 Forbidden`: Usuário não autorizado a acessar a organização especificada.
        *   `500 Internal Server Error`.

*   **`GET /api/v1/audit/organizations/:orgId/frameworks/:frameworkId/compliance-score`**
    *   **Descrição:** Calcula e retorna o score geral de conformidade para um framework dentro de uma organização.
    *   **Autenticação:** JWT Obrigatório. O `organization_id` no token do usuário deve corresponder ao `:orgId` no path.
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

*   **`GET /api/v1/audit/organizations/:orgId/frameworks/:frameworkId/c2m2-maturity-summary`**
    *   **Descrição:** Calcula e retorna um sumário da maturidade C2M2 para um framework específico dentro de uma organização, agregado por Função NIST.
    *   **Autenticação:** JWT Obrigatório. O `organization_id` no token do usuário deve corresponder ao `:orgId` no path.
    *   **Parâmetros de Path:** `orgId` (UUID da organização), `frameworkId` (UUID do framework).
    *   **Respostas:**
        *   `200 OK`: Objeto `C2M2MaturityFrameworkSummaryResponse`.
            ```json
            {
                "framework_id": "uuid-framework",
                "framework_name": "NIST Cybersecurity Framework 2.0",
                "organization_id": "uuid-org",
                "summary_by_function": [
                    {
                        "nist_component_type": "Function",
                        "nist_component_name": "Identify",
                        "achieved_mil": 2, // Nível C2M2 (0-3) agregado para esta função (ex: moda dos MILs dos controles)
                        "evaluated_controls": 10, // Número de controles NIST com C2M2MaturityLevel preenchido nesta função
                        "total_controls": 15,     // Número total de controles NIST nesta função
                        "mil_distribution": {    // Distribuição dos MILs dos controles avaliados
                            "mil0": 1,
                            "mil1": 2,
                            "mil2": 5,
                            "mil3": 2
                        }
                    }
                    // ... Outras Funções NIST (Protect, Detect, Respond, Recover, Govern)
                ]
                // "summary_by_category": [] // Opcional, pode ser adicionado no futuro se necessário
            }
            ```
        *   `400 Bad Request`: IDs inválidos.
        *   `403 Forbidden`: Acesso negado à organização.
        *   `404 Not Found`: Framework não encontrado.
        *   `500 Internal Server Error`.

---

### 8. Autenticação de Múltiplos Fatores (MFA) (`/api/v1/users/me/2fa`)

Endpoints para o usuário autenticado gerenciar suas configurações de 2FA.

#### 8.1. TOTP (Time-based One-Time Password)

*   **`POST /api/v1/users/me/2fa/totp/setup`**
    *   **Descrição:** Inicia o processo de configuração do TOTP para o usuário autenticado. Gera um novo segredo TOTP, armazena-o (associado ao usuário, `IsTOTPEnabled` ainda é `false`) e retorna o segredo e um QR code para o usuário escanear em seu aplicativo autenticador.
    *   **Autenticação:** JWT Obrigatório.
    *   **Nota de Segurança:** O segredo TOTP gerado é armazenado de forma segura no backend (criptografado em repouso).
    *   **Respostas:**
        *   `200 OK`:
            ```json
            {
                "secret": "BASE32_ENCODED_SECRET_KEY", // Segredo em texto plano para o usuário configurar no app autenticador
                "qr_code": "data:image/png;base64,BASE64_ENCODED_PNG_IMAGE_OF_QR_CODE",
                "account": "user@example.com", // Email do usuário
                "issuer": "PhoenixGRC",       // Nome da aplicação (configurável)
                "backup_codes_generated": false // Indica se os códigos de backup foram gerados automaticamente (atualmente falso, gerados via endpoint dedicado)
            }
            ```
        *   `500 Internal Server Error`: Falha ao gerar chave TOTP, QR code, ou salvar segredo.

*   **`POST /api/v1/users/me/2fa/totp/verify`**
    *   **Descrição:** Verifica um código TOTP fornecido pelo usuário. Se for a primeira verificação após o setup, ativa o TOTP para o usuário (`IsTOTPEnabled = true`) e pode acionar a geração de códigos de backup.
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

#### 8.2. Códigos de Backup

*   **`POST /api/v1/users/me/2fa/backup-codes/generate`**
    *   **Descrição:** Gera um novo conjunto de códigos de backup para o usuário. Invalida quaisquer códigos de backup anteriores. Os códigos retornados devem ser armazenados de forma segura pelo usuário, pois são exibidos apenas uma vez.
    *   **Autenticação:** JWT Obrigatório. Requer que TOTP esteja habilitado.
    *   **Respostas:**
        *   `200 OK`:
            ```json
            {
                "backup_codes": ["code1-plain-text", "code2-plain-text", "..."]
            }
            ```
        *   `400 Bad Request`: Se TOTP não estiver habilitado.
        *   `500 Internal Server Error`: Falha ao gerar ou salvar os hashes dos códigos.

*   **`POST /auth/login/2fa/backup-code/verify`**
    *   **Descrição:** Verifica um código de backup fornecido pelo usuário durante a etapa de 2FA do login. Se válido, o código é consumido (não pode ser reutilizado) e o login prossegue com a emissão de um token JWT.
    *   **Autenticação:** Nenhuma (parte do fluxo de login multi-etapa).
    *   **Payload da Requisição (`application/json`):** (Já documentado na Seção 2 - Autenticação)
        ```json
        {
            "user_id": "uuid-string-do-usuario",
            "backup_code": "string-codigo-backup"
        }
        ```
    *   **Respostas:** (Já documentado na Seção 2 - Autenticação)

---

### 9. Gerenciamento de Arquivos (`/api/v1/files`)

Endpoints para operações relacionadas a arquivos, como obter URLs de acesso seguro.

*   **`GET /api/v1/files/signed-url`**
    *   **Descrição:** Gera uma URL assinada de curta duração para acessar um objeto de arquivo armazenado (ex: evidências de auditoria, logos). Os arquivos são armazenados de forma privada e esta URL fornece acesso temporário.
    *   **Autenticação:** JWT Obrigatório.
    *   **Query Params:**
        *   `objectKey` (string, obrigatório): A chave/path do objeto no bucket de armazenamento. Este é o valor que agora é armazenado em campos como `AuditAssessment.EvidenceURL` ou `Organization.LogoURL` quando se referem a um arquivo carregado pela aplicação.
        *   `durationMinutes` (int, opcional, default: 15): Duração em minutos para a validade da URL assinada (máximo usualmente permitido pelos provedores é 7 dias, ou 10080 minutos).
    *   **Respostas:**
        *   `200 OK`:
            ```json
            {
                "signed_url": "https://storage.provider.com/path/to/object?signature=..."
            }
            ```
        *   `400 Bad Request`: `objectKey` ausente ou `durationMinutes` inválido.
        *   `404 Not Found`: Se o `objectKey` de alguma forma não for encontrado ou o usuário não tiver permissão para o bucket implícito (embora a autorização aqui seja mais sobre o acesso ao endpoint em si).
        *   `500 Internal Server Error`: Falha ao gerar URL assinada ou provedor de armazenamento não configurado.

---
**TODOs Gerais da Documentação:**
*   Revisar e detalhar as validações específicas para cada campo nos payloads (ex: formatos de data exatos se não YYYY-MM-DD, limites de string específicos além de min/max básicos, etc.).
*   Confirmar e detalhar as regras de autorização para todas as operações de modificação (PUT, POST, DELETE), especialmente quem pode modificar recursos de outros usuários ou da organização.
*   Expandir exemplos de payloads de requisição e resposta onde for útil.
*   Detalhar a estrutura exata do `config_json` para cada `provider_type` de IdentityProvider.
*   Adicionar uma seção sobre códigos de erro comuns e seus significados.

---

## Configuração do Backend (Variáveis de Ambiente Relevantes)

A seguir, uma lista de variáveis de ambiente importantes para configurar o comportamento do backend, especialmente para funcionalidades como OAuth2 global e comportamento de provisionamento.

*   **`APP_ROOT_URL`**
    *   **Descrição:** A URL raiz da aplicação frontend. Usada para construir URIs de redirecionamento corretos para fluxos OAuth2 e SAML, e em links enviados em emails/notificações. Deve ser a URL base que o usuário acessa no navegador.
    *   **Exemplo:** `http://localhost:3000` (para desenvolvimento local com frontend na porta 3000) ou `https://app.suaempresa.com`.

*   **`GOOGLE_CLIENT_ID`**
    *   **Descrição:** O Client ID fornecido pelo Google Cloud Console para o seu projeto OAuth2. Necessário se o login global com Google estiver habilitado via `/api/public/social-identity-providers`.
    *   **Exemplo:** `xxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.apps.googleusercontent.com`

*   **`GOOGLE_CLIENT_SECRET`**
    *   **Descrição:** O Client Secret fornecido pelo Google Cloud Console. Necessário se o login global com Google estiver habilitado.
    *   **Exemplo:** `GOCSPX-xxxxxxxxxxxxxxxxxxxxxxx`

*   **`GITHUB_CLIENT_ID`**
    *   **Descrição:** O Client ID do seu aplicativo OAuth do GitHub. Necessário se o login global com GitHub estiver habilitado via `/api/public/social-identity-providers`.
    *   **Exemplo:** `Iv1.xxxxxxxxxxxxxxxx`

*   **`GITHUB_CLIENT_SECRET`**
    *   **Descrição:** O Client Secret do seu aplicativo OAuth do GitHub. Necessário se o login global com GitHub estiver habilitado.
    *   **Exemplo:** `xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

*   **`ALLOW_GLOBAL_SSO_USER_CREATION`**
    *   **Descrição:** Um booleano (`true` ou `false`) que determina se novos usuários podem ser criados no sistema quando eles fazem login pela primeira vez através de um provedor de identidade OAuth2 global (identificado como `"global"`).
    *   **Default:** `false` (o comportamento exato se não definido pode depender da função `getEnvAsBool`, mas é recomendado definir explicitamente).
    *   **Exemplo:** `true`

*   **`DEFAULT_ORGANIZATION_ID_FOR_GLOBAL_SSO`**
    *   **Descrição:** O UUID de uma organização existente para a qual novos usuários, criados via SSO global (quando `ALLOW_GLOBAL_SSO_USER_CREATION` é `true`), serão automaticamente associados. Se esta variável estiver definida com um UUID válido, os novos usuários globais pertencerão a esta organização. Se estiver vazia ou inválida, os novos usuários globais serão criados sem associação direta a uma organização (terão `OrganizationID` nulo no banco de dados).
    *   **Exemplo:** `a1b2c3d4-e5f6-7890-1234-567890abcdef`

*   **`ENCRYPTION_KEY_HEX`**
    *   **Descrição:** Chave de criptografia de 32 bytes (representada como 64 caracteres hexadecimais) usada para criptografar dados sensíveis em repouso, como o segredo TOTP dos usuários. **Esta chave é crítica para a segurança. Deve ser gerada de forma segura (ex: usando um gerador de números aleatórios criptograficamente seguro) e mantida em segredo.** Não deve ser commitada no repositório de código.
    *   **Exemplo:** `your_super_secret_and_randomly_generated_64_hex_characters_long_key`

*   **`FRONTEND_BASE_URL`** (Nota: `APP_ROOT_URL` é preferível para consistência)
    *   **Descrição:** URL base do frontend. Usada em alguns contextos para construir links, como em notificações de webhook. É recomendado usar `APP_ROOT_URL` de forma consistente para todos os casos de URLs base do frontend.
    *   **Exemplo:** `http://localhost:3000`

Outras variáveis de ambiente para configuração de banco de dados (ex: `POSTGRES_HOST`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`), JWT (`JWT_SECRET_KEY`, `JWT_TOKEN_LIFESPAN_HOURS`), provedores de armazenamento de arquivos (ex: `GCS_PROJECT_ID`, `AWS_S3_BUCKET`), etc., também são cruciais e geralmente definidas no arquivo `.env` (para desenvolvimento) ou no ambiente de implantação.
```
