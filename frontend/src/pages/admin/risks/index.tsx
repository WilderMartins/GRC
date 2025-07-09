import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import ApprovalDecisionModal from '@/components/risks/ApprovalDecisionModal';
import RiskBulkUploadModal from '@/components/risks/RiskBulkUploadModal';
import PaginationControls from '@/components/common/PaginationControls';
import { useNotifier } from '@/hooks/useNotifier';
import { useAuth } from '@/contexts/AuthContext';
import { useEffect, useState, useCallback } from 'react';
import apiClient from '@/lib/axios';
import { useDebounce } from '@/hooks/useDebounce'; // Supondo que este hook exista
import {
    Risk,
    RiskOwner, // RiskOwner é específico aqui, mas poderia ser UserLookup se os campos baterem
    ApprovalWorkflow,
    UserLookup,
    PaginatedResponse,
    RiskStatus,      // Usado para filterStatus
    RiskImpact,      // Usado para filterImpact
    RiskProbability, // Usado para filterProbability
    RiskCategory,    // Usado para filterCategory
    SortOrder
} from '@/types';

// Definições locais de tipos e interfaces removidas

const RisksPageContent = () => {
  const notify = useNotifier();
  const { user } = useAuth();

  const [risks, setRisks] = useState<Risk[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null); // Erro principal da busca

  // Paginação
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalPages, setTotalPages] = useState(0);
  const [totalItems, setTotalItems] = useState(0);

  // Filtros
  const [filterCategory, setFilterCategory] = useState<RiskCategoryFilter>("");
  const [filterImpact, setFilterImpact] = useState<RiskImpactFilter>("");
  const [filterProbability, setFilterProbability] = useState<RiskProbabilityFilter>("");
  const [filterStatus, setFilterStatus] = useState<RiskStatusFilter>("");
  const [filterOwnerId, setFilterOwnerId] = useState<string>("");
  const [searchTermTitle, setSearchTermTitle] = useState<string>("");
  const debouncedSearchTitle = useDebounce(searchTermTitle, 500);
  const [ownersForFilter, setOwnersForFilter] = useState<UserLookup[]>([]);
  const [isLoadingOwners, setIsLoadingOwners] = useState(false);

  // Ordenação
  const [sortBy, setSortBy] = useState<string>('created_at');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');

  // Modais
  const [showDecisionModal, setShowDecisionModal] = useState(false);
  const [selectedRiskForDecision, setSelectedRiskForDecision] = useState<Risk | null>(null);
  const [pendingApprovalWorkflowId, setPendingApprovalWorkflowId] = useState<string | null>(null);
  const [showUploadModal, setShowUploadModal] = useState(false);

  // Buscar proprietários para o filtro
  useEffect(() => {
    const fetchOwners = async () => {
      if (!user) return;
      setIsLoadingOwners(true);
      try {
        const response = await apiClient.get<UserLookup[]>('/users/organization-lookup');
        setOwnersForFilter(response.data || []);
      } catch (err) {
        console.error("Erro ao buscar proprietários para filtro:", err);
        notify.error("Falha ao carregar lista de proprietários para o filtro.");
      } finally {
        setIsLoadingOwners(false);
      }
    };
    fetchOwners();
  }, [user, notify]);

  const fetchRisks = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const params: any = {
        page: currentPage,
        page_size: pageSize,
        sort_by: sortBy,
        order: sortOrder,
      };
      if (filterCategory) params.category = filterCategory;
      if (filterImpact) params.impact = filterImpact;
      if (filterProbability) params.probability = filterProbability;
      if (filterStatus) params.status = filterStatus;
      if (filterOwnerId) params.owner_id = filterOwnerId;
      if (debouncedSearchTitle) params.title_like = debouncedSearchTitle;

      const response = await apiClient.get<PaginatedRisksResponse>('/risks', { params });

      // Processar hasPendingApproval (mantendo lógica N+1 por enquanto)
      const risksData = response.data.items || [];
      const processedRisks = await Promise.all(
        risksData.map(async (risk) => {
          if (user && risk.owner_id === user.id) { // Apenas checa para o usuário logado
            try {
              const historyResponse = await apiClient.get<ApprovalWorkflow[]>(`/risks/${risk.id}/approval-history`);
              const pendingApproval = historyResponse.data.find(wf => wf.status === 'pendente');
              return { ...risk, hasPendingApproval: !!pendingApproval };
            } catch (historyErr) {
              return { ...risk, hasPendingApproval: false };
            }
          }
          return { ...risk, hasPendingApproval: false };
        })
      );

      setRisks(processedRisks);
      setTotalItems(response.data.total_items);
      setTotalPages(response.data.total_pages);
      // A API retorna a página e page_size, mas vamos confiar nos estados do frontend
      // setCurrentPage(response.data.page);
      // setPageSize(response.data.page_size);

    } catch (err: any) {
      console.error("Erro ao buscar riscos:", err);
      setError(err.response?.data?.error || "Falha ao buscar riscos.");
      setRisks([]);
      setTotalItems(0);
      setTotalPages(0);
    } finally {
      setIsLoading(false);
    }
  }, [currentPage, pageSize, sortBy, sortOrder, filterCategory, filterImpact, filterProbability, filterStatus, filterOwnerId, debouncedSearchTitle, user]);

  useEffect(() => {
    fetchRisks();
  }, [fetchRisks]);

  // Resetar para primeira página ao mudar filtros ou ordenação
  useEffect(() => {
    if (currentPage !== 1) {
        setCurrentPage(1);
    }
  }, [filterCategory, filterImpact, filterProbability, filterStatus, filterOwnerId, debouncedSearchTitle, sortBy, sortOrder]);


  const handlePageChange = (newPage: number) => {
    setCurrentPage(newPage);
  };

  const handleSort = (newSortBy: string) => {
    if (sortBy === newSortBy) {
      setSortOrder(prevOrder => prevOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(newSortBy);
      setSortOrder('asc');
    }
  };

  const clearFilters = () => {
    setFilterCategory("");
    setFilterImpact("");
    setFilterProbability("");
    setFilterStatus("");
    setFilterOwnerId("");
    setSearchTermTitle("");
    setSortBy('created_at');
    setSortOrder('desc');
    if (currentPage !== 1) setCurrentPage(1);
  };

  const handleOpenDecisionModal = async (risk: Risk) => {
    setIsLoading(true);
    try {
      const historyResponse = await apiClient.get<ApprovalWorkflow[]>(`/risks/${risk.id}/approval-history`);
      const pendingWF = historyResponse.data.find(wf => wf.status === 'pendente');
      if (pendingWF) {
        setSelectedRiskForDecision(risk);
        setPendingApprovalWorkflowId(pendingWF.id);
        setShowDecisionModal(true);
      } else {
        notify.info("Não foi encontrado um workflow de aprovação pendente para este risco.");
        fetchRisks();
      }
    } catch (err) {
      notify.error("Erro ao verificar status de aprovação do risco.");
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
    fetchRisks();
    // O modal já se fecha
  };

  const handleSubmitForAcceptance = async (riskId: string, riskTitle: string) => {
    if (window.confirm(`Tem certeza que deseja submeter o risco "${riskTitle}" para aceite?`)) {
        try {
            await apiClient.post(`/risks/${riskId}/submit-acceptance`);
            notify.success(`Risco "${riskTitle}" submetido para aceite com sucesso.`);
            fetchRisks();
        } catch (err: any) {
            notify.error(err.response?.data?.error || "Falha ao submeter risco para aceite.");
        }
    }
  };

  const handleDeleteRisk = async (riskId: string, riskTitle: string) => {
    if (window.confirm(`Tem certeza que deseja deletar o risco "${riskTitle}"? Esta ação não pode ser desfeita.`)) {
      setIsLoading(true);
      try {
        await apiClient.delete(`/risks/${riskId}`);
        notify.success(`Risco "${riskTitle}" deletado com sucesso.`);
        if (risks.length === 1 && currentPage > 1) {
            setCurrentPage(currentPage - 1); // Trigger fetch via useEffect
        } else {
            fetchRisks();
        }
      } catch (err: any) {
        notify.error(err.response?.data?.error || "Falha ao deletar risco.");
      } finally {
        // setIsLoading(false); // fetchRisks() cuidará disso
      }
    }
  };

  const TableHeader: React.FC<{ field: string; label: string }> = ({ field, label }) => (
    <th scope="col" className="py-3.5 px-3 text-left text-sm font-semibold text-gray-900 dark:text-white cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600 whitespace-nowrap"
        onClick={() => handleSort(field)}>
      {label}
      {sortBy === field && (sortOrder === 'asc' ? ' ▲' : ' ▼')}
    </th>
  );

  return (
    <AdminLayout title="Gestão de Riscos - Phoenix GRC">
      <Head><title>Gestão de Riscos - Phoenix GRC</title></Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex flex-col sm:flex-row justify-between items-center mb-6 gap-3">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">Gestão de Riscos</h1>
          <div className="flex space-x-3">
            <button onClick={() => setShowUploadModal(true)}
              className="inline-flex items-center justify-center rounded-md border border-transparent bg-green-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800">
              Importar CSV
            </button>
            <Link href="/admin/risks/new" legacyBehavior>
              <a className="inline-flex items-center justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800">
                Novo Risco
              </a>
            </Link>
          </div>
        </div>

        {/* Filtros UI */}
        <div className="my-6 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg shadow">
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7 gap-4 items-end">
            <div>
              <label htmlFor="searchTermTitle" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Título</label>
              <input type="text" id="searchTermTitle" value={searchTermTitle} onChange={(e) => setSearchTermTitle(e.target.value)}
                     placeholder="Buscar por título..."
                     className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md"/>
            </div>
            <div>
              <label htmlFor="filterCategory" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Categoria</label>
              <select id="filterCategory" value={filterCategory} onChange={(e) => setFilterCategory(e.target.value as RiskCategoryFilter)}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                <option value="">Todas</option>
                <option value="tecnologico">Tecnológico</option>
                <option value="operacional">Operacional</option>
                <option value="legal">Legal</option>
              </select>
            </div>
             <div>
              <label htmlFor="filterImpact" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Impacto</label>
              <select id="filterImpact" value={filterImpact} onChange={(e) => setFilterImpact(e.target.value as RiskImpactFilter)}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                <option value="">Todos</option>
                <option value="Crítico">Crítico</option>
                <option value="Alto">Alto</option>
                <option value="Médio">Médio</option>
                <option value="Baixo">Baixo</option>
              </select>
            </div>
            <div>
              <label htmlFor="filterProbability" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Probabilidade</label>
              <select id="filterProbability" value={filterProbability} onChange={(e) => setFilterProbability(e.target.value as RiskProbabilityFilter)}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                <option value="">Todas</option>
                <option value="Crítico">Crítico</option> {/* Assumindo que probabilidade também pode ser Crítico */}
                <option value="Alto">Alto</option>
                <option value="Médio">Médio</option>
                <option value="Baixo">Baixo</option>
              </select>
            </div>
            <div>
              <label htmlFor="filterStatus" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Status</label>
              <select id="filterStatus" value={filterStatus} onChange={(e) => setFilterStatus(e.target.value as RiskStatusFilter)}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                <option value="">Todos</option>
                <option value="aberto">Aberto</option>
                <option value="em_andamento">Em Andamento</option>
                <option value="mitigado">Mitigado</option>
                <option value="aceito">Aceito</option>
              </select>
            </div>
            <div>
              <label htmlFor="filterOwnerId" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Proprietário</label>
              <select id="filterOwnerId" value={filterOwnerId} onChange={(e) => setFilterOwnerId(e.target.value)}
                      disabled={isLoadingOwners}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md disabled:opacity-50">
                <option value="">Todos</option>
                {isLoadingOwners && <option value="" disabled>Carregando...</option>}
                {ownersForFilter.map(owner => (
                  <option key={owner.id} value={owner.id}>{owner.name}</option>
                ))}
              </select>
            </div>
            <div>
              <button onClick={clearFilters}
                      className="w-full inline-flex items-center justify-center rounded-md border border-gray-300 dark:border-gray-500 bg-white dark:bg-gray-600 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-100 shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2">
                Limpar Filtros
              </button>
            </div>
          </div>
        </div>

        {isLoading && <p className="text-center text-gray-500 dark:text-gray-400 py-4">Carregando riscos...</p>}
        {error && <p className="text-center text-red-500 py-4">Erro ao carregar riscos: {error}</p>}

        {!isLoading && !error && risks.length === 0 && (
             <div className="text-center py-10">
                <p className="text-gray-500 dark:text-gray-400">Nenhum risco encontrado com os filtros aplicados.</p>
            </div>
        )}

        {!isLoading && !error && risks.length > 0 && (
          <>
            <div className="mt-8 flow-root">
              <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                  <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                    <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                      <thead className="bg-gray-50 dark:bg-gray-700">
                        <tr>
                          <TableHeader field="title" label="Título" />
                          <TableHeader field="category" label="Categoria" />
                          <TableHeader field="impact" label="Impacto" />
                          <TableHeader field="probability" label="Probabilidade" />
                          <TableHeader field="status" label="Status" />
                          <TableHeader field="owner.name" label="Proprietário" />
                          {/* Ordenar por owner.name pode precisar de ajuste no backend se owner for uma relação */}
                          <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6"><span className="sr-only">Ações</span></th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                        {risks.map((risk) => (
                          <tr key={risk.id}>
                            <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">
                              {risk.title}
                              {risk.hasPendingApproval && (
                                <span className="ml-2 px-2 py-0.5 inline-flex text-xs leading-5 font-semibold rounded-full bg-yellow-100 text-yellow-800 dark:bg-yellow-700 dark:text-yellow-100 animate-pulse" title="Aprovação Pendente">
                                  Pendente
                                </span>
                              )}
                              {risk.hasPendingApproval && user?.id === risk.owner_id && (
                                <button onClick={() => handleOpenDecisionModal(risk)}
                                  className="ml-2 px-2 py-0.5 text-xs bg-blue-500 text-white rounded-full hover:bg-blue-600"
                                  title="Tomar Decisão sobre Aceite">
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
                              <button onClick={() => handleDeleteRisk(risk.id, risk.title)}
                                className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-200"
                                disabled={isLoading}>
                                Deletar
                              </button>
                              {(user?.role === 'admin' || user?.role === 'manager') && risk.status !== 'aceito' && risk.status !== 'mitigado' && !risk.hasPendingApproval && (
                                <button onClick={() => handleSubmitForAcceptance(risk.id, risk.title)}
                                  className="ml-2 text-green-600 hover:text-green-900 dark:text-green-400 dark:hover:text-green-200"
                                  disabled={isLoading}>
                                  Submeter p/ Aceite
                                </button>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                  <PaginationControls
                    currentPage={currentPage}
                    totalPages={totalPages}
                    totalItems={totalItems}
                    pageSize={pageSize}
                    onPageChange={handlePageChange}
                    isLoading={isLoading}
                  />
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
            currentApproverId={selectedRiskForDecision.owner_id}
            onClose={handleCloseDecisionModal}
            onSubmitSuccess={handleDecisionSubmitSuccess}
        />
      )}
      <RiskBulkUploadModal
        isOpen={showUploadModal}
        onClose={() => setShowUploadModal(false)}
        onUploadSuccess={() => {
          clearFilters(); // Limpar filtros e ir para a primeira página para ver os novos riscos importados
          // fetchRisks será chamado pelo useEffect devido à mudança de currentPage/filtros
        }}
      />
    </AdminLayout>
  );
};

export default WithAuth(RisksPageContent);
