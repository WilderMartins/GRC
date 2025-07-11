import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import { useRouter } from 'next/router';
import { useAuth } from '@/contexts/AuthContext';
import { useEffect, useState, useCallback } from 'react';
import apiClient from '@/lib/axios';
import WebhookForm from '@/components/admin/WebhookForm'; // Importar o formulário

// Tipos (placeholders, idealmente de um arquivo compartilhado/gerado)
interface WebhookConfiguration {
  id: string;
  name: string;
  url: string;
  event_types: string; // No backend é string separada por vírgula, ou JSON string
  event_types_list?: string[]; // Para exibição no frontend
  is_active: boolean;
}

const OrgWebhooksPageContent = () => {
  const router = useRouter();
  const { orgId } = router.query;
  const { user, isLoading: authIsLoading } = useAuth();
  const [canAccess, setCanAccess] = useState(false);
  const [pageError, setPageError] = useState<string | null>(null); // Erro de acesso à página

  const [webhooks, setWebhooks] = useState<WebhookConfiguration[]>([]);
  const [isLoadingData, setIsLoadingData] = useState(true);
  const [dataError, setDataError] = useState<string | null>(null);

  const [showWebhookModal, setShowWebhookModal] = useState(false);
  const [editingWebhook, setEditingWebhook] = useState<WebhookConfiguration | null>(null);

  const fetchWebhooks = useCallback(async () => {
    if (!canAccess || !orgId || typeof orgId !== 'string') return;

    setIsLoadingData(true);
    setDataError(null);
    try {
      const response = await apiClient.get<WebhookConfiguration[]>(`/organizations/${orgId}/webhooks`);
      // Processar event_types_list
      const processedWebhooks = response.data.map(wh => ({
        ...wh,
        event_types_list: wh.event_types ? wh.event_types.split(',') : [],
      }));
      setWebhooks(processedWebhooks || []);
    } catch (err: any) {
      console.error("Erro ao buscar webhooks:", err);
      setDataError(err.response?.data?.error || err.message || "Falha ao buscar webhooks.");
      setWebhooks([]);
    } finally {
      setIsLoadingData(false);
    }
  }, [orgId, canAccess]);

  useEffect(() => {
    if (authIsLoading) return; // Esperar o auth carregar

    if (!user) {
      setPageError("Usuário não autenticado.");
      setCanAccess(false);
      return;
    }
    if (user.organization_id !== orgId) {
      // TODO: Adicionar lógica para superadmin no futuro
      setPageError("Você não tem permissão para acessar as configurações desta organização.");
      setCanAccess(false);
      return;
    }
    setCanAccess(true);
    setPageError(null);
    fetchWebhooks(); // Chamar fetchWebhooks aqui quando o acesso for permitido
  }, [orgId, user, authIsLoading, fetchWebhooks]); // Adicionar fetchWebhooks como dependência

  if (authIsLoading) {
    return <AdminLayout title="Carregando..."><div className="p-6 text-center">Verificando permissões...</div></AdminLayout>;
  }

  if (!canAccess && pageError) {
    return <AdminLayout title="Acesso Negado"><div className="p-6 text-center text-red-500">{pageError}</div></AdminLayout>;
  }

  if (!canAccess && !pageError) { // Ainda pode estar determinando o acesso se user.organization_id não estiver pronto
    return <AdminLayout title="Carregando..."><div className="p-6 text-center">Verificando organização...</div></AdminLayout>;
  }

  const handleAddNewWebhook = () => {
    setEditingWebhook(null);
    setShowWebhookModal(true);
  };

  const handleEditWebhook = (webhook: WebhookConfiguration) => {
    setEditingWebhook(webhook);
    setShowWebhookModal(true);
  };

  const handleCloseWebhookModal = () => {
    setShowWebhookModal(false);
    setEditingWebhook(null);
  };

  const handleWebhookSubmitSuccess = () => {
    fetchWebhooks(); // Re-carrega a lista de webhooks
    // O WebhookForm já chama onClose, que está em handleCloseWebhookModal
  };

  const handleDeleteWebhook = async (webhookId: string, webhookName: string) => {
    if (!orgId || typeof orgId !== 'string') {
        setDataError("ID da Organização não encontrado para deletar webhook."); // Usar setDataError
        return;
    }
    if (window.confirm(`Tem certeza que deseja deletar o webhook "${webhookName}"? Esta ação não pode ser desfeita.`)) {
      setIsLoadingData(true);
      setDataError(null);
      try {
        await apiClient.delete(`/organizations/${orgId}/webhooks/${webhookId}`);
        // alert(`Webhook "${webhookName}" deletado com sucesso.`); // Opcional, pois a lista será recarregada
        fetchWebhooks(); // Re-buscar a lista para refletir a remoção
      } catch (err: any) {
        console.error("Erro ao deletar webhook:", err);
        setDataError(err.response?.data?.error || err.message || "Falha ao deletar webhook.");
        setIsLoadingData(false);
      }
      // setIsLoadingData(false) será chamado pelo fetchWebhooks em caso de sucesso
    }
  };


  return (
    <AdminLayout title={`Webhooks - Organização ${orgId ? String(orgId).substring(0,8) : ''} - Phoenix GRC`}>
      <Head>
        <title>Gerenciar Webhooks - Phoenix GRC</title>
      </Head>

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex flex-col sm:flex-row justify-between items-center mb-6 gap-3">
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
              Gerenciar Webhooks
            </h1>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Organização: {orgId}
            </p>
          </div>
          {/* O botão pode abrir um Modal */}
          <button
            onClick={handleAddNewWebhook}
            className="inline-flex items-center justify-center rounded-md border border-transparent bg-brand-primary px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 dark:focus:ring-offset-gray-800 transition-colors"
          >
            Adicionar Novo Webhook
          </button>
        </div>

        {isLoadingData && <p className="text-center text-gray-500 dark:text-gray-400 py-4">Carregando webhooks...</p>}
        {dataError && <p className="text-center text-red-500 py-4">Erro ao carregar webhooks: {dataError}</p>}

        {!isLoadingData && !dataError && (
          <div className="mt-8 flow-root">
            <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
              <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                  <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                    <thead className="bg-gray-50 dark:bg-gray-700">
                      <tr>
                        <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-6">Nome</th>
                        <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">URL (Parcial)</th>
                        <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Eventos</th>
                        <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Status</th>
                        <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6"><span className="sr-only">Ações</span></th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                      {webhooks.length === 0 && (
                        <tr><td colSpan={5} className="text-center py-4 px-6 text-sm text-gray-500 dark:text-gray-400">Nenhum webhook configurado.</td></tr>
                      )}
                      {webhooks.map((webhook) => (
                        <tr key={webhook.id}>
                          <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">{webhook.name}</td>
                          <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300" title={webhook.url}>
                            {webhook.url.length > 50 ? webhook.url.substring(0, 47) + "..." : webhook.url}
                          </td>
                          <td className="px-3 py-4 text-sm text-gray-500 dark:text-gray-300">
                            {(webhook.event_types_list || []).map(event => (
                                <span key={event} className="mr-1 mb-1 inline-block px-2 py-0.5 text-xs font-semibold rounded-full bg-blue-100 text-blue-800 dark:bg-blue-700 dark:text-blue-100">
                                    {event}
                                </span>
                            ))}
                          </td>
                          <td className="whitespace-nowrap px-3 py-4 text-sm">
                            <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                              webhook.is_active ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' : 'bg-red-100 text-red-800 dark:bg-red-700 dark:text-red-100'
                            }`}>
                              {webhook.is_active ? 'Ativo' : 'Inativo'}
                            </span>
                          </td>
                          <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6 space-x-2">
                            <button onClick={() => handleEditWebhook(webhook)} className="text-brand-primary hover:text-brand-primary/80 dark:text-brand-primary dark:hover:text-brand-primary/80 transition-colors" disabled={isLoadingData}>Editar</button>
                            <button onClick={() => handleDeleteWebhook(webhook.id, webhook.name)} className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 transition-colors" disabled={isLoadingData}>Deletar</button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
      {showWebhookModal && orgId && typeof orgId === 'string' && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
            <WebhookForm
              organizationId={orgId}
              initialData={editingWebhook || undefined}
              isEditing={!!editingWebhook}
              onClose={handleCloseWebhookModal}
              onSubmitSuccess={handleWebhookSubmitSuccess}
            />
          </div>
        </div>
      )}
    </AdminLayout>
  );
};

export default WithAuth(OrgWebhooksPageContent);
