// User Roles
export type UserRole = "admin" | "manager" | "user" | ""; // "" para filtros "todos"

// Risk Related Enums
export type RiskStatus = "aberto" | "em_andamento" | "mitigado" | "aceito" | "";
export type RiskImpact = "Baixo" | "Médio" | "Alto" | "Crítico" | "";
export type RiskProbability = "Baixo" | "Médio" | "Alto" | "Crítico" | "";
export type RiskCategory = "tecnologico" | "operacional" | "legal" | "";

// Vulnerability Related Enums
export type VulnerabilitySeverity = "Baixo" | "Médio" | "Alto" | "Crítico" | "";
export type VulnerabilityStatus = "descoberta" | "em_correcao" | "corrigida" | "";

// Audit Related Enums
export type AuditAssessmentStatus = "conforme" | "nao_conforme" | "parcialmente_conforme" | "";
// Usado no filtro da página de detalhes do Framework, inclui a opção "Não Avaliado"
export type AuditAssessmentStatusFilter = AuditAssessmentStatus | "nao_avaliado" | "";


// Identity Provider Related Enums
export type IdentityProviderType = "saml" | "oauth2_google" | "oauth2_github" | "";

// Approval Workflow Related Enums
export type ApprovalDecision = "aprovado" | "rejeitado";
export type ApprovalStatus = "pendente" | "aprovado" | "rejeitado" | ""; // "" para filtros

// General Sort Order
export type SortOrder = "asc" | "desc";
