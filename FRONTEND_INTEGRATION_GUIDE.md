# Guia de Integração Backend para Frontend - Phoenix GRC

## 1. Introdução

Este documento serve como um guia para desenvolvedores frontend integrarem com a API backend do Phoenix GRC. Ele detalha os fluxos de autenticação, os principais endpoints da API, os payloads esperados e as respostas, com foco nas funcionalidades prontas para serem consumidas pela interface do usuário.

Para a documentação completa e detalhada de todos os endpoints da API, consulte o arquivo [API_DOCUMENTATION.md](API_DOCUMENTATION.md).

**URL Base da API (Exemplo Docker Local com Nginx):** `http://localhost` (se Nginx na porta 80)
*   Endpoints de autenticação não versionados: `/auth/...`
*   Endpoints públicos não versionados: `/api/public/...`
*   Endpoints protegidos versionados: `/api/v1/...`

O frontend deve ser configurado para enviar o token JWT no header `Authorization: Bearer <token>` para todos os endpoints sob `/api/v1/`.

## 2. Configuração Inicial e Variáveis de Ambiente do Frontend

O frontend precisará de algumas configurações para interagir corretamente com o backend:

*   **`NEXT_PUBLIC_API_BASE_URL`**: A URL base completa para a API do backend.
    *   Exemplo em desenvolvimento local (com Nginx): `http://localhost` (se o Nginx estiver na porta 80 e fazendo proxy para o backend).
    *   Exemplo em desenvolvimento local (acessando backend diretamente): `http://localhost:8080` (se o backend estiver na porta 8080).
    *   Em produção: A URL pública da sua instância do Phoenix GRC (ex: `https://app.suaempresa.com`).
*   **`NEXT_PUBLIC_APP_ROOT_URL`**: A URL raiz da própria aplicação frontend. Esta deve corresponder à variável `APP_ROOT_URL` configurada no backend. É crucial para o correto funcionamento dos redirecionamentos OAuth2.
    *   Exemplo: `http://localhost:3000` (se o Next.js em dev roda na 3000) ou `https://app.suaempresa.com`.

## 3. Autenticação

### 3.1. Login Padrão (Email/Senha)

*   **Endpoint:** `POST /auth/login`
*   **Descrição:** Autentica um usuário com email e senha.
*   **Payload:**
    ```json
    {
        "email": "user@example.com",
        "password": "yourpassword"
    }
    ```

#### 5.3.9. Obter Sumário de Maturidade C2M2 por Função NIST

*   **Endpoint:** `GET /audit/organizations/{orgId}/frameworks/{frameworkId}/c2m2-maturity-summary`
*   **Descrição:** Calcula e retorna um sumário da maturidade C2M2 para um framework específico dentro de uma organização, agregado por Função NIST (Identify, Protect, Detect, Respond, Recover, Govern).
*   **Resposta de Sucesso (200 OK):**
    ```json
    {
        "framework_id": "uuid-framework",
        "framework_name": "NIST Cybersecurity Framework 2.0",
        "organization_id": "uuid-org",
        "summary_by_function": [
            {
                "nist_component_type": "Function",
                "nist_component_name": "Identify",
                "achieved_mil": 2, // Nível C2M2 (0-3) agregado (ex: moda dos MILs dos controles da função)
                "evaluated_controls": 10, // Controles com C2M2MaturityLevel preenchido nesta função
                "total_controls": 15,     // Total de controles NIST nesta função
                "mil_distribution": {    // Distribuição dos MILs dos controles avaliados
                    "mil0": 1,
                    "mil1": 2,
                    "mil2": 5,
                    "mil3": 2
                }
            }
            // ... Outras Funções NIST ...
        ]
    }
    ```
*   **Notas Frontend:**
    *   Usar estes dados para construir visualizações (gráficos, tabelas) da postura de maturidade C2M2 da organização em relação às Funções do NIST CSF.
    *   O `achieved_mil` é uma agregação simplificada (moda). A lógica exata de como um MIL é "alcançado" para uma função inteira pode ser mais complexa no C2M2 e pode ser refinada no backend no futuro.
*   **Resposta de Sucesso (200 OK - 2FA não habilitado):**
    ```json
    {
        "token": "jwt.token.string", // Armazenar este token (ex: localStorage, cookie seguro)
        "user_id": "uuid-string",
        "email": "user@example.com",
        "name": "User Name",
        "role": "admin",
        "organization_id": "org-uuid-string"
    }
    ```
    O frontend deve armazenar o `token` e usá-lo para chamadas autenticadas. Outros dados do usuário podem ser usados para popular o estado da UI.
*   **Resposta de Sucesso (200 OK - 2FA habilitado):**
    ```json
    {
        "2fa_required": true,
        "user_id": "uuid-string",
        "message": "Password verified. Please provide TOTP token."
    }
    ```
    O frontend deve então solicitar o código TOTP ou de backup (ver Seção 3.2).
*   **Respostas de Erro:**
    *   `400 Bad Request`: Payload inválido.
    *   `401 Unauthorized`: Credenciais inválidas ou usuário inativo.

### 3.2. Autenticação de Dois Fatores (2FA) - Pós Login

Se o login inicial (`POST /auth/login`) retornar `{"2fa_required": true, "user_id": "..."}`, o frontend deve exibir uma interface para o usuário inserir seu código TOTP ou um código de backup.

#### 3.2.1. Verificação TOTP

*   **Endpoint:** `POST /auth/login/2fa/verify`
*   **Payload:**
    ```json
    {
        "user_id": "uuid-string-do-usuario", // user_id retornado pelo /auth/login
        "token": "123456"                   // Código TOTP de 6 dígitos
    }
    ```
*   **Resposta de Sucesso (200 OK):**
    ```json
    {
        "token": "jwt.token.string", // Token JWT completo
        "user_id": "uuid-string",
        "email": "user@example.com",
        // ... outros dados do usuário
    }
    ```
    Armazenar o token como no login padrão.
*   **Respostas de Erro:**
    *   `400 Bad Request`: Payload inválido.
    *   `401 Unauthorized`: Token TOTP inválido.

#### 3.2.2. Verificação com Código de Backup

*   **Endpoint:** `POST /auth/login/2fa/backup-code/verify`
*   **Payload:**
    ```json
    {
        "user_id": "uuid-string-do-usuario",
        "backup_code": "codigo-backup-alfanumerico"
    }
    ```
*   **Resposta de Sucesso (200 OK):** Similar à verificação TOTP, retorna o token JWT completo.
*   **Respostas de Erro:**
    *   `400 Bad Request`.
    *   `401 Unauthorized`: Código de backup inválido.

### 3.3. Login Social via OAuth2 (Google, Github)

O fluxo de login social envolve redirecionamentos entre o frontend, o backend e o provedor OAuth2.

**Passo 1: Listar Provedores de Identidade Disponíveis**

*   **Endpoint:** `GET /api/public/social-identity-providers`
*   **Descrição:** O frontend deve chamar este endpoint ao carregar a página de login para buscar as opções de login social disponíveis.
*   **Resposta de Sucesso (200 OK):**
    ```json
    [
        {
            "id": "global", // Ou UUID para IdP específico da organização
            "name": "Google (Global)",
            "type": "oauth2_google",
            "provider_slug": "google", // Usar para construir a URL de login
            "icon_url": "/path/to/google_icon.svg" // Opcional, pode ser gerenciado pelo frontend
        },
        // ... outros provedores
    ]
    ```
*   **Ação do Frontend:** Para cada provedor retornado, exibir um botão de login (ex: "Login com Google", "Login com GitHub da Acme Corp").

**Passo 2: Iniciar Fluxo de Login OAuth2**

*   Quando o usuário clica em um botão de login social, o frontend deve redirecionar o navegador do usuário para o endpoint de login do backend correspondente.
*   **URL de Redirecionamento (construída pelo frontend):**
    `{NEXT_PUBLIC_API_BASE_URL}/auth/oauth2/{provider_slug}/{id}/login`
    *   `{provider_slug}`: Vem do campo `provider_slug` da resposta do Passo 1 (ex: "google", "github").
    *   `{id}`: Vem do campo `id` da resposta do Passo 1 (pode ser "global" ou um UUID).
    *   Exemplo: `http://localhost/auth/oauth2/google/global/login`
    *   Exemplo: `http://localhost/auth/oauth2/github/uuid-do-idp-da-org/login`

*   **Ação do Backend:** O backend redirecionará o usuário para a página de autorização do provedor OAuth2 (Google, GitHub).

**Passo 3: Tratamento do Callback no Frontend**

*   Após o usuário autorizar (ou negar) no site do provedor OAuth2, o provedor redirecionará de volta para o endpoint de callback do backend (`/auth/oauth2/{provider_slug}/{id}/callback`).
*   O backend processará o código de autorização, obterá as informações do usuário, provisionará/logará o usuário no Phoenix GRC, gerará um token JWT e, finalmente, redirecionará o navegador do usuário de volta para o frontend.
*   **URL de Redirecionamento para o Frontend (enviada pelo backend):**
    `{NEXT_PUBLIC_APP_ROOT_URL}/oauth2/callback?token={JWT_TOKEN}&sso_success=true&provider={provider_slug}`
    *   Exemplo: `http://localhost:3000/oauth2/callback?token=ey...&sso_success=true&provider=google`

*   **Ação do Frontend:**
    1.  O frontend deve ter uma rota/página configurada para lidar com `/oauth2/callback` (ou o path configurado em `APP_ROOT_URL` no backend para este redirecionamento).
    2.  Nesta página, extrair o `token` JWT dos query parameters da URL.
    3.  Armazenar o token JWT (como no login padrão).
    4.  Verificar `sso_success=true`.
    5.  Opcionalmente, usar o parâmetro `provider` para feedback ao usuário.
    6.  Redirecionar o usuário para o dashboard ou página principal da aplicação.
    7.  Se `sso_success` não for `true` ou o token estiver ausente, exibir uma mensagem de erro.

## 4. Gerenciamento de Usuário Autenticado

Estes endpoints requerem o token JWT no header `Authorization: Bearer <token>`.

### 4.1. Obter Dados do Usuário Logado

*   **Endpoint:** `GET /api/v1/me`
*   **Resposta de Sucesso (200 OK):**
    ```json
    {
        "user_id": "uuid-string",
        "email": "user@example.com",
        "name": "User Name",
        "role": "admin",
        "organization_id": "org-uuid-string",
        // Adicionar outros campos do modelo User se relevantes para o frontend,
        // como is_totp_enabled, etc. (Verificar API_DOCUMENTATION.md para o DTO exato)
    }
    ```
    O frontend deve verificar o DTO exato em `API_DOCUMENTATION.md` para `UserResponse`.

### 4.2. Configurações de 2FA do Usuário

Localização: `/api/v1/users/me/2fa/...`

#### 4.2.1. Setup de TOTP

*   **Endpoint:** `POST /api/v1/users/me/2fa/totp/setup`
*   **Descrição:** Inicia o setup do TOTP. O backend gera um segredo e um QR code.
*   **Resposta de Sucesso (200 OK):**
    ```json
    {
        "secret": "BASE32_ENCODED_SECRET_KEY", // Mostrar ao usuário para input manual
        "qr_code": "data:image/png;base64,BASE64_ENCODED_PNG_IMAGE_OF_QR_CODE", // Renderizar como imagem
        "account": "user@example.com",
        "issuer": "PhoenixGRC",
        "backup_codes_generated": false // Atualmente, códigos de backup são gerados em endpoint separado
    }
    ```
*   **Ação do Frontend:** Exibir o QR code para o usuário escanear com seu app autenticador (ex: Google Authenticator, Authy) e também o `secret` para configuração manual. Após o usuário configurar no app, ele precisará verificar um token (próximo passo).

#### 4.2.2. Verificar e Ativar TOTP

*   **Endpoint:** `POST /api/v1/users/me/2fa/totp/verify`
*   **Descrição:** Usuário envia um token TOTP do seu app para verificar e ativar o TOTP.
*   **Payload:**
    ```json
    {
        "token": "123456" // Código TOTP do app autenticador
    }
    ```
*   **Resposta de Sucesso (200 OK):**
    ```json
    { "message": "TOTP successfully verified and enabled." }
    // Ou "TOTP token verified successfully." se já estava habilitado (raro neste fluxo)
    ```
*   **Ação do Frontend:** Após sucesso, informar ao usuário que TOTP está ativo. Recomendar a geração de códigos de backup.

#### 4.2.3. Gerar Códigos de Backup

*   **Endpoint:** `POST /api/v1/users/me/2fa/backup-codes/generate`
*   **Descrição:** Gera um novo conjunto de códigos de backup. Invalida os anteriores.
*   **Requisito:** TOTP deve estar habilitado.
*   **Resposta de Sucesso (200 OK):**
    ```json
    {
        "backup_codes": ["code1-plain-text", "code2-plain-text", ...] // Array de 10 códigos
    }
    ```
*   **Ação do Frontend:** Exibir estes códigos ao usuário UMA ÚNICA VEZ. O usuário deve ser instruído a armazená-los em local seguro.

#### 4.2.4. Desabilitar TOTP

*   **Endpoint:** `POST /api/v1/users/me/2fa/totp/disable`
*   **Descrição:** Desabilita TOTP para o usuário. Requer a senha atual.
*   **Payload:**
    ```json
    {
        "password": "current_user_password"
    }
    ```
*   **Resposta de Sucesso (200 OK):**
    ```json
    { "message": "TOTP has been successfully disabled." }
    ```

### 4.3. Resumo do Dashboard do Usuário

*   **Endpoint:** `GET /api/v1/me/dashboard/summary`
*   **Descrição:** Retorna contagens para o dashboard do usuário.
*   **Resposta de Sucesso (200 OK):**
    ```json
    {
        "assigned_risks_open_count": 5,
        "assigned_vulnerabilities_open_count": 2,
        "pending_approval_tasks_count": 1
    }
    ```

## 5. Módulos Principais

Esta seção detalha os endpoints para as funcionalidades centrais de GRC. Todos os endpoints aqui estão sob `/api/v1/` e requerem autenticação JWT.

### 5.1. Gestão de Riscos (`/risks`)

#### 5.1.1. Criar Risco

*   **Endpoint:** `POST /risks`
*   **Descrição:** Cria um novo risco. O `owner_id` é opcional; se não fornecido, o usuário que cria se torna o proprietário. `RiskLevel` é calculado pelo backend.
*   **Payload:**
    ```json
    {
        "title": "string (obrigatório, min 3, max 255)",
        "description": "string (opcional)",
        "category": "string (opcional, um de: tecnologico, operacional, legal)",
        "impact": "string (opcional, um de: Baixo, Médio, Alto, Crítico)",
        "probability": "string (opcional, um de: Baixo, Médio, Alto, Crítico)",
        "status": "string (opcional, um de: aberto, em_andamento, mitigado, aceito, default: aberto)",
        "owner_id": "string (UUID do usuário, opcional)",
        "next_review_date": "string (YYYY-MM-DD, opcional)",
        "mitigation_details": "string (opcional)",
        "acceptance_justification": "string (opcional)",
        "custom_fields": {} // Objeto JSON para campos customizados (opcional)
    }
    ```
*   **Resposta de Sucesso (201 Created):** Objeto do risco criado (modelo `Risk`).
*   **Notas Frontend:**
    *   Fornecer seletores para `category`, `impact`, `probability`, `status`.
    *   Para `owner_id`, usar o endpoint de lookup de usuários (Seção 6.1).
    *   O campo `custom_fields` permite flexibilidade, mas o frontend precisaria de uma forma de definir e renderizar esses campos se forem usados extensivamente.

#### 5.1.2. Listar Riscos

*   **Endpoint:** `GET /risks`
*   **Descrição:** Lista riscos da organização do usuário, com paginação e filtros.
*   **Query Params:**
    *   `page` (int, opcional, default: 1)
    *   `page_size` (int, opcional, default: 10)
    *   `status` (string, opcional)
    *   `impact` (string, opcional)
    *   `probability` (string, opcional)
    *   `category` (string, opcional)
    *   `owner_id` (string UUID, opcional)
    *   `title_like` (string, opcional, busca parcial no título)
    *   `sort_by` (string, opcional, ex: `created_at`, `title`, `risk_level`, `status`, `next_review_date`)
    *   `sort_order` (string, opcional, `asc` ou `desc`, default: `desc` para `created_at`, `asc` para outros)
*   **Resposta de Sucesso (200 OK):**
    ```json
    {
        "items": [ /* array de modelos Risk, com Owner pré-carregado */ ],
        "total_items": 150,
        "total_pages": 15,
        "page": 1,
        "page_size": 10
    }
    ```
*   **Notas Frontend:** Implementar controles de paginação e filtros na UI.

#### 5.1.3. Obter Risco Específico

*   **Endpoint:** `GET /risks/{riskId}`
*   **Descrição:** Obtém detalhes de um risco pelo seu ID.
*   **Resposta de Sucesso (200 OK):** Objeto `Risk` (com `Owner` pré-carregado).

#### 5.1.4. Atualizar Risco

*   **Endpoint:** `PUT /risks/{riskId}`
*   **Descrição:** Atualiza um risco. `RiskLevel` é recalculado se necessário.
    *   **Autorização:** Proprietário do risco, Admin ou Manager da organização.
*   **Payload:** Similar ao de Criar Risco.
*   **Resposta de Sucesso (200 OK):** Objeto `Risk` atualizado.

#### 5.1.5. Deletar Risco

*   **Endpoint:** `DELETE /risks/{riskId}`
*   **Descrição:** Deleta um risco.
    *   **Autorização:** Proprietário do risco, Admin ou Manager da organização.
*   **Resposta de Sucesso (200 OK):** `{ "message": "Risk deleted successfully" }`

#### 5.1.6. Upload em Massa de Riscos (CSV)

*   **Endpoint:** `POST /risks/bulk-upload-csv`
*   **Descrição:** Permite criar múltiplos riscos enviando um arquivo CSV.
*   **Requisição:** `multipart/form-data`, com o arquivo no campo `file`.
*   **Formato CSV (Cabeçalhos obrigatórios: title, impact, probability):** Ver `API_DOCUMENTATION.md`.
*   **Respostas:**
    *   `200 OK`: `{ "successfully_imported": N, "failed_rows": [] }`
    *   `207 Multi-Status`: `{ "successfully_imported": M, "failed_rows": [ ... ] }`
*   **Notas Frontend:** Fornecer interface de upload de arquivo. Exibir resultados, incluindo erros por linha se houver.

#### 5.1.7. Gerenciamento de Stakeholders do Risco

Base Path: `/risks/{riskId}/stakeholders`

*   **Adicionar Stakeholder:** `POST /`
    *   **Payload:** `{ "user_id": "string (UUID do usuário)" }`
    *   **Resposta (201 Created):** `{ "message": "Stakeholder added successfully" }`
*   **Listar Stakeholders:** `GET /`
    *   **Resposta (200 OK):** Array de `UserStakeholderResponse` (`{ id, name, email, role }`).
*   **Remover Stakeholder:** `DELETE /{userId}`
    *   **Resposta (200 OK):** `{ "message": "Stakeholder removed successfully" }`
*   **Notas Frontend:** Para adicionar, usar o lookup de usuários. Listar stakeholders na página de detalhes do risco.

#### 5.1.8. Workflow de Aceite de Risco

Base Path: `/risks/{riskId}/approval` (ajustado para consistência, verificar API_DOC)
*Corrigindo base path conforme API_DOC para Workflow de Aceite:*
O `approvalId` é parte do path para a decisão. A submissão e listagem são no nível do risco.

*   **Submeter Risco para Aceite:** `POST /risks/{riskId}/submit-acceptance`
    *   **Descrição:** Admin/Manager submete o risco para aceite pelo proprietário do risco.
    *   **Resposta (201 Created):** Objeto `ApprovalWorkflow`.
*   **Decidir sobre Aceite:** `POST /risks/{riskId}/approval/{approvalId}/decide`
    *   **Descrição:** Proprietário do risco (aprovador) aprova ou rejeita.
    *   **Payload:** `{ "decision": "aprovado" | "rejeitado", "comments": "string (opcional)" }`
    *   **Resposta (200 OK):** Objeto `ApprovalWorkflow` atualizado.
*   **Histórico de Aprovações:** `GET /risks/{riskId}/approval-history`
    *   **Descrição:** Lista todos os workflows de aprovação para o risco.
    *   **Resposta (200 OK):** Lista paginada de `ApprovalWorkflow`.
*   **Notas Frontend:**
    *   Exibir status de aprovação do risco.
    *   Se o risco tiver um proprietário, permitir que Admin/Manager submetam para aceite.
    *   Se o usuário logado for o proprietário de um risco com aceite pendente, permitir que ele aprove/rejeite.
    *   Mostrar histórico de aprovações.

### 5.2. Gestão de Vulnerabilidades (`/vulnerabilities`)

#### 5.2.1. Criar Vulnerabilidade

*   **Endpoint:** `POST /vulnerabilities`
*   **Descrição:** Cria uma nova vulnerabilidade.
*   **Payload:**
    ```json
    {
        "title": "string (obrigatório, min 3, max 255)",
        "description": "string (opcional)",
        "cve_id": "string (opcional, max 50, ex: CVE-2023-12345)",
        "severity": "string (obrigatório, um de: Baixo, Médio, Alto, Crítico)",
        "status": "string (opcional, um de: descoberta, em_correcao, corrigida, aceita_risco, default: descoberta)",
        "asset_affected": "string (opcional, max 255)",
        "remediation_details": "string (opcional)",
        "cvss_score": "number (opcional, ex: 7.5)"
    }
    ```
*   **Resposta de Sucesso (201 Created):** Objeto da vulnerabilidade criada (modelo `Vulnerability`).
*   **Notas Frontend:** Fornecer seletores para `severity` e `status`.

#### 5.2.2. Listar Vulnerabilidades

*   **Endpoint:** `GET /vulnerabilities`
*   **Descrição:** Lista vulnerabilidades da organização, com paginação e filtros.
*   **Query Params:**
    *   `page` (int, opcional, default: 1)
    *   `page_size` (int, opcional, default: 10)
    *   `status` (string, opcional)
    *   `severity` (string, opcional)
    *   `title_like` (string, opcional)
    *   `cve_id` (string, opcional)
    *   `asset_affected_like` (string, opcional)
    *   `sort_by` (string, opcional, ex: `created_at`, `title`, `severity`, `status`, `cvss_score`)
    *   `sort_order` (string, opcional, `asc` ou `desc`)
*   **Resposta de Sucesso (200 OK):**
    ```json
    {
        "items": [ /* array de modelos Vulnerability */ ],
        "total_items": 50,
        "total_pages": 5,
        "page": 1,
        "page_size": 10
    }
    ```
*   **Notas Frontend:** Implementar UI para paginação e filtros.

#### 5.2.3. Obter Vulnerabilidade Específica

*   **Endpoint:** `GET /vulnerabilities/{vulnId}`
*   **Descrição:** Obtém detalhes de uma vulnerabilidade pelo seu ID.
*   **Resposta de Sucesso (200 OK):** Objeto `Vulnerability`.

#### 5.2.4. Atualizar Vulnerabilidade

*   **Endpoint:** `PUT /vulnerabilities/{vulnId}`
*   **Descrição:** Atualiza uma vulnerabilidade.
    *   **Autorização:** Admin ou Manager da organização.
*   **Payload:** Similar ao de Criar Vulnerabilidade.
*   **Resposta de Sucesso (200 OK):** Objeto `Vulnerability` atualizado.

#### 5.2.5. Deletar Vulnerabilidade

*   **Endpoint:** `DELETE /vulnerabilities/{vulnId}`
*   **Descrição:** Deleta uma vulnerabilidade.
    *   **Autorização:** Admin ou Manager da organização.
*   **Resposta de Sucesso (200 OK):** `{ "message": "Vulnerability deleted successfully" }`

### 5.3. Auditoria e Conformidade (`/audit`)

#### 5.3.1. Listar Frameworks de Auditoria

*   **Endpoint:** `GET /audit/frameworks`
*   **Descrição:** Lista todos os frameworks de auditoria globais disponíveis no sistema (ex: NIST CSF, ISO 27001).
*   **Resposta de Sucesso (200 OK):** Array de `AuditFramework` (`{ ID, Name, Description, Version, CreatedAt, UpdatedAt }`).

#### 5.3.2. Listar Famílias de Controles de um Framework

*   **Endpoint:** `GET /audit/frameworks/{frameworkId}/control-families`
*   **Descrição:** Lista as famílias de controles únicas para um framework específico.
*   **Resposta de Sucesso (200 OK):** `{ "families": ["Família 1", "Família 2"] }`
*   **Notas Frontend:** Útil para construir filtros na UI de controles.

#### 5.3.3. Listar Controles de um Framework (com Avaliações da Organização)

*   **Endpoint:** `GET /audit/frameworks/{frameworkId}/controls`
*   **Descrição:** Lista todos os controles para um framework, incluindo a avaliação atual da organização do usuário para cada controle (se existir).
*   **Resposta de Sucesso (200 OK):** Array de `AuditControlWithAssessmentResponse`.
    ```json
    [
        {
            // Campos de models.AuditControl
            "ID": "uuid-controle-1",
            "FrameworkID": "uuid-framework-1",
            "ControlID": "GV.OC-1", // Identificador textual do controle
            "Description": "Descrição do controle...",
            "Family": "Governança Organizacional (GV.OC)",
            // ...
            "assessment": { // Objeto models.AuditAssessment (pode ser null se não avaliado)
                "ID": "uuid-assessment-1",
                "Status": "conforme",
                "EvidenceURL": "objectName-ou-urlExterna", // Usar com GET /files/signed-url se for objectName
                "Score": 100,
                "AssessmentDate": "timestamp",
                "Comments": "Comentários da avaliação principal...",
                "c2m2_maturity_level": 2, // Exemplo, pode ser null/omitido
                "c2m2_assessment_date": "timestamp", // Exemplo, pode ser null/omitido
                "c2m2_comments": "Comentários da avaliação C2M2..." // Exemplo, pode ser null/omitido
                // ...
            }
        }
    ]
    ```
*   **Notas Frontend:**
    *   A UI deve permitir ao usuário ver o status da avaliação e os dados de maturidade C2M2 de cada controle.
    *   Se `assessment.EvidenceURL` for um `objectName` (não uma URL http/https), o frontend precisa chamar `GET /files/signed-url?objectKey={assessment.EvidenceURL}` para obter uma URL de download/visualização temporária.

#### 5.3.4. Criar/Atualizar Avaliação de Controle

*   **Endpoint:** `POST /audit/assessments`
*   **Descrição:** Cria ou atualiza (upsert) uma avaliação para um controle específico na organização do usuário.
*   **Requisição:** `multipart/form-data`
    *   Campo `data` (string JSON obrigatória):
        ```json
        {
            "audit_control_id": "uuid-do-audit-control", // Obrigatório
            "status": "string (conforme, nao_conforme, parcialmente_conforme, nao_aplicavel)", // Obrigatório
            "evidence_url": "string (URL externa, opcional)",
            "score": "integer (0-100, opcional)",
            "assessment_date": "string (YYYY-MM-DD, opcional, default: hoje)",
            "comments": "string (opcional)",
            // Campos C2M2 (opcionais)
            "c2m2_maturity_level": "integer (0-3, opcional)",
            "c2m2_assessment_date": "string (YYYY-MM-DD, opcional)",
            "c2m2_comments": "string (opcional)"
        }
        ```
    *   Campo `evidence_file` (arquivo, opcional): Arquivo de evidência. Se fornecido, seu `objectName` será armazenado em `EvidenceURL`.
*   **Resposta de Sucesso (200 OK):** Objeto `AuditAssessment` criado/atualizado.
*   **Notas Frontend:**
    *   Permitir upload de arquivo ou input de URL externa para evidência.
    *   Lembre-se que `EvidenceURL` na resposta conterá o `objectName` se um arquivo foi carregado.

#### 5.3.5. Obter Avaliação de um Controle Específico

*   **Endpoint:** `GET /audit/assessments/control/{controlId}`
    *   `{controlId}` é o UUID do `AuditControl`.
*   **Descrição:** Obtém a avaliação de um controle para a organização do usuário.
*   **Resposta de Sucesso (200 OK):** Objeto `AuditAssessment`. (Pode ser 404 se não avaliado).

#### 5.3.6. Remover Evidência de uma Avaliação

*   **Endpoint:** `DELETE /audit/assessments/{assessmentId}/evidence`
    *   `{assessmentId}` é o UUID da `AuditAssessment`.
*   **Descrição:** Remove o arquivo de evidência do storage (se aplicável) e limpa `EvidenceURL` no DB.
*   **Resposta de Sucesso (200 OK):** `{ "message": "Evidence deleted..." }`

#### 5.3.7. Listar Avaliações de uma Organização para um Framework

*   **Endpoint:** `GET /audit/organizations/{orgId}/frameworks/{frameworkId}/assessments`
*   **Descrição:** Lista todas as avaliações de uma organização para um framework. Útil para visões gerais de conformidade ou relatórios. O `{orgId}` deve ser o da organização do usuário logado (ou o handler imporá isso).
*   **Query Params:** `page`, `page_size`.
*   **Resposta de Sucesso (200 OK):** Lista paginada de `AuditAssessment` (com `AuditControl` pré-carregado).

#### 5.3.8. Obter Score de Conformidade

*   **Endpoint:** `GET /audit/organizations/{orgId}/frameworks/{frameworkId}/compliance-score`
*   **Descrição:** Calcula o score geral de conformidade para um framework na organização.
*   **Resposta de Sucesso (200 OK):**
    ```json
    {
        "framework_id": "uuid-framework",
        "framework_name": "Nome do Framework",
        "organization_id": "uuid-org",
        "compliance_score": 75.5, // Média dos scores
        "total_controls": 120,
        "evaluated_controls": 80,
        // ... outras contagens de status
    }
    ```

#### 5.3.9. Obter Sumário de Maturidade C2M2 por Função NIST

*   **Endpoint:** `GET /audit/organizations/{orgId}/frameworks/{frameworkId}/c2m2-maturity-summary`
*   **Descrição:** Calcula e retorna um sumário da maturidade C2M2 para um framework específico dentro de uma organização, agregado por Função NIST (Identify, Protect, Detect, Respond, Recover, Govern).
*   **Resposta de Sucesso (200 OK):**
    ```json
    {
        "framework_id": "uuid-framework",
        "framework_name": "NIST Cybersecurity Framework 2.0",
        "organization_id": "uuid-org",
        "summary_by_function": [
            {
                "nist_component_type": "Function",
                "nist_component_name": "Identify",
                "achieved_mil": 2, // Nível C2M2 (0-3) agregado (ex: moda dos MILs dos controles da função)
                "evaluated_controls": 10, // Controles com C2M2MaturityLevel preenchido nesta função
                "total_controls": 15,     // Total de controles NIST nesta função
                "mil_distribution": {    // Distribuição dos MILs dos controles avaliados
                    "mil0": 1,
                    "mil1": 2,
                    "mil2": 5,
                    "mil3": 2
                }
            }
            // ... Outras Funções NIST ...
        ]
    }
    ```
*   **Notas Frontend:**
    *   Usar estes dados para construir visualizações (gráficos, tabelas) da postura de maturidade C2M2 da organização em relação às Funções do NIST CSF.
    *   O `achieved_mil` é uma agregação simplificada (moda). A lógica exata de como um MIL é "alcançado" para uma função inteira pode ser mais complexa no C2M2 e pode ser refinada no backend no futuro.

### 5.4. Administração da Organização (`/organizations/{orgId}/...`)

Estes endpoints são para administradores (`admin`) ou gerentes (`manager`) da organização especificada por `{orgId}`. O `{orgId}` no path deve corresponder ao `organization_id` do usuário autenticado (o backend valida isso).

#### 5.4.1. Branding da Organização

Base Path: `/organizations/{orgId}/branding`

*   **Atualizar Branding:** `PUT /`
    *   **Descrição:** Atualiza logo e cores da organização.
    *   **Requisição:** `multipart/form-data`
        *   Campo `data` (string JSON): `{ "primary_color": "#RRGGBB", "secondary_color": "#RRGGBB" }` (ambos opcionais)
        *   Campo `logo_file` (arquivo, opcional): Imagem do logo.
    *   **Resposta (200 OK):** Objeto `Organization` atualizado. `LogoURL` conterá `objectName` se logo foi carregado.
    *   **Notas Frontend:**
        *   Para exibir o logo, usar `LogoURL` (que é o `objectName`) com o endpoint `GET /files/signed-url`.
        *   Fornecer color pickers para as cores.

*   **Obter Branding:** `GET /`
    *   **Descrição:** Obtém as configurações de branding.
    *   **Resposta (200 OK):** `{ id, name, logo_url (objectName), primary_color, secondary_color }`.

#### 5.4.2. Provedores de Identidade (SSO/OAuth2 da Organização)

Base Path: `/organizations/{orgId}/identity-providers`

*   **Adicionar IdP:** `POST /`
    *   **Payload:**
        ```json
        {
            "provider_type": "string (saml, oauth2_google, oauth2_github)",
            "name": "string (nome amigável)",
            "is_active": "boolean (default: true)",
            "is_public": "boolean (default: false)", // Se true, aparece em /api/public/social-identity-providers
            "config_json": { /* ...config específica do provedor... */ },
            "attribute_mapping_json": { /* ...mapeamento de atributos SAML... */ } // Opcional
        }
        ```
        *   Exemplos de `config_json`:
            *   `oauth2_google`: `{ "client_id": "...", "client_secret": "..." }`
            *   `saml`: `{ "idp_entity_id": "...", "idp_sso_url": "...", "idp_x509_cert": "..." }`
    *   **Resposta (201 Created):** Objeto `IdentityProvider`.
*   **Listar IdPs:** `GET /` (paginado)
    *   **Resposta (200 OK):** Lista paginada de `IdentityProvider`.
*   **Obter IdP:** `GET /{idpId}`
*   **Atualizar IdP:** `PUT /{idpId}` (Payload similar ao POST)
*   **Deletar IdP:** `DELETE /{idpId}`
*   **Notas Frontend:**
    *   A UI deve ter formulários diferentes para cada `provider_type` para coletar o `config_json` correto.
    *   SAML ainda é experimental no backend.

#### 5.4.3. Webhooks da Organização

Base Path: `/organizations/{orgId}/webhooks`

*   **Criar Webhook:** `POST /`
    *   **Payload:**
        ```json
        {
            "name": "string",
            "url": "string (URL válida)",
            "event_types": ["string"], // Ex: ["risk_created", "vulnerability_status_changed"]
            "is_active": "boolean (default: true)",
            "secret_token": "string (opcional, para verificar payloads)"
        }
        ```
    *   **Resposta (201 Created):** Objeto `WebhookConfiguration`.
*   **Listar Webhooks:** `GET /` (paginado)
*   **Obter Webhook:** `GET /{webhookId}`
*   **Atualizar Webhook:** `PUT /{webhookId}` (Payload similar ao POST)
*   **Deletar Webhook:** `DELETE /{webhookId}`
*   **Notas Frontend:** Listar `event_types` disponíveis para seleção.

#### 5.4.4. Gerenciamento de Usuários da Organização

Base Path: `/organizations/{orgId}/users`

*   **Listar Usuários da Organização:** `GET /` (paginado)
    *   **Resposta (200 OK):** Lista paginada de `UserResponse` (DTO sem PasswordHash).
*   **Obter Usuário Específico:** `GET /{userId}`
    *   **Resposta (200 OK):** Objeto `UserResponse`.
*   **Atualizar Role do Usuário:** `PUT /{userId}/role`
    *   **Payload:** `{ "role": "string (admin, manager, user)" }`
    *   **Resposta (200 OK):** Objeto `UserResponse` atualizado.
    *   **Notas Frontend:** Cuidado com lógicas de não poder rebaixar o último admin.
*   **Atualizar Status do Usuário (Ativar/Desativar):** `PUT /{userId}/status`
    *   **Payload:** `{ "is_active": "boolean" }`
    *   **Resposta (200 OK):** Objeto `UserResponse` atualizado.
    *   **Notas Frontend:** Cuidado com lógicas de não poder desativar o último admin ativo.

## 6. Recursos Utilitários

Estes endpoints fornecem funcionalidades de apoio para a UI.

### 6.1. Lookup de Usuários da Organização

*   **Endpoint:** `GET /api/v1/users/organization-lookup`
*   **Descrição:** Retorna uma lista simplificada de usuários (ID, Nome) da organização do usuário autenticado. Útil para preencher seletores de "proprietário", "stakeholder", "avaliador", etc. Retorna apenas usuários ativos.
*   **Resposta de Sucesso (200 OK):**
    ```json
    [
        { "id": "uuid-user1", "name": "Nome do Usuário 1" },
        { "id": "uuid-user2", "name": "Nome do Usuário 2" }
    ]
    ```

### 6.2. Obter URL Assinada para Arquivos

*   **Endpoint:** `GET /api/v1/files/signed-url`
*   **Descrição:** Gera uma URL de curta duração para acessar um objeto de arquivo privado (ex: logos de organização, evidências de auditoria).
*   **Query Params:**
    *   `objectKey` (string, obrigatório): A chave do objeto no storage. Este é o valor retornado em campos como `Organization.LogoURL` ou `AuditAssessment.EvidenceURL` quando um arquivo foi carregado pela aplicação.
    *   `durationMinutes` (int, opcional, default: 15): Duração da validade da URL.
*   **Resposta de Sucesso (200 OK):**
    ```json
    {
        "signed_url": "https://storage.provider.com/path/to/object?signature=..."
    }
    ```
*   **Notas Frontend:**
    *   Sempre que um campo de modelo (ex: `AuditAssessment.EvidenceURL`) contiver um `objectName` (e não uma URL `http://` ou `https://` completa), o frontend deve usar este endpoint para obter uma URL temporária para exibir ou permitir o download do arquivo.
    *   Se `EvidenceURL` já for uma URL externa completa (ex: fornecida manualmente pelo usuário), ela pode ser usada diretamente.

## 7. Tratamento de Erros da API

A API utiliza códigos de status HTTP padrão para indicar o sucesso ou falha de uma requisição.

*   **`200 OK`**: Requisição bem-sucedida.
*   **`201 Created`**: Recurso criado com sucesso (geralmente em respostas a `POST`).
*   **`204 No Content`**: Requisição bem-sucedida, sem conteúdo para retornar (geralmente em respostas a `DELETE`).
*   **`400 Bad Request`**: A requisição foi malformada ou contém dados inválidos. O corpo da resposta geralmente contém um JSON com mais detalhes:
    ```json
    { "error": "Mensagem detalhando o erro de validação ou payload." }
    ```
*   **`401 Unauthorized`**: Autenticação falhou ou é necessária.
    *   Pode ocorrer se o token JWT estiver ausente, inválido ou expirado.
    *   Também usado para credenciais de login inválidas ou falhas de 2FA.
    *   Resposta: `{ "error": "Mensagem específica da falha de autenticação." }`
*   **`403 Forbidden`**: O usuário autenticado não tem permissão para realizar a ação solicitada no recurso especificado.
    *   Resposta: `{ "error": "Usuário não autorizado a realizar esta ação." }`
*   **`404 Not Found`**: O recurso solicitado não foi encontrado.
    *   Resposta: `{ "error": "Recurso não encontrado." }` (ou mensagem mais específica).
*   **`409 Conflict`**: A requisição não pôde ser processada devido a um conflito com o estado atual do recurso (ex: tentar criar um recurso que já existe com um identificador único, ou tentar modificar um recurso de forma inconsistente).
    *   Resposta: `{ "error": "Mensagem detalhando o conflito." }`
*   **`422 Unprocessable Entity`**: A requisição foi bem formada mas não pôde ser seguida devido a erros semânticos (usado, por exemplo, no upload de CSV com linhas inválidas, retornando `207 Multi-Status` que é um tipo de `422` mais específico).
*   **`500 Internal Server Error`**: Um erro inesperado ocorreu no servidor.
    *   Resposta: `{ "error": "Erro interno do servidor." }`
    *   Estes erros devem ser raros e investigados. O backend loga mais detalhes.

**Boas Práticas para o Frontend:**
*   Verificar o `Content-Type` da resposta para garantir que é `application/json` antes de tentar decodificar o corpo de erro.
*   Exibir mensagens de erro amigáveis para o usuário, mas logar os detalhes do erro (ou o corpo JSON completo do erro) no console do desenvolvedor para facilitar a depuração.
*   Para `401 Unauthorized` devido a token expirado, implementar um fluxo para deslogar o usuário e/ou redirecioná-lo para a página de login (possivelmente com uma tentativa de refresh de token se essa funcionalidade for implementada no futuro).

## 8. Considerações sobre o Setup Inicial via Wizard (Frontend)

O backend suporta um fluxo de setup inicial que é idealmente conduzido por uma interface de usuário (Wizard) no frontend.

**Detecção de Necessidade de Setup:**
*   O frontend pode tentar fazer uma chamada para um endpoint simples que requer que a organização e o admin já existam (ex: `GET /api/v1/me` após uma tentativa de login "fictícia" ou um endpoint de status de setup dedicado - este último não existe atualmente).
*   Alternativamente, se o backend for iniciado e não conseguir se conectar ao banco de dados (conforme configurado no `.env` do backend), ele não iniciará corretamente. O frontend, ao não conseguir se comunicar com a API, pode inferir um problema de setup ou de conexão.
*   **Melhoria Sugerida para o Backend:** Criar um endpoint público (ex: `/api/public/setup-status`) que o frontend possa consultar para saber se o setup básico (conexão com DB, migrações, organização/admin inicial) já foi concluído.

**Fluxo do Wizard (Interação Frontend-Backend):**

1.  **Configuração do Banco de Dados (Frontend informa, Backend usa):**
    *   O Wizard no frontend coletaria os dados de conexão do banco de dados do usuário (host, port, user, password, dbname, sslmode).
    *   **Ação:** Estes dados NÃO são enviados para um endpoint. O usuário deve ser instruído a colocar esses dados no arquivo `.env` do **backend** e reiniciar o backend. O frontend pode, então, tentar verificar a conexão.
2.  **Verificar Conexão com o Banco e Executar Migrações (Backend):**
    *   Após o usuário configurar o `.env` do backend e reiniciá-lo, o backend tentará se conectar ao DB.
    *   Se o comando `setup` for executado (`docker-compose run --rm backend setup` ou `./server setup`), o backend:
        *   Conecta-se ao DB.
        *   Executa as migrações (`database.MigrateDB(dbURL)`).
    *   **Feedback para o Frontend:** O frontend precisaria de uma forma de saber se esta etapa foi bem-sucedida. Se o Wizard for uma aplicação frontend separada que orquestra o setup, ela poderia invocar o comando de setup do backend e analisar sua saída, ou depender de um endpoint de status.
3.  **Criar Primeira Organização e Usuário Admin (Backend):**
    *   O comando `setup` do backend, após as migrações, solicita interativamente no console os detalhes da primeira organização e do usuário admin.
    *   **Desafio para Wizard no Browser:** Este fluxo interativo no console do backend não se traduz diretamente para um Wizard no browser.
    *   **Soluções Possíveis para Wizard no Browser:**
        *   **Opção A (Recomendada):** O backend expõe endpoints específicos para o Wizard após as migrações terem sido executadas com sucesso.
            *   Ex: `POST /api/setup/initialize` (público, mas talvez protegido por um token de setup único ou primeira execução)
            *   Payload: `{ "organization_name": "...", "admin_name": "...", "admin_email": "...", "admin_password": "..." }`
            *   O backend então cria a organização e o admin.
        *   **Opção B:** O frontend instrui o usuário a executar o comando `docker-compose run --rm backend setup` separadamente no terminal e depois voltar ao frontend.

**Fluxo Simplificado Assumindo Endpoints de Setup (Opção A):**

1.  **Frontend (Página de Setup):**
    *   Instrui o usuário a configurar as variáveis de DB no `.env` do backend e reiniciar o backend.
    *   Após o usuário confirmar, o frontend chama um endpoint (ex: `POST /api/public/setup/check-db-and-migrate`).
        *   Backend tenta conectar ao DB e rodar migrações. Retorna sucesso/falha.
    *   Se sucesso, o frontend exibe formulários para:
        *   Nome da Organização.
        *   Nome, Email, Senha do Admin.
    *   Frontend envia esses dados para `POST /api/public/setup/create-admin-org`.
        *   Backend cria a organização e o admin. Retorna sucesso/falha.
    *   Se sucesso, redireciona para a página de login.

**Nota:** Os endpoints `/api/public/setup/*` mencionados acima são sugestões e **não existem atualmente**. A implementação atual do setup é via CLI interativa do backend. Para um Wizard no browser, o backend precisaria ser adaptado para expor tais endpoints.

## 9. Apêndice: Lista de Endpoints Chave (Resumo)

*   `GET /health`
*   `GET /api/public/social-identity-providers`
*   `POST /auth/login`
*   `POST /auth/login/2fa/verify`
*   `POST /auth/login/2fa/backup-code/verify`
*   `GET /auth/oauth2/{provider}/{idpId}/login`
*   `GET /api/v1/me`
*   `GET /api/v1/me/dashboard/summary`
*   CRUD em `/api/v1/risks`
*   CRUD em `/api/v1/vulnerabilities`
*   Endpoints em `/api/v1/audit/...`
*   Endpoints de administração em `/api/v1/organizations/{orgId}/...`
*   `GET /api/v1/users/organization-lookup`
*   `GET /api/v1/files/signed-url`
*   `GET /metrics`

Consulte `API_DOCUMENTATION.md` para detalhes completos de cada um.
