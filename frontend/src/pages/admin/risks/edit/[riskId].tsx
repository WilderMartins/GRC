import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';
import RiskForm from '@/components/risks/RiskForm'; // Importar o formulário
import { useEffect, useState, useCallback } from 'react'; // Adicionado useCallback
import apiClient from '@/lib/axios'; // Ajuste o path
import { useAuth } from '@/contexts/AuthContext';
import {
    Risk,
    RiskImpact, // Assegure que estes sejam os tipos de enum corretos ou strings literais
    RiskProbability,
    RiskStatus,
    ApprovalWorkflow,
    UserLookup, // Para o select de usuários e lista de stakeholders
} from '@/types'; // Importar tipos globais
import { useNotifier } from '@/hooks/useNotifier'; // Para notificações
import { useTranslation } from 'next-i18next'; // Para traduções


const EditRiskPageContent = () => {
  const router = useRouter();
  const notify = useNotifier();
  const { t } = useTranslation(['risks', 'common']);
  const { riskId } = router.query;

  const [initialData, setInitialData] = useState<Risk | null>(null);
  const [approvalHistory, setApprovalHistory] = useState<ApprovalWorkflow[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isHistoryLoading, setIsHistoryLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { user } = useAuth();

  // Estados para Stakeholders
import useOrganizationUsersLookup from '@/hooks/useOrganizationUsersLookup'; // Importar o hook

// ... (outras importações)

const EditRiskPageContent = () => {
    // ... (outros hooks)
    const { user } = useAuth();

    // Estados para Stakeholders
    const [stakeholders, setStakeholders] = useState<UserLookup[]>([]);
    const [isLoadingStakeholders, setIsLoadingStakeholders] = useState(true);
    const [stakeholdersError, setStakeholdersError] = useState<string | null>(null);
    const { users: organizationUsers, isLoading: isLoadingOrgUsers, fetchUsers: fetchOrganizationUsers } = useOrganizationUsersLookup();
    const [selectedUserToAddAsStakeholder, setSelectedUserToAddAsStakeholder] = useState<string>('');
    const [isSubmittingStakeholder, setIsSubmittingStakeholder] = useState(false);
    const [isRemovingStakeholderId, setIsRemovingStakeholderId] = useState<string | null>(null);


    const fetchRiskData = useCallback(async () => {
    if (riskId && typeof riskId === 'string') {
      setIsLoading(true);
      setError(null);
      try {
        const riskResponse = await apiClient.get<Risk>(`/api/v1/risks/${riskId}`); // Usar tipo Risk importado
        setInitialData(riskResponse.data);
      } catch (err: any) {
        console.error("Erro ao buscar dados do risco:", err);
        setError(err.response?.data?.error || err.message || "Falha ao buscar dados do risco.");
      } finally {
        setIsLoading(false);
      }
    } else if (riskId) { // Se riskId existe mas não é string (ex: string[])
        setError("ID do Risco inválido.");
        setIsLoading(false);
    }
    // Se riskId for undefined, isLoading permanecerá true até router.isReady e riskId serem definidos
  }, [riskId]);

  const fetchApprovalHistory = useCallback(async () => {
    if (riskId && typeof riskId === 'string') {
      setIsHistoryLoading(true);
      try {
        const historyResponse = await apiClient.get<ApprovalWorkflow[]>(`/api/v1/risks/${riskId}/approval-history`);
        setApprovalHistory(historyResponse.data || []);
      } catch (err: any) {
        console.error("Erro ao buscar histórico de aprovação:", err);
      } finally {
        setIsHistoryLoading(false);
      }
    }
  }, [riskId]);

  const fetchStakeholders = useCallback(async () => {
    if (riskId && typeof riskId === 'string') {
      setIsLoadingStakeholders(true);
      setStakeholdersError(null);
      try {
        const response = await apiClient.get<UserLookup[]>(`/api/v1/risks/${riskId}/stakeholders`);
        setStakeholders(response.data || []);
      } catch (err: any) {
        console.error("Erro ao buscar stakeholders:", err);
        setStakeholdersError(t('stakeholders.error_loading_stakeholders', { ns: 'risks' }));
      } finally {
        setIsLoadingStakeholders(false);
      }
    }
  }, [riskId, t]);

  // A função fetchOrganizationUsers foi removida e substituída pelo hook useOrganizationUsersLookup
  // A chamada ao hook já está no topo do componente.

  useEffect(() => {
    if (router.isReady && riskId) {
      fetchRiskData();
      fetchApprovalHistory();
      fetchStakeholders();
      fetchOrganizationUsers(); // Carregar usuários para o select de adicionar stakeholder
    } else if (router.isReady && !riskId) {
      setError("ID do Risco não fornecido na URL.");
      setIsLoading(false);
      setIsHistoryLoading(false);
      setIsLoadingStakeholders(false);
    }
  }, [riskId, router.isReady, fetchRiskData, fetchApprovalHistory, fetchStakeholders, fetchOrganizationUsers]);

  const handleAddStakeholder = async () => {
    if (!selectedUserToAddAsStakeholder || !riskId) {
      notify.warn(t('stakeholders.warn_select_user', { ns: 'risks'}));
      return;
    }
    setIsSubmittingStakeholder(true);
    try {
      await apiClient.post(`/api/v1/risks/${riskId}/stakeholders`, { user_id: selectedUserToAddAsStakeholder });
      notify.success(t('stakeholders.success_stakeholder_added', { ns: 'risks'}));
      setSelectedUserToAddAsStakeholder('');
      fetchStakeholders(); // Re-fetch a lista de stakeholders
    } catch (err: any) {
      notify.error(t('stakeholders.error_adding_stakeholder', { message: err.response?.data?.error || t('common:unknown_error'), ns: 'risks' }));
    } finally {
      setIsSubmittingStakeholder(false);
    }
  };

  const handleRemoveStakeholder = async (userIdToRemove: string) => {
    if (!riskId) return;
    if (window.confirm(t('stakeholders.confirm_remove_stakeholder', { ns: 'risks'}))) {
      setIsRemovingStakeholderId(userIdToRemove);
      try {
        await apiClient.delete(`/api/v1/risks/${riskId}/stakeholders/${userIdToRemove}`);
        notify.success(t('stakeholders.success_stakeholder_removed', { ns: 'risks'}));
        fetchStakeholders(); // Re-fetch
      } catch (err: any) {
        notify.error(t('stakeholders.error_removing_stakeholder', { message: err.response?.data?.error || t('common:unknown_error'), ns: 'risks' }));
      } finally {
        setIsRemovingStakeholderId(null);
      }
    }
  };

  const handleSuccess = () => {
    // Usar notifier para uma mensagem mais elegante
    notify.success(t('edit_risk.success_updated_risk', { ns: 'risks' }));
    router.push('/admin/risks');
  };

  if (isLoading && !initialData && !error) {
    return <AdminLayout title={t('common:loading_ellipsis')}><div className="text-center p-10">{t('common:loading_data', { item_name: t('common_item_names.risk', { ns: 'common'})})}</div></AdminLayout>;
  }

  if (error) {
    return <AdminLayout title={t('common:error_loading_title')}><div className="text-center p-10 text-red-500">{t('common:error_message_text', { error_details: error })}</div></AdminLayout>;
  }

  if (!initialData && !isLoading) {
     return <AdminLayout title={t('common:item_not_found_title', { item_name: t('common_item_names.risk', { ns: 'common'})})}><div className="text-center p-10">{t('common:item_not_found_or_invalid_id', { item_name: t('common_item_names.risk', { ns: 'common'})})}</div></AdminLayout>;
  }


  return (
    <AdminLayout title={t('edit_risk.page_title_prefix', { ns: 'risks' }) + ` - ${initialData?.title || ''}`}>
      <Head>
        <title>{t('edit_risk.browser_title_prefix', { ns: 'risks', risk_title: initialData?.title || '' })} - {t('common:app_name')}</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            {t('edit_risk.header_prefix', { ns: 'risks' })}: <span className="text-brand-primary dark:text-brand-primary">{initialData?.title}</span>
          </h1>
          <Link href="/admin/risks" legacyBehavior>
            <a className="text-brand-primary hover:text-brand-primary/80 dark:hover:text-brand-primary/70 transition-colors">
              &larr; {t('common:back_to_list_link', { list_name: t('common_item_names.risks_plural', { ns: 'common'})})}
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
                impact: initialData.impact as RiskImpact,
                probability: initialData.probability as RiskProbability,
                status: initialData.status as RiskStatus,
                owner_id: initialData.owner_id,
              }}
              isEditing={true}
              onSubmitSuccess={handleSuccess}
            />
          )}
        </div>

        {/* Seção de Stakeholders */}
        <div className="mt-10">
          <h2 className="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white mb-4">
            {t('stakeholders.section_title', { ns: 'risks' })}
          </h2>
          <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg p-6 md:p-8">
            <div className="mb-6">
              <label htmlFor="add-stakeholder-select" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {t('stakeholders.add_new_label', { ns: 'risks' })}
              </label>
              <div className="mt-1 flex items-center space-x-2">
                <select
                  id="add-stakeholder-select"
                  value={selectedUserToAddAsStakeholder}
                  onChange={(e) => setSelectedUserToAddAsStakeholder(e.target.value)}
                  className="block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white sm:text-sm p-2 flex-grow"
                  disabled={isSubmittingStakeholder || organizationUsers.length === 0}
                >
                  <option value="">{t('stakeholders.select_user_placeholder', { ns: 'risks' })}</option>
                  {organizationUsers.map(orgUser => (
                    <option key={orgUser.id} value={orgUser.id}>{orgUser.name} ({orgUser.email})</option>
                  ))}
                </select>
                <button
                  type="button"
                  onClick={handleAddStakeholder}
                  disabled={!selectedUserToAddAsStakeholder || isSubmittingStakeholder}
                  className="px-4 py-2 text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 rounded-md shadow-sm disabled:opacity-50 flex items-center"
                >
                  {isSubmittingStakeholder && <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>}
                  {t('stakeholders.add_button', { ns: 'risks' })}
                </button>
              </div>
            </div>

            {isLoadingStakeholders && <p className="text-gray-500 dark:text-gray-400">{t('stakeholders.loading_stakeholders', { ns: 'risks' })}</p>}
            {stakeholdersError && <p className="text-red-500 dark:text-red-400">{stakeholdersError}</p>}
            {!isLoadingStakeholders && !stakeholdersError && stakeholders.length === 0 && (
              <p className="text-gray-500 dark:text-gray-400">{t('stakeholders.no_stakeholders_found', { ns: 'risks' })}</p>
            )}
            {!isLoadingStakeholders && !stakeholdersError && stakeholders.length > 0 && (
              <ul role="list" className="divide-y divide-gray-200 dark:divide-gray-700">
                {stakeholders.map((stakeholder) => (
                  <li key={stakeholder.id} className="py-3 flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-gray-900 dark:text-white">{stakeholder.name}</p>
                      {stakeholder.email && <p className="text-xs text-gray-500 dark:text-gray-400">{stakeholder.email}</p>}
                    </div>
                    <button
                      type="button"
                      onClick={() => handleRemoveStakeholder(stakeholder.id)}
                      disabled={isRemovingStakeholderId === stakeholder.id}
                      className="ml-4 px-3 py-1 text-xs font-medium text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 border border-red-300 dark:border-red-500 rounded-md hover:bg-red-50 dark:hover:bg-red-700/30 disabled:opacity-50 flex items-center"
                    >
                      {isRemovingStakeholderId === stakeholder.id && <svg className="animate-spin -ml-1 mr-2 h-3 w-3" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>}
                      {t('stakeholders.remove_button', { ns: 'risks' })}
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>

        {/* Seção de Histórico de Aprovação */}
        <div className="mt-10">
            <h2 className="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white mb-4">
                {t('approval_history.section_title', { ns: 'risks' })}
            </h2>
            {isHistoryLoading && <p className="text-gray-500 dark:text-gray-400">{t('approval_history.loading_history', { ns: 'risks' })}</p>}
            {!isHistoryLoading && approvalHistory.length === 0 && (
                <p className="text-gray-500 dark:text-gray-400">{t('approval_history.no_history_found', { ns: 'risks' })}</p>
            )}
            {!isHistoryLoading && approvalHistory.length > 0 && (
                <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg overflow-hidden">
                    <ul role="list" className="divide-y divide-gray-200 dark:divide-gray-700">
                        {approvalHistory.map((wf) => (
                            <li key={wf.id} className="px-4 py-4 sm:px-6">
                                <div className="flex items-center justify-between">
                                    <p className="text-sm font-medium text-brand-primary dark:text-brand-primary truncate">
                                        {t('approval_history.decision_label', { ns: 'risks' })}: <span className={`font-bold ${
                                            wf.status === 'aprovado' ? 'text-green-600 dark:text-green-400' :
                                            wf.status === 'rejeitado' ? 'text-red-600 dark:text-red-400' :
                                            'text-yellow-600 dark:text-yellow-400'
                                        }`}>{t(`approval_history.status_${wf.status}`, { ns: 'risks', defaultValue: wf.status.toUpperCase() })}</span>
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
                                            {t('approval_history.requester_label', { ns: 'risks' })}: {wf.requester?.name || wf.requester_id.substring(0,8)}
                                        </p>
                                        <p className="mt-2 flex items-center text-sm text-gray-500 dark:text-gray-400 sm:mt-0 sm:ml-6">
                                            <svg className="flex-shrink-0 mr-1.5 h-5 w-5 text-gray-400 dark:text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                                                <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
                                            </svg>
                                            {t('approval_history.approver_label', { ns: 'risks' })}: {wf.approver?.name || wf.approver_id.substring(0,8)}
                                        </p>
                                    </div>
                                </div>
                                {wf.comments && (
                                    <div className="mt-2 text-sm text-gray-700 dark:text-gray-300">
                                        <p>{t('approval_history.comments_label', { ns: 'risks' })}: <span className="italic">{wf.comments}</span></p>
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
