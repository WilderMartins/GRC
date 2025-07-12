// Interface genérica para respostas de API paginadas
export interface PaginatedResponse<T> {
  items: T[];
  total_items: number;
  total_pages: number;
  page: number;
  page_size: number;
}

// Pode-se adicionar outros tipos de resposta de API comuns aqui,
// por exemplo, uma estrutura de erro padrão, se houver.
// export interface ApiErrorResponse {
//   error: string;
//   message?: string;
//   details?: Record<string, any>;
// }

// Exemplo de como poderia ser usado com os modelos:
// import { Risk } from './models';
// type PaginatedRisks = PaginatedResponse<Risk>;

// Resposta para o endpoint de lookup de usuários da organização
export interface UserLookupResponse {
  id: string; // UUID do usuário
  name: string; // Nome do usuário
}

// Resposta para o endpoint de sumário do dashboard do usuário
export interface UserDashboardSummary {
  assigned_risks_open_count: number;
  assigned_vulnerabilities_open_count: number;
  pending_approval_tasks_count: number;
}

// Resposta para o endpoint de score de conformidade
export interface ComplianceScoreResponse {
    framework_id: string;
    framework_name: string;
    organization_id: string;
    compliance_score: number;
    total_controls: number;
    evaluated_controls: number;
    conformant_controls: number;
    partially_conformant_controls: number;
    non_conformant_controls: number;
}

// Resposta para o endpoint de sumário de maturidade C2M2
export interface C2M2MaturityFrameworkSummaryResponse {
    framework_id: string;
    framework_name: string;
    organization_id: string;
    summary_by_function: C2M2MaturitySummaryItem[];
}

export interface C2M2MaturitySummaryItem {
    nist_component_type: string;
    nist_component_name: string;
    achieved_mil: number;
    evaluated_controls: number;
    total_controls: number;
    mil_distribution: {
        mil0: number;
        mil1: number;
        mil2: number;
        mil3: number;
    };
}
