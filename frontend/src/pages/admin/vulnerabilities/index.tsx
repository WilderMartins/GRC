import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useEffect, useState, useCallback } from 'react';
import apiClient from '@/lib/axios';
import PaginationControls from '@/components/common/PaginationControls';
import { useNotifier } from '@/hooks/useNotifier';
import { useDebounce } from '@/hooks/useDebounce';
import {
    Vulnerability,
    VulnerabilitySeverity,
    VulnerabilityStatus,
    PaginatedResponse,
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
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'vulnerabilities'])),
  },
});

const VulnerabilitiesPageContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['vulnerabilities', 'common']);
  const notify = useNotifier();
  const [vulnerabilities, setVulnerabilities] = useState<Vulnerability[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalPages, setTotalPages] = useState(0);
  const [totalItems, setTotalItems] = useState(0);

  const [filterSeverity, setFilterSeverity] = useState<VulnerabilitySeverity>("");
  const [filterStatus, setFilterStatus] = useState<VulnerabilityStatus>("");
  const [searchAssetInput, setSearchAssetInput] = useState<string>("");
  const debouncedSearchAsset = useDebounce(searchAssetInput, 500);

  const [sortBy, setSortBy] = useState<string>('created_at');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');

  const fetchVulnerabilities = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const params: any = {
        page: currentPage,
        page_size: pageSize,
        sort_by: sortBy,
        order: sortOrder,
      };
      if (filterSeverity) params.severity = filterSeverity;
      if (filterStatus) params.status = filterStatus;
      if (debouncedSearchAsset) params.asset_affected_like = debouncedSearchAsset;

      const response = await apiClient.get<PaginatedResponse<Vulnerability>>('/vulnerabilities', { params });
      setVulnerabilities(response.data.items || []);
      setTotalItems(response.data.total_items);
      setTotalPages(response.data.total_pages);
    } catch (err: any) {
      console.error("Erro ao buscar vulnerabilidades:", err);
      const apiError = err.response?.data?.error || t('common:unknown_error');
      setError(t('list.error_loading_vulnerabilities', { message: apiError }));
      setVulnerabilities([]);
      setTotalItems(0);
      setTotalPages(0);
    } finally {
      setIsLoading(false);
    }
  }, [currentPage, pageSize, sortBy, sortOrder, filterSeverity, filterStatus, debouncedSearchAsset, t]);

  useEffect(() => {
    fetchVulnerabilities();
  }, [fetchVulnerabilities]);

  useEffect(() => {
    if (currentPage !== 1) {
        setCurrentPage(1);
    }
  }, [filterSeverity, filterStatus, debouncedSearchAsset, sortBy, sortOrder]);


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
    setFilterSeverity("");
    setFilterStatus("");
    setSearchAssetInput("");
    setSortBy('created_at');
    setSortOrder('desc');
    if (currentPage !== 1) setCurrentPage(1);
    else fetchVulnerabilities(); // Forçar re-fetch se já estava na página 1
  };

  const handleDeleteVulnerability = async (vulnId: string, vulnTitle: string) => {
    if (window.confirm(t('list.confirm_delete_message', { vulnTitle }))) {
      setIsLoading(true);
      try {
        await apiClient.delete(`/vulnerabilities/${vulnId}`);
        notify.success(t('list.delete_success_message', { vulnTitle }));
        if (vulnerabilities.length === 1 && currentPage > 1) {
            setCurrentPage(currentPage - 1);
        } else {
            fetchVulnerabilities();
        }
      } catch (err: any) {
        console.error("Erro ao deletar vulnerabilidade:", err);
        notify.error(t('list.delete_error_message', { message: err.response?.data?.error || t('common:unknown_error') }));
         setIsLoading(false); // Reset loading on error if fetchVulnerabilities is not called
      }
      // setIsLoading(false) é tratado pelo fetchVulnerabilities no sucesso
    }
  };

  const TableHeader: React.FC<{ field: string; labelKey: string }> = ({ field, labelKey }) => (
    <th scope="col" className="py-3.5 px-3 text-left text-sm font-semibold text-gray-900 dark:text-white cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600"
        onClick={() => handleSort(field)}>
      {t(labelKey)}
      {sortBy === field && (sortOrder === 'asc' ? ' ▲' : ' ▼')}
    </th>
  );

  return (
    <AdminLayout title={t('list.page_title')}>
      <Head>
        <title>{t('list.page_title')} - {t('common:app_name')}</title>
      </Head>

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex flex-col sm:flex-row justify-between items-center mb-6 gap-3">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            {t('list.header')}
          </h1>
          <Link href="/admin/vulnerabilities/new" legacyBehavior>
            <a className="inline-flex items-center justify-center rounded-md border border-transparent bg-brand-primary px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 dark:focus:ring-offset-gray-800 transition-colors">
              {t('list.add_new_button')}
            </a>
          </Link>
        </div>

        <div className="my-6 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg shadow">
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4 items-end">
            <div>
              <label htmlFor="filterSeverity" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('list.filter_severity_label')}</label>
              <select id="filterSeverity" value={filterSeverity} onChange={(e) => setFilterSeverity(e.target.value as VulnerabilitySeverity)}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                <option value="">{t('common:all_option')}</option>
                <option value="Crítico">{t('severity_options.Critical', { ns: 'vulnerabilities' })}</option>
                <option value="Alto">{t('severity_options.High', { ns: 'vulnerabilities' })}</option>
                <option value="Médio">{t('severity_options.Medium', { ns: 'vulnerabilities' })}</option>
                <option value="Baixo">{t('severity_options.Low', { ns: 'vulnerabilities' })}</option>
              </select>
            </div>
            <div>
              <label htmlFor="filterStatus" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('list.filter_status_label')}</label>
              <select id="filterStatus" value={filterStatus} onChange={(e) => setFilterStatus(e.target.value as VulnerabilityStatus)}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                <option value="">{t('common:all_option')}</option>
                <option value="descoberta">{t('status_options.descoberta', { ns: 'vulnerabilities' })}</option>
                <option value="em_correcao">{t('status_options.em_correcao', { ns: 'vulnerabilities' })}</option>
                <option value="corrigida">{t('status_options.corrigida', { ns: 'vulnerabilities' })}</option>
              </select>
            </div>
            <div>
              <label htmlFor="searchAssetInput" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('list.filter_asset_label')}</label>
              <input type="text" id="searchAssetInput" value={searchAssetInput} onChange={(e) => setSearchAssetInput(e.target.value)}
                     placeholder={t('list.filter_asset_placeholder')}
                     className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md"/>
            </div>
            <div>
              <button onClick={clearFilters}
                      className="w-full inline-flex items-center justify-center rounded-md border border-gray-300 dark:border-gray-500 bg-white dark:bg-gray-600 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-100 shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2">
                {t('list.clear_filters_button')}
              </button>
            </div>
          </div>
        </div>

        {isLoading && <p className="text-center text-gray-500 dark:text-gray-400 py-4">{t('list.loading_vulnerabilities')}</p>}
        {error && <p className="text-center text-red-500 py-4">{error}</p>}

        {!isLoading && !error && vulnerabilities.length === 0 && (
            <div className="text-center py-10">
                <p className="text-gray-500 dark:text-gray-400">{t('list.no_vulnerabilities_found')}</p>
            </div>
        )}

        {!isLoading && !error && vulnerabilities.length > 0 && (
          <>
            <div className="mt-8 flow-root">
              <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                  <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                    <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                      <thead className="bg-gray-50 dark:bg-gray-700">
                        <tr>
                          <TableHeader field="title" labelKey="list.table_header_title" />
                          <TableHeader field="cve_id" labelKey="list.table_header_cve_id" />
                          <TableHeader field="severity" labelKey="list.table_header_severity" />
                          <TableHeader field="status" labelKey="list.table_header_status" />
                          <TableHeader field="asset_affected" labelKey="list.table_header_asset" />
                          <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6"><span className="sr-only">{t('list.table_header_actions')}</span></th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                        {vulnerabilities.map((vuln) => (
                          <tr key={vuln.id}>
                            <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">{vuln.title}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{vuln.cve_id || '-'}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{t(`severity_options.${vuln.severity}`, {ns: 'vulnerabilities', defaultValue: vuln.severity})}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">
                               <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                    vuln.status === 'descoberta' ? 'bg-orange-100 text-orange-800 dark:bg-orange-700 dark:text-orange-100' :
                                    vuln.status === 'em_correcao' ? 'bg-blue-100 text-blue-800 dark:bg-blue-700 dark:text-blue-100' :
                                    vuln.status === 'corrigida' ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' :
                                    'bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-200'
                                }`}>
                                    {t(`status_options.${vuln.status}`, {ns: 'vulnerabilities', defaultValue: vuln.status})}
                                </span>
                            </td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{vuln.asset_affected}</td>
                            <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6 space-x-2">
                              <Link href={`/admin/vulnerabilities/edit/${vuln.id}`} legacyBehavior><a className="text-brand-primary hover:text-brand-primary/80 dark:text-brand-primary dark:hover:text-brand-primary/80 transition-colors">{t('list.action_edit')}</a></Link>
                              <button
                                onClick={() => handleDeleteVulnerability(vuln.id, vuln.title)}
                                className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-200 transition-colors"
                                disabled={isLoading}
                              >
                                {t('list.action_delete')}
                              </button>
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
    </AdminLayout>
  );
};

export default WithAuth(VulnerabilitiesPageContent);
