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
import { useDebounce } from '@/hooks/useDebounce';
import {
    Risk,
    RiskOwner, // Mantendo RiskOwner aqui se ele tiver campos além de UserLookup, senão usar UserLookup
    ApprovalWorkflow,
    UserLookup,
    PaginatedResponse,
    RiskStatus,
    RiskImpact,
    RiskProbability,
    RiskCategory,
    SortOrder
} from '@/types';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';

type Props = {
  // Props from getStaticProps
}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'risks', 'auth'])), // Incluir 'auth' se necessário para o user.name
  },
});

const RisksPageContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['risks', 'common']);
  const notify = useNotifier();
  const { user } = useAuth();

  const [risks, setRisks] = useState<Risk[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalPages, setTotalPages] = useState(0);
  const [totalItems, setTotalItems] = useState(0);

  const [filterCategory, setFilterCategory] = useState<RiskCategory>("");
  const [filterImpact, setFilterImpact] = useState<RiskImpact>("");
  const [filterProbability, setFilterProbability] = useState<RiskProbability>("");
  const [filterStatus, setFilterStatus] = useState<RiskStatus>("");
  const [filterOwnerId, setFilterOwnerId] = useState<string>("");
  const [searchTermTitle, setSearchTermTitle] = useState<string>("");
  const debouncedSearchTitle = useDebounce(searchTermTitle, 500);
  const [ownersForFilter, setOwnersForFilter] = useState<UserLookup[]>([]);
  const [isLoadingOwners, setIsLoadingOwners] = useState(false);

  const [sortBy, setSortBy] = useState<string>('created_at');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');

  const [showDecisionModal, setShowDecisionModal] = useState(false);
  const [selectedRiskForDecision, setSelectedRiskForDecision] = useState<Risk | null>(null);
  const [pendingApprovalWorkflowId, setPendingApprovalWorkflowId] = useState<string | null>(null);
  const [showUploadModal, setShowUploadModal] = useState(false);

  useEffect(() => {
    const fetchOwners = async () => {
      if (!user) return;
      setIsLoadingOwners(true);
      try {
        const response = await apiClient.get<UserLookup[]>('/users/organization-lookup');
        setOwnersForFilter(response.data || []);
      } catch (err) {
        console.error("Erro ao buscar proprietários para filtro:", err);
        notify.error(t('common:error_loading_list', {list_name: t('list.filter_owner_label')}));
      } finally {
        setIsLoadingOwners(false);
      }
    };
    fetchOwners();
  }, [user, notify, t]);

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

      const response = await apiClient.get<PaginatedResponse<Risk>>('/risks', { params });

      const risksData = response.data.items || [];
      const processedRisks = await Promise.all(
        risksData.map(async (risk) => {
          if (user && risk.owner_id === user.id) {
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
    } catch (err: any) {
      console.error("Erro ao buscar riscos:", err);
      const apiError = err.response?.data?.error || t('common:unknown_error');
      setError(t('list.error_loading_risks', {message: apiError}));
      setRisks([]);
      setTotalItems(0);
      setTotalPages(0);
    } finally {
      setIsLoading(false);
    }
  }, [currentPage, pageSize, sortBy, sortOrder, filterCategory, filterImpact, filterProbability, filterStatus, filterOwnerId, debouncedSearchTitle, user, t]);

  useEffect(() => {
    fetchRisks();
  }, [fetchRisks]);

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
    else fetchRisks(); // Forçar re-fetch se já estiver na página 1
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
        notify.info(t('list.no_pending_approval_message'));
        fetchRisks();
      }
    } catch (err) {
      notify.error(t('list.error_checking_approval_status'));
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
  };

  const handleSubmitForAcceptance = async (riskId: string, riskTitle: string) => {
    if (window.confirm(t('list.submit_acceptance_confirm_message', { riskTitle }))) {
        try {
            await apiClient.post(`/risks/${riskId}/submit-acceptance`);
            notify.success(t('list.submit_acceptance_success_message', { riskTitle }));
            fetchRisks();
        } catch (err: any) {
            notify.error(t('list.submit_acceptance_error_message', { message: err.response?.data?.error || t('common:unknown_error') }));
        }
    }
  };

  const handleDeleteRisk = async (riskId: string, riskTitle: string) => {
    if (window.confirm(t('list.confirm_delete_message', { riskTitle }))) {
      setIsLoading(true);
      try {
        await apiClient.delete(`/risks/${riskId}`);
        notify.success(t('list.delete_success_message', { riskTitle }));
        if (risks.length === 1 && currentPage > 1) {
            setCurrentPage(currentPage - 1);
        } else {
            fetchRisks();
        }
      } catch (err: any) {
        notify.error(t('list.delete_error_message', { message: err.response?.data?.error || t('common:unknown_error') }));
      } finally {
         // fetchRisks() set isLoading to false
      }
    }
  };

  const TableHeader: React.FC<{ field: string; labelKey: string }> = ({ field, labelKey }) => (
    <th scope="col" className="py-3.5 px-3 text-left text-sm font-semibold text-gray-900 dark:text-white cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600 whitespace-nowrap"
        onClick={() => handleSort(field)}>
      {t(labelKey)}
      {sortBy === field && (sortOrder === 'asc' ? ' ▲' : ' ▼')}
    </th>
  );

  return (
    <AdminLayout title={t('list.page_title')}>
      <Head><title>{t('list.page_title')} - {t('common:app_name')}</title></Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex flex-col sm:flex-row justify-between items-center mb-6 gap-3">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">{t('list.header')}</h1>
          <div className="flex space-x-3">
            <button onClick={() => setShowUploadModal(true)}
              className="inline-flex items-center justify-center rounded-md border border-transparent bg-green-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800">
              {t('list.import_csv_button')}
            </button>
            <Link href="/admin/risks/new" legacyBehavior>
              <a className="inline-flex items-center justify-center rounded-md border border-transparent bg-brand-primary px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 dark:focus:ring-offset-gray-800 transition-colors">
                {t('list.add_new_risk_button')}
              </a>
            </Link>
          </div>
        </div>

        <div className="my-6 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg shadow">
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7 gap-4 items-end">
            <div>
              <label htmlFor="searchTermTitle" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('list.filter_title_label')}</label>
              <input type="text" id="searchTermTitle" value={searchTermTitle} onChange={(e) => setSearchTermTitle(e.target.value)}
                     placeholder={t('list.filter_title_placeholder')}
                     className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md"/>
            </div>
            <div>
              <label htmlFor="filterCategory" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('list.filter_category_label')}</label>
              <select id="filterCategory" value={filterCategory} onChange={(e) => setFilterCategory(e.target.value as RiskCategory)}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                <option value="">{t('list.filter_all_option')}</option>
                <option value="tecnologico">{t('form.option_category_tech', { ns: 'risks' })}</option>
                <option value="operacional">{t('form.option_category_op', { ns: 'risks' })}</option>
                <option value="legal">{t('form.option_category_legal', { ns: 'risks' })}</option>
              </select>
            </div>
             <div>
              <label htmlFor="filterImpact" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('list.filter_impact_label')}</label>
              <select id="filterImpact" value={filterImpact} onChange={(e) => setFilterImpact(e.target.value as RiskImpact)}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                <option value="">{t('list.filter_all_option')}</option>
                <option value="Crítico">{t('form.option_impact_critical', { ns: 'risks' })}</option>
                <option value="Alto">{t('form.option_impact_high', { ns: 'risks' })}</option>
                <option value="Médio">{t('form.option_impact_medium', { ns: 'risks' })}</option>
                <option value="Baixo">{t('form.option_impact_low', { ns: 'risks' })}</option>
              </select>
            </div>
            <div>
              <label htmlFor="filterProbability" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('list.filter_probability_label')}</label>
              <select id="filterProbability" value={filterProbability} onChange={(e) => setFilterProbability(e.target.value as RiskProbability)}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                <option value="">{t('list.filter_all_option')}</option>
                <option value="Crítico">{t('form.option_probability_critical', { ns: 'risks' })}</option>
                <option value="Alto">{t('form.option_probability_high', { ns: 'risks' })}</option>
                <option value="Médio">{t('form.option_probability_medium', { ns: 'risks' })}</option>
                <option value="Baixo">{t('form.option_probability_low', { ns: 'risks' })}</option>
              </select>
            </div>
            <div>
              <label htmlFor="filterStatus" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('list.filter_status_label')}</label>
              <select id="filterStatus" value={filterStatus} onChange={(e) => setFilterStatus(e.target.value as RiskStatus)}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                <option value="">{t('list.filter_all_option')}</option>
                <option value="aberto">{t('form.option_status_open', { ns: 'risks' })}</option>
                <option value="em_andamento">{t('form.option_status_in_progress', { ns: 'risks' })}</option>
                <option value="mitigado">{t('form.option_status_mitigated', { ns: 'risks' })}</option>
                <option value="aceito">{t('form.option_status_accepted', { ns: 'risks' })}</option>
              </select>
            </div>
            <div>
              <label htmlFor="filterOwnerId" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('list.filter_owner_label')}</label>
              <select id="filterOwnerId" value={filterOwnerId} onChange={(e) => setFilterOwnerId(e.target.value)}
                      disabled={isLoadingOwners}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md disabled:opacity-50">
                <option value="">{t('list.filter_all_option')}</option>
                {isLoadingOwners && <option value="" disabled>{t('common:loading_options')}</option>}
                {ownersForFilter.map(owner => (
                  <option key={owner.id} value={owner.id}>{owner.name}</option>
                ))}
              </select>
            </div>
            <div>
              <button onClick={clearFilters}
                      className="w-full inline-flex items-center justify-center rounded-md border border-gray-300 dark:border-gray-500 bg-white dark:bg-gray-600 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-100 shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2">
                {t('list.clear_filters_button')}
              </button>
            </div>
          </div>
        </div>

        {isLoading && <p className="text-center text-gray-500 dark:text-gray-400 py-4">{t('list.loading_risks')}</p>}
        {error && <p className="text-center text-red-500 py-4">{error}</p>} {/* Error já é traduzido no fetchRisks */}

        {!isLoading && !error && risks.length === 0 && (
             <div className="text-center py-10">
                <p className="text-gray-500 dark:text-gray-400">{t('list.no_risks_found')}</p>
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
                          <TableHeader field="title" labelKey="list.table_header_title" />
                          <TableHeader field="category" labelKey="list.table_header_category" />
                          <TableHeader field="impact" labelKey="list.table_header_impact" />
                          <TableHeader field="probability" labelKey="list.table_header_probability" />
                          <TableHeader field="status" labelKey="list.table_header_status" />
                          <TableHeader field="owner.name" labelKey="list.table_header_owner" />
                          <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6"><span className="sr-only">{t('list.table_header_actions')}</span></th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                        {risks.map((risk) => (
                          <tr key={risk.id}>
                            <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">
                              {risk.title}
                              {risk.hasPendingApproval && (
                                <span className="ml-2 px-2 py-0.5 inline-flex text-xs leading-5 font-semibold rounded-full bg-yellow-100 text-yellow-800 dark:bg-yellow-700 dark:text-yellow-100 animate-pulse" title={t('list.pending_approval_badge')}>
                                  {t('list.pending_approval_badge')}
                                </span>
                              )}
                              {risk.hasPendingApproval && user?.id === risk.owner_id && (
                                <button onClick={() => handleOpenDecisionModal(risk)}
                                  className="ml-2 px-2 py-0.5 text-xs bg-blue-500 text-white rounded-full hover:bg-blue-600"
                                  title={t('list.action_decide')}>
                                  {t('list.action_decide')}
                                </button>
                              )}
                            </td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{t(`form.option_category_${risk.category}`, {ns: 'risks', defaultValue: risk.category})}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{t(`form.option_impact_${risk.impact.toLowerCase()}`, {ns: 'risks', defaultValue: risk.impact})}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{t(`form.option_probability_${risk.probability.toLowerCase()}`, {ns: 'risks', defaultValue: risk.probability})}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">
                              <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                    risk.status === 'aberto' ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-700 dark:text-yellow-100' :
                                    risk.status === 'em_andamento' ? 'bg-blue-100 text-blue-800 dark:bg-blue-700 dark:text-blue-100' :
                                    risk.status === 'mitigado' ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' :
                                    risk.status === 'aceito' ? 'bg-purple-100 text-purple-800 dark:bg-purple-700 dark:text-purple-100' :
                                    'bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-200'
                                }`}>
                                    {t(`form.option_status_${risk.status.replace('_', '')}`, {ns: 'risks', defaultValue: risk.status})}
                                </span>
                            </td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{risk.owner?.name || risk.owner_id}</td>
                            <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6 space-x-2">
                              <Link href={`/admin/risks/edit/${risk.id}`} legacyBehavior><a className="text-brand-primary hover:text-brand-primary/80 dark:text-brand-primary dark:hover:text-brand-primary/80 transition-colors">{t('list.action_edit')}</a></Link>
                              <button onClick={() => handleDeleteRisk(risk.id, risk.title)}
                                className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-200 transition-colors"
                                disabled={isLoading}>
                                {t('list.action_delete')}
                              </button>
                              {(user?.role === 'admin' || user?.role === 'manager') && risk.status !== 'aceito' && risk.status !== 'mitigado' && !risk.hasPendingApproval && (
                                <button onClick={() => handleSubmitForAcceptance(risk.id, risk.title)}
                                  className="ml-2 text-green-600 hover:text-green-900 dark:text-green-400 dark:hover:text-green-200 transition-colors"
                                  disabled={isLoading}>
                                  {t('list.action_submit_for_acceptance')}
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
          clearFilters();
        }}
      />
    </AdminLayout>
  );
};

export default WithAuth(RisksPageContent);
