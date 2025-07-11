import {
    UserRole,
    RiskStatus, RiskImpact, RiskProbability, RiskCategory,
    VulnerabilitySeverity, VulnerabilityStatus,
    AuditAssessmentStatus, IdentityProviderType, ApprovalStatus
} from './enums';

// User Related
export interface User {
  id: string;
  name: string;
  email: string;
  role: UserRole | string; // string para acomodar roles não previstas no enum, se vier da API
  organization_id: string;
  is_active?: boolean;      // Comum em listagens de usuários
  is_totp_enabled?: boolean; // Adicionado para status de MFA
  created_at?: string;      // Timestamps são opcionais em alguns contextos de formulário
  updated_at?: string;
}

export interface UserLookup { // Para seletores
    id: string;
    name: string;
}

// Organization (básico, pode ser expandido)
export interface Organization {
    id: string;
    name: string;
    created_at?: string;
    updated_at?: string;
}

// Risk Related
export interface RiskOwner extends UserLookup { // Pode ter mais campos se necessário
    email?: string;
}

export interface Risk {
  id: string;
  organization_id: string;
  title: string;
  description: string;
  category: RiskCategory | string; // string para acomodar valores não previstos
  impact: RiskImpact | string;
  probability: RiskProbability | string;
  status: RiskStatus | string;
  owner_id: string;
  owner?: RiskOwner;
  created_at: string;
  updated_at: string;
  hasPendingApproval?: boolean; // Adicionado na listagem de riscos
}

// Vulnerability Related
export interface Vulnerability {
  id: string;
  organization_id?: string; // Pode vir do contexto do usuário logado
  title: string;
  description: string; // Backend tem, frontend form não tinha, mas é bom ter
  cve_id?: string;
  severity: VulnerabilitySeverity | string;
  status: VulnerabilityStatus | string;
  asset_affected: string;
  created_at: string;
  updated_at: string;
}

// Audit Related
export interface AuditFramework {
  id: string;
  name: string;
  description?: string; // Adicionando, pode ser útil
  created_at?: string;
  updated_at?: string;
}

export interface AuditControl {
  id: string;
  control_id: string;
  name?: string; // Adicionando, para um nome mais amigável além do ID
  description: string;
  family: string;
  framework_id?: string; // Para referência
}

export interface AuditAssessment {
  id: string;
  organization_id: string;
  audit_control_id: string;
  status: AuditAssessmentStatus | string;
  score?: number;
  evidence_url?: string;
  assessment_date?: string;
  created_at: string;
  updated_at: string;
  AuditControl?: AuditControl; // Para dados aninhados
}

export interface ControlWithAssessment extends AuditControl { // Usado na página de detalhes do Framework
  assessment?: AuditAssessment;
}

// Identity Provider Related
export interface IdentityProviderConfigSaml {
    idp_entity_id: string;
    idp_sso_url: string;
    idp_x509_cert: string;
    sign_request?: boolean;
    want_assertions_signed?: boolean;
    // Outros campos SAML...
}
export interface IdentityProviderConfigOAuth2 {
    client_id: string;
    client_secret: string; // Sensível, talvez não deva trafegar para o frontend após criação
    scopes?: string[];
    // Outros campos OAuth2...
}
export type IdentityProviderConfig = IdentityProviderConfigSaml | IdentityProviderConfigOAuth2 | Record<string, any>;

export interface AttributeMapping {
    email?: string;
    name?: string;
    // Outros atributos mapeáveis...
}

export interface IdentityProvider {
  id: string;
  organization_id: string;
  provider_type: IdentityProviderType | string;
  name: string;
  is_active: boolean;
  config_json: string; // String JSON da API
  attribute_mapping_json?: string; // String JSON da API (opcional)
  created_at?: string;
  updated_at?: string;

  // Campos parseados para uso no frontend (não vêm da API diretamente assim)
  config_json_parsed?: IdentityProviderConfig;
  attribute_mapping_json_parsed?: AttributeMapping;
}

// Para o formulário de IdP, pode ser ligeiramente diferente
export interface IdentityProviderFormData {
    name: string;
    provider_type: IdentityProviderType | string;
    is_active: boolean;
    config_json_string: string;
    attribute_mapping_json_string: string;
}


// Approval Workflow Related
export interface ApprovalWorkflow {
  id: string;
  risk_id: string; // Ou genericamente target_id, target_type
  status: ApprovalStatus | string;
  requester_id?: string;
  requester_name?: string; // Denormalizado
  approver_id?: string;
  approver_name?: string; // Denormalizado
  comments?: string;
  created_at?: string;
  updated_at?: string;
}

// Dashboard Related
export interface AdminStatistics {
  active_users_count?: number;
  total_risks_count?: number;
  active_frameworks_count?: number;
  open_vulnerabilities_count?: number;
}

export interface ActivityLog {
  id: string;
  timestamp: string;
  actor_name: string;
  actor_id?: string;
  action_description: string;
  target_type?: string;
  target_id?: string;
  target_link?: string;
}

export interface UserDashboardSummary {
  assigned_risks_open_count?: number;
  assigned_vulnerabilities_open_count?: number;
  pending_approval_tasks_count?: number;
}

// Tipo para IdentityProvider usado na página de login (mais simples)
export interface LoginIdentityProvider {
  id: string; // ID da *configuração* do IdP no sistema Phoenix GRC
  name: string; // Nome amigável para exibir no botão
  type: IdentityProviderType | string; // Tipo do provedor
  login_url: string; // URL completa para iniciar o fluxo, fornecida pelo backend
}
