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
