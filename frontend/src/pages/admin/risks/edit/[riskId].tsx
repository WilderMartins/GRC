import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';
import RiskForm from '@/components/risks/RiskForm'; // Importar o formulário
import { useEffect, useState, useCallback } from 'react'; // Adicionado useCallback
import apiClient from '@/lib/axios'; // Ajuste o path
import { useAuth } from '@/contexts/AuthContext'; // Para o user (se necessário para permissões futuras)

// Tipos (idealmente compartilhados)
type RiskStatus = "aberto" | "em_andamento" | "mitigado" | "aceito";
type RiskImpact = "Baixo" | "Médio" | "Alto" | "Crítico";
type RiskProbability = "Baixo" | "Médio" | "Alto" | "Crítico";

interface UserInfo { // Para Requester e Approver
    id: string;
    name: string;
    email: string;
}
interface ApprovalWorkflow {
  id: string;
  risk_id: string;
  requester_id: string;
  requester?: UserInfo;
  approver_id: string;
  approver?: UserInfo;
  status: string; // "pendente", "aprovado", "rejeitado"
  comments: string;
  created_at: string;
  updated_at: string;
}
interface RiskOwner { id: string; name: string; email: string; } // Já definido antes
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
  owner?: RiskOwner;
  created_at: string;
  updated_at: string;
  // ApprovalWorkflows (opcional, se o endpoint de Risco detalhado já trouxer)
}


const EditRiskPageContent = () => {
  const router = useRouter();
  const { riskId } = router.query;
  const [initialData, setInitialData] = useState<Risk | null>(null);
  const [approvalHistory, setApprovalHistory] = useState<ApprovalWorkflow[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isHistoryLoading, setIsHistoryLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { user } = useAuth();


  const fetchRiskData = useCallback(async () => {
    if (riskId && typeof riskId === 'string') {
      setIsLoading(true);
      setError(null);
      try {
        const riskResponse = await apiClient.get<Risk>(`/risks/${riskId}`);
        setInitialData(riskResponse.data);
      } catch (err: any) {
        console.error("Erro ao buscar dados do risco:", err);
        setError(err.response?.data?.error || err.message || "Falha ao buscar dados do risco.");
      } finally {
        setIsLoading(false);
      }
    } else if (riskId) {
        setError("ID do Risco inválido.");
        setIsLoading(false);
    }
  }, [riskId]);

  const fetchApprovalHistory = useCallback(async () => {
    if (riskId && typeof riskId === 'string') {
      setIsHistoryLoading(true);
      try {
        const historyResponse = await apiClient.get<ApprovalWorkflow[]>(`/risks/${riskId}/approval-history`);
        setApprovalHistory(historyResponse.data || []);
      } catch (err: any) {
        console.error("Erro ao buscar histórico de aprovação:", err);
        // Não setar erro principal para não sobrescrever erro de fetch do risco
      } finally {
        setIsHistoryLoading(false);
      }
    }
  }, [riskId]);


  useEffect(() => {
    if (router.isReady) { // Garante que router.query está populado
        fetchRiskData();
        fetchApprovalHistory();
    }
  }, [riskId, router.isReady, fetchRiskData, fetchApprovalHistory]);


  const handleSuccess = () => {
    alert('Risco atualizado com sucesso!'); // Placeholder
    router.push('/admin/risks');
  };

  if (isLoading && !initialData) { // Mostrar loading apenas se não houver dados ainda
    return <AdminLayout title="Carregando..."><div className="text-center p-10">Carregando dados do risco...</div></AdminLayout>;
  }

  if (error) {
    return <AdminLayout title="Erro"><div className="text-center p-10 text-red-500">Erro: {error}</div></AdminLayout>;
  }

  if (!initialData && !isLoading) { // Se terminou de carregar e não encontrou dados (ou ID inválido)
     return <AdminLayout title="Risco não encontrado"><div className="text-center p-10">Risco não encontrado.</div></AdminLayout>;
  }


  return (
    <AdminLayout title={`Editar Risco - Phoenix GRC`}>
      <Head>
        <title>Editar Risco {initialData?.title || ''} - Phoenix GRC</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            Editar Risco: <span className="text-indigo-600 dark:text-indigo-400">{initialData?.title}</span>
          </h1>
          <Link href="/admin/risks" legacyBehavior>
            <a className="text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-200">
              &larr; Voltar para Lista de Riscos
            </a>
          </Link>
        </div>

        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg p-6 md:p-8">
          {initialData && (
            <RiskForm
              initialData={{
                id: initialData.id,
                title: initialData.title,
                description: initialData.description,
                category: initialData.category as any,
                impact: initialData.impact,
                probability: initialData.probability,
                status: initialData.status,
                owner_id: initialData.owner_id,
              }}
              isEditing={true}
              onSubmitSuccess={handleSuccess}
            />
          )}
        </div>

        {/* Seção de Histórico de Aprovação */}
        <div className="mt-10">
            <h2 className="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white mb-4">
                Histórico de Aprovação de Aceite
            </h2>
            {isHistoryLoading && <p className="text-gray-500 dark:text-gray-400">Carregando histórico...</p>}
            {!isHistoryLoading && approvalHistory.length === 0 && (
                <p className="text-gray-500 dark:text-gray-400">Nenhum workflow de aprovação encontrado para este risco.</p>
            )}
            {!isHistoryLoading && approvalHistory.length > 0 && (
                <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg overflow-hidden">
                    <ul role="list" className="divide-y divide-gray-200 dark:divide-gray-700">
                        {approvalHistory.map((wf) => (
                            <li key={wf.id} className="px-4 py-4 sm:px-6">
                                <div className="flex items-center justify-between">
                                    <p className="text-sm font-medium text-indigo-600 dark:text-indigo-400 truncate">
                                        Decisão: <span className={`font-bold ${
                                            wf.status === 'aprovado' ? 'text-green-600 dark:text-green-400' :
                                            wf.status === 'rejeitado' ? 'text-red-600 dark:text-red-400' :
                                            'text-yellow-600 dark:text-yellow-400'
                                        }`}>{wf.status.toUpperCase()}</span>
                                    </p>
                                    <div className="ml-2 flex-shrink-0 flex">
                                        <p className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200">
                                            {new Date(wf.updated_at).toLocaleString()}
                                        </p>
                                    </div>
                                </div>
                                <div className="mt-2 sm:flex sm:justify-between">
                                    <div className="sm:flex">
                                        <p className="flex items-center text-sm text-gray-500 dark:text-gray-400">
                                            <svg className="flex-shrink-0 mr-1.5 h-5 w-5 text-gray-400 dark:text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                                                <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
                                            </svg>
                                            Requisitante: {wf.requester?.name || wf.requester_id.substring(0,8)}
                                        </p>
                                        <p className="mt-2 flex items-center text-sm text-gray-500 dark:text-gray-400 sm:mt-0 sm:ml-6">
                                            <svg className="flex-shrink-0 mr-1.5 h-5 w-5 text-gray-400 dark:text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                                                <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
                                            </svg>
                                            Aprovador: {wf.approver?.name || wf.approver_id.substring(0,8)}
                                        </p>
                                    </div>
                                </div>
                                {wf.comments && (
                                    <div className="mt-2 text-sm text-gray-700 dark:text-gray-300">
                                        <p>Comentários: <span className="italic">{wf.comments}</span></p>
                                    </div>
                                )}
                            </li>
                        ))}
                    </ul>
                </div>
            )}
        </div>

      </div>
    </AdminLayout>
  );
};

export default WithAuth(EditRiskPageContent);
