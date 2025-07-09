import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { useEffect, useState, useCallback } from 'react';
import apiClient from '@/lib/axios';
import { useAuth } from '@/contexts/AuthContext';
import AssessmentForm from '@/components/audit/AssessmentForm';
import PaginationControls from '@/components/common/PaginationControls';
import {
    AuditFramework, // Usar no lugar de AuditFrameworkInfo
    AuditControl,
    AuditAssessment,
    ControlWithAssessment,
    PaginatedResponse, // Usar no lugar de PaginatedControlsResponse e PaginatedAssessmentsResponse
    AuditAssessmentStatusFilter, // Importar o enum/type para o filtro de status
    // ControlFamiliesResponse pode ser mantido local ou tipado se for simples como { families: string[] }
} from '@/types';

// Tipos locais removidos ou substituídos pelos importados

// Interface para a resposta da API de famílias de controle (pode ser mantida ou movida para api.ts se for mais complexa)
interface ControlFamiliesResponse {
    families: string[];
}


const FrameworkDetailPageContent = () => {
  const router = useRouter();
  const { frameworkId } = router.query;
  const { user, isLoading: authIsLoading } = useAuth();

  const [frameworkInfo, setFrameworkInfo] = useState<Partial<AuditFramework> | null>(null); // Usar Partial<AuditFramework>
  const [controlsWithAssessments, setControlsWithAssessments] = useState<ControlWithAssessment[]>([]);

  const [isLoadingData, setIsLoadingData] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [controlsCurrentPage, setControlsCurrentPage] = useState(1);
  const [controlsPageSize, setControlsPageSize] = useState(10);
  const [controlsTotalPages, setControlsTotalPages] = useState(0);
  const [controlsTotalItems, setControlsTotalItems] = useState(0);

  const [availableControlFamilies, setAvailableControlFamilies] = useState<string[]>([]);
  const [filterFamily, setFilterFamily] = useState<string>("");
  const [filterAssessmentStatus, setFilterAssessmentStatus] = useState<AssessmentStatusFilter>("");

  const [showAssessmentModal, setShowAssessmentModal] = useState(false);
  const [selectedControlForAssessment, setSelectedControlForAssessment] = useState<ControlWithAssessment | null>(null);

  // Fetch inicial para nome do framework e famílias de controle
  useEffect(() => {
    if (frameworkId && typeof frameworkId === 'string' && !authIsLoading && user) {
      setIsLoadingData(true); // Set loading true for this initial fetch part
      apiClient.get<AuditFrameworkInfo[]>('/audit/frameworks')
        .then(response => {
          const currentFramework = response.data.find(f => f.id === frameworkId);
          setFrameworkInfo(currentFramework || {id: frameworkId, name: "Framework Desconhecido"});
        })
        .catch(err => {
          console.error("Erro ao buscar nome do framework:", err);
          setError(prev => prev ? prev + "; Falha ao carregar informações do framework." : "Falha ao carregar informações do framework.");
        });

      apiClient.get<ControlFamiliesResponse>(`/audit/frameworks/${frameworkId}/control-families`)
        .then(response => {
          setAvailableControlFamilies(response.data?.families?.sort() || []);
        })
        .catch(err => {
          console.warn("Endpoint /control-families não encontrado ou falhou, tentando alternativa:", err);
          apiClient.get<PaginatedControlsResponse | AuditControl[]>(`/audit/frameworks/${frameworkId}/controls?page_size=10000`)
            .then(response => {
                let allControls: AuditControl[] = [];
                if (Array.isArray(response.data)) {
                    allControls = response.data;
                } else if (response.data && Array.isArray(response.data.items)) {
                    allControls = response.data.items;
                }
                const families = Array.from(new Set(allControls.map(c => c.family))).sort();
                setAvailableControlFamilies(families);
            })
            .catch(deepErr => {
                console.error("Erro ao buscar todas as famílias de controle (alternativa):", deepErr);
                setError(prev => prev ? prev + "; Falha ao carregar filtros de família." : "Falha ao carregar filtros de família.");
            });
        });
        // setIsLoadingData(false) // Loading data will be set to false in fetchControlsAndCombineAssessments
    }
  }, [frameworkId, authIsLoading, user]);

  const fetchControlsAndCombineAssessments = useCallback(async () => {
    if (!frameworkId || typeof frameworkId !== 'string' || !user?.organization_id || authIsLoading) {
      setIsLoadingData(false);
      if (router.isReady && !frameworkId && !error) setError("ID do Framework não encontrado na URL."); // Set error if appropriate
      if (!authIsLoading && !user?.organization_id && !error) setError("ID da Organização do usuário não encontrado.");
      return;
    }
    setIsLoadingData(true);
    // Don't reset main error if it was set by initial fetches for frameworkInfo/families
    // setError(null);

    try {
      const controlsParams: { page: number; page_size: number; family?: string } = {
        page: controlsCurrentPage,
        page_size: controlsPageSize
      };
      if (filterFamily) {
        controlsParams.family = filterFamily;
      }
      const controlsResponse = await apiClient.get<PaginatedControlsResponse>(`/audit/frameworks/${frameworkId}/controls`, { params: controlsParams });
      const fetchedControls = controlsResponse.data.items || [];
      setControlsTotalItems(controlsResponse.data.total_items);
      setControlsTotalPages(controlsResponse.data.total_pages);

      const assessmentsResponse = await apiClient.get<PaginatedAssessmentsResponse | AuditAssessment[]>(
        `/audit/organizations/${user.organization_id}/frameworks/${frameworkId}/assessments?page_size=100000`
      );
      let allAssessmentsForFramework: AuditAssessment[] = [];
      if (Array.isArray(assessmentsResponse.data)) {
        allAssessmentsForFramework = assessmentsResponse.data;
      } else if (assessmentsResponse.data && Array.isArray(assessmentsResponse.data.items)) {
        allAssessmentsForFramework = assessmentsResponse.data.items;
      }

      let combined: ControlWithAssessment[] = fetchedControls.map(control => {
        const assessment = allAssessmentsForFramework.find(a => a.audit_control_id === control.id);
        return { ...control, assessment };
      });

      if (filterAssessmentStatus) {
        combined = combined.filter(item => {
          if (filterAssessmentStatus === "nao_avaliado") {
            return !item.assessment;
          }
          return item.assessment?.status === filterAssessmentStatus;
        });
      }
      setControlsWithAssessments(combined);
      if (fetchedControls.length === 0 && controlsResponse.data.total_items > 0 && controlsCurrentPage > 1) {
        // Se a página atual não tem itens, mas há itens no total (ex: após aplicar um filtro que esvazia a página atual)
        // e não estamos na primeira página, volte para a primeira página.
        setControlsCurrentPage(1);
      }


    } catch (err: any) {
      console.error("Erro ao buscar dados de controles e avaliações:", err);
      const fetchError = err.response?.data?.error || err.message || "Falha ao buscar dados de controles e/ou avaliações.";
      setError(prev => prev ? `${prev}; ${fetchError}` : fetchError);
      setControlsWithAssessments([]);
    } finally {
      setIsLoadingData(false);
    }
  }, [
    frameworkId,
    user?.organization_id,
    authIsLoading,
    controlsCurrentPage,
    controlsPageSize,
    filterFamily,
    filterAssessmentStatus,
    router.isReady
  ]);

  useEffect(() => {
    if (router.isReady && frameworkId && user && !authIsLoading) {
        fetchControlsAndCombineAssessments();
    }
  }, [fetchControlsAndCombineAssessments, router.isReady, frameworkId, user, authIsLoading]);


  const handleOpenAssessmentModal = (controlItem: ControlWithAssessment) => {
    setSelectedControlForAssessment(controlItem);
    setShowAssessmentModal(true);
  };

  const handleCloseAssessmentModal = () => {
    setSelectedControlForAssessment(null);
    setShowAssessmentModal(false);
  };

  const handleAssessmentSubmitSuccess = () => {
    fetchControlsAndCombineAssessments();
    handleCloseAssessmentModal();
  };

  const handleControlsPageChange = (newPage: number) => {
    if (newPage !== controlsCurrentPage) {
        setControlsCurrentPage(newPage);
    }
  };

  const handleFilterFamilyChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setFilterFamily(e.target.value);
    setControlsCurrentPage(1);
  };

  const handleFilterAssessmentStatusChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setFilterAssessmentStatus(e.target.value as AssessmentStatusFilter);
    // Re-chamada para aplicar o filtro de status no frontend.
    // Se a paginação/filtro de família também estiverem envolvidos, o useEffect principal cuidará.
    // Mas se apenas o status mudar, precisamos garantir que a lista seja reprocessada.
    // A dependência de filterAssessmentStatus no useCallback de fetchControlsAndCombineAssessments já faz isso.
    // No entanto, se quisermos uma reavaliação imediata dos dados já carregados:
    // A melhor forma é deixar o useEffect principal lidar com isso, pois ele já depende de filterAssessmentStatus
    // através do useCallback.
  };

  const clearFilters = () => {
    const hadFilters = filterFamily !== "" || filterAssessmentStatus !== "";
    setFilterFamily("");
    setFilterAssessmentStatus("");
    // Se a página atual não for 1, ou se havia filtros, a mudança de estado
    // (filterFamily, filterAssessmentStatus, controlsCurrentPage) acionará o useEffect para rebuscar.
    if (controlsCurrentPage !== 1) {
        setControlsCurrentPage(1);
    } else if (hadFilters) {
        // Se já estava na página 1 mas tinha filtros, precisa forçar o re-fetch
        // A mudança de filterFamily/filterAssessmentStatus já deve disparar o useEffect.
        // Para garantir, podemos chamar explicitamente se as dependências não forem suficientes.
        // A dependência no useCallback de fetchControlsAndCombineAssessments deve ser suficiente.
    }
  };

  const frameworkName = frameworkInfo?.name || `Framework ${frameworkId ? String(frameworkId).substring(0,8) : ''}...`;

  if (authIsLoading || (!router.isReady && !frameworkInfo)) {
    return <AdminLayout title="Carregando..."><div className="p-6 text-center">Carregando informações do framework...</div></AdminLayout>;
  }

  // Mostrar erro principal se ocorreu e não há dados para exibir na tabela, ou se o frameworkInfo não carregou
  if (error && (!frameworkInfo || frameworkInfo.name === "Framework Desconhecido" || controlsWithAssessments.length === 0) && !isLoadingData ) {
    return (
        <AdminLayout title={`Erro - ${frameworkName}`}>
            <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
                <h1 className="text-2xl font-bold text-red-600 dark:text-red-400 mb-4">Erro ao Carregar Dados</h1>
                <p className="text-red-500 dark:text-red-300">{error}</p>
                <Link href="/admin/audit/frameworks" legacyBehavior>
                    <a className="mt-4 inline-flex items-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700">
                    Voltar para Frameworks
                    </a>
                </Link>
            </div>
        </AdminLayout>
    );
  }

  return (
    <AdminLayout title={`Controles: ${frameworkName} - Phoenix GRC`}>
      <Head>
        <title>Controles: ${frameworkName} - Phoenix GRC</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="sm:flex sm:items-center sm:justify-between mb-6">
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
              {frameworkInfo ? frameworkInfo.name : 'Carregando nome...'}
            </h1>
            <p className="mt-2 text-sm text-gray-700 dark:text-gray-400">
              Lista de controles e suas avaliações de conformidade para sua organização.
            </p>
          </div>
          <div className="mt-4 sm:mt-0 sm:ml-16 sm:flex-none">
            <Link href="/admin/audit/frameworks" legacyBehavior>
                <a className="inline-flex items-center rounded-md border border-transparent bg-gray-200 dark:bg-gray-700 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-300 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2">
                &larr; Voltar para Frameworks
                </a>
            </Link>
          </div>
        </div>

        <div className="my-6 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg shadow">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 items-end">
            <div>
              <label htmlFor="filterFamily" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Filtrar por Família</label>
              <select id="filterFamily" name="filterFamily" value={filterFamily} onChange={handleFilterFamilyChange}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md disabled:opacity-50"
                      disabled={availableControlFamilies.length === 0 && !isLoadingData}
              >
                <option value="">Todas as Famílias</option>
                {availableControlFamilies.map(family => (
                  <option key={family} value={family}>{family}</option>
                ))}
              </select>
            </div>
            <div>
              <label htmlFor="filterAssessmentStatus" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Filtrar por Status da Avaliação</label>
              <select id="filterAssessmentStatus" name="filterAssessmentStatus" value={filterAssessmentStatus} onChange={handleFilterAssessmentStatusChange}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                <option value="">Todos os Status</option>
                <option value="conforme">Conforme</option>
                <option value="nao_conforme">Não Conforme</option>
                <option value="parcialmente_conforme">Parcialmente Conforme</option>
                <option value="nao_avaliado">Não Avaliado</option>
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

        {isLoadingData && <p className="text-center py-10">Carregando controles e avaliações...</p>}
        {error && !isLoadingData && controlsWithAssessments.length === 0 && <p className="text-center text-red-500 py-10">Erro ao carregar dados: {error}</p>}

        {!isLoadingData && !error && controlsWithAssessments.length === 0 && (
            <div className="text-center py-10">
                <p className="text-gray-500 dark:text-gray-400">Nenhum controle encontrado com os filtros aplicados.</p>
            </div>
        )}

        {/* Renderizar tabela e paginação apenas se não estiver carregando, sem erro principal E houver controles */}
        {!isLoadingData && !error && controlsWithAssessments.length > 0 && (
          <div className="mt-8 flow-root">
            <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
              <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                  <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                    <thead className="bg-gray-50 dark:bg-gray-700">
                      <tr>
                        <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-6">ID Controle</th>
                        <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Descrição</th>
                        <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Família</th>
                        <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Status Avaliação</th>
                        <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Score</th>
                        <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6">
                          <span className="sr-only">Ações</span>
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                      {controlsWithAssessments.map((item) => (
                        <tr key={item.id}>
                          <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">{item.control_id}</td>
                          <td className="px-3 py-4 text-sm text-gray-500 dark:text-gray-300 max-w-md truncate hover:whitespace-normal">{item.description}</td>
                          <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{item.family}</td>
                          <td className="whitespace-nowrap px-3 py-4 text-sm">
                            {item.assessment ? (
                               <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                  item.assessment.status === 'conforme' ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' :
                                  item.assessment.status === 'parcialmente_conforme' ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-700 dark:text-yellow-100' :
                                  item.assessment.status === 'nao_conforme' ? 'bg-red-100 text-red-800 dark:bg-red-700 dark:text-red-100' :
                                  'bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-200'
                              }`}>
                                  {item.assessment.status}
                              </span>
                            ) : (
                              <span className="text-xs text-gray-400 dark:text-gray-500">Não Avaliado</span>
                            )}
                          </td>
                          <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{item.assessment?.score ?? '-'}</td>
                          <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6">
                            <button
                              onClick={() => handleOpenAssessmentModal(item)}
                              className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-200"
                            >
                              {item.assessment ? 'Editar Avaliação' : 'Avaliar'}
                            </button>
                             {item.assessment?.evidence_url && (
                              <a
                                  href={item.assessment.evidence_url}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className="ml-3 text-blue-600 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-200"
                                  title={item.assessment.evidence_url}
                              >
                                  Ver Evidência
                              </a>
                             )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                {controlsTotalPages > 0 && ( // Mostrar paginação apenas se houver páginas
                    <PaginationControls
                        currentPage={controlsCurrentPage}
                        totalPages={controlsTotalPages}
                        totalItems={controlsTotalItems}
                        pageSize={controlsPageSize}
                        onPageChange={handleControlsPageChange}
                        isLoading={isLoadingData}
                    />
                )}
              </div>
            </div>
          </div>
        )}

        {showAssessmentModal && selectedControlForAssessment && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm transition-opacity duration-300 ease-in-out">
            <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-xl max-h-[90vh] overflow-y-auto">
              <AssessmentForm
                controlId={selectedControlForAssessment.id}
                controlDisplayId={selectedControlForAssessment.control_id}
                initialData={selectedControlForAssessment.assessment ? {
                  audit_control_id: selectedControlForAssessment.assessment.audit_control_id,
                  status: selectedControlForAssessment.assessment.status as any,
                  score: selectedControlForAssessment.assessment.score,
                  assessment_date: selectedControlForAssessment.assessment.assessment_date ? selectedControlForAssessment.assessment.assessment_date.split('T')[0] : new Date().toISOString().split('T')[0],
                  evidence_url: selectedControlForAssessment.assessment.evidence_url || '',
                } : undefined }
                onClose={handleCloseAssessmentModal}
                onSubmitSuccess={handleAssessmentSubmitSuccess}
              />
            </div>
          </div>
        )}
      </div>
    </AdminLayout>
  );
};

export default WithAuth(FrameworkDetailPageContent);
