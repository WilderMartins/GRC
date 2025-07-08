import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import ApprovalDecisionModal from '@/components/risks/ApprovalDecisionModal';
import RiskBulkUploadModal from '@/components/risks/RiskBulkUploadModal'; // Importar o modal de upload

import { useEffect, useState, useCallback } from 'react'; // Adicionado useCallback
import apiClient from '@/lib/axios'; // Ajuste o path se necessário
import { useAuth } from '@/contexts/AuthContext'; // Para verificar role do usuário

// Tipos do backend (idealmente compartilhados ou gerados)
type RiskStatus = "aberto" | "em_andamento" | "mitigado" | "aceito";
type RiskImpact = "Baixo" | "Médio" | "Alto" | "Crítico";
type RiskProbability = "Baixo" | "Médio" | "Alto" | "Crítico";

interface RiskOwner { // Supondo que o preload de Owner retorne pelo menos isso
    id: string;
    name: string;
    email: string;
}
interface Risk {
  id: string;
  organization_id: string;
  title: string;
  description: string;
  category: string;
  impact: RiskImpact;
  probability: RiskProbability;
  status: RiskStatus;
  owner_id: string;
  owner?: RiskOwner; // GORM Preload pode popular isso
  created_at: string;
  updated_at: string;
  hasPendingApproval?: boolean; // Novo campo para UI
}

// Adicionar tipo para ApprovalWorkflow (simplificado)
interface ApprovalWorkflow {
  id: string;
  risk_id: string;
  status: string; // "pendente", "aprovado", "rejeitado"
  // Outros campos se necessário para a lógica de exibição
}


interface PaginatedRisksResponse {
  items: Risk[];
  total_items: number;
  total_pages: number;
  page: number;
  page_size: number;
}

const RisksPageContent = () => {
  const [risks, setRisks] = useState<Risk[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10); // Pode ser configurável
  const [totalPages, setTotalPages] = useState(0);
  const [totalItems, setTotalItems] = useState(0);
  const { user } = useAuth(); // Para verificar a role do usuário

  const [showDecisionModal, setShowDecisionModal] = useState(false);
  const [selectedRiskForDecision, setSelectedRiskForDecision] = useState<Risk | null>(null);
  const [pendingApprovalWorkflowId, setPendingApprovalWorkflowId] = useState<string | null>(null);

  const [showUploadModal, setShowUploadModal] = useState(false); // Estado para o modal de upload


  const fetchRisks = useCallback(async (page: number, size: number) => { // Envolver com useCallback
    setIsLoading(true);
    setError(null);
    try {
      const response = await apiClient.get<PaginatedRisksResponse>('/risks', {
        params: { page, page_size: size },
      });

      const risksData = response.data.items || [];
      const processedRisks = await Promise.all(
        risksData.map(async (risk) => {
          if (user && risk.owner_id === user.id) {
            try {
              const historyResponse = await apiClient.get<ApprovalWorkflow[]>(`/risks/${risk.id}/approval-history`);
              const pendingApproval = historyResponse.data.find(wf => wf.status === 'pendente');
              return { ...risk, hasPendingApproval: !!pendingApproval };
            } catch (historyErr) {
              console.error(`Erro ao buscar histórico de aprovação para risco ${risk.id}:`, historyErr);
              return { ...risk, hasPendingApproval: false }; // Assume não pendente em caso de erro
            }
          }
          return { ...risk, hasPendingApproval: false };
        })
      );

      setRisks(processedRisks);
      setTotalItems(response.data.total_items);
      setTotalPages(response.data.total_pages);
      setCurrentPage(response.data.page);
      setPageSize(response.data.page_size);

    } catch (err: any) {
      console.error("Erro ao buscar riscos:", err);
      setError(err.response?.data?.error || err.message || "Falha ao buscar riscos.");
      setRisks([]); // Limpar riscos em caso de erro
    } finally {
      setIsLoading(false);
    }
  };

  const handleOpenDecisionModal = async (risk: Risk) => {
    // Precisamos do ID do ApprovalWorkflow pendente.
    // A flag `hasPendingApproval` apenas indica que existe um.
    // Vamos buscar o histórico e pegar o ID do pendente.
    // Isso poderia ser otimizado se a API de riscos já trouxesse o ID do workflow pendente.
    setIsLoading(true); // Usar um loader específico para esta ação seria melhor
    try {
      const historyResponse = await apiClient.get<ApprovalWorkflow[]>(`/risks/${risk.id}/approval-history`);
      const pendingWF = historyResponse.data.find(wf => wf.status === 'pendente');
      if (pendingWF) {
        setSelectedRiskForDecision(risk);
        setPendingApprovalWorkflowId(pendingWF.id);
        setShowDecisionModal(true);
      } else {
        alert("Não foi encontrado um workflow de aprovação pendente para este risco. A lista pode estar desatualizada.");
        fetchRisks(currentPage, pageSize); // Re-sincronizar
      }
    } catch (err) {
      console.error("Erro ao buscar workflow pendente:", err);
      alert("Erro ao verificar status de aprovação do risco.");
    } finally {
      setIsLoading(false);
    }
  };

  const handleCloseDecisionModal = () => {
    setShowDecisionModal(false);
    setSelectedRiskForDecision(null);
    setPendingApprovalWorkflowId(null);
  };

  const handleDecisionSubmitSuccess = () => {
    fetchRisks(currentPage, pageSize); // Re-fetch para atualizar status do risco e remover badge/botão
    // O modal já se fecha no seu próprio onSubmitSuccess
  };

  const handleSubmitForAcceptance = async (riskId: string, riskTitle: string) => {
    // Idealmente, um estado de loading específico para esta ação
    // const [isSubmitting, setIsSubmitting] = useState(false);
    // setIsSubmitting(true);
    if (window.confirm(`Tem certeza que deseja submeter o risco "${riskTitle}" para aceite?`)) {
        try {
            await apiClient.post(`/risks/${riskId}/submit-acceptance`);
            alert(`Risco "${riskTitle}" submetido para aceite com sucesso.`);
            // Atualizar a UI: pode ser recarregando os dados ou atualizando o status do risco localmente.
            // Recarregar é mais simples por enquanto.
            fetchRisks(currentPage, pageSize);
        } catch (err: any) {
            console.error("Erro ao submeter risco para aceite:", err);
            setError(err.response?.data?.error || err.message || "Falha ao submeter risco para aceite.");
            // Limpar o erro após um tempo ou quando o usuário interagir novamente
            setTimeout(() => setError(null), 5000);
        } finally {
            // setIsSubmitting(false);
        }
    }
  };

  useEffect(() => {
    fetchRisks(currentPage, pageSize);
  }, [currentPage, pageSize, fetchRisks]); // Adicionar fetchRisks às dependências do useEffect


  const handlePreviousPage = () => {
    if (currentPage > 1) {
      setCurrentPage(currentPage - 1);
    }
  };

  const handleNextPage = () => {
    if (currentPage < totalPages) {
      setCurrentPage(currentPage + 1);
    }
  };

  const handleDeleteRisk = async (riskId: string, riskTitle: string) => {
    if (window.confirm(`Tem certeza que deseja deletar o risco "${riskTitle}"? Esta ação não pode ser desfeita.`)) {
      // Idealmente, ter um estado de loading específico para a deleção da linha
      // Para simplificar, vamos reusar o isLoading geral ou adicionar um novo se necessário.
      // setIsLoading(true); // Ou um setLoadingDelete(true)
      try {
        await apiClient.delete(`/risks/${riskId}`);
        alert(`Risco "${riskTitle}" deletado com sucesso.`); // Placeholder para notificação melhor
        // Re-buscar os riscos para atualizar a lista
        // Se estiver na última página e ela ficar vazia, ajustar currentPage
        if (risks.length === 1 && currentPage > 1) {
            setCurrentPage(currentPage - 1);
        } else {
            fetchRisks(currentPage, pageSize); // Re-fetch a página atual
        }
      } catch (err: any) {
        console.error("Erro ao deletar risco:", err);
        setError(err.response?.data?.error || err.message || "Falha ao deletar risco.");
      } finally {
        // setIsLoading(false); // Ou setLoadingDelete(false)
      }
    }
  };


  return (
    <AdminLayout title="Gestão de Riscos - Phoenix GRC">
      <Head>
        <title>Gestão de Riscos - Phoenix GRC</title>
      </Head>

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex flex-col sm:flex-row justify-between items-center mb-6 gap-3">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            Gestão de Riscos
          </h1>
          <div className="flex space-x-3">
            <button
              onClick={() => setShowUploadModal(true)}
              className="inline-flex items-center justify-center rounded-md border border-transparent bg-green-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800"
            >
              Importar Riscos CSV
            </button>
            <Link href="/admin/risks/new" legacyBehavior>
              <a className="inline-flex items-center justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800">
                Adicionar Novo Risco
              </a>
            </Link>
          </div>
        </div>

        {isLoading && <p className="text-center text-gray-500 dark:text-gray-400 py-4">Carregando riscos...</p>}
        {error && <p className="text-center text-red-500 py-4">Erro ao carregar riscos: {error}</p>}

        {!isLoading && !error && (
          <>
            <div className="mt-8 flow-root">
              {/* ... (código da tabela e paginação existente) ... */}
              <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                  <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                    <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                      <thead className="bg-gray-50 dark:bg-gray-700">
                        <tr>
                          <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-6">Título</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Categoria</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Impacto</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Probabilidade</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Status</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Proprietário</th>
                          <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6">
                            <span className="sr-only">Ações</span>
                          </th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                        {risks.map((risk) => (
                          <tr key={risk.id}>
                            <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">
                              {risk.title}
                              {risk.hasPendingApproval && (
                                <span className="ml-2 px-2 py-0.5 inline-flex text-xs leading-5 font-semibold rounded-full bg-yellow-500 text-white animate-pulse" title="Aprovação Pendente">
                                  Pendente
                                </span>
                              )}
                              {risk.hasPendingApproval && user?.id === risk.owner_id && (
                                <button
                                  onClick={() => handleOpenDecisionModal(risk)}
                                  className="ml-2 px-2 py-0.5 text-xs bg-blue-500 text-white rounded-full hover:bg-blue-600"
                                  title="Tomar Decisão sobre Aceite"
                                >
                                  Decidir
                                </button>
                              )}
                            </td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{risk.category}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{risk.impact}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{risk.probability}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">
                              <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                    risk.status === 'aberto' ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-700 dark:text-yellow-100' :
                                    risk.status === 'em_andamento' ? 'bg-blue-100 text-blue-800 dark:bg-blue-700 dark:text-blue-100' :
                                    risk.status === 'mitigado' ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' :
                                    risk.status === 'aceito' ? 'bg-purple-100 text-purple-800 dark:bg-purple-700 dark:text-purple-100' :
                                    'bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-200'
                                }`}>
                                    {risk.status}
                                </span>
                            </td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{risk.owner?.name || risk.owner_id}</td>
                            <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6 space-x-2">
                              <Link href={`/admin/risks/edit/${risk.id}`} legacyBehavior><a className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-200">Editar</a></Link>
                              <button
                                onClick={() => handleDeleteRisk(risk.id, risk.title)}
                                className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-200"
                                disabled={isLoading}
                              >
                                Deletar
                              </button>
                              {(user?.role === 'admin' || user?.role === 'manager') && risk.status !== 'aceito' && risk.status !== 'mitigado' && !risk.hasPendingApproval && (
                                <button
                                  onClick={() => handleSubmitForAcceptance(risk.id, risk.title)}
                                  className="ml-2 text-green-600 hover:text-green-900 dark:text-green-400 dark:hover:text-green-200"
                                  disabled={isLoading}
                                >
                                  Submeter p/ Aceite
                                </button>
                              )}
                            </td>
                          </tr>
                        ))}
                        {risks.length === 0 && (
                            <tr>
                                <td colSpan={7} className="text-center py-4 text-sm text-gray-500 dark:text-gray-400">
                                    Nenhum risco encontrado.
                                </td>
                            </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                  {totalPages > 0 && (
                    <nav
                      className="flex items-center justify-between border-t border-gray-200 bg-white dark:bg-gray-800 px-4 py-3 sm:px-6"
                      aria-label="Paginação"
                    >
                      <div className="hidden sm:block">
                        <p className="text-sm text-gray-700 dark:text-gray-300">
                          Mostrando <span className="font-medium">{(currentPage - 1) * pageSize + 1}</span>
                          {' '}a <span className="font-medium">{Math.min(currentPage * pageSize, totalItems)}</span>
                          {' '}de <span className="font-medium">{totalItems}</span> resultados
                        </p>
                      </div>
                      <div className="flex flex-1 justify-between sm:justify-end">
                        <button
                          onClick={handlePreviousPage}
                          disabled={currentPage <= 1 || isLoading}
                          className="relative inline-flex items-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
                        >
                          Anterior
                        </button>
                        <button
                          onClick={handleNextPage}
                          disabled={currentPage >= totalPages || isLoading}
                          className="relative ml-3 inline-flex items-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
                        >
                          Próxima
                        </button>
                      </div>
                    </nav>
                  )}
                </div>
              </div>
            </div>
          </>
        )}
      </div>
      {showDecisionModal && selectedRiskForDecision && pendingApprovalWorkflowId && (
        <ApprovalDecisionModal
            riskId={selectedRiskForDecision.id}
            riskTitle={selectedRiskForDecision.title}
            approvalId={pendingApprovalWorkflowId}
            currentApproverId={selectedRiskForDecision.owner_id} // O backend valida se o user logado é este
            onClose={handleCloseDecisionModal}
            onSubmitSuccess={handleDecisionSubmitSuccess}
        />
      )}

      <RiskBulkUploadModal
        isOpen={showUploadModal}
        onClose={() => setShowUploadModal(false)}
        onUploadSuccess={() => {
          fetchRisks(1, pageSize); // Voltar para a primeira página após upload ou manter a atual? Por ora, primeira.
          setShowUploadModal(false); // Fechar modal após o upload ser processado no componente filho
        }}
      />
    </AdminLayout>
  );
};

export default WithAuth(RisksPageContent);
