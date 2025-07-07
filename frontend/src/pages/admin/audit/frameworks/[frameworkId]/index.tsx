import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { useEffect, useState, useCallback } from 'react'; // Adicionado useCallback
import apiClient from '@/lib/axios'; // Ajuste o path se necessário
import { useAuth } from '@/contexts/AuthContext'; // Para obter orgId
import AssessmentForm from '@/components/audit/AssessmentForm'; // Importar o formulário

// Tipos (idealmente de um arquivo compartilhado)
interface AuditFrameworkInfo { // Para buscar o nome do framework
    id: string;
    name: string;
}
interface AuditControl {
  id: string; // UUID do AuditControl
  control_id: string; // ID textual, ex: "AC-1"
  description: string;
  family: string;
  // framework_id: string; // Já temos pelo contexto da página
}
interface AuditAssessment {
  id: string; // UUID da Assessment
  organization_id: string;
  audit_control_id: string; // FK para AuditControl.ID
  status: string; // "conforme", "nao_conforme", etc.
  score?: number;
  evidence_url?: string;
  assessment_date?: string; // Formato YYYY-MM-DD
  created_at: string;
  updated_at: string;
  AuditControl?: AuditControl; // GORM pode preencher isso
}
interface ControlWithAssessment extends AuditControl {
  assessment?: AuditAssessment;
}

interface PaginatedAssessmentsResponse { // Se a API de assessments for paginada
    items: AuditAssessment[];
    total_items: number;
    total_pages: number;
    page: number;
    page_size: number;
}


const FrameworkDetailPageContent = () => {
  const router = useRouter();
  const { frameworkId } = router.query;
  const { user } = useAuth(); // Para obter organization_id

  const [frameworkInfo, setFrameworkInfo] = useState<AuditFrameworkInfo | null>(null);
  const [controlsWithAssessments, setControlsWithAssessments] = useState<ControlWithAssessment[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [showAssessmentModal, setShowAssessmentModal] = useState(false);
  const [selectedControlForAssessment, setSelectedControlForAssessment] = useState<ControlWithAssessment | null>(null);

  // TODO: Adicionar estados para paginação de assessments se a API for paginada

  const fetchData = useCallback(async () => { // Envolver em useCallback
    if (!frameworkId || typeof frameworkId !== 'string' || !user?.organization_id) {
        // Se frameworkId ou orgId não estiverem prontos, não fazer nada ou setar erro.
        // Pode acontecer na renderização inicial se o router não estiver pronto.
        setIsLoading(false); // Para evitar loading infinito se IDs não chegarem
        if (!frameworkId && router.isReady) setError("ID do Framework não encontrado na URL.");
        if (!user?.organization_id && !auth.isLoading) setError("ID da Organização não encontrado.");
        return;
    }
    setIsLoading(true);
    setError(null);
    try {
        // 1. Buscar informações do Framework (nome)
        const frameworksResponse = await apiClient.get<AuditFrameworkInfo[]>('/audit/frameworks');
        const currentFramework = frameworksResponse.data.find(f => f.id === frameworkId);
        setFrameworkInfo(currentFramework || {id: frameworkId, name: "Framework Desconhecido"});

        // 2. Buscar Controles do Framework
        const controlsResponse = await apiClient.get<AuditControl[]>(`/audit/frameworks/${frameworkId}/controls`);
        const fetchedControls = controlsResponse.data || [];

        // 3. Buscar Avaliações da Organização para este Framework
        const assessmentsResponse = await apiClient.get<PaginatedAssessmentsResponse | AuditAssessment[]>(
        `/audit/organizations/${user.organization_id}/frameworks/${frameworkId}/assessments`
        );

        let fetchedAssessments: AuditAssessment[] = [];
        if (Array.isArray(assessmentsResponse.data)) {
        fetchedAssessments = assessmentsResponse.data;
        } else if (assessmentsResponse.data && Array.isArray(assessmentsResponse.data.items)) {
        fetchedAssessments = assessmentsResponse.data.items;
        }

        // 4. Combinar Controles com Avaliações
        const combined: ControlWithAssessment[] = fetchedControls.map(control => {
        const assessment = fetchedAssessments.find(a => a.audit_control_id === control.id);
        return { ...control, assessment };
        });
        setControlsWithAssessments(combined);

    } catch (err: any) {
        console.error("Erro ao buscar dados do framework:", err);
        setError(err.response?.data?.error || err.message || "Falha ao buscar dados do framework.");
    } finally {
        setIsLoading(false);
    }
  }, [frameworkId, user?.organization_id, auth.isLoading, router.isReady]); // Adicionar router.isReady

  useEffect(() => {
    fetchData();
  }, [fetchData]); // Agora fetchData é estável


  const handleOpenAssessmentModal = (controlItem: ControlWithAssessment) => {
    setSelectedControlForAssessment(controlItem);
    setShowAssessmentModal(true);
  };

  const handleCloseAssessmentModal = () => {
    setSelectedControlForAssessment(null);
    setShowAssessmentModal(false);
  };

  const handleAssessmentSubmitSuccess = (updatedAssessment: AuditAssessment) => {
    // Atualizar a lista localmente ou re-buscar dados
    // Re-buscar é mais simples para garantir consistência
    fetchData();
    // Poderia também atualizar localmente:
    // setControlsWithAssessments(prev => prev.map(item =>
    //   item.id === updatedAssessment.audit_control_id
    //     ? { ...item, assessment: updatedAssessment }
    //     : item
    // ));
    handleCloseAssessmentModal();
    // alert("Avaliação salva com sucesso!"); // O form já pode ter um alerta
  };


  const frameworkName = frameworkInfo?.name || `Framework ${frameworkId ? String(frameworkId).substring(0,8) : ''}...`;

  return (
    <AdminLayout title={`Controles: ${frameworkName} - Phoenix GRC`}>
      <Head>
        <title>Controles: ${frameworkName} - Phoenix GRC</title>
        try {
          // 1. Buscar informações do Framework (nome) - Opcional se já tivermos o nome
          //    Poderíamos buscar todos os frameworks e encontrar o nome, ou ter um endpoint /frameworks/{id}
          //    Por simplicidade, vamos assumir que o nome pode ser buscado ou já é conhecido.
          //    Para este exemplo, vamos mockar o nome ou buscá-lo se tivermos um endpoint.
          //    apiClient.get(`/audit/frameworks/${frameworkId}`).then(res => setFrameworkInfo(res.data));
          //    Como não temos esse endpoint, vamos buscar todos e filtrar pelo ID.
          const frameworksResponse = await apiClient.get<AuditFrameworkInfo[]>('/audit/frameworks');
          const currentFramework = frameworksResponse.data.find(f => f.id === frameworkId);
          setFrameworkInfo(currentFramework || {id: frameworkId, name: "Framework Desconhecido"});


          // 2. Buscar Controles do Framework
          const controlsResponse = await apiClient.get<AuditControl[]>(`/audit/frameworks/${frameworkId}/controls`);
          const fetchedControls = controlsResponse.data || [];

          // 3. Buscar Avaliações da Organização para este Framework
          // A API é /api/v1/audit/organizations/{orgId}/frameworks/{frameworkId}/assessments
          // Ela já retorna as avaliações com o AuditControl aninhado.
          const assessmentsResponse = await apiClient.get<PaginatedAssessmentsResponse | AuditAssessment[]>(
            `/audit/organizations/${user.organization_id}/frameworks/${frameworkId}/assessments`
            // TODO: Adicionar params de paginação se a API de assessments for paginada
          );

          let fetchedAssessments: AuditAssessment[] = [];
          if (Array.isArray(assessmentsResponse.data)) {
            fetchedAssessments = assessmentsResponse.data;
          } else if (assessmentsResponse.data && Array.isArray(assessmentsResponse.data.items)) {
            fetchedAssessments = assessmentsResponse.data.items;
            // TODO: setar dados de paginação de assessments
          }


          // 4. Combinar Controles com Avaliações
          const combined: ControlWithAssessment[] = fetchedControls.map(control => {
            const assessment = fetchedAssessments.find(a => a.audit_control_id === control.id);
            return { ...control, assessment };
          });
          setControlsWithAssessments(combined);

        } catch (err: any) {
          console.error("Erro ao buscar dados do framework:", err);
          setError(err.response?.data?.error || err.message || "Falha ao buscar dados do framework.");
        } finally {
          setIsLoading(false);
        }
      };
      fetchData();
    }
  }, [frameworkId, user?.organization_id]);


  const frameworkName = frameworkInfo?.name || `Framework ${frameworkId ? String(frameworkId).substring(0,8) : ''}...`;

  return (
    <AdminLayout title={`Controles: ${frameworkName} - Phoenix GRC`}>
      <Head>
        <title>Controles: ${frameworkName} - Phoenix GRC</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="sm:flex sm:items-center sm:justify-between mb-6">
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
              {frameworkName}
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

        {/* TODO: Adicionar filtros por família de controle, status da avaliação */}

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
                    {controlsWithAssessments.length === 0 && (
                        <tr>
                            <td colSpan={6} className="text-center py-4 text-sm text-gray-500 dark:text-gray-400">
                                Nenhum controle encontrado para este framework.
                            </td>
                        </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
        {/* TODO: Adicionar paginação para controles/avaliações se necessário */}

        {showAssessmentModal && selectedControlForAssessment && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm transition-opacity duration-300 ease-in-out">
            <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-xl max-h-[90vh] overflow-y-auto">
              <AssessmentForm
                controlId={selectedControlForAssessment.id}
                controlDisplayId={selectedControlForAssessment.control_id}
                initialData={selectedControlForAssessment.assessment ? {
                  // Mapear dados da avaliação existente para AssessmentFormData
                  audit_control_id: selectedControlForAssessment.assessment.audit_control_id,
                  status: selectedControlForAssessment.assessment.status as any, // Cast se necessário
                  score: selectedControlForAssessment.assessment.score,
                  assessment_date: selectedControlForAssessment.assessment.assessment_date ? selectedControlForAssessment.assessment.assessment_date.split('T')[0] : new Date().toISOString().split('T')[0],
                  evidence_url: selectedControlForAssessment.assessment.evidence_url || '',
                } : undefined } // Passar undefined se não houver avaliação prévia
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
