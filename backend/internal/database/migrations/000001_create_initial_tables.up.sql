-- Migração Inicial: Criação das tabelas base do Phoenix GRC

-- Organizações
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    name VARCHAR(255) NOT NULL UNIQUE,
    logo_url TEXT, -- Armazenará o objectName do GCS/S3 ou URL externa
    primary_color VARCHAR(7),   -- Ex: #RRGGBB
    secondary_color VARCHAR(7) -- Ex: #RRGGBB
);
CREATE INDEX IF NOT EXISTS idx_organizations_deleted_at ON organizations(deleted_at);

-- Usuários
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL, -- Usuário pode ficar sem org se a org for deletada
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user', -- admin, manager, user
    is_active BOOLEAN DEFAULT TRUE,
    sso_provider VARCHAR(100),
    social_login_id TEXT, -- ID do usuário no provedor social
    totp_secret TEXT, -- Criptografado
    is_totp_enabled BOOLEAN DEFAULT FALSE,
    mfa_backup_codes_hashed TEXT, -- JSON array de hashes de códigos de backup
    last_login_at TIMESTAMP WITH TIME ZONE,
    failed_login_attempts INT DEFAULT 0,
    lockout_until TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_organization_id ON users(organization_id);

-- Provedores de Identidade (para SSO por organização)
CREATE TABLE IF NOT EXISTS identity_providers (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    provider_type VARCHAR(50) NOT NULL, -- ex: saml, oauth2_google, oauth2_github
    config_json TEXT NOT NULL, -- Configurações específicas do provedor em JSON
    attribute_mapping_json TEXT, -- Mapeamento de atributos em JSON
    is_active BOOLEAN DEFAULT TRUE,
    is_public BOOLEAN DEFAULT FALSE -- Se deve ser listado em /api/public/social-identity-providers
);
CREATE INDEX IF NOT EXISTS idx_identity_providers_deleted_at ON identity_providers(deleted_at);
CREATE INDEX IF NOT EXISTS idx_identity_providers_organization_id ON identity_providers(organization_id);

-- Riscos
CREATE TABLE IF NOT EXISTS risks (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    owner_id UUID REFERENCES users(id) ON DELETE SET NULL, -- Proprietário do risco
    title VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100), -- tecnologico, operacional, legal
    impact VARCHAR(50),     -- Baixo, Médio, Alto, Crítico
    probability VARCHAR(50),-- Baixa, Média, Alta, Crítica
    risk_level VARCHAR(50), -- Calculado: Baixo, Médio, Alto, Crítico
    status VARCHAR(50) DEFAULT 'aberto', -- aberto, em_andamento, mitigado, aceito
    next_review_date TIMESTAMP WITH TIME ZONE,
    last_reviewed_at TIMESTAMP WITH TIME ZONE,
    mitigation_details TEXT,
    acceptance_justification TEXT,
    custom_fields JSONB -- Para campos customizáveis
);
CREATE INDEX IF NOT EXISTS idx_risks_deleted_at ON risks(deleted_at);
CREATE INDEX IF NOT EXISTS idx_risks_organization_id ON risks(organization_id);
CREATE INDEX IF NOT EXISTS idx_risks_owner_id ON risks(owner_id);
CREATE INDEX IF NOT EXISTS idx_risks_status ON risks(status);

-- Vulnerabilidades
CREATE TABLE IF NOT EXISTS vulnerabilities (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    cve_id VARCHAR(50),
    severity VARCHAR(50) NOT NULL, -- Baixo, Médio, Alto, Crítico
    status VARCHAR(50) DEFAULT 'descoberta', -- descoberta, em_correcao, corrigida, aceita_risco
    asset_affected TEXT,
    remediation_details TEXT,
    cvss_score NUMERIC(3,1) -- Ex: 7.5
);
CREATE INDEX IF NOT EXISTS idx_vulnerabilities_deleted_at ON vulnerabilities(deleted_at);
CREATE INDEX IF NOT EXISTS idx_vulnerabilities_organization_id ON vulnerabilities(organization_id);
CREATE INDEX IF NOT EXISTS idx_vulnerabilities_status ON vulnerabilities(status);
CREATE INDEX IF NOT EXISTS idx_vulnerabilities_severity ON vulnerabilities(severity);

-- Frameworks de Auditoria
CREATE TABLE IF NOT EXISTS audit_frameworks (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    version VARCHAR(50)
);
CREATE INDEX IF NOT EXISTS idx_audit_frameworks_deleted_at ON audit_frameworks(deleted_at);

-- Controles de Auditoria
CREATE TABLE IF NOT EXISTS audit_controls (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    framework_id UUID NOT NULL REFERENCES audit_frameworks(id) ON DELETE CASCADE,
    control_id VARCHAR(100) NOT NULL, -- Ex: AC-1, NIST.CM-3
    description TEXT NOT NULL,
    family VARCHAR(255), -- Ex: Access Control, Configuration Management
    UNIQUE (framework_id, control_id)
);
CREATE INDEX IF NOT EXISTS idx_audit_controls_deleted_at ON audit_controls(deleted_at);
CREATE INDEX IF NOT EXISTS idx_audit_controls_framework_id ON audit_controls(framework_id);

-- Avaliações de Auditoria
CREATE TABLE IF NOT EXISTS audit_assessments (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    audit_control_id UUID NOT NULL REFERENCES audit_controls(id) ON DELETE CASCADE,
    assessed_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL, -- conforme, nao_conforme, parcialmente_conforme, nao_aplicavel
    evidence_url TEXT, -- Armazenará o objectName do GCS/S3 ou URL externa
    score INTEGER, -- 0-100
    assessment_date TIMESTAMP WITH TIME ZONE,
    comments TEXT,
    UNIQUE (organization_id, audit_control_id) -- Uma avaliação por controle por organização
);
CREATE INDEX IF NOT EXISTS idx_audit_assessments_deleted_at ON audit_assessments(deleted_at);
CREATE INDEX IF NOT EXISTS idx_audit_assessments_organization_id ON audit_assessments(organization_id);
CREATE INDEX IF NOT EXISTS idx_audit_assessments_audit_control_id ON audit_assessments(audit_control_id);

-- Configurações de Webhook
CREATE TABLE IF NOT EXISTS webhook_configurations (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    url TEXT NOT NULL,
    secret_token TEXT, -- Para verificar a autenticidade do payload
    event_types TEXT NOT NULL, -- CSV de eventos, ex: "risk_created,risk_updated"
    is_active BOOLEAN DEFAULT TRUE
);
CREATE INDEX IF NOT EXISTS idx_webhook_configurations_deleted_at ON webhook_configurations(deleted_at);
CREATE INDEX IF NOT EXISTS idx_webhook_configurations_organization_id ON webhook_configurations(organization_id);

-- Tabela de Junção: Stakeholders de Risco
CREATE TABLE IF NOT EXISTS risk_stakeholders (
    risk_id UUID NOT NULL REFERENCES risks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (risk_id, user_id)
);

-- Workflows de Aprovação (ex: para aceite de risco)
CREATE TABLE IF NOT EXISTS approval_workflows (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    risk_id UUID UNIQUE REFERENCES risks(id) ON DELETE CASCADE, -- Um workflow de aprovação ativo por risco
    requester_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    approver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- Geralmente o Owner do Risco
    status VARCHAR(50) NOT NULL DEFAULT 'pendente', -- pendente, aprovado, rejeitado
    decision_date TIMESTAMP WITH TIME ZONE,
    comments TEXT
);
CREATE INDEX IF NOT EXISTS idx_approval_workflows_deleted_at ON approval_workflows(deleted_at);
CREATE INDEX IF NOT EXISTS idx_approval_workflows_organization_id ON approval_workflows(organization_id);
CREATE INDEX IF NOT EXISTS idx_approval_workflows_risk_id ON approval_workflows(risk_id);
CREATE INDEX IF NOT EXISTS idx_approval_workflows_status ON approval_workflows(status);

-- Adicionar outras tabelas conforme necessário (ex: Ativos, Políticas, etc.)
-- Lembre-se de adicionar índices para colunas frequentemente usadas em queries (filtros, joins).
-- As constraints de chave estrangeira com ON DELETE CASCADE ou ON DELETE SET NULL são importantes.
-- GORM pode adicionar alguns índices automaticamente (ex: para chaves primárias e estrangeiras),
-- mas é bom ser explícito para outros campos de busca comum.
-- O tipo TEXT é usado para campos que podem ser longos. VARCHAR(255) para strings mais curtas.
-- JSONB é usado para campos customizáveis para flexibilidade.
-- Timestamps são WITH TIME ZONE para consistência.
-- `deleted_at` é para soft delete, GORM o usa automaticamente.
-- UNIQUE constraints são importantes para integridade (ex: email do usuário, nome da organização).
-- Default values são definidos onde apropriado.
-- A ordem de criação das tabelas importa se houver FKs sem DEFERRABLE.
-- Aqui, assumimos que o SGBD lida com a ordem ou as FKs são criadas após todas as tabelas.
-- Para PostgreSQL, geralmente não é um problema se todas as tabelas referenciadas existirem.
-- Para um primeiro schema, esta é uma base sólida.
-- Em um projeto real, este arquivo seria gerado com muito mais precisão e detalhes.
-- Incluindo todos os índices, constraints e tipos de dados exatos que o GORM geraria.
-- E também dados de seed iniciais, se necessário (ex: frameworks de auditoria padrão).
-- (Fim do arquivo 000001_create_initial_tables.up.sql)
